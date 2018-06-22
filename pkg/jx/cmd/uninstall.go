package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"fmt"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/jx/cmd/util"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"gopkg.in/AlecAivazis/survey.v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	jxClient, _, err := o.JXClient()
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

	err = o.cleanupConfig()
	if err != nil {
		return err
	}
	envNames, err := kube.GetEnvironmentNames(jxClient, namespace)
	if err != nil {
		log.Warnf("Failed to find Environments. Probably not installed yet?. Error: %s\n", err)
	}
	helmBinary, err := o.TeamHelmBin()
	if err != nil {
		return err
	}
	for _, env := range envNames {
		release := namespace + "-" + env
		err := o.runCommandQuietly(helmBinary, "status", release)
		if err != nil {
			continue
		}
		err = o.runCommand(helmBinary, "delete", "--purge", release)
		if err != nil {
			log.Warnf("Failed to uninstall environment chart %s: %s\n", release, err)
		}
	}
	err = o.runCommand(helmBinary, "delete", "--purge", "jenkins-x")
	if err != nil {
		return err
	}
	err = jxClient.JenkinsV1().Environments(namespace).DeleteCollection(&meta_v1.DeleteOptions{}, meta_v1.ListOptions{})
	if err != nil {
		return err
	}

	client, _, err := o.KubeClient()
	if err != nil {
		return err
	}
	err = client.CoreV1().Namespaces().Delete(namespace, &meta_v1.DeleteOptions{})
	if err != nil {
		return err
	}
	for _, env := range envNames {
		envNamespace := namespace + "-" + env
		_, err := client.CoreV1().Namespaces().Get(envNamespace, meta_v1.GetOptions{})
		if err != nil {
			continue
		}
		err = client.CoreV1().Namespaces().Delete(envNamespace, &meta_v1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	log.Success("Jenkins X has been successfully uninstalled ")
	return nil
}

func (o *UninstallOptions) cleanupConfig() error {
	authConfigSvc, err := o.Factory.CreateAuthConfigService(util.JenkinsAuthConfigFile)
	if err != nil {
		return nil
	}
	server := authConfigSvc.Config().CurrentServer
	err = authConfigSvc.DeleteServer(server)
	if err != nil {
		return err
	}

	chartConfigSvc, err := o.Factory.CreateChartmuseumAuthConfigService()
	if err != nil {
		return err
	}
	server = chartConfigSvc.Config().CurrentServer
	return chartConfigSvc.DeleteServer(server)
}
