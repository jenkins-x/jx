package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	deleteGitServerLong = templates.LongDesc(`
		Deletes one or more Git servers from your local settings
`)

	deleteGitServerExample = templates.Examples(`
		# Deletes a Git provider
		jx delete git server MyProvider
	`)
)

// DeleteGitServerOptions the options for the create spring command
type DeleteGitServerOptions struct {
	CommonOptions

	IgnoreMissingServer bool
}

// NewCmdDeleteGitServer defines the command
func NewCmdDeleteGitServer(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteGitServerOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "server",
		Short:   "Deletes one or more Git servers",
		Long:    deleteGitServerLong,
		Example: deleteGitServerExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.IgnoreMissingServer, "ignore-missing", "i", false, "Silently ignore attempts to remove a Git server name that does not exist")
	return cmd
}

// Run implements the command
func (o *DeleteGitServerOptions) Run() error {
	args := o.Args
	if len(args) == 0 {
		return fmt.Errorf("Missing Git server name argument")
	}
	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()

	serverNames := config.GetServerNames()
	for _, arg := range args {
		idx := config.IndexOfServerName(arg)
		if idx < 0 {
			if o.IgnoreMissingServer {
				return nil
			}
			return util.InvalidArg(arg, serverNames)
		}
		server := config.Servers[idx]
		if server != nil {
			err = o.deleteServerResources(server)
			if err != nil {
				return err
			}
		}
		config.Servers = append(config.Servers[0:idx], config.Servers[idx+1:]...)
	}
	err = authConfigSvc.SaveConfig()
	if err != nil {
		return err
	}
	log.Infof("Deleted Git servers: %s from local settings\n", util.ColorInfo(strings.Join(args, ", ")))
	return nil
}

func (o *DeleteGitServerOptions) deleteServerResources(server *auth.AuthServer) error {
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	kubeClient, err := o.KubeClient()
	if err != nil {
		return err
	}
	secrets, err := o.LoadPipelineSecrets(kube.ValueKindGit, server.Kind)
	if err != nil {
		return err
	}
	for _, secret := range secrets.Items {
		ann := secret.Annotations
		if ann != nil && ann[kube.AnnotationURL] == server.URL {
			name := secret.Name
			log.Infof("Deleting Secret %s\n", util.ColorInfo(name))

			err = kubeClient.CoreV1().Secrets(ns).Delete(name, nil)
			if err != nil {
				return err
			}
		}
	}
	gitServiceResources := jxClient.JenkinsV1().GitServices(ns)
	gitServices, err := gitServiceResources.List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, gitService := range gitServices.Items {
		if gitService.Spec.URL == server.URL {
			name := gitService.Name
			log.Infof("Deleting GitService %s\n", util.ColorInfo(name))
			err = gitServiceResources.Delete(name, nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
