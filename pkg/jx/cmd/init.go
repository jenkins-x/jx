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
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
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
		This command installs the Jenkins-X platform on a connected kubernetes cluster
`)

	initExample = templates.Examples(`
		jx init
`)
)

// NewCmdInit creates a command object for the generic "init" action, which
// primes a kubernetes cluster so it's ready for jenkins-x to be installed
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
		Short:   "Init Jenkins-X",
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
		log.Fatalf("init failed: %v", err)
		os.Exit(-1)
	}

	// draft init
	err = o.initDraft()
	if err != nil {
		log.Fatalf("init failed: %v", err)
		os.Exit(-1)
	}

	// install ingress
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

	log.Success("tiller running")

	return nil
}

func (o *InitOptions) initDraft() error {
	f := o.Factory
	client, _, err := f.CreateClient()
	if err != nil {
		return err
	}

	running, err := kube.IsDeploymentRunning(client, "draftd", "kube-system")
	if running {
		return nil
	}
	if err == nil && !running {
		return errors.New("existing draftd deployment found but not running, please check the kube-system namespace and resolve any issues")
	}

	if !running {
		err = o.runCommand("draft", "init", "--auto-accept")
		if err != nil {
			if !strings.Contains(err.Error(), "already exists") {
				return err
			}
		}
		err = o.runCommand("draft", "pack-repo", "add", "https://github.com/jenkins-x/draft-repo")
		if err != nil {
			if !strings.Contains(err.Error(), "already exists") {
				return err
			}
		}
	}

	err = kube.WaitForDeploymentToBeReady(client, "draftd", "kube-system", 5*time.Minute)
	if err != nil {
		return err
	}

	log.Success("draftd running")

	return nil
}
