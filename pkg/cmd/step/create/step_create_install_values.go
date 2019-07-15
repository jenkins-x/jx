package create

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/cloud/gke/externaldns"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	createInstallValuesLong = templates.LongDesc(`
		Creates any missing cluster values into the cluster/values.yaml file 
`)

	createInstallValuesExample = templates.Examples(`
		# populate the cluster/values.yaml file
		jx step create install values
	
			`)
)

// StepCreateInstallValuesOptions contains the command line flags
type StepCreateInstallValuesOptions struct {
	opts.StepOptions

	Dir              string
	Namespace        string
	Provider         string
	IngressNamespace string
	IngressService   string
	ExternalIP       string
	LazyCreate       bool
	LazyCreateFlag   string
}

// StepCreateInstallValuesResults stores the generated results
type StepCreateInstallValuesResults struct {
	Pipeline    *pipelineapi.Pipeline
	Task        *pipelineapi.Task
	PipelineRun *pipelineapi.PipelineRun
}

// NewCmdStepCreateInstallValues Creates a new Command object
func NewCmdStepCreateInstallValues(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepCreateInstallValuesOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "install values",
		Short:   "Creates any missing cluster values into the cluster/values.yaml file ",
		Long:    createInstallValuesLong,
		Example: createInstallValuesExample,
		Aliases: []string{"version pullrequest"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Dir, "dir", "d", ".", "the directory to look for the values.yaml file")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "the namespace to install into. Defaults to $DEPLOY_NAMESPACE if not")

	cmd.Flags().StringVarP(&options.IngressNamespace, "ingress-namespace", "", opts.DefaultIngressNamesapce, "The namespace for the Ingress controller")
	cmd.Flags().StringVarP(&options.IngressService, "ingress-service", "", opts.DefaultIngressServiceName, "The name of the Ingress controller Service")
	cmd.Flags().StringVarP(&options.ExternalIP, "external-ip", "", "", "The external IP used to access ingress endpoints from outside the Kubernetes cluster. For bare metal on premise clusters this is often the IP of the Kubernetes master. For cloud installations this is often the external IP of the ingress LoadBalancer.")
	cmd.Flags().StringVarP(&options.Provider, "provider", "", "", "Cloud service providing the Kubernetes cluster.  Supported providers: "+cloud.KubernetesProviderOptions())
	cmd.Flags().StringVarP(&options.LazyCreateFlag, "lazy-create", "", "", fmt.Sprintf("Specify true/false as to whether to lazily create missing resources. If not specified it is enabled if Terraform is not specified in the %s file", config.RequirementsConfigFileName))
	return cmd
}

// Run implements this command
func (o *StepCreateInstallValuesOptions) Run() error {
	var err error
	if o.Dir == "" {
		o.Dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	clusterDir := filepath.Join(o.Dir, "cluster")
	err = os.MkdirAll(clusterDir, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create cluster dir: %s", clusterDir)
	}

	valuesFile := filepath.Join(clusterDir, helm.ValuesFileName)
	values, err := helm.LoadValuesFile(valuesFile)
	if err != nil {
		return errors.Wrapf(err, "failed to load helm values: %s", valuesFile)
	}

	values, err = o.defaultMissingValues(values)
	if err != nil {
		return errors.Wrapf(err, "failed to default helm values into: %s", valuesFile)
	}

	err = helm.SaveFile(valuesFile, values)
	if err != nil {
		return errors.Wrapf(err, "failed to save helm values: %s", valuesFile)
	}
	log.Logger().Infof("wrote %s\n", util.ColorInfo(valuesFile))
	return nil
}

func (o *StepCreateInstallValuesOptions) defaultMissingValues(values map[string]interface{}) (map[string]interface{}, error) {
	info := util.ColorInfo
	ns := o.Namespace
	if ns == "" {
		ns = os.Getenv("DEPLOY_NAMESPACE")
	}
	if ns != "" {
		if ns == "" {
			return values, fmt.Errorf("no default namespace found")
		}
	}
	requirements, requirementsFileName, err := config.LoadRequirementsConfig(o.Dir)
	if err != nil {
		return values, errors.Wrapf(err, "failed to load Jenkins X requirements")
	}

	o.LazyCreate, err = requirements.IsLazyCreateSecrets(o.LazyCreateFlag)
	if err != nil {
		return values, errors.Wrapf(err, "failed to see if lazy create flag is set %s", o.LazyCreateFlag)
	}

	if requirements.Cluster.Provider == "" {
		log.Logger().Warnf("No provider configured\n")
	}

	if requirements.Ingress.Domain == "" {
		requirements.Ingress.Domain, err = o.discoverIngressDomain()
		if err != nil {
			return values, errors.Wrapf(err, "failed to discover the Ingress domain")
		}
	}

	util.SetMapValueViaPath(values, "domain", requirements.Ingress.Domain)

	subDomain := "." + ns + "."

	// if we're using GKE and folks have provided a domain, i.e. we're  not using the Jenkins X default nip.io
	// then let's enable external dns automatically.
	if !strings.Contains(requirements.Ingress.Domain, "nip.io") && requirements.Cluster.Provider == cloud.GKE {
		log.Logger().Info("using a custom domain and GKE so enabling external dns, you can also now enable TLS")
		requirements.Ingress.ExternalDNS = true
		log.Logger().Infof("validating the external-dns secret in namespace %s\n", info(ns))

		kubeClient, err := o.KubeClient()
		if err != nil {
			return values, errors.Wrap(err, "creating kubernetes client")
		}

		serviceAccountName := gke.GcpServiceAccountSecretName(kube.DefaultExternalDNSReleaseName)

		err = kube.ValidateSecret(kubeClient, serviceAccountName, externaldns.ServiceAccountSecretKey, ns)

		if err != nil {
			if o.LazyCreate {

				log.Logger().Infof("attempting to lazily create the external-dns secret %s\n", info(ns))

				_, err = externaldns.CreateExternalDNSGCPServiceAccount(o.GCloud(), kubeClient, kube.DefaultExternalDNSReleaseName, ns, requirements.Cluster.ClusterName, requirements.Cluster.ProjectID)
				if err != nil {
					return values, errors.Wrap(err, "creating the ExternalDNS GCP Service Account")
				}
				// lets rerun the verify step to ensure its all sorted now
				err = kube.ValidateSecret(kubeClient, serviceAccountName, externaldns.ServiceAccountSecretKey, ns)
			}
		}
		if err != nil {
			return values, errors.Wrap(err, "validating external-dns secret")
		}

		// for external dns to work using dns we need to use `-` and not `.`
		subDomain = "-" + ns + "."

		err = o.GCloud().EnableAPIs(requirements.Cluster.ProjectID, "dns")
		if err != nil {
			return values, errors.Wrap(err, "unable to enable 'dns' api")
		}
	} else {
		log.Logger().Info("Disabling using external-dns as it currently only works on GKE and not nip.io domains")
		requirements.Ingress.ExternalDNS = false
	}
	// TLS uses cert-manager to ask LetsEncrypt for a signed certificate
	if requirements.Ingress.TLS.Enabled {
		if requirements.Cluster.Provider != cloud.GKE {
			return values, errors.New("TLS is currently only supported on Google Container Engine")
		}

		if strings.Contains(requirements.Ingress.Domain, "nip.io") {
			return values, errors.New("TLS is not supported with nip.io, you will need to use a domain you own")
		}
		_, err = mail.ParseAddress(requirements.Ingress.TLS.Email)
		if err != nil {
			return values, errors.Wrap(err, "You must provide a valid email address to enable TLS so you can receive notifications from LetsEncrypt about your certificates")
		}

		util.SetMapValueViaPath(values, "tls", true)
	}

	util.SetMapValueViaPath(values, "namespaceSubDomain", subDomain)

	projectID := util.GetMapValueAsStringViaPath(values, "projectID")
	if projectID == "" {
		util.SetMapValueViaPath(values, "projectID", requirements.Cluster.ProjectID)
	}

	requirements.SaveConfig(requirementsFileName)
	if err != nil {
		return values, nil
	}

	return values, nil
}

func (o *StepCreateInstallValuesOptions) discoverIngressDomain() (string, error) {
	client, err := o.KubeClient()
	if err != nil {
		return "", errors.Wrap(err, "getting the kubernetes client")
	}
	requirements, _, err := config.LoadRequirementsConfig(o.Dir)
	if err != nil {
		return "failed to load Jenkins X requirements", err
	}
	if requirements.Ingress.Domain != "" {
		return requirements.Ingress.Domain, nil
	}
	if o.Provider == "" {
		o.Provider = requirements.Cluster.Provider
		if o.Provider == "" {
			log.Logger().Warnf("No provider configured\n")
		}
	}
	domain, err := o.GetDomain(client, "",
		o.Provider,
		o.IngressNamespace,
		o.IngressService,
		o.ExternalIP)
	if err != nil {
		return "", errors.Wrapf(err, "getting a domain for ingress service %s/%s", o.IngressNamespace, o.IngressService)
	}
	if domain == "" {
		hasHost, err := o.waitForIngressControllerHost(client, o.IngressNamespace, o.IngressService)
		if err != nil {
			return domain, err
		}
		if hasHost {
			domain, err = o.GetDomain(client, "",
				o.Provider,
				o.IngressNamespace,
				o.IngressService,
				o.ExternalIP)
			if err != nil {
				return "", errors.Wrapf(err, "getting a domain for ingress service %s/%s", o.IngressNamespace, o.IngressService)
			}
		}
	}
	return domain, nil
}

func (o *StepCreateInstallValuesOptions) waitForIngressControllerHost(kubeClient kubernetes.Interface, ns, serviceName string) (bool, error) {
	loggedWait := false
	serviceInterface := kubeClient.CoreV1().Services(ns)

	if serviceName == "" || ns == "" {
		return false, nil
	}
	_, err := serviceInterface.Get(serviceName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	fn := func() (bool, error) {
		svc, err := serviceInterface.Get(serviceName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// lets get the ingress service status
		for _, lb := range svc.Status.LoadBalancer.Ingress {
			if lb.Hostname != "" || lb.IP != "" {
				return true, nil
			}
		}

		if !loggedWait {
			loggedWait = true
			log.Logger().Infof("waiting for external Host on the ingress service %s in namespace %s ...", serviceName, ns)
		}
		return false, nil
	}
	err = o.RetryUntilTrueOrTimeout(time.Minute*5, time.Second*3, fn)
	if err != nil {
		return false, err
	}
	return true, nil
}
