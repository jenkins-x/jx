package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
	"path/filepath"
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

	ImportFileResolver jenkinsfile.ImportFileResolver
}

// NewCmdBuildPackApply Creates a new Command object
func NewCmdBuildPackApply(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepBuildPackApplyOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "buildpack apply",
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
	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "The directory to query to find the projects .git directory")
	cmd.Flags().StringVarP(&options.Jenkinsfile, "jenkinsfile", "", "", "The name of the Jenkinsfile to use. If not specified then 'Jenkinsfile' will be used")
	cmd.Flags().StringVarP(&options.DraftPack, "pack", "", "", "The name of the pack to use")
	cmd.Flags().BoolVarP(&options.DisableJenkinsfileCheck, "no-jenkinsfile", "", false, "Disable defaulting a Jenkinsfile if its missing")
	return cmd
}

// Run implements this command
func (o *StepBuildPackApplyOptions) Run() error {
	dir := o.Dir

	if o.ImportFileResolver == nil {
		o.ImportFileResolver = o.resolveImportFile
	}

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

	args := &InvokeDraftPack{
		Dir:                     dir,
		CustomDraftPack:         o.DraftPack,
		Jenkinsfile:             jenkinsfile,
		DefaultJenkinsfile:      defaultJenkinsfile,
		WithRename:              withRename,
		InitialisedGit:          true,
		DisableJenkinsfileCheck: o.DisableJenkinsfileCheck,
	}
	_, err := o.invokeDraftPack(args)
	if err != nil {
		return err
	}
	return nil
}

// resolveImportFile resolve an import name and file 
func (o *StepBuildPackApplyOptions) resolveImportFile(importFile *jenkinsfile.ImportFile) (string, error) {
	return importFile.File, fmt.Errorf("not implemented")
}
