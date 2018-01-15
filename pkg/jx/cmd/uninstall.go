package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"fmt"

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"gopkg.in/AlecAivazis/survey.v1"
)

type UninstallOptions struct {
	CommonOptions
}

var (
	uninstall_long = templates.LongDesc(`
		Uninstalls the Jenkins X platform from a kubernetes cluster`)
	uninstall_example = templates.Examples(`
		# Uninstall the Jenkins X platform
		jx uninstall`)
)

func NewCmdUninstall(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &UninstallOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "uninstall",
		Short:   "Uninstall the Jenkins X platform",
		Long:    uninstall_long,
		Example: uninstall_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	return cmd
}

func (o *UninstallOptions) Run() error {
	config, _, err := kube.LoadConfig()
	if err != nil {
		return err
	}
	server := kube.CurrentServer(config)
	namespace := kube.CurrentNamespace(config)
	confirm := &survey.Confirm{
		Message: fmt.Sprintf("Are you sure you wish to remove the Jenkins X platform from the '%s' namespace on cluster '%s'? :", namespace, server),
		Default: false,
	}
	flag := false
	err = survey.AskOne(confirm, &flag, nil)
	if err != nil {
		return err
	}
	if !flag {
		return nil
	}
	err = o.runCommand("helm", "delete", "--purge", "jenkins-x")
	if err != nil {
		return err
	}
	log.Success("Jenkins X has been successfully uninstalled ")
	return nil
}
