package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/surveyutils"

	"github.com/jenkins-x/jx/pkg/kube/services"

	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/jenkins-x/jx/pkg/cloud/iks"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	rbacv1 "k8s.io/api/rbac/v1"

	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// InitOptions the options for running init
type InitOptions struct {
	CommonOptions
	Client clientset.Clientset
	Flags  InitFlags
}

// InitFlags the flags for running init
type InitFlags struct {
	Domain                     string
	Provider                   string
	Namespace                  string
	UserClusterRole            string
	TillerClusterRole          string
	IngressClusterRole         string
	TillerNamespace            string
	IngressNamespace           string
	IngressService             string
	IngressDeployment          string
	ExternalIP                 string
	DraftClient                bool
	HelmClient                 bool
	Helm3                      bool
	HelmBin                    string
	RecreateExistingDraftRepos bool
	NoTiller                   bool
	RemoteTiller               bool
	GlobalTiller               bool
	SkipIngress                bool
	SkipTiller                 bool
	OnPremise                  bool
	Http                       bool
	NoGitValidate              bool
}

const (
	optionUsername        = "username"
	optionNamespace       = "namespace"
	optionTillerNamespace = "tiller-namespace"

	// JenkinsBuildPackURL URL of Draft packs for Jenkins X
	JenkinsBuildPackURL = "https://github.com/jenkins-x/draft-packs.git"
	// INGRESS_SERVICE_NAME service name for ingress controller
	INGRESS_SERVICE_NAME = "jxing-nginx-ingress-controller"
	// DEFAULT_CHARTMUSEUM_URL default URL for Jenkins X ChartMuseum
	DEFAULT_CHARTMUSEUM_URL = "http://chartmuseum.jenkins-x.io"
)

var (
	initLong = templates.LongDesc(`
		This command initializes the connected Kubernetes cluster for Jenkins X platform installation
`)

	initExample = templates.Examples(`
		jx init
`)
)

// NewCmdInit creates a command object for the generic "init" action, which
// primes a Kubernetes cluster so it's ready for Jenkins X to be installed
func NewCmdInit(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &InitOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,

			Out: out,
			Err: errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Init Jenkins X",
		Long:    initLong,
		Example: initExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.Flags.Provider, "provider", "", "", "Cloud service providing the Kubernetes cluster.  Supported providers: "+KubernetesProviderOptions())
	cmd.Flags().StringVarP(&options.Flags.Namespace, optionNamespace, "", "jx", "The namespace the Jenkins X platform should be installed into")
	options.addInitFlags(cmd)
	return cmd
}

func (o *InitOptions) addInitFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.Flags.Domain, "domain", "", "", "Domain to expose ingress endpoints.  Example: jenkinsx.io")
	cmd.Flags().StringVarP(&o.Username, optionUsername, "", "", "The Kubernetes username used to initialise helm. Usually your email address for your Kubernetes account")
	cmd.Flags().StringVarP(&o.Flags.UserClusterRole, "user-cluster-role", "", "cluster-admin", "The cluster role for the current user to be able to administer helm")
	cmd.Flags().StringVarP(&o.Flags.TillerClusterRole, "tiller-cluster-role", "", "cluster-admin", "The cluster role for Helm's tiller")
	cmd.Flags().StringVarP(&o.Flags.TillerNamespace, optionTillerNamespace, "", "kube-system", "The namespace for the Tiller when using a global tiller")
	cmd.Flags().StringVarP(&o.Flags.IngressClusterRole, "ingress-cluster-role", "", "cluster-admin", "The cluster role for the Ingress controller")
	cmd.Flags().StringVarP(&o.Flags.IngressNamespace, "ingress-namespace", "", "kube-system", "The namespace for the Ingress controller")
	cmd.Flags().StringVarP(&o.Flags.IngressService, "ingress-service", "", INGRESS_SERVICE_NAME, "The name of the Ingress controller Service")
	cmd.Flags().StringVarP(&o.Flags.IngressDeployment, "ingress-deployment", "", INGRESS_SERVICE_NAME, "The name of the Ingress controller Deployment")
	cmd.Flags().StringVarP(&o.Flags.ExternalIP, "external-ip", "", "", "The external IP used to access ingress endpoints from outside the Kubernetes cluster. For bare metal on premise clusters this is often the IP of the Kubernetes master. For cloud installations this is often the external IP of the ingress LoadBalancer.")
	cmd.Flags().BoolVarP(&o.Flags.DraftClient, "draft-client-only", "", false, "Only install draft client")
	cmd.Flags().BoolVarP(&o.Flags.HelmClient, "helm-client-only", "", false, "Only install helm client")
	cmd.Flags().BoolVarP(&o.Flags.RecreateExistingDraftRepos, "recreate-existing-draft-repos", "", false, "Delete existing helm repos used by Jenkins X under ~/draft/packs")
	cmd.Flags().BoolVarP(&o.Flags.GlobalTiller, "global-tiller", "", true, "Whether or not to use a cluster global tiller")
	cmd.Flags().BoolVarP(&o.Flags.RemoteTiller, "remote-tiller", "", true, "If enabled and we are using tiller for helm then run tiller remotely in the kubernetes cluster. Otherwise we run the tiller process locally.")
	cmd.Flags().BoolVarP(&o.Flags.NoTiller, "no-tiller", "", false, "Whether to disable the use of tiller with helm. If disabled we use 'helm template' to generate the YAML from helm charts then we use 'kubectl apply' to install it to avoid using tiller completely.")
	cmd.Flags().BoolVarP(&o.Flags.SkipIngress, "skip-ingress", "", false, "Don't install an ingress controller")
	cmd.Flags().BoolVarP(&o.Flags.SkipTiller, "skip-setup-tiller", "", false, "Don't setup the Helm Tiller service - lets use whatever tiller is already setup for us.")
	cmd.Flags().BoolVarP(&o.Flags.Helm3, "helm3", "", false, "Use helm3 to install Jenkins X which does not use Tiller")
	cmd.Flags().BoolVarP(&o.Flags.OnPremise, "on-premise", "", false, "If installing on an on premise cluster then lets default the 'external-ip' to be the Kubernetes master IP address")
}

// Run performs initialization
func (o *InitOptions) Run() error {
	var err error
	if !o.Flags.RemoteTiller || o.Flags.NoTiller {
		o.Flags.HelmClient = true
		o.Flags.SkipTiller = true
		o.Flags.GlobalTiller = false
	}
	o.Flags.Provider, err = o.GetCloudProvider(o.Flags.Provider)
	if err != nil {
		return err
	}

	if !o.Flags.NoGitValidate {
		err = o.validateGit()
		if err != nil {
			return err
		}
	}

	err = o.enableClusterAdminRole()
	if err != nil {
		return err
	}

	// So a user doesn't need to specify ingress options if provider is ICP: we will use ICP's own ingress controller
	// and by default, the tiller namespace "jx"
	if o.Flags.Provider == ICP {
		o.configureForICP()
	}

	// Needs to be done early as is an ingress availablility is an indicator of cluster readyness
	if o.Flags.Provider == IKS {
		err = o.initIKSIngress()
		if err != nil {
			return err
		}
	}

	// helm init, this has been seen to fail intermittently on public clouds, so let's retry a couple of times
	err = o.retry(3, 2*time.Second, func() (err error) {
		err = o.initHelm()
		return
	})

	if err != nil {
		log.Fatalf("helm init failed: %v", err)
		return err
	}

	// draft init
	_, err = o.initBuildPacks()
	if err != nil {
		log.Fatalf("initialise build packs failed: %v", err)
		return err
	}

	// install ingress
	if !o.Flags.SkipIngress {
		err = o.initIngress()
		if err != nil {
			log.Fatalf("ingress init failed: %v", err)
			return err
		}
	}

	return nil
}

func (o *InitOptions) enableClusterAdminRole() error {
	client, err := o.KubeClient()
	if err != nil {
		return err
	}

	if o.Username == "" {
		o.Username, err = o.GetClusterUserName()
		if err != nil {
			return err
		}
	}
	if o.Username == "" {
		return util.MissingOption(optionUsername)
	}
	userFormatted := kube.ToValidName(o.Username)

	clusterRoleBindingName := kube.ToValidName(userFormatted + "-" + o.Flags.UserClusterRole + "-binding")

	clusterRoleBindingInterface := client.RbacV1().ClusterRoleBindings()
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "User",
				Name:     o.Username,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     o.Flags.UserClusterRole,
		},
	}

	return o.retry(3, 10*time.Second, func() (err error) {
		_, err = clusterRoleBindingInterface.Get(clusterRoleBindingName, metav1.GetOptions{})
		if err != nil {
			log.Infof("Trying to create ClusterRoleBinding %s for role: %s for user %s\n %v\n", clusterRoleBindingName, o.Flags.UserClusterRole, o.Username, err)

			//args := []string{"create", "clusterrolebinding", clusterRoleBindingName, "--clusterrole=" + role, "--user=" + user}

			_, err = clusterRoleBindingInterface.Create(clusterRoleBinding)
			if err == nil {
				log.Infof("Created ClusterRoleBinding %s\n", clusterRoleBindingName)
			}
		}
		return err
	})
}

func (o *InitOptions) initHelm() error {
	var err error

	if o.Flags.Helm3 {
		log.Infof("Using %s\n", util.ColorInfo("helm3"))
		o.Flags.SkipTiller = true
	} else {
		log.Infof("Using %s\n", util.ColorInfo("helm2"))
	}

	if !o.Flags.SkipTiller {
		log.Infof("Configuring %s\n", util.ColorInfo("tiller"))
		client, curNs, err := o.KubeClientAndNamespace()
		if err != nil {
			return err
		}

		serviceAccountName := "tiller"
		tillerNamespace := o.Flags.TillerNamespace

		if o.Flags.GlobalTiller {
			if tillerNamespace == "" {
				return util.MissingOption(optionTillerNamespace)
			}
		} else {
			ns := o.Flags.Namespace
			if ns == "" {
				ns = curNs
			}
			if ns == "" {
				return util.MissingOption(optionNamespace)
			}
			tillerNamespace = ns
		}

		err = o.ensureServiceAccount(tillerNamespace, serviceAccountName)
		if err != nil {
			return err
		}

		if o.Flags.GlobalTiller {
			clusterRoleBindingName := serviceAccountName
			role := o.Flags.TillerClusterRole

			err = o.ensureClusterRoleBinding(clusterRoleBindingName, role, tillerNamespace, serviceAccountName)
			if err != nil {
				return err
			}
		} else {
			// lets create a tiller service account
			roleName := "tiller-manager"
			roleBindingName := "tiller-binding"

			_, err = client.RbacV1().Roles(tillerNamespace).Get(roleName, metav1.GetOptions{})
			if err != nil {
				// lets create a Role for tiller
				role := &rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      roleName,
						Namespace: tillerNamespace,
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{"", "extensions", "apps"},
							Resources: []string{"*"},
							Verbs:     []string{"*"},
						},
					},
				}
				_, err = client.RbacV1().Roles(tillerNamespace).Create(role)
				if err != nil {
					return fmt.Errorf("Failed to create Role %s in namespace %s: %s", roleName, tillerNamespace, err)
				}
				log.Infof("Created Role %s in namespace %s\n", util.ColorInfo(roleName), util.ColorInfo(tillerNamespace))
			}
			_, err = client.RbacV1().RoleBindings(tillerNamespace).Get(roleBindingName, metav1.GetOptions{})
			if err != nil {
				// lets create a RoleBinding for tiller
				roleBinding := &rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      roleBindingName,
						Namespace: tillerNamespace,
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      serviceAccountName,
							Namespace: tillerNamespace,
						},
					},
					RoleRef: rbacv1.RoleRef{
						Kind:     "Role",
						Name:     roleName,
						APIGroup: "rbac.authorization.k8s.io",
					},
				}
				_, err = client.RbacV1().RoleBindings(tillerNamespace).Create(roleBinding)
				if err != nil {
					return fmt.Errorf("Failed to create RoleBinding %s in namespace %s: %s", roleName, tillerNamespace, err)
				}
				log.Infof("Created RoleBinding %s in namespace %s\n", util.ColorInfo(roleName), util.ColorInfo(tillerNamespace))
			}
		}

		running, err := kube.IsDeploymentRunning(client, "tiller-deploy", tillerNamespace)
		if running {
			log.Infof("Tiller Deployment is running in namespace %s\n", util.ColorInfo(tillerNamespace))
			return nil
		}
		if err == nil && !running {
			return fmt.Errorf("existing tiller deployment found but not running, please check the %s namespace and resolve any issues", tillerNamespace)
		}

		if !running {
			log.Infof("Initialising helm using ServiceAccount %s in namespace %s\n", util.ColorInfo(serviceAccountName), util.ColorInfo(tillerNamespace))

			err = o.Helm().Init(false, serviceAccountName, tillerNamespace, false)
			if err != nil {
				return err
			}
			err = kube.WaitForDeploymentToBeReady(client, "tiller-deploy", tillerNamespace, 10*time.Minute)
			if err != nil {
				return err
			}

			err = o.Helm().Init(false, serviceAccountName, tillerNamespace, true)
			if err != nil {
				return err
			}
		}

		log.Infof("Waiting for tiller-deploy to be ready in tiller namespace %s\n", tillerNamespace)
		err = kube.WaitForDeploymentToBeReady(client, "tiller-deploy", tillerNamespace, 10*time.Minute)
		if err != nil {
			return err
		}
	} else {
		log.Infof("Skipping %s\n", util.ColorInfo("tiller"))
	}

	if o.Flags.Helm3 {
		err = o.Helm().Init(false, "", "", false)
		if err != nil {
			return err
		}
	} else if o.Flags.HelmClient || o.Flags.SkipTiller {
		err = o.Helm().Init(true, "", "", false)
		if err != nil {
			return err
		}
	}

	err = o.Helm().AddRepo("jenkins-x", DEFAULT_CHARTMUSEUM_URL, "", "")
	if err != nil {
		return err
	}
	log.Success("helm installed and configured")

	return nil
}

func (o *InitOptions) configureForICP() {
	icpDefaultTillerNS := "default"
	icpDefaultNS := "jx"

	log.Infoln("")
	log.Infoln(util.ColorInfo("IBM Cloud Private installation of Jenkins X"))
	log.Infoln("Configuring Jenkins X options for IBM Cloud Private: ensure your Kubernetes context is already " +
		"configured to point to the cluster jx will be installed into.")
	log.Infoln("")

	log.Infoln(util.ColorInfo("Permitting image repositories to be used"))
	log.Infoln("If you have a clusterimagepolicy, ensure that this policy permits pulling from the following additional repositories: " +
		"the scope of which can be narrowed down once you are sure only images from certain repositories are being used:")
	log.Infoln("- name: docker.io/* \n" +
		"- name: gcr.io/* \n" +
		"- name: quay.io/* \n" +
		"- name: k8s.gcr.io/* \n" +
		"- name: <your ICP cluster name>:8500/* \n")

	log.Infoln(util.ColorInfo("IBM Cloud Private defaults"))
	log.Infoln("By default, with IBM Cloud Private the Tiller namespace for jx will be \"" + icpDefaultTillerNS + "\" and the namespace " +
		"where Jenkins X resources will be installed into is \"" + icpDefaultNS + "\".")
	log.Infoln("")

	log.Infoln(util.ColorInfo("Using the IBM Cloud Private Docker registry"))
	log.Infoln("To use the IBM Cloud Private Docker registry, when environments (namespaces) are created, " +
		"create a Docker registry secret and patch the default service account in the created namespace to use the secret, adding it as an ImagePullSecret. " +
		"This is required so that pods in the created namespace can pull images from the registry.")
	log.Infoln("")

	o.Flags.IngressNamespace = "kube-system"
	o.Flags.IngressDeployment = "default-backend"
	o.Flags.IngressService = "default-backend"
	o.Flags.TillerNamespace = icpDefaultTillerNS
	o.Flags.Namespace = icpDefaultNS
	//o.Flags.NoTiller = true // eventually desirable once ICP tiller version is 2.10 or better

	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	ICPExternalIP := ""
	ICPDomain := ""

	if !(o.BatchMode) {
		if o.Flags.ExternalIP != "" {
			log.Infoln("An external IP has already been specified: otherwise you will be prompted for one to use")
			return
		}

		prompt := &survey.Input{
			Message: "Provide the external IP Jenkins X should use: typically your IBM Cloud Private proxy node IP address",
			Default: "", // Would be useful to set this as the public IP automatically
			Help:    "",
		}
		survey.AskOne(prompt, &ICPExternalIP, nil, surveyOpts)

		o.Flags.ExternalIP = ICPExternalIP

		prompt = &survey.Input{
			Message: "Provide the domain Jenkins X should be available at: typically your IBM Cloud Private proxy node IP address but with a domain added to the end",
			Default: ICPExternalIP + ".nip.io",
			Help:    "",
		}

		survey.AskOne(prompt, &ICPDomain, nil, surveyOpts)

		o.Flags.Domain = ICPDomain
	}
}

func (o *InitOptions) initIKSIngress() error {
	log.Infoln("Wait for Ingress controller to be injected into IBM Kubernetes Service Cluster")
	kubeClient, err := o.KubeClient()
	if err != nil {
		return err
	}

	ingressNamespace := o.Flags.IngressNamespace

	clusterID, err := iks.GetKubeClusterID(kubeClient)
	if err != nil || clusterID == "" {
		clusterID, err = iks.GetClusterID()
		if err != nil {
			return err
		}
	}
	o.Flags.IngressDeployment = "public-cr" + strings.ToLower(clusterID) + "-alb1"
	o.Flags.IngressService = "public-cr" + strings.ToLower(clusterID) + "-alb1"

	return kube.WaitForDeploymentToBeCreatedAndReady(kubeClient, o.Flags.IngressDeployment, ingressNamespace, 30*time.Minute)
}

func (o *InitOptions) initIngress() error {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	client, err := o.KubeClient()
	if err != nil {
		return err
	}

	ingressNamespace := o.Flags.IngressNamespace

	err = kube.EnsureNamespaceCreated(client, ingressNamespace, map[string]string{"jenkins.io/kind": "ingress"}, nil)
	if err != nil {
		return fmt.Errorf("Failed to ensure the ingress namespace %s is created: %s\nIs this an RBAC issue on your cluster?", ingressNamespace, err)
	}

	currentContext, err := o.getCommandOutput("", "kubectl", "config", "current-context")
	if err != nil {
		return err
	}
	if currentContext == "minikube" {
		if o.Flags.Provider == "" {
			o.Flags.Provider = MINIKUBE
		}
		addons, err := o.getCommandOutput("", "minikube", "addons", "list")
		if err != nil {
			return err
		}
		if strings.Contains(addons, "- ingress: enabled") {
			log.Success("nginx ingress controller already enabled")
			return nil
		}
		err = o.RunCommand("minikube", "addons", "enable", "ingress")
		if err != nil {
			return err
		}
		log.Success("nginx ingress controller now enabled on Minikube")
		return nil

	}

	if isOpenShiftProvider(o.Flags.Provider) {
		log.Infoln("Not installing ingress as using OpenShift which uses Route and its own mechanism of ingress")
		return nil
	}

	podCount, err := kube.DeploymentPodCount(client, o.Flags.IngressDeployment, ingressNamespace)
	if podCount == 0 {
		installIngressController := false
		if o.BatchMode {
			installIngressController = true
		} else {
			prompt := &survey.Confirm{
				Message: "No existing ingress controller found in the " + ingressNamespace + " namespace, shall we install one?",
				Default: true,
				Help:    "An ingress controller works with an external loadbalancer so you can access Jenkins X and your applications",
			}
			survey.AskOne(prompt, &installIngressController, nil, surveyOpts)
		}

		if !installIngressController {
			return nil
		}

		values := []string{"rbac.create=true" /*,"rbac.serviceAccountName="+ingressServiceAccount*/}
		valuesFiles := []string{}
		valuesFiles, err = helm.AppendMyValues(valuesFiles)
		if err != nil {
			return errors.Wrap(err, "failed to append the myvalues file")
		}
		if o.Flags.Provider == AWS || o.Flags.Provider == EKS {
			// we can only enable one port for NLBs right now
			enableHTTP := "false"
			enableHTTPS := "true"
			if o.Flags.Http {
				enableHTTP = "true"
				enableHTTPS = "false"
			}
			yamlText := `---
rbac:
 create: true

controller:
 service:
   annotations:
     service.beta.kubernetes.io/aws-load-balancer-type: nlb
   enableHttp: ` + enableHTTP + `
   enableHttps: ` + enableHTTPS + `
`

			f, err := ioutil.TempFile("", "ing-values-")
			if err != nil {
				return err
			}
			fileName := f.Name()
			err = ioutil.WriteFile(fileName, []byte(yamlText), DefaultWritePermissions)
			if err != nil {
				return err
			}
			log.Infof("Using helm values file: %s\n", fileName)
			valuesFiles = append(valuesFiles, fileName)
		}

		i := 0
		for {
			log.Infof("Installing using helm binary: %s\n", util.ColorInfo(o.Helm().HelmBinary()))
			err = o.Helm().InstallChart("stable/nginx-ingress", "jxing", ingressNamespace, nil, nil, values,
				valuesFiles, "", "", "")
			if err != nil {
				if i >= 3 {
					log.Errorf("Failed to install ingress chart: %s", err)
					break
				}
				i++
				time.Sleep(time.Second)
			} else {
				break
			}
		}
		err = kube.WaitForDeploymentToBeReady(client, o.Flags.IngressDeployment, ingressNamespace, 10*time.Minute)
		if err != nil {
			return err
		}

	} else {
		log.Info("existing ingress controller found, no need to install a new one\n")
	}

	if o.Flags.Provider != MINIKUBE && o.Flags.Provider != MINISHIFT && o.Flags.Provider != OPENSHIFT {

		log.Infof("Waiting for external loadbalancer to be created and update the nginx-ingress-controller service in %s namespace\n", ingressNamespace)

		if o.Flags.Provider == OKE {
			log.Infof("Note: this loadbalancer will fail to be provisioned if you have insufficient quotas, this can happen easily on a OCI free account\n")
		}

		if o.Flags.Provider == GKE {
			log.Infof("Note: this loadbalancer will fail to be provisioned if you have insufficient quotas, this can happen easily on a GKE free account. To view quotas run: %s\n", util.ColorInfo("gcloud compute project-info describe"))
		}

		externalIP := o.Flags.ExternalIP
		if externalIP == "" && o.Flags.OnPremise {
			// lets find the Kubernetes master IP
			config, err := o.CreateKubeConfig()
			if err != nil {
				return err
			}
			host := config.Host
			if host == "" {
				log.Warnf("No API server host is defined in the local kube config!\n")
			} else {
				externalIP, err = util.UrlHostNameWithoutPort(host)
				if err != nil {
					return fmt.Errorf("Could not parse Kubernetes master URI: %s as got: %s\nTry specifying the external IP address directly via: --external-ip", host, err)
				}
			}
		}

		if externalIP == "" {
			err = services.WaitForExternalIP(client, o.Flags.IngressService, ingressNamespace, 10*time.Minute)
			if err != nil {
				return err
			}
			log.Infof("External loadbalancer created\n")
		} else {
			log.Infof("Using external IP: %s\n", util.ColorInfo(externalIP))
		}

		o.Flags.Domain, err = o.GetDomain(client, o.Flags.Domain, o.Flags.Provider, ingressNamespace, o.Flags.IngressService, externalIP)
		if err != nil {
			return err
		}
	}

	log.Success("nginx ingress controller installed and configured")

	return nil
}

func (o *InitOptions) ingressNamespace() string {
	ingressNamespace := "kube-system"
	if !o.Flags.GlobalTiller {
		ingressNamespace = o.Flags.Namespace
	}
	return ingressNamespace
}

// validateGit validates that git is configured correctly
func (o *InitOptions) validateGit() error {
	// lets ignore errors which indicate no value set
	userName, _ := o.Git().Username("")
	userEmail, _ := o.Git().Email("")
	var err error
	if userName == "" {
		if !o.BatchMode {
			userName, err = util.PickValue("Please enter the name you wish to use with git: ", "", true, "", o.In, o.Out, o.Err)
			if err != nil {
				return err
			}
		}
		if userName == "" {
			return fmt.Errorf("No Git user.name is defined. Please run the command: git config --global --add user.name \"MyName\"")
		}
		err = o.Git().SetUsername("", userName)
		if err != nil {
			return err
		}
	}
	if userEmail == "" {
		if !o.BatchMode {
			userEmail, err = util.PickValue("Please enter the email address you wish to use with git: ", "", true, "", o.In, o.Out, o.Err)
			if err != nil {
				return err
			}
		}
		if userEmail == "" {
			return fmt.Errorf("No Git user.email is defined. Please run the command: git config --global --add user.email \"me@acme.com\"")
		}
		err = o.Git().SetEmail("", userEmail)
		if err != nil {
			return err
		}
	}
	log.Infof("Git configured for user: %s and email %s\n", util.ColorInfo(userName), util.ColorInfo(userEmail))
	return nil
}

// HelmBinary returns name of configured Helm binary
func (o *InitOptions) HelmBinary() string {
	if o.Flags.Helm3 {
		return "helm3"
	}
	testHelmBin := o.Flags.HelmBin
	if testHelmBin != "" {
		return testHelmBin
	}
	return "helm"
}

// GetDomain returns the domain name, calculating it if possible else prompting the user
func (o *CommonOptions) GetDomain(client kubernetes.Interface, domain string, provider string, ingressNamespace string, ingressService string, externalIP string) (string, error) {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	address := externalIP
	if address == "" {
		if provider == MINIKUBE {
			ip, err := o.getCommandOutput("", "minikube", "ip")
			if err != nil {
				return "", err
			}
			address = ip
		} else if provider == MINISHIFT {
			ip, err := o.getCommandOutput("", "minishift", "ip")
			if err != nil {
				return "", err
			}
			address = ip
		} else {
			info := util.ColorInfo
			log.Infof("Waiting to find the external host name of the ingress controller Service in namespace %s with name %s\n", info(ingressNamespace), info(ingressService))
			if provider == KUBERNETES {
				log.Infof("If you are installing Jenkins X on premise you may want to use the '--on-premise' flag or specify the '--external-ip' flags. See: %s\n", info("https://jenkins-x.io/getting-started/install-on-cluster/#installing-jenkins-x-on-premise"))
			}
			svc, err := client.CoreV1().Services(ingressNamespace).Get(ingressService, metav1.GetOptions{})
			if err != nil {
				return "", err
			}
			if svc != nil {
				for _, v := range svc.Status.LoadBalancer.Ingress {
					if v.IP != "" {
						address = v.IP
					} else if v.Hostname != "" {
						address = v.Hostname
					}
				}
			}
		}
	}
	defaultDomain := address

	if provider == AWS || provider == EKS {
		if domain != "" {
			err := amazon.RegisterAwsCustomDomain(domain, address)
			return domain, err
		}
		log.Infof("\nOn AWS we recommend using a custom DNS name to access services in your Kubernetes cluster to ensure you can use all of your Availability Zones\n")
		log.Infof("If you do not have a custom DNS name you can use yet, then you can register a new one here: %s\n\n", util.ColorInfo("https://console.aws.amazon.com/route53/home?#DomainRegistration:"))

		for {
			if util.Confirm("Would you like to register a wildcard DNS ALIAS to point at this ELB address? ", true,
				"When using AWS we need to use a wildcard DNS alias to point at the ELB host name so you can access services inside Jenkins X and in your Environments.", o.In, o.Out, o.Err) {
				customDomain := ""
				prompt := &survey.Input{
					Message: "Your custom DNS name: ",
					Help:    "Enter your custom domain that we can use to setup a Route 53 ALIAS record to point at the ELB host: " + address,
				}
				survey.AskOne(prompt, &customDomain, nil, surveyOpts)
				if customDomain != "" {
					err := amazon.RegisterAwsCustomDomain(customDomain, address)
					return customDomain, err
				}
			} else {
				break
			}
		}
	}

	if provider == IKS {
		if domain != "" {
			log.Infof("\nIBM Kubernetes Service will use provided domain. Ensure name is registrered with DNS (ex. CIS) and pointing the cluster ingress IP: %s\n", util.ColorInfo(address))
			return domain, nil
		}
		clusterName, err := iks.GetClusterName()
		clusterRegion, err := iks.GetKubeClusterRegion(client)
		if err == nil && clusterName != "" && clusterRegion != "" {
			customDomain := clusterName + "." + clusterRegion + ".containers.appdomain.cloud"
			log.Infof("\nIBM Kubernetes Service will use the default cluster domain: ")
			log.Infof("%s\n", util.ColorInfo(customDomain))
			return customDomain, nil
		} else {
			log.Infof("ERROR getting IBM Kubernetes Service will use the default cluster domain:")
			log.Infof(err.Error())
		}
	}

	if address != "" {
		addNip := true
		aip := net.ParseIP(address)
		if aip == nil {
			log.Infof("The Ingress address %s is not an IP address. We recommend we try resolve it to a public IP address and use that for the domain to access services externally.\n", util.ColorInfo(address))

			addressIP := ""
			if util.Confirm("Would you like wait and resolve this address to an IP address and use it for the domain?", true,
				"Should we convert "+address+" to an IP address so we can access resources externally", o.In, o.Out, o.Err) {

				log.Infof("Waiting for %s to be resolvable to an IP address...\n", util.ColorInfo(address))
				f := func() error {
					ips, err := net.LookupIP(address)
					if err == nil {
						for _, ip := range ips {
							t := ip.String()
							if t != "" && !ip.IsLoopback() {
								addressIP = t
								return nil
							}
						}
					}
					return fmt.Errorf("Address cannot be resolved yet %s", address)
				}

				o.retryQuiet(5*6, time.Second*10, f)
			}
			if addressIP == "" {
				addNip = false
				log.Infof("Still not managed to resolve address %s into an IP address. Please try figure out the domain by hand\n", address)
			} else {
				log.Infof("%s resolved to IP %s\n", util.ColorInfo(address), util.ColorInfo(addressIP))
				address = addressIP
			}
		}
		if addNip && !strings.HasSuffix(address, ".amazonaws.com") {
			defaultDomain = fmt.Sprintf("%s.nip.io", address)
		}
	}
	if domain == "" {
		if o.BatchMode {
			log.Successf("No domain flag provided so using default %s to generate Ingress rules", defaultDomain)
			return defaultDomain, nil
		}
		log.Successf("You can now configure a wildcard DNS pointing to the new loadbalancer address %s", address)
		log.Info("\nIf you do not have a custom domain setup yet, Ingress rules will be set for magic dns nip.io.")
		log.Infof("\nOnce you have a custom domain ready, you can update with the command %s", util.ColorInfo("jx upgrade ingress --cluster"))

		log.Infof("\nIf you don't have a wildcard DNS setup then setup a DNS (A) record and point it at: %s then use the DNS domain in the next input...\n", address)

		if domain == "" {
			prompt := &survey.Input{
				Message: "Domain",
				Default: defaultDomain,
				Help:    "Enter your custom domain that is used to generate Ingress rules, defaults to the magic dns nip.io",
			}
			survey.AskOne(prompt, &domain,
				survey.ComposeValidators(survey.Required, surveyutils.NoWhiteSpaceValidator()), surveyOpts)
		}
		if domain == "" {
			domain = defaultDomain
		}
	} else {
		if domain != defaultDomain {
			log.Successf("You can now configure your wildcard DNS %s to point to %s\n", domain, address)
		}
	}

	return domain, nil
}
