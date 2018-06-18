package cmd

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
	IngressService             string
	IngressDeployment          string
	ExternalIP                 string
	DraftClient                bool
	HelmClient                 bool
	RecreateExistingDraftRepos bool
	GlobalTiller               bool
	SkipIngress                bool
	SkipTiller                 bool
	OnPremise                  bool
}

const (
	optionUsername        = "username"
	optionNamespace       = "namespace"
	optionTillerNamespace = "tiller-namespace"

	JenkinsBuildPackURL = "https://github.com/jenkins-x/draft-packs.git"

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

	cmd.Flags().StringVarP(&options.Flags.Provider, "provider", "", "", "Cloud service providing the kubernetes cluster.  Supported providers: "+KubernetesProviderOptions())
	cmd.Flags().StringVarP(&options.Flags.Namespace, optionNamespace, "", "jx", "The namespace the Jenkins X platform should be installed into")
	options.addInitFlags(cmd)
	return cmd
}

func (options *InitOptions) addInitFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&options.Flags.Domain, "domain", "", "", "Domain to expose ingress endpoints.  Example: jenkinsx.io")
	cmd.Flags().StringVarP(&options.Flags.Username, optionUsername, "", "", "The kubernetes username used to initialise helm. Usually your email address for your kubernetes account")
	cmd.Flags().StringVarP(&options.Flags.UserClusterRole, "user-cluster-role", "", "cluster-admin", "The cluster role for the current user to be able to administer helm")
	cmd.Flags().StringVarP(&options.Flags.TillerClusterRole, "tiller-cluster-role", "", "cluster-admin", "The cluster role for Helm's tiller")
	cmd.Flags().StringVarP(&options.Flags.TillerNamespace, optionTillerNamespace, "", "kube-system", "The namespace for the Tiller when using a gloabl tiller")
	cmd.Flags().StringVarP(&options.Flags.IngressClusterRole, "ingress-cluster-role", "", "cluster-admin", "The cluster role for the Ingress controller")
	cmd.Flags().StringVarP(&options.Flags.IngressNamespace, "ingress-namespace", "", "kube-system", "The namespace for the Ingress controller")
	cmd.Flags().StringVarP(&options.Flags.IngressService, "ingress-service", "", INGRESS_SERVICE_NAME, "The name of the Ingress controller Service")
	cmd.Flags().StringVarP(&options.Flags.IngressDeployment, "ingress-deployment", "", INGRESS_SERVICE_NAME, "The namespace for the Ingress controller Deployment")
	cmd.Flags().StringVarP(&options.Flags.ExternalIP, "external-ip", "", "", "The external IP used to access ingress endpoints from outside the kubernetes cluster. For bare metal on premise clusters this is often the IP of the kubernetes master. For cloud installations this is often the external IP of the ingress LoadBalancer.")
	cmd.Flags().BoolVarP(&options.Flags.DraftClient, "draft-client-only", "", false, "Only install draft client")
	cmd.Flags().BoolVarP(&options.Flags.HelmClient, "helm-client-only", "", false, "Only install helm client")
	cmd.Flags().BoolVarP(&options.Flags.RecreateExistingDraftRepos, "recreate-existing-draft-repos", "", false, "Delete existing helm repos used by Jenkins X under ~/draft/packs")
	cmd.Flags().BoolVarP(&options.Flags.GlobalTiller, "global-tiller", "", true, "Whether or not to use a cluster global tiller")
	cmd.Flags().BoolVarP(&options.Flags.SkipIngress, "skip-ingress", "", false, "Dont install an ingress controller")
	cmd.Flags().BoolVarP(&options.Flags.SkipTiller, "skip-tiller", "", false, "Dont install a Helms Tiller service")
	cmd.Flags().BoolVarP(&options.Flags.OnPremise, "on-premise", "", false, "If installing on an on premise cluster then lets default the 'external-ip' to be the kubernetes master IP address")
}

func (o *InitOptions) Run() error {

	var err error
	o.Flags.Provider, err = o.GetCloudProvider(o.Flags.Provider)
	if err != nil {
		return err
	}

	err = o.validateGit()
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
	client, _, err := o.KubeClient()
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

		err := o.retry(3, 10*time.Second, func() (err error) {
			return o.runCommand("kubectl", args...)
		})
		if err != nil {
			return err
		}

		o.Printf("Created ClusterRoleBinding %s\n", clusterRoleBindingName)
	}
	return nil
}

func (o *InitOptions) initHelm() error {
	var err error

	if !o.Flags.SkipTiller {

		client, curNs, err := o.KubeClient()
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

			_, err = client.RbacV1().Roles(tillerNamespace).Get(roleName, meta_v1.GetOptions{})
			if err != nil {
				// lets create a Role for tiller
				role := &rbacv1.Role{
					ObjectMeta: meta_v1.ObjectMeta{
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
				o.Printf("Created Role %s in namespace %s\n", util.ColorInfo(roleName), util.ColorInfo(tillerNamespace))
			}
			_, err = client.RbacV1().RoleBindings(tillerNamespace).Get(roleBindingName, meta_v1.GetOptions{})
			if err != nil {
				// lets create a RoleBinding for tiller
				roleBinding := &rbacv1.RoleBinding{
					ObjectMeta: meta_v1.ObjectMeta{
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
				o.Printf("Created RoleBinding %s in namespace %s\n", util.ColorInfo(roleName), util.ColorInfo(tillerNamespace))
			}
		}

		running, err := kube.IsDeploymentRunning(client, "tiller-deploy", tillerNamespace)
		if running {
			o.Printf("Tiller Deployment is running in namespace %s\n", util.ColorInfo(tillerNamespace))
			return nil
		}
		if err == nil && !running {
			return fmt.Errorf("existing tiller deployment found but not running, please check the %s namespace and resolve any issues", tillerNamespace)
		}

		if !running {
			o.Printf("Initialising helm using ServiceAccount %s in namespace %s\n", util.ColorInfo(serviceAccountName), util.ColorInfo(tillerNamespace))

			err = o.runCommand("helm", "init", "--service-account", serviceAccountName, "--tiller-namespace", tillerNamespace)
			if err != nil {
				return err
			}
			err = o.runCommand("helm", "init", "--upgrade", "--service-account", serviceAccountName, "--tiller-namespace", tillerNamespace)
			if err != nil {
				return err
			}
		}

		err = kube.WaitForDeploymentToBeReady(client, "tiller-deploy", tillerNamespace, 10*time.Minute)
		if err != nil {
			return err
		}
	}

	if o.Flags.HelmClient || o.Flags.SkipTiller {
		err = o.runCommand("helm", "init", "--client-only")
		if err != nil {
			return err
		}
	}

	err = o.runCommand("helm", "repo", "add", "jenkins-x", DEFAULT_CHARTMUSEUM_URL)
	if err != nil {
		return err
	}
	log.Success("helm installed and configured")

	return nil
}

// initBuildPacks initalise the build packs
func (o *InitOptions) initBuildPacks() (string, error) {
	settings, err := o.TeamSettings()

	if err != nil {
		return "", err
	}

	packUrl := settings.BuildPackURL
	packRef := settings.BuildPackRef

	u, err := url.Parse(strings.TrimSuffix(packUrl, ".git"))
	if err != nil {
		return "", fmt.Errorf("Failed to parse build pack URL: %s: %s", packUrl, err)
	}

	draftDir, err := util.DraftDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(draftDir, "packs", u.Host, u.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("Could not create %s: %s", dir, err)
	}

	err = gits.GitCloneOrPull(packUrl, dir)
	if err != nil {
		return "", err
	}
	if packRef != "master" {
		err = gits.CheckoutRemoteBranch(dir, packRef)
	}
	return filepath.Join(dir, "packs"), err
}

func (o *InitOptions) initIngress() error {
	client, _, err := o.KubeClient()
	if err != nil {
		return err
	}

	ingressNamespace := o.Flags.IngressNamespace

	err = kube.EnsureNamespaceCreated(client, ingressNamespace, map[string]string{"jenkins.io/kind": "ingress"}, nil)
	if err != nil {
		return fmt.Errorf("Failed to ensure the ingress namespace %s is created: %s\nIs this an RBAC issue on your cluster?", ingressNamespace, err)
	}

	/*
		ingressServiceAccount := "ingress"
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

	if isOpenShiftProvider(o.Flags.Provider) {
		o.Printf("Not installing ingress as using OpenShift which uses Route and its own mechanism of ingress\n")
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
			survey.AskOne(prompt, &installIngressController, nil)
		}

		if !installIngressController {
			return nil
		}

		i := 0
		for {
			//err = o.runCommand("helm", "install", "--name", "jxing", "stable/nginx-ingress", "--namespace", ingressNamespace, "--set", "rbac.create=true", "--set", "rbac.serviceAccountName="+ingressServiceAccount)
			err = o.runCommandVerbose("helm", "install", "--name", "jxing", "stable/nginx-ingress", "--namespace", ingressNamespace, "--set", "rbac.create=true")
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

		err = kube.WaitForDeploymentToBeReady(client, o.Flags.IngressDeployment, ingressNamespace, 10*time.Minute)
		if err != nil {
			return err
		}

	} else {
		log.Info("existing ingress controller found, no need to install a new one\n")
	}

	if o.Flags.Provider != MINIKUBE && o.Flags.Provider != MINISHIFT && o.Flags.Provider != OPENSHIFT {
		log.Infof("Waiting for external loadbalancer to be created and update the nginx-ingress-controller service in %s namespace\n", ingressNamespace)

		if o.Flags.Provider == GKE {
			log.Infof("Note: this loadbalancer will fail to be provisioned if you have insufficient quotas, this can happen easily on a GKE free account. To view quotas run: %s\n", util.ColorInfo("gcloud compute project-info describe"))
		}
		externalIP := o.Flags.ExternalIP
		if externalIP == "" && o.Flags.OnPremise {
			// lets find the kubernetes master IP
			config, err := o.Factory.CreateKubeConfig()
			if err != nil {
				return err
			}
			host := config.Host
			if host == "" {
				o.warnf("No API server host is defined in the local kube config!\n")
			} else {
				externalIP, err = util.UrlHostNameWithoutPort(host)
				if err != nil {
					return fmt.Errorf("Could not parse kubernetes master URI: %s as got: %s\nTry specifying the external IP address directly via: --external-ip", host, err)
				}
			}
		}

		if externalIP == "" {
			err = kube.WaitForExternalIP(client, o.Flags.IngressService, ingressNamespace, 10*time.Minute)
			if err != nil {
				return err
			}
			log.Infof("External loadbalancer created\n")
		} else {
			o.Printf("Using external IP: %s\n", util.ColorInfo(externalIP))
		}

		o.Flags.Domain, err = o.GetDomain(client, o.Flags.Domain, o.Flags.Domain, ingressNamespace, o.Flags.IngressService, externalIP)
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
	userName, _ := o.getCommandOutput("", "git", "config", "--global", "--get", "user.name")
	userEmail, _ := o.getCommandOutput("", "git", "config", "--global", "--get", "user.email")
	var err error

	if userName == "" {
		if !o.BatchMode {
			userName, err = util.PickValue("Please enter the name you wish to use with git: ", "", true)
			if err != nil {
				return err
			}
		}
		if userName == "" {
			return fmt.Errorf("No git user.name is defined. Please run the command: git config --global --add user.name \"MyName\"")
		}
		err = o.runCommandFromDir("", "git", "config", "--global", "--add", "user.name", userName)
		if err != nil {
			return err
		}
	}
	if userEmail == "" {
		if !o.BatchMode {
			userEmail, err = util.PickValue("Please enter the email address you wish to use with git: ", "", true)
			if err != nil {
				return err
			}
		}
		if userEmail == "" {
			return fmt.Errorf("No git user.email is defined. Please run the command: git config --global --add user.email \"me@acme.com\"")
		}
		err = o.runCommandFromDir("", "git", "config", "--global", "--add", "user.email", userEmail)
		if err != nil {
			return err
		}
	}
	o.Printf("Git configured for user: %s and email %s\n", util.ColorInfo(userName), util.ColorInfo(userEmail))
	return nil
}

func (o *CommonOptions) GetDomain(client kubernetes.Interface, domain string, provider string, ingressNamespace string, ingressService string, externalIP string) (string, error) {
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
			o.Printf("Waiting to find the external host name of the ingress controller Service in namespace %s with name %s\n", info(ingressNamespace), info(ingressService))
			if provider == KUBERNETES {
				o.Printf("If you are installing Jenkins X on premise you may want to use the '--on-premise' flag or specify the '--external-ip' flags. See: %s\n", info("https://jenkins-x.io/getting-started/install-on-cluster/#installing-jenkins-x-on-premise"))
			}
			svc, err := client.CoreV1().Services(ingressNamespace).Get(ingressService, meta_v1.GetOptions{})
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
	if address != "" {
		addNip := true
		aip := net.ParseIP(address)
		if aip == nil {
			o.Printf("The Ingress address %s is not an IP address. We recommend we try resolve it to a public IP address and use that for the domain to access services externally.\n", util.ColorInfo(address))

			addressIP := ""
			if util.Confirm("Would you like wait and resolve this address to an IP address and use it for the domain?", true,
				"Should we convert "+address+" to an IP address so we can access resources externally") {

				o.Printf("Waiting for %s to be resolvable to an IP address...\n", util.ColorInfo(address))
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
				o.warnf("Still not managed to resolve address %s into an IP address. Please try figure out the domain by hand\n", address)
			} else {
				o.Printf("%s resolved to IP %s\n", util.ColorInfo(address), util.ColorInfo(addressIP))
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
		log.Infof("If you don't have a wildcard DNS yet you can use the default %s", defaultDomain)

		if domain == "" {
			prompt := &survey.Input{
				Message: "Domain",
				Default: defaultDomain,
				Help:    "Enter your custom domain that is used to generate Ingress rules, defaults to the magic dns nip.io",
			}
			survey.AskOne(prompt, &domain, survey.Required)
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
