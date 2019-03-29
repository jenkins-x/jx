package app

import (
	"github.com/jenkins-x/jx/cmd/codegen/generator"
	"github.com/jenkins-x/jx/cmd/codegen/util"
	"go/build"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"

	jxutil "github.com/jenkins-x/jx/pkg/util"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd"
)

// ClientSetGenerationOptions contain the options for the clientset generation.
type ClientSetGenerationOptions struct {
	GenerateOptions
	Generators []string
}

var (
	createClientGoLong = templates.LongDesc(`This command code generates clients for the specified custom resources.`)

	createClientGoExample = templates.Examples(`
		# lets generate a client
		codegen clientset
			--output-package=github.com/jenkins-x/jx/pkg/client \
			--input-package=github.com/jenkins-x/pkg-apis \
			--group-with-version=jenkins.io:v1
		
		# You will normally want to add a target to your Makefile that looks like:

		generate-clients:
			codegen clientset
				--output-package=github.com/jenkins-x/jx/pkg/client \
				--input-package=github.com/jenkins-x/jx/pkg/apis \
				--group-with-version=jenkins.io:v1
		
		# and then call:

		make generate-clients
`)
)

// NewGenerateClientSetCmd creates the command
func NewGenerateClientSetCmd(commonOpts *cmd.CommonOptions) *cobra.Command {
	o := &ClientSetGenerationOptions{
		GenerateOptions: GenerateOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "clientset",
		Short:   "Creates Go client for Custom Resources",
		Long:    createClientGoLong,
		Example: createClientGoExample,

		Run: func(c *cobra.Command, args []string) {
			o.Cmd = c
			o.Args = args
			err := o.Run()
			cmd.CheckErr(err)
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
		util.AppLogger().Warnf("Error getting working directory for %v\n", err)
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
	cmd.Flags().StringVarP(&o.InputBase, optionInputBase, "", wd, "Input base, defaults working directory")

	return cmd
}

// Run executes this command.
func (o *ClientSetGenerationOptions) Run() error {
	var err error
	o.BoilerplateFile, err = generator.GetBoilerplateFile(o.BoilerplateFile)
	if err != nil {
		return errors.Wrapf(err, "reading file %s specified by %s", o.BoilerplateFile, optionBoilerplateFile)
	}
	if len(o.GroupsWithVersions) < 1 {
		return jxutil.InvalidOptionf(optionGroupWithVersion, o.GroupsWithVersions, "must specify at least once")
	}
	if o.InputPackage == "" {
		return jxutil.MissingOption(optionInputPackage)
	}
	if o.OutputPackage == "" {
		return jxutil.MissingOption(optionOutputPackage)
	}

	err = o.configure()
	if err != nil {
		return errors.Wrapf(err, "ensure GOPATH is set correctly")
	}

	err = generator.InstallCodeGenerators(o.ClientGenVersion)
	if err != nil {
		return errors.Wrapf(err, "installing kubernetes code generator tools")
	}
	util.AppLogger().Infof("generating Go code to %s in package %s from package %s\n", o.OutputBase, o.GoPathOutputPackage, o.GoPathInputPackage)
	return generator.GenerateClient(o.Generators, o.GroupsWithVersions, o.GoPathInputPackage, o.GoPathOutputPackage,
		filepath.Join(build.Default.GOPATH, "src"), o.BoilerplateFile)
}
