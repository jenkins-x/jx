package cmd

import (
	"go/build"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"

	"github.com/jenkins-x/jx/pkg/log"

	"github.com/jenkins-x/jx/pkg/util"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// CreateClientGoOptions the options for the create client go command
type CreateClientGoOptions struct {
	CreateClientOptions
	Generators []string
}

var (
	createClientGoLong = templates.LongDesc(`This command code generates clients for the specified custom resources.
 
`)

	createClientGoExample = templates.Examples(`
		# lets generate a client
		jx create client
			--output-package=github.com/jenkins-x/jx/pkg/client \
			--input-package=github.com/jenkins-x/pkg-apis \
			--group-with-version=jenkins.io:v1
		
		# You will normally want to add a target to your Makefile that looks like:

		generate-clients:
			jx create client
				--output-package=github.com/jenkins-x/jx/pkg/client \
				--input-package=github.com/jenkins-x/jx/pkg/apis \
				--group-with-version=jenkins.io:v1
		
		# and then call:

		make generate-clients
`)
)

// NewCmdCreateClientGo creates the command
func NewCmdCreateClientGo(f Factory, in terminal.FileReader, out terminal.FileWriter,
	errOut io.Writer) *cobra.Command {
	o := &CreateClientGoOptions{
		CreateClientOptions: CreateClientOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					In:      in,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "go",
		Short:   "Creates Go client for Custom Resources",
		Long:    createClientGoLong,
		Example: createClientGoExample,

		Run: func(cmd *cobra.Command, args []string) {
			o.Cmd = cmd
			o.Args = args
			err := o.Run()
			CheckErr(err)
		},
	}

	availableGenerators := []string{
		"deepcopy",
		"clientset",
		"listers",
		"informers",
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Warnf("Error getting working directory for %v\n", err)
	}

	cmd.Flags().StringArrayVarP(&o.Generators, "generator", "", availableGenerators, "Enable a generator")
	cmd.Flags().StringVarP(&o.OutputBase, "output-base", "", wd, "Output base directory, "+
		"by the current working directory")
	cmd.Flags().StringVarP(&o.BoilerplateFile, optionBoilerplateFile, "", "custom-boilerplate.go.txt",
		"Custom boilerplate to add to all files if the file is missing it will be ignored")
	cmd.Flags().StringArrayVarP(&o.GroupsWithVersions, optionGroupWithVersion, "g", make([]string, 0),
		"group name:version (e.g. jenkins.io:v1) to generate, must specify at least once")
	cmd.Flags().StringVarP(&o.InputPackage, optionInputPackage, "i", "", "Input package, must specify")
	cmd.Flags().StringVarP(&o.OutputPackage, optionOutputPackage, "o", "", "Output package, must specify")
	cmd.Flags().StringVarP(&o.ClientGenVersion, "client-generator-version", "", "kubernetes-1.11.3",
		"Version (really a commit-ish) of github.com/kubernetes/code-generator")
	cmd.Flags().BoolVarP(&o.Verbose, optionVerbose, "v", false, "Enables verbose logging")
	cmd.Flags().StringVarP(&o.InputBase, optionInputBase, "", wd, "Input base, defaults working directory")

	return cmd
}

// Run implements this command
func (o *CreateClientGoOptions) Run() error {
	var err error
	o.BoilerplateFile, err = kube.GetBoilerplateFile(o.BoilerplateFile, o.Verbose)
	if err != nil {
		return errors.Wrapf(err, "reading file %s specified by %s", o.BoilerplateFile, optionBoilerplateFile)
	}
	if len(o.GroupsWithVersions) < 1 {
		return util.InvalidOptionf(optionGroupWithVersion, o.GroupsWithVersions, "must specify at least once")
	}
	if o.InputPackage == "" {
		return util.MissingOption(optionInputPackage)
	}
	if o.OutputPackage == "" {
		return util.MissingOption(optionOutputPackage)
	}

	err = o.configureGoPath()
	if err != nil {
		return errors.Wrapf(err, "ensuring GOPATH is set correctly")
	}

	err = kube.InstallGen(o.ClientGenVersion, o.Git())
	if err != nil {
		return errors.Wrapf(err, "installing kubernetes code generator tools")
	}
	log.Infof("Generating Go code to %s in package %s from package %s\n", o.OutputBase, o.GoPathOutputPackage, o.GoPathInputPackage)
	return kube.GenerateClient(o.Generators, o.GroupsWithVersions, o.GoPathInputPackage, o.GoPathOutputPackage,
		filepath.Join(build.Default.GOPATH, "src"), o.BoilerplateFile, o.Verbose)
}
