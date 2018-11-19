package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
)

var (
	createJenkinsfileLong = templates.LongDesc(`
		Creates a Knative jenkinsfile resource for a project
`)

	createJenkinsfileExample = templates.Examples(`
		# create a Knative jenkinsfile and render to the console
		jx step create jenkinsfile

		# create a Knative jenkinsfile
		jx step create jenkinsfile -o myjenkinsfile.yaml

			`)
)

// StepCreateJenkinsfileOptions contains the command line flags
type StepCreateJenkinsfileOptions struct {
	StepOptions

	Dir       string
	OutputDir string

	ImportFileResolver jenkinsfile.ImportFileResolver
}

// NewCmdCreateJenkinsfile Creates a new Command object
func NewCmdCreateJenkinsfile(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepCreateJenkinsfileOptions{
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
		Use:     "create jenkinsfile",
		Short:   "Creates a Jenkinsfile for a project using build packs and templates",
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
	cmd.Flags().StringVarP(&options.OutputDir, "output-dir", "o", "", "The directory where the generated jenkinsfile yaml files will be output to")
	return cmd
}

// Run implements this command
func (o *StepCreateJenkinsfileOptions) Run() error {
	return nil
}


// GenerateJenkinsfile generates the jenkinsfile 
func (o *StepCreateJenkinsfileOptions) GenerateJenkinsfile(arguments *jenkinsfile.CreateJenkinsfileArguments) error {
	err := arguments.Validate()
	if err != nil {
		return err
	}
	resolver := o.ImportFileResolver
	if resolver == nil {
		resolver = o.resolveImportFile
	}
	config, err := jenkinsfile.LoadPipelineConfig(arguments.ConfigFile, resolver)
	if err != nil {
		return err
	}

	templateFile := arguments.TemplateFile

	data, err := ioutil.ReadFile(templateFile)
	if err != nil {
		return errors.Wrapf(err, "failed to load template %s", templateFile)
	}

	t, err := template.New("myJenkinsfile").Parse(string(data))
	if err != nil {
		return errors.Wrapf(err, "failed to parse template %s", templateFile)
	}
	outFile := arguments.OutputFile
	outDir, _ := filepath.Split(outFile)
	err = os.MkdirAll(outDir, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to make directory %s", outDir)
	}
	file, err := os.Create(outFile)
	if err != nil {
		return errors.Wrapf(err, "failed to create file %s", outFile)
	}
	defer file.Close()

	err = t.Execute(file, config)
	if err != nil {
		return errors.Wrapf(err, "failed to write file %s", outFile)
	}
	return nil
}

// resolveImportFile resolve an import name and file 
func (o *StepCreateJenkinsfileOptions) resolveImportFile(importFile *jenkinsfile.ImportFile) (string, error) {
	return importFile.File, fmt.Errorf("not implemented")
}
