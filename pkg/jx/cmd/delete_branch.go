package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

var (
	deleteBranchLong = templates.LongDesc(`
		Deletes one or more branches in repositories.

		Note that command will ask for confirmation before doing anything!
`)

	deleteBranchExample = templates.Examples(`
		# Selects the repositories to delete from the given GitHub organisation
		jx delete branch --org myname --name myrepo -f updatebot- -a
	`)
)

// DeleteBranchOptions the options for the create spring command
type DeleteBranchOptions struct {
	CreateOptions

	Organisation      string
	Repositories      []string
	GitHost           string
	GitHub            bool
	SelectAll         bool
	SelectFilter      string
	SelectAllRepos    bool
	SelectFilterRepos string
}

// NewCmdDeleteBranch creates a command object for the "delete repo" command
func NewCmdDeleteBranch(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteBranchOptions{
		CreateOptions: CreateOptions{
			CommonOptions: commoncmd.CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "branch",
		Short:   "Deletes one or more branches in git repositories",
		Aliases: []string{"repository"},
		Long:    deleteBranchLong,
		Example: deleteBranchExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	//addDeleteFlags(cmd, &options.CreateOptions)

	cmd.Flags().StringVarP(&options.Organisation, "org", "o", "", "Specify the Git provider organisation that includes the repository to delete")
	cmd.Flags().StringArrayVarP(&options.Repositories, "name", "n", []string{}, "Specify the Git repository names to delete")
	cmd.Flags().StringVarP(&options.GitHost, "git-host", "g", "", "The Git server host if not using GitHub")
	cmd.Flags().BoolVarP(&options.GitHub, "github", "", false, "If you wish to pick the repositories from GitHub to import")
	cmd.Flags().BoolVarP(&options.SelectAll, "all", "a", false, "If selecting branches to remove this defaults to selecting them all")
	cmd.Flags().StringVarP(&options.SelectFilter, "filter", "f", "", "If selecting branches to remove this filters the list of repositories")
	cmd.Flags().BoolVarP(&options.SelectAllRepos, "all-repos", "", false, "If selecting projects to remove branches this defaults to selecting them all")
	cmd.Flags().StringVarP(&options.SelectFilterRepos, "filter-repos", "", "", "If selecting projects to remove brancehs this filters the list of repositories")
	cmd.Flags().BoolVarP(&options.BatchMode, "batch-mode", "b", false, "Run without being prompted. WARNING! You will not be asked to confirm deletions if you use this flag.")
	return cmd
}

// Run implements the command
func (o *DeleteBranchOptions) Run() error {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
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
			server, err = config.PickServer("Pick the Git server to search for repositories", o.BatchMode, o.In, o.Out, o.Err)
			if err != nil {
				return err
			}
		}
	}
	if server == nil {
		return fmt.Errorf("No Git server provided")
	}
	userAuth, err := config.PickServerUserAuth(server, "Git user name", o.BatchMode, "", o.In, o.Out, o.Err)
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
		org, err = gits.PickOrganisation(provider, username, o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
	}

	if org == "" {
		org = username
	}

	names := o.Repositories
	if len(names) == 0 {
		repos, err := gits.PickRepositories(provider, org, "Which repositories do you want to remove branches from:", o.SelectAllRepos, o.SelectFilterRepos, o.In, o.Out, o.Err)
		if err != nil {
			return err
		}

		for _, r := range repos {
			names = append(names, r.Name)
		}
	}

	for _, name := range names {
		repo, err := provider.GetRepository(org, name)
		if err != nil {
			return errors.Wrapf(err, "Failed to find repository for %s/%s", org, name)
		}

		dir, err := o.cloneOrPullRepository(org, name, repo.SSHURL)
		if err != nil {
			return errors.Wrapf(err, "Failed to clone/pull repository for %s/%s", org, name)
		}

		err = o.Git().PullRemoteBranches(dir)
		if err != nil {
			return errors.Wrapf(err, "Failed to pull remote branches for %s/%s", org, name)
		}
		branchNames, err := o.Git().RemoteBranchNames(dir, "remotes/origin/")
		if err != nil {
			return errors.Wrapf(err, "Failed to get remote branches for %s/%s", org, name)
		}

		branches, err := util.SelectNamesWithFilter(branchNames, "Which remote branches do you to to delete: ", o.SelectAll, o.SelectFilter, "", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
		if len(branches) == 0 {
			return fmt.Errorf("No branches selected!")
		}
		if !o.BatchMode {
			log.Warnf("You are about to delete these branches '%s' on the Git provider. This operation CANNOT be undone!",
				strings.Join(branches, ","))

			flag := false
			prompt := &survey.Confirm{
				Message: "Are you sure you want to delete these all these branches?",
				Default: false,
			}
			err = survey.AskOne(prompt, &flag, nil, surveyOpts)
			if err != nil {
				return err
			}
			if !flag {
				return nil
			}
		}

		info := util.ColorInfo
		for _, branch := range branches {
			err := o.Git().DeleteRemoteBranch(dir, "origin", branch)
			if err != nil {
				return errors.Wrapf(err, "Failed to delete remote branche %s from %s/%s", branch, org, name)
			}
			log.Infof("Deleted branch in repo %s/%s branch: %s\n", info(org), info(name), info(branch))
		}
	}
	return nil
}

func (o *DeleteBranchOptions) cloneOrPullRepository(org string, repo string, gitURL string) (string, error) {
	environmentsDir, err := util.EnvironmentsDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(environmentsDir, org, repo)

	// now lets clone the fork and push it...
	exists, err := util.FileExists(dir)
	if err != nil {
		return dir, err
	}

	if exists {
		// lets check the git remote URL is setup correctly
		err = o.Git().SetRemoteURL(dir, "origin", gitURL)
		if err != nil {
			return dir, err
		}
		err = o.Git().Stash(dir)
		return dir, err
	} else {
		err := os.MkdirAll(dir, util.DefaultWritePermissions)
		if err != nil {
			return dir, fmt.Errorf("Failed to create directory %s due to %s", dir, err)
		}
		info := util.ColorInfo
		log.Infof("Cloning repository %s/%s to %s\n", info(org), info(repo), info(dir))
		err = o.Git().Clone(gitURL, dir)
		if err != nil {
			return dir, err
		}
		err = o.Git().SetRemoteURL(dir, "origin", gitURL)
		if err != nil {
			return dir, err
		}
		return dir, err
	}
}
