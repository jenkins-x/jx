package create

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/log"
	"k8s.io/client-go/kubernetes"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	createInstallValuesLong = templates.LongDesc(`
		Creates any mising cluster values into the cluster/values.yaml file 
`)

	createInstallValuesExample = templates.Examples(`
		# populate the cluster/values.yaml file
		jx step create install vales
	
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
		Short:   "Creates any mising cluster values into the cluster/values.yaml file ",
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
	ns := o.Namespace
	if ns == "" {
		ns = os.Getenv("DEPLOY_NAMESPACE")
	}
	if ns != "" {
		current := util.GetMapValueAsStringViaPath(values, "namespaceSubDomain")
		if current == "" {
			subDomain := "." + ns + "."
			util.SetMapValueViaPath(values, "namespaceSubDomain", subDomain)
		}
	}

	domain := util.GetMapValueAsStringViaPath(values, "domain")
	if domain == "" {
		domain, err := o.discoverIngressDomain(values)
		if err != nil {
			return values, errors.Wrapf(err, "failed to discover the Ingress domain")
		}
		if domain == "" {
			return values, fmt.Errorf("could not detect a domain. Pleae configure one at 'domain' in the init-values.yaml")
		}
		util.SetMapValueViaPath(values, "domain", domain)
	}
	return values, nil
}

func (o *StepCreateInstallValuesOptions) discoverIngressDomain(values map[string]interface{}) (string, error) {
	client, err := o.KubeClient()
	if err != nil {
		return "", errors.Wrap(err, "getting the kubernetes client")
	}
	if o.Provider == "" {
		// TODO lets see if the provider is in the config
		log.Logger().Warnf("No provider configured\n")
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
