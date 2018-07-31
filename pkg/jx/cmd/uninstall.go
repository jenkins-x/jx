package cmd

import (
	"fmt"
	"io"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/AlecAivazis/survey.v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type UninstallOptions struct {
	CommonOptions

	Namespace string
	Confirm   bool
}

var (
	uninstall_long = templates.LongDesc(`
		Uninstalls the Jenkins X platform from a kubernetes cluster`)
	uninstall_example = templates.Examples(`
		# Uninstall the Jenkins X platform
		jx uninstall`)
)

func NewCmdUninstall(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
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
			CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "The team namespace to uninstall. Defaults to the current namespace.")
	cmd.Flags().BoolVarP(&options.Confirm, "yes", "y", false, "Confirms we should uninstall this installation")
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
	namespace := o.Namespace
	if namespace == "" {
		namespace = kube.CurrentNamespace(config)
	}
	if o.BatchMode {
		if !o.Confirm {
			return fmt.Errorf("In batch mode you must specify the '-y' flag to confirm")
		}
	} else {
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
	}
	log.Infof("Removing installation of Jenkins X in team namespace %s\n", util.ColorInfo(namespace))

	err = o.cleanupConfig()
	if err != nil {
		return err
	}
	envNames, err := kube.GetEnvironmentNames(jxClient, namespace)
	if err != nil {
		log.Warnf("Failed to find Environments. Probably not installed yet?. Error: %s\n", err)
	}
	for _, env := range envNames {
		release := namespace + "-" + env
		err := o.Helm().StatusRelease(release)
		if err != nil {
			continue
		}
		err = o.Helm().DeleteRelease(release, true)
		if err != nil {
			log.Warnf("Failed to uninstall environment chart %s: %s\n", release, err)
		}
	}
	o.Helm().DeleteRelease("jx-prow", true)
	err = o.Helm().DeleteRelease("jenkins-x", true)
	if err != nil {
		errc := o.cleanupNamesapces(namespace, envNames)
		if errc != nil {
			errc = errors.Wrap(errc, "failed to cleanup the jenkins-x platform")
			return errc
		}
		return errors.Wrap(err, "failed to purge the jenkins-x chart")
	}
	err = jxClient.JenkinsV1().Environments(namespace).DeleteCollection(&meta_v1.DeleteOptions{}, meta_v1.ListOptions{})
	if err != nil {
		return err
	}
	err = o.cleanupNamesapces(namespace, envNames)
	if err != nil {
		return err
	}
	log.Successf("Jenkins X has been successfully uninstalled from team namespace %s", namespace)
	return nil
}

func (o *UninstallOptions) cleanupNamesapces(namespace string, envNames []string) error {
	client, _, err := o.KubeClient()
	if err != nil {
		return errors.Wrap(err, "failed to get the kube client")
	}
	err = o.deleteNamespace(namespace)
	if err != nil {
		return errors.Wrap(err, "failed to delete team namespace namespace")
	}
	for _, env := range envNames {
		envNamespace := namespace + "-" + env
		_, err := client.CoreV1().Namespaces().Get(envNamespace, meta_v1.GetOptions{})
		if err != nil {
			continue
		}
		err = o.deleteNamespace(envNamespace)
		if err != nil {
			return errors.Wrap(err, "failed to delete environment namespace")
		}
	}
	return nil
}

func (o *UninstallOptions) deleteNamespace(namespace string) error {
	client, _, err := o.KubeClient()
	if err != nil {
		return errors.Wrap(err, "failed to get the kube client")
	}
	err = client.CoreV1().Namespaces().Delete(namespace, &meta_v1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to delete the namespace '%s'", namespace)
	}
	return nil
}

func (o *UninstallOptions) cleanupConfig() error {
	authConfigSvc, err := o.Factory.CreateAuthConfigService(JenkinsAuthConfigFile)
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
