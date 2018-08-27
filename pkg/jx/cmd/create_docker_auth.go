package cmd

import (
	"io"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/AlecAivazis/survey.v1"
)

const (
	username = "user"
	host     = "host"
)

var (
	createDockerAuthLong = templates.LongDesc(`
		Creates/updates an entry for secret in the config.json for a given user, host
`)

	createDockerAuthExample = templates.Examples(`
		# Create/update docker auth entry in the config.json file
		jx create auth --host "angoothachap.private.docker.registry" --user "angoothachap" --secret "AngoothachapDockerHubToken" 
	`)
)

// CreateIssueOptions the options for the create spring command
type CreateDockerAuthOptions struct {
	CreateOptions

	Host   string
	User   string
	Secret string
}

// NewCmdCreateIssue creates a command object for the "create" command
func NewCmdCreateDockerAuth(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateDockerAuthOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "docker auth",
		Short:   "Create/update docker auth for a given host and user in the config.json file",
		Long:    createDockerAuthLong,
		Example: createDockerAuthExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Host, host, "h", "", "The docker host")
	cmd.Flags().StringVarP(&options.User, username, "u", "", "The title of the issue to create")
	cmd.Flags().StringVarP(&options.Secret, "secret", "s", "", "The secret to associate auth component of config.json")
	options.addCommonFlags(cmd)
	return cmd
}

// Run implements the command
func (o *CreateDockerAuthOptions) Run() error {
	if o.Host == "" {
		return util.MissingOption(host)
	}
	if o.User == "" {
		return util.MissingOption(username)
	}
	secret := o.Secret
	if secret == "" {
		prompt := &survey.Input{
			Message: "Please provide secret for the host: " + o.Host + "  and user: " + o.User,
		}
		survey.AskOne(prompt, &secret, nil)
	}
	kubeClient, currentNs, err := o.KubeClient()
	if err != nil {
		return err
	}
	secretFromConfig, err := kubeClient.CoreV1().Secrets(currentNs).Get("jenkins-docker-cfg", metav1.GetOptions{})
	if err != nil {
		return nil
	}
	secretFromConfig.Data["config.json"] = ""
	return nil
}
