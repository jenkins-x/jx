package cmd

import (
	"fmt"
	"go/build"
	"io"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

const (
	optionGroupWithVersion = "group-with-version"
	optionInputPackage     = "input-package"
	optionOutputPackage    = "output-package"

	optionInputBase       = "input-base"
	optionOutputBase      = "output-base"
	optionBoilerplateFile = "boilerplate-file"
	optionModuleName      = "module-name"
)

// CreateClientOptions the options for the create client command
type CreateClientOptions struct {
	CreateOptions
	OutputBase          string
	BoilerplateFile     string
	GroupsWithVersions  []string
	InputPackage        string
	GoPathInputPackage  string
	GoPathOutputPackage string
	GoPathOutputBase    string
	OutputPackage       string
	ClientGenVersion    string
	InputBase           string
}

var (
	createClientLong = templates.LongDesc(`Generates clients, OpenAPI spec and API docs for custom resources.

Custom resources are defined using Go structs.

Available generators include:

* jx create client openapi # Generates OpenAPI specs, required to generate API docs and clients other than Go
* jx create client docs # Generates API docs from the OpenAPI specs
* jx create client go # Generates a Go client directly from custom resources

`)

	createClientExample = templates.Examples(`
`)
)

// NewCmdCreateClient creates clients for CRDs
func NewCmdCreateClient(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	o := &CreateClientOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "client",
		Short:   "Creates clients for Custom Resources",
		Long:    createClientLong,
		Example: createClientExample,

		Run: func(cmd *cobra.Command, args []string) {
			o.Cmd = cmd
			o.Args = args
			err := o.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateClientDocs(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateClientOpenAPI(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateClientGo(f, in, out, errOut))

	return cmd
}

// Run implements this command
func (o *CreateClientOptions) Run() error {
	return o.Cmd.Help()
}

func (o *CreateClientOptions) configureGoPath() error {

	if build.Default.GOPATH == "" {
		return errors.New("GOPATH must be set")
	}

	inputPath := filepath.Join(o.InputBase, o.InputPackage)
	outputPath := filepath.Join(o.OutputBase, o.OutputPackage)

	if !strings.HasPrefix(inputPath, build.Default.GOPATH) {
		return errors.Errorf("input %s is not in GOPATH (%s)", inputPath, build.Default.GOPATH)
	}

	if !strings.HasPrefix(outputPath, build.Default.GOPATH) {
		return errors.Errorf("output %s is not in GOPATH (%s)", outputPath, build.Default.GOPATH)
	}

	// Work out the InputPackage relative to GOROOT
	o.GoPathInputPackage = strings.TrimPrefix(inputPath,
		fmt.Sprintf("%s/", filepath.Join(build.Default.GOPATH, "src")))

	// Work out the OutputPackage relative to GOROOT
	o.GoPathOutputPackage = strings.TrimPrefix(outputPath,
		fmt.Sprintf("%s/", filepath.Join(build.Default.GOPATH, "src")))
	return nil
}
