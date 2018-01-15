package cmd

import (
	"errors"
	"io"

	"os"

	"time"

	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// InitOptions the flags for running init
type InitOptions struct {
	CommonOptions
	Client   clientset.Clientset
	Flags    InitFlags
	Provider KubernetesProvider
}

type InitFlags struct {
}

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

	return cmd
}

func (o *InitOptions) Run() error {

	// helm init
	err := o.initHelm()
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

	if running {
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
		return err
	}

	if !running {
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
	}
	survey.AskOne(prompt, &installIngressController, nil)

	if !installIngressController {
		return nil
	}

	err = o.runCommand("helm", "install", "stable/nginx-ingress", "--namespace", "kube-system")
	if err != nil {
		return err
	}

	log.Success("nginx ingress controller installed and configured")

	return nil
}
