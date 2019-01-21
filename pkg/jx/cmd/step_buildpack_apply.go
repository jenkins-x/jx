package cmd

import (
	"io"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

var (
	createJenkinsfileLong = templates.LongDesc(`
		Applies the build pack for a project to add any missing files like a Jenkinsfile
`)

	createJenkinsfileExample = templates.Examples(`
		# applies the current build pack for the current team adding any missing files like Jenkinsfile
		jx step buildpack apply

		# applies the 'maven' build pack to the current project
		jx step buildpack apply --pack maven

			`)
)

// StepBuildPackApplyOptions contains the command line flags
type StepBuildPackApplyOptions struct {
	StepOptions

	Dir                     string
	Jenkinsfile             string
	DraftPack               string
	DisableJenkinsfileCheck bool
}

// NewCmdStepBuildPackApply Creates a new Command object
func NewCmdStepBuildPackApply(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepBuildPackApplyOptions{
		StepOptions: StepOptions{
			CommonOptions: commoncmd.CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "apply",
		Short:   "Applies the current teams build pack to the project to add any missing resources like a Jenkinsfile",
		Long:    createJenkinsfileLong,
		Example: createJenkinsfileExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.AddCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "The directory to query to find the projects .git directory")
	cmd.Flags().StringVarP(&options.Jenkinsfile, "jenkinsfile", "", "", "The name of the Jenkinsfile to use. If not specified then 'Jenkinsfile' will be used")
	cmd.Flags().StringVarP(&options.DraftPack, "pack", "", "", "The name of the pack to use")
	cmd.Flags().BoolVarP(&options.DisableJenkinsfileCheck, "no-jenkinsfile", "", false, "Disable defaulting a Jenkinsfile if its missing")
	return cmd
}

// Run implements this command
func (o *StepBuildPackApplyOptions) Run() error {
	var err error
	dir := o.Dir
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	settings, err := o.CommonOptions.TeamSettings()
	if err != nil {
		return err
	}
	log.Infof("build pack is %s\n", settings.BuildPackURL)

	defaultJenkinsfile := filepath.Join(dir, jenkins.DefaultJenkinsfile)
	jenkinsfile := jenkins.DefaultJenkinsfile
	withRename := false
	if o.Jenkinsfile != "" {
		jenkinsfile = o.Jenkinsfile
		withRename = true
	}
	if !filepath.IsAbs(jenkinsfile) {
		jenkinsfile = filepath.Join(dir, jenkinsfile)
	}

	args := &commoncmd.InvokeDraftPack{
		Dir:                     dir,
		CustomDraftPack:         o.DraftPack,
		Jenkinsfile:             jenkinsfile,
		DefaultJenkinsfile:      defaultJenkinsfile,
		WithRename:              withRename,
		InitialisedGit:          true,
		DisableJenkinsfileCheck: o.DisableJenkinsfileCheck,
	}
	_, err = o.InvokeDraftPack(args)
	if err != nil {
		return err
	}
	return nil
}
