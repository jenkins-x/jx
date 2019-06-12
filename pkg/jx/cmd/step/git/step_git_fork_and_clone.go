package git

import (
	"os"

	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"

	"github.com/jenkins-x/jx/pkg/log"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
)

// StepGitForkAndCloneOptions contains the command line flags
type StepGitForkAndCloneOptions struct {
	opts.StepOptions
	Dir         string
	BaseRef     string
	PrintOutDir bool

	OutputDir string
}

var (
	// StepGitForkAndCloneLong command long description
	StepGitForkAndCloneLong = templates.LongDesc(`
		This pipeline step will clone a git repo, creating a fork if required. The fork is created if the owner of the 
		repo is not the current git user (and that forking the git repo is allowed).

`)
	// StepGitForkAndCloneExample command example
	StepGitForkAndCloneExample = templates.Examples(`
		# Fork and clone the jx repo
		jx step git fork-and-clone https://github.com/jenkins-x/jx.git

`)
)

// NewCmdStepGitForkAndClone create the 'step git envs' command
func NewCmdStepGitForkAndClone(commonOpts *opts.CommonOptions) *cobra.Command {
	options := StepGitForkAndCloneOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "fork-and-clone",
		Short:   "Forks and clones a git repo",
		Long:    StepGitForkAndCloneLong,
		Example: StepGitForkAndCloneExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Dir, "dir", "", "", "The directory in which the git repo is checked out, by default the working directory")
	cmd.Flags().StringVarP(&options.BaseRef, "base", "", "master", "The base ref to start from")
	cmd.Flags().BoolVarP(&options.BatchMode, opts.OptionBatchMode, "b", false, "Enable batch mode")
	cmd.Flags().BoolVarP(&options.PrintOutDir, "print-out-dir", "", false, "prints the directory the fork has been cloned to on stdout")
	return cmd
}

// Run implements the command
func (o *StepGitForkAndCloneOptions) Run() error {
	if len(o.Args) != 1 {
		return errors.Errorf("Must specify exactly one git url but was %v", o.Args)
	}
	if o.Dir == "" {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		o.Dir = dir
	}
	gitURL := o.Args[0]
	provider, err := o.GitProviderForURL(gitURL, "git username")
	if err != nil {
		return errors.Wrapf(err, "getting git provider for %s", gitURL)
	}
	dir, baseRef, gitInfo, err := gits.ForkAndPullPullRepo(gitURL, o.Dir, o.BaseRef, "", provider, o.Git(), nil)
	if err != nil {
		return errors.Wrapf(err, "forking and pulling %s", gitURL)
	}
	o.OutDir = dir
	if o.PrintOutDir {
		// Output the directory so it can be used in a script
		log.Logger().Infof(dir)
	}
	if gitInfo.Fork {
		log.Logger().Infof("Forked %s and pulled it into %s checking out %s", gitURL, dir, baseRef)
	} else {
		log.Logger().Infof("Pulled %s (%s) into %s", gitURL, baseRef, dir)
	}

	return nil
}
