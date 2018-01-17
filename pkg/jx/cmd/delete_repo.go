package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/gits"
	"gopkg.in/AlecAivazis/survey.v1"
	"github.com/jenkins-x/jx/pkg/auth"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

var (
	delete_repo_long = templates.LongDesc(`
		Deletes one or more repositories.

		This command will require the delete repo role on your Persona Access Token. 

		Note that command will ask for confirmation before doing anything!
`)

	delete_repo_example = templates.Examples(`
		# Selects the repositories to delete from the given github organisation
		jx delete repo --github --org myname 

        # Selects all the repositories in organisation myname that contain 'foo'
        # you get a chance to select which ones not to delete
		jx delete repo --github --org myname --all --filter foo 
	`)
)

// DeleteRepoOptions the options for the create spring command
type DeleteRepoOptions struct {
	CreateOptions

	Organisation string
	Repository   string
	GitHost      string
	GitHub       bool
	SelectAll    bool
	SelectFilter string
}

// NewCmdDeleteRepo creates a command object for the "delete repo" command
func NewCmdDeleteRepo(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteRepoOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "repo",
		Short:   "Deletes one or more git repositories",
		Aliases: []string{"repository"},
		Long:    delete_repo_long,
		Example: delete_repo_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	//addDeleteFlags(cmd, &options.CreateOptions)

	cmd.Flags().StringVarP(&options.Organisation, "org", "o", "", "Specify the git provider organisation to import the project into (if it is not already in one)")
	cmd.Flags().StringVarP(&options.Repository, "name", "n", "", "Specify the git repository name to import the project into (if it is not already in one)")
	cmd.Flags().StringVarP(&options.GitHost, "git-host", "g", "", "The Git server host if not using GitHub")
	cmd.Flags().BoolVarP(&options.GitHub, "github", "", false, "If you wis to pick the repositories from GitHub to import")
	cmd.Flags().BoolVarP(&options.SelectAll, "all", "", false, "If selecting projects to import from a git provider this defaults to selecting them all")
	cmd.Flags().StringVarP(&options.SelectFilter, "filter", "", "", "If selecting projects to import from a git provider this filters the list of repositories")
	return cmd
}

// Run implements the command
func (o *DeleteRepoOptions) Run() error {
	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	var server *auth.AuthServer
	config := authConfigSvc.Config()
	if o.GitHub {
		server = config.GetOrCreateServer(gits.GitHubHost)
	} else {
		if o.GitHost != "" {
			server = config.GetOrCreateServer(o.GitHost)
		} else {
			server, err = config.PickServer("Pick the git server to search for repositories")
			if err != nil {
				return err
			}
		}
	}
	if server == nil {
		return fmt.Errorf("No git server provided!");
	}
	userAuth, err := config.PickServerUserAuth(server, "git user name")
	if err != nil {
		return err
	}
	provider, err := gits.CreateProvider(server, &userAuth)
	if err != nil {
		return err
	}
	username := userAuth.Username
	org := o.Organisation
	if org == "" {
		org, err = gits.PickOrganisation(provider, username)
		if err != nil {
			return err
		}
	}
	repos, err := gits.PickRepositories(provider, org, "Which repositories do you want to delete:", o.SelectAll, o.SelectFilter)
	if err != nil {
		return err
	}

	names := []string{}
	for _, r := range repos {
		names = append(names, r.Name)
	}

	o.warnf("You are about to delete these repositories on the git provider. This operation CANNOT be undone!")

	flag := false
	prompt := &survey.Confirm{
		Message: "Are you sure you want to delete these all these repositories?",
		Default: false,
	}
	err = survey.AskOne(prompt, &flag, nil)
	if err != nil {
		return err
	}
	if !flag {
		return nil
	}

	owner := org
	if owner == "" {
		owner = username
	}
	info := util.ColorInfo
	for _, r := range repos {
		name := r.Name
		err = provider.DeleteRepository(owner, name)
		if err != nil {
			o.warnf("%s\n", err)
		} else {
			o.Printf("Deleted repository %s/%s\n", info(owner), info(name))
		}
	}
	return nil
}
