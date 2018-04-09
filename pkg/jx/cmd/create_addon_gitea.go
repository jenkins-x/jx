package cmd

import (
	"github.com/spf13/cobra"
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/AlecAivazis/survey.v1"
)

const (
	optionChart   = "chart"
	optionRelease = "release"

	defaultGiteaReleaseName = "gitea"
)

var (
	create_addon_gitea_long = templates.LongDesc(`
		Creates the gitea addon (hosted git server)
`)

	create_addon_gitea_example = templates.Examples(`
		# Create the gitea addon 
		jx create addon gitea
	`)
)

// CreateAddonGiteaOptions the options for the create spring command
type CreateAddonGiteaOptions struct {
	CreateAddonOptions

	Chart    string
	Username string
	Password string
	Email    string
	IsAdmin  bool
	NoUser   bool
	NoToken  bool
}

// NewCmdCreateAddonGitea creates a command object for the "create" command
func NewCmdCreateAddonGitea(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateAddonGiteaOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "gitea",
		Short:   "Create a gitea addon for hosting git repositories",
		Aliases: []string{"env"},
		Long:    create_addon_gitea_long,
		Example: create_addon_gitea_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd, "", defaultGiteaReleaseName)

	cmd.Flags().StringVarP(&options.Username, "username", "u", "", "The name for the user to create in gitea. Note that gitea tends to reject 'admin'")
	cmd.Flags().StringVarP(&options.Password, "password", "p", "", "The password for the user to create in gitea. Note that gitea tends to reject passwords less than 6 characters")
	cmd.Flags().StringVarP(&options.Email, "email", "e", "", "The email address of the new user to create in gitea")
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The version of the gitea addon to use")
	cmd.Flags().StringVarP(&options.Chart, optionChart, "c", kube.ChartGitea, "The name of the chart to use")
	cmd.Flags().BoolVarP(&options.IsAdmin, "admin", "", false, "Should the new user created be an admin of the gitea server")
	cmd.Flags().BoolVarP(&options.NoUser, "no-user", "", false, "If true disable trying to create a new user in the gitea server")
	cmd.Flags().BoolVarP(&options.NoToken, "no-token", "", false, "If true disable trying to create a new token in the gitea server")
	return cmd
}

// Run implements the command
func (o *CreateAddonGiteaOptions) Run() error {
	if o.ReleaseName == "" {
		return util.MissingOption(optionRelease)
	}
	if o.Chart == "" {
		return util.MissingOption(optionChart)
	}
	err := o.installChart(o.ReleaseName, o.Chart, o.Version, o.Namespace, true, nil)
	if err != nil {
		return err
	}
	err = o.createGitServer()
	if err != nil {
		return err
	}
	if !o.NoUser {
		// now to add the git server + a user
		if !o.BatchMode {
			if o.Username == "" {
				prompt := &survey.Input{
					Message: "Enter the user name to create in gitea: ",
				}
				err = survey.AskOne(prompt, &o.Username, nil)
				if err != nil {
					return err
				}
			}
			if o.Username != "" {
				if o.Password == "" {
					prompt := &survey.Password{
						Message: "Enter the password for the new user in gitea: ",
					}
					err = survey.AskOne(prompt, &o.Password, nil)
					if err != nil {
						return err
					}
				}
				if o.Password != "" {
					if o.Email == "" {
						prompt := &survey.Input{
							Message: "Enter the email address of the user to create in gitea: ",
						}
						err = survey.AskOne(prompt, &o.Email, nil)
						if err != nil {
							return err
						}
					}
				}
			}
		}
		if o.Username != "" && o.Password != "" && o.Email != "" {
			err = o.createGitUser()
			if err != nil {
				return err
			}
		}
	}
	if !o.NoUser && o.Username != "" && o.Password != "" {
		return o.createGitToken()
	}
	return nil
}

func (o *CreateAddonGiteaOptions) createGitServer() error {
	options := &CreateGitServerOptions{
		CreateOptions: o.CreateOptions,
	}
	options.Args = []string{"gitea"}
	return options.Run()
}

func (o *CreateAddonGiteaOptions) createGitUser() error {
	o.Printf("Generating user: %s with email: %s\n", util.ColorInfo(o.Username), util.ColorInfo(o.Email))
	options := &CreateGitUserOptions{
		CreateOptions: o.CreateOptions,
		Username:      o.Username,
		Password:      o.Password,
		Email:         o.Email,
		IsAdmin:       o.IsAdmin,
	}
	options.CommonOptions.Args = []string{}
	options.ServerFlags.ServerName = "gitea"
	return options.Run()
}

func (o *CreateAddonGiteaOptions) createGitToken() error {
	o.Printf("Generating token for user %s with email %s\n", util.ColorInfo(o.Username), util.ColorInfo(o.Email))
	options := &CreateGitTokenOptions{
		CreateOptions: o.CreateOptions,
		Username:      o.Username,
		Password:      o.Password,
	}
	options.CommonOptions.Args = []string{}
	options.ServerFlags.ServerName = "gitea"
	return options.Run()
}
