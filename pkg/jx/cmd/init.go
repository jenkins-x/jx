package cmd

import (
	"errors"
	"io"

	"os"

	"time"

	"strings"

	"fmt"

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// InitOptions the flags for running init
type InitOptions struct {
	CommonOptions
	Client clientset.Clientset
	Flags  InitFlags
}

type InitFlags struct {
	Domain   string
	Provider string
}

const (
	INGRESS_SERVICE_NAME = "jxing-nginx-ingress-controller"
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

	cmd.Flags().StringVarP(&options.Flags.Provider, "provider", "", "", "Cloud service providing the kubernetes cluster.  Supported providers: [minikube,gke,aks]")
	cmd.Flags().StringVarP(&options.Flags.Domain, "domain", "d", "", "Domain to expose ingress endpoints.  Example: jenkinsx.io")
	return cmd
}

func (o *InitOptions) Run() error {

	var err error
	o.Flags.Provider, err = o.GetCloudProvider(o.Flags.Provider)
	if err != nil {
		return err
	}

	// helm init
	err = o.initHelm()
	if err != nil {
		log.Fatalf("helm init failed: %v", err)
		os.Exit(-1)
	}

	// draft init
	err = o.initDraft()
	if err != nil {
		log.Fatalf("draft init failed: %v", err)
		os.Exit(-1)
	}

	// install ingress
	err = o.initIngress()
	if err != nil {
		log.Fatalf("ingress init failed: %v", err)
		os.Exit(-1)
	}

	return nil
}

func (o *InitOptions) initHelm() error {
	f := o.Factory
	client, _, err := f.CreateClient()
	if err != nil {
		return err
	}

	running, err := kube.IsDeploymentRunning(client, "tiller-deploy", "kube-system")
	if running {
		return nil
	}
	if err == nil && !running {
		return errors.New("existing tiller deployment found but not running, please check the kube-system namespace and resolve any issues")
	}

	if !running {
		err = o.runCommand("helm", "init")
		if err != nil {
			return err
		}
	}

	err = kube.WaitForDeploymentToBeReady(client, "tiller-deploy", "kube-system", 5*time.Minute)
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

	err = o.removeDraftRepoIfInstalled("github.com/Azure/draft")
	if err != nil {
		return err
	}

	if running || o.Flags.Provider == GKE || o.Flags.Provider == AKS {
		err = o.runCommand("draft", "init", "--auto-accept", "--client-only")

	} else {
		err = o.runCommand("draft", "init", "--auto-accept")

	}
	if err != nil {
		return err
	}

	err = o.removeDraftRepoIfInstalled("github.com/jenkins-x/draft-repo")
	if err != nil {
		return err
	}

	err = o.runCommand("draft", "pack-repo", "add", "https://github.com/jenkins-x/draft-repo")
	if err != nil {
		log.Warn("error adding pack to draft, if you are using git 2.16.1 take a look at this issue for a workaround https://github.com/jenkins-x/jx/issues/176#issuecomment-361897946")
		return err
	}

	if !running && o.Flags.Provider != GKE && o.Flags.Provider != AKS {
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
	text, err := o.getCommandOutput("", "draft", "pack-repo", "list")
	if err != nil {
		// if pack-repo list fails then it's because no repos currently exist
		return nil
	}
	if strings.Contains(text, repo) {
		log.Warnf("existing repo %s found, we recommend to remove and let draft init recreate, shall we do this now?", repo)
		return o.runCommandInteractive(true, "draft", "pack-repo", "remove", repo)
	}
	return nil
}

func (o *InitOptions) initIngress() error {
	f := o.Factory
	client, _, err := f.CreateClient()
	if err != nil {
		return err
	}

	currentContext, err := o.getCommandOutput("", "kubectl", "config", "current-context")
	if err != nil {
		return err
	}
	if currentContext == "minikube" {
		if  o.Flags.Provider == "" {
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
	podList, err := client.CoreV1().Pods("kube-system").List(options)
	if err != nil {
		return err
	}

	if podList != nil && len(podList.Items) > 0 {
		log.Info("existing nginx ingress controller found, no need to install")
		return nil
	}

	installIngressController := false
	prompt := &survey.Confirm{
		Message: "No existing ingress controller found in the kube-system namespace, shall we install one?",
		Default: true,
		Help:    "An ingress controller works with an external loadbalancer so you can access Jenkins X and your applications",
	}
	survey.AskOne(prompt, &installIngressController, nil)

	if !installIngressController {
		return nil
	}

	i := 0
	for {
		err = o.runCommand("helm", "install", "--name", "jxing", "stable/nginx-ingress", "--namespace", "kube-system")
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

	err = kube.WaitForDeploymentToBeReady(client, INGRESS_SERVICE_NAME, "kube-system", 10*time.Minute)
	if err != nil {
		return err
	}

	if o.Flags.Provider == GKE || o.Flags.Provider == AKS {
		log.Info("Waiting for external loadbalancer to be created and update the nginx-ingress-controller service in kube-system namespace\n")
		err = kube.WaitForExternalIP(client, INGRESS_SERVICE_NAME, "kube-system", 10*time.Minute)
		if err != nil {
			return err
		}

		log.Infof("External loadbalancer created\n")

		o.Flags.Domain, err = o.GetDomain(client, o.Flags.Domain, o.Flags.Domain)
		if err != nil {
			return err
		}
	}

	log.Success("nginx ingress controller installed and configured")

	return nil
}

func (o *CommonOptions) GetDomain(client *kubernetes.Clientset, domain string, provider string) (string, error) {
	var address string
	if provider == MINIKUBE {
		ip, err := o.getCommandOutput("", "minikube", "ip")
		if err != nil {
		  return "", err
		}
		address = ip
	} else {
		svc, err := client.CoreV1().Services("kube-system").Get(INGRESS_SERVICE_NAME, meta_v1.GetOptions{})
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
