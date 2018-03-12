package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/homedir"
)

// InitOptions the flags for running init
type InitOptions struct {
	CommonOptions
	Client clientset.Clientset
	Flags  InitFlags
}

type InitFlags struct {
	Domain                     string
	Provider                   string
	Namespace                  string
	Username                   string
	UserClusterRole            string
	TillerClusterRole          string
	IngressClusterRole         string
	TillerNamespace            string
	IngressNamespace           string
	DraftClient                bool
	HelmClient                 bool
	RecreateExistingDraftRepos bool
	GlobalTiller               bool
}

const (
	optionUsername  = "username"
	optionNamespace = "namespace"

	INGRESS_SERVICE_NAME    = "jxing-nginx-ingress-controller"
	DEFAULT_CHARTMUSEUM_URL = "http://chartmuseum.build.cd.jenkins-x.io"
)

var (
	initLong = templates.LongDesc(`
		This command installs the Jenkins X platform on a connected kubernetes cluster
`)

	initExample = templates.Examples(`
		jx init
`)
)

// NewCmdInit creates a command object for the generic "init" action, which
// primes a kubernetes cluster so it's ready for jenkins x to be installed
func NewCmdInit(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &InitOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
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
			cmdutil.CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.Flags.Provider, "provider", "", "", "Cloud service providing the kubernetes cluster.  Supported providers: [aks,eks,gke,kubernetes,minikube]")
	cmd.Flags().StringVarP(&options.Flags.Namespace, optionNamespace, "", "jx", "The namespace the Jenkins X platform should be installed into")
	options.addInitFlags(cmd)
	return cmd
}

func (options *InitOptions) addInitFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&options.Flags.Domain, "domain", "", "", "Domain to expose ingress endpoints.  Example: jenkinsx.io")
	cmd.Flags().StringVarP(&options.Flags.Username, optionUsername, "", "", "The kubernetes username used to initialise helm. Usually your email address for your kubernetes account")
	cmd.Flags().StringVarP(&options.Flags.UserClusterRole, "user-cluster-role", "", "cluster-admin", "The cluster role for the current user to be able to administer helm")
	cmd.Flags().StringVarP(&options.Flags.TillerClusterRole, "tiller-cluster-role", "", "cluster-admin", "The cluster role for Helm's tiller")
	cmd.Flags().StringVarP(&options.Flags.TillerNamespace, "tiller-namespace", "", "kube-system", "The namespace for the Tiller when using a gloabl tiller")
	cmd.Flags().StringVarP(&options.Flags.IngressClusterRole, "ingress-cluster-role", "", "cluster-admin", "The cluster role for the Ingress controller")
	cmd.Flags().StringVarP(&options.Flags.IngressNamespace, "ingress-namespace", "", "kube-system", "The namespace for the Ingress controller")
	cmd.Flags().BoolVarP(&options.Flags.DraftClient, "draft-client-only", "", false, "Only install draft client")
	cmd.Flags().BoolVarP(&options.Flags.HelmClient, "helm-client-only", "", false, "Only install helm client")
	cmd.Flags().BoolVarP(&options.Flags.RecreateExistingDraftRepos, "recreate-existing-draft-repos", "", false, "Delete existing helm repos used by Jenkins X under ~/draft/packs")
	cmd.Flags().BoolVarP(&options.Flags.GlobalTiller, "global-tiller", "", true, "Whether or not to use a cluster global tiller")
}

func (o *InitOptions) Run() error {

	var err error
	o.Flags.Provider, err = o.GetCloudProvider(o.Flags.Provider)
	if err != nil {
		return err
	}

	err = o.enableClusterAdminRole()
	if err != nil {
		return err
	}

	// helm init, this has been seen to fail intermittently on public clouds, so lets retry a couple of times
	err = o.retry(3, 2*time.Second, func() (err error) {
		err = o.initHelm()
		return
	})

	if err != nil {
		log.Fatalf("helm init failed: %v", err)
		return err
	}

	// draft init
	err = o.initDraft()
	if err != nil {
		log.Fatalf("draft init failed: %v", err)
		return err
	}

	// install ingress
	err = o.initIngress()
	if err != nil {
		log.Fatalf("ingress init failed: %v", err)
		return err
	}

	return nil
}

func (o *InitOptions) enableClusterAdminRole() error {
	f := o.Factory
	client, _, err := f.CreateClient()
	if err != nil {
		return err
	}

	user := o.Flags.Username
	if user == "" {
		config, _, err := kube.LoadConfig()
		if err != nil {
			return err
		}
		if config == nil || config.Contexts == nil || len(config.Contexts) == 0 {
			return fmt.Errorf("No kubernetes contexts available! Try create or connect to cluster?")
		}
		contextName := config.CurrentContext
		if contextName == "" {
			return fmt.Errorf("No kuberentes context selected. Please select one (e.g. via jx context) first")
		}
		context := config.Contexts[contextName]
		if context == nil {
			return fmt.Errorf("No kuberentes context available for context %s", contextName)
		}
		user = context.AuthInfo
	}
	if user == "" {
		return util.MissingOption(optionUsername)
	}
	user = kube.ToValidName(user)

	role := o.Flags.UserClusterRole
	clusterRoleBindingName := kube.ToValidName(user + "-" + role + "-binding")

	_, err = client.RbacV1().ClusterRoleBindings().Get(clusterRoleBindingName, meta_v1.GetOptions{})
	if err != nil {
		o.Printf("Trying to create ClusterRoleBinding %s for role: %s for user %s\n", clusterRoleBindingName, role, user)
		args := []string{"create", "clusterrolebinding", clusterRoleBindingName, "--clusterrole=" + role, "--user=" + user}
		err := o.runCommand("kubectl", args...)
		if err != nil {
			return err
		}

		o.Printf("Created ClusterRoleBinding %s\n", clusterRoleBindingName)
	}
	return nil
}

func (o *InitOptions) initHelm() error {
	f := o.Factory
	client, curNs, err := f.CreateClient()
	if err != nil {
		return err
	}

	serviceAccountName := "tiller"
	tillerNamespace := o.Flags.TillerNamespace
	ns := o.Flags.Namespace
	if ns == "" {
		ns = curNs
	}
	if ns == "" {
		return util.MissingOption(optionNamespace)
	}

	if o.Flags.GlobalTiller {
		clusterRoleBindingName := serviceAccountName
		role := o.Flags.TillerClusterRole

		err = o.ensureClusterRoleBinding(clusterRoleBindingName, role, tillerNamespace, serviceAccountName)
		if err != nil {
			return err
		}
	} else {
		tillerNamespace = ns
		_, err = client.CoreV1().Namespaces().Get(ns, meta_v1.GetOptions{})
		if err != nil {
			n := &corev1.Namespace{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: ns,
				},
			}
			_, err = client.CoreV1().Namespaces().Create(n)
			if err != nil {
				return fmt.Errorf("Failed to create Namespace %s: %s", ns, err)
			}
			o.Printf("Created Namespace %s\n", util.ColorInfo(ns))
		}

		// lets create a tiller service account
		roleName := "tiller-manager"
		roleBindingName := "tiller-binding"

		err = o.ensureServiceAccount(ns, serviceAccountName)
		if err != nil {
			return err
		}
		_, err = client.RbacV1().Roles(ns).Get(roleName, meta_v1.GetOptions{})
		if err != nil {
			// lets create a Role for tiller
			role := &rbacv1.Role{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      roleName,
					Namespace: ns,
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"", "extensions", "apps"},
						Resources: []string{"*"},
						Verbs:     []string{"*"},
					},
				},
			}
			_, err = client.RbacV1().Roles(ns).Create(role)
			if err != nil {
				return fmt.Errorf("Failed to create Role %s in namespace %s: %s", roleName, ns, err)
			}
			o.Printf("Created Role %s in namespace %s\n", util.ColorInfo(roleName), util.ColorInfo(ns))
		}
		_, err = client.RbacV1().RoleBindings(ns).Get(roleBindingName, meta_v1.GetOptions{})
		if err != nil {
			// lets create a RoleBinding for tiller
			roleBinding := &rbacv1.RoleBinding{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      roleBindingName,
					Namespace: ns,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      serviceAccountName,
						Namespace: ns,
					},
				},
				RoleRef: rbacv1.RoleRef{
					Kind:     "Role",
					Name:     roleName,
					APIGroup: "rbac.authorization.k8s.io",
				},
			}
			_, err = client.RbacV1().RoleBindings(ns).Create(roleBinding)
			if err != nil {
				return fmt.Errorf("Failed to create RoleBinding %s in namespace %s: %s", roleName, ns, err)
			}
			o.Printf("Created RoleBinding %s in namespace %s\n", util.ColorInfo(roleName), util.ColorInfo(ns))
		}
	}

	if o.Flags.HelmClient {
		err = o.runCommand("helm", "init", "--client-only")
		if err != nil {
			return err
		}
	}

	running, err := kube.IsDeploymentRunning(client, "tiller-deploy", tillerNamespace)
	if running {
		o.Printf("Tiller Deployment is running in namespace %s\n", util.ColorInfo(tillerNamespace))
		return nil
	}
	if err == nil && !running {
		return errors.New("existing tiller deployment found but not running, please check the kube-system namespace and resolve any issues")
	}

	if !running {
		args := []string{"init"}
		if !o.Flags.GlobalTiller {
			args = []string{"init", "--service-account", serviceAccountName, "--tiller-namespace", ns}
		}
		o.Printf("Initialising helm via: %s\n", util.ColorInfo("helm "+strings.Join(args, " ")))

		err = o.runCommand("helm", args...)
		if err != nil {
			return err
		}
		err = o.runCommand("helm", "init", "--upgrade")
		if err != nil {
			return err
		}
	}

	err = kube.WaitForDeploymentToBeReady(client, "tiller-deploy", tillerNamespace, 5*time.Minute)
	if err != nil {
		return err
	}

	err = o.runCommand("helm", "repo", "add", "jenkins-x", DEFAULT_CHARTMUSEUM_URL)
	if err != nil {
		return err
	}

	log.Success("helm installed and configured")

	return nil
}

func (o *InitOptions) initDraft() error {
	f := o.Factory
	client, _, err := f.CreateClient()
	if err != nil {
		return err
	}

	running, err := kube.IsDeploymentRunning(client, "draftd", "kube-system")

	if err == nil && !running {
		return errors.New("existing draftd deployment found but not running, please check the kube-system namespace and resolve any issues")
	}

	err = o.removeDraftRepoIfInstalled("Azure")
	if err != nil {
		return err
	}

	if running || o.Flags.Provider == GKE || o.Flags.Provider == AKS || o.Flags.Provider == EKS || o.Flags.Provider == KUBERNETES || o.Flags.DraftClient {
		err = o.runCommand("draft", "init", "--auto-accept", "--client-only")

	} else {
		err = o.runCommand("draft", "init", "--auto-accept")

	}
	if err != nil {
		return err
	}

	err = o.removeDraftRepoIfInstalled("jenkins-x")
	if err != nil {
		return err
	}

	err = o.runCommand("draft", "pack-repo", "add", "https://github.com/jenkins-x/draft-packs")
	if err != nil {
		log.Warn("error adding pack to draft, if you are using git 2.16.1 take a look at this issue for a workaround https://github.com/jenkins-x/jx/issues/176#issuecomment-361897946")
		return err
	}

	if !running && o.Flags.Provider != GKE && o.Flags.Provider != AKS && o.Flags.Provider != EKS && o.Flags.Provider != KUBERNETES && !o.Flags.DraftClient {
		err = kube.WaitForDeploymentToBeReady(client, "draftd", "kube-system", 5*time.Minute)
		if err != nil {
			return err
		}

	}
	log.Success("draft installed and configured")

	return nil
}

// this happens in `draft init` too, except there seems to be a timing issue where the repo add fails if done straight after their repo remove.
func (o *InitOptions) removeDraftRepoIfInstalled(repo string) error {

	pack := filepath.Join(homedir.HomeDir(), ".draft", "packs", "github.com", repo)
	if _, err := os.Stat(pack); err == nil {
		recreate := o.Flags.RecreateExistingDraftRepos
		if !recreate {
			prompt := &survey.Confirm{
				Message: fmt.Sprintf("Delete existing %s draft pack repo and get latest?", repo),
				Default: true,
				Help:    "Draft pack repos contain the files and folders used to install applications on Kubernetes, we recommend getting the latest",
			}
			survey.AskOne(prompt, &recreate, nil)
		}
		if recreate {
			os.RemoveAll(pack)
		}
	}
	return nil
}

func (o *InitOptions) initIngress() error {
	f := o.Factory
	client, _, err := f.CreateClient()
	if err != nil {
		return err
	}

	ingressServiceAccount := "ingress"
	ingressNamespace := o.Flags.IngressNamespace
	/*
		err = o.ensureServiceAccount(ingressNamespace, ingressServiceAccount)
		if err != nil {
			return err
		}

		role := o.Flags.IngressClusterRole
		clusterRoleBindingName := kube.ToValidName(ingressServiceAccount + "-" + role + "-binding")

		err = o.ensureClusterRoleBinding(clusterRoleBindingName, role, ingressNamespace, ingressServiceAccount)
		if err != nil {
			return err
		}
	*/

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
		err = o.runCommand("minikube", "addons", "enable", "ingress")
		if err != nil {
			return err
		}
		log.Success("nginx ingress controller now enabled on minikube")
		return nil

	}

	podLabels := labels.SelectorFromSet(labels.Set(map[string]string{"app": "nginx-ingress", "component": "controller"}))
	options := meta_v1.ListOptions{LabelSelector: podLabels.String()}
	podList, err := client.CoreV1().Pods(ingressNamespace).List(options)
	if err != nil {
		return err
	}

	if podList != nil && len(podList.Items) > 0 {
		log.Info("existing nginx ingress controller found, no need to install")
		return nil
	}

	installIngressController := false
	if o.BatchMode {
		installIngressController = true
	} else {
		prompt := &survey.Confirm{
			Message: "No existing ingress controller found in the " + ingressNamespace + " namespace, shall we install one?",
			Default: true,
			Help:    "An ingress controller works with an external loadbalancer so you can access Jenkins X and your applications",
		}
		survey.AskOne(prompt, &installIngressController, nil)
	}

	if !installIngressController {
		return nil
	}

	i := 0
	for {
		err = o.runCommand("helm", "install", "--name", "jxing", "stable/nginx-ingress", "--namespace", ingressNamespace, "--set", "rbac.create=true", "--set", "rbac.serviceAccountName="+ingressServiceAccount)
		if err != nil {
			if i >= 3 {
				break
			}
			i++
			time.Sleep(time.Second)
		} else {
			break
		}
	}

	err = kube.WaitForDeploymentToBeReady(client, INGRESS_SERVICE_NAME, ingressNamespace, 10*time.Minute)
	if err != nil {
		return err
	}

	if o.Flags.Provider == GKE || o.Flags.Provider == AKS || o.Flags.Provider == EKS || o.Flags.Provider == KUBERNETES {
		log.Infof("Waiting for external loadbalancer to be created and update the nginx-ingress-controller service in %s namespace\n", ingressNamespace)
		err = kube.WaitForExternalIP(client, INGRESS_SERVICE_NAME, ingressNamespace, 10*time.Minute)
		if err != nil {
			return err
		}

		log.Infof("External loadbalancer created\n")

		o.Flags.Domain, err = o.GetDomain(client, o.Flags.Domain, o.Flags.Domain, ingressNamespace)
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

func (o *CommonOptions) GetDomain(client *kubernetes.Clientset, domain string, provider string, ingressNamespace string) (string, error) {
	var address string
	if provider == MINIKUBE {
		ip, err := o.getCommandOutput("", "minikube", "ip")
		if err != nil {
			return "", err
		}
		address = ip
	} else {
		svc, err := client.CoreV1().Services(ingressNamespace).Get(INGRESS_SERVICE_NAME, meta_v1.GetOptions{})
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
	defaultDomain := fmt.Sprintf("%s.nip.io", address)
	if domain == "" {

		if o.BatchMode {
			log.Successf("No domain flag provided so using default %s to generate Ingress rules", defaultDomain)
			return defaultDomain, nil
		}
		log.Successf("You can now configure a wildcard DNS pointing to the new loadbalancer address %s", address)
		log.Infof("If you don't have a wildcard DNS yet you can use the default %s", defaultDomain)

		if domain == "" {
			prompt := &survey.Input{
				Message: "Domain",
				Default: defaultDomain,
				Help:    "Enter your custom domain that is used to generate Ingress rules, defaults to the magic dns nip.io",
			}
			survey.AskOne(prompt, &domain, nil)
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
