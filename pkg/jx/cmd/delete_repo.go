package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
)

var (
	deleteRepoLong = templates.LongDesc(`
		Deletes one or more repositories.

		This command will require the delete repo role on your Persona Access Token. 

		Note that command will ask for confirmation before doing anything!
`)

	deleteRepoExample = templates.Examples(`
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
	Repositories []string
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
		Long:    deleteRepoLong,
		Example: deleteRepoExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	//addDeleteFlags(cmd, &options.CreateOptions)

	cmd.Flags().StringVarP(&options.Organisation, "org", "o", "", "Specify the git provider organisation that includes the repository to delete")
	cmd.Flags().StringArrayVarP(&options.Repositories, "name", "n", []string{}, "Specify the git repository names to delete")
	cmd.Flags().StringVarP(&options.GitHost, "git-host", "g", "", "The Git server host if not using GitHub")
	cmd.Flags().BoolVarP(&options.GitHub, "github", "", false, "If you wish to pick the repositories from GitHub to import")
	cmd.Flags().BoolVarP(&options.SelectAll, "all", "a", false, "If selecting projects to import from a git provider this defaults to selecting them all")
	cmd.Flags().StringVarP(&options.SelectFilter, "filter", "f", "", "If selecting projects to import from a git provider this filters the list of repositories")
	cmd.Flags().BoolVarP(&options.BatchMode, "batch-mode", "b", false, "Run without being prompted. WARNING! You will not be asked to confirm deletions if you use this flag.")
	return cmd
}

// Run implements the command
func (o *DeleteRepoOptions) Run() error {
	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	var server *auth.AuthServer
	config := authConfigSvc.Config()
	if o.GitHub {
		server = config.GetOrCreateServer(gits.GitHubURL)
	} else {
		if o.GitHost != "" {
			server = config.GetOrCreateServer(o.GitHost)
		} else {
			server, err = config.PickServer("Pick the git server to search for repositories", o.BatchMode)
			if err != nil {
				return err
			}
		}
	}
	if server == nil {
		return fmt.Errorf("No git server provided")
	}
	userAuth, err := config.PickServerUserAuth(server, "git user name", o.BatchMode)
	if err != nil {
		return err
	}
	provider, err := gits.CreateProvider(server, userAuth, o.Git())
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

	if org == "" {
		org = username
	}

	names := o.Repositories
	if len(names) == 0 {
		repos, err := gits.PickRepositories(provider, org, "Which repositories do you want to delete:", o.SelectAll, o.SelectFilter)
		if err != nil {
			return err
		}

		for _, r := range repos {
			names = append(names, r.Name)
		}
	}

	if !o.BatchMode {
		log.Warnf("You are about to delete these repositories '%s' on the git provider. This operation CANNOT be undone!",
			strings.Join(names, ","))

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
	}

	owner := org
	if owner == "" {
		owner = username
	}
	info := util.ColorInfo
	for _, name := range names {
		err = provider.DeleteRepository(owner, name)
		if err != nil {
			log.Warnf("Ensure Git Token has delete repo permissions or manually delete, for GitHub check https://github.com/settings/tokens\n")
			log.Warnf("%s\n", err)
		} else {
			log.Infof("Deleted repository %s/%s\n", info(owner), info(name))
		}
	}
	return nil
}
