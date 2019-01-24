package cmd

import (
	"go/build"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"

	"github.com/jenkins-x/jx/pkg/log"

	"github.com/jenkins-x/jx/pkg/util"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// CreateClientOpenAPIOptions the options for the create client openapi command
type CreateClientOpenAPIOptions struct {
	CreateClientOptions
	Title                string
	Version              string
	ReferenceDocsVersion string
	OpenAPIDependencies  []string
	OpenAPIGenVersion    string
	OpenAPIOutputDir     string
	ModuleName           string
}

var (
	createClientOpenAPILong = templates.LongDesc(`This command code generates OpenAPI specs for
	the specified custom resources.
 
`)

	createClientOpenAPIExample = templates.Examples(`
		# lets generate client docs
		jx create client openapi
			--output-package=github.com/jenkins-x/jx/pkg/client \
			--input-package=github.com/jenkins-x/pkg-apis \
			--group-with-version=jenkins.io:v1
			--version=1.2.3
			--title=Jenkins X
		
		# You will normally want to add a target to your Makefile that looks like:

		generate-openapi:
			jx create client openapi
				--output-package=github.com/jenkins-x/jx/pkg/client \
				--input-package=github.com/jenkins-x/jx/pkg/apis \
				--group-with-version=jenkins.io:v1
				--version=${VERSION}
				--title=${TITLE}
		
		# and then call:

		make generate-openapi
`)
)

// NewCmdCreateClientOpenAPI creates the command
func NewCmdCreateClientOpenAPI(f Factory, in terminal.FileReader, out terminal.FileWriter,
	errOut io.Writer) *cobra.Command {
	o := &CreateClientOpenAPIOptions{
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
		Use:     "openapi",
		Short:   "Creates OpenAPI specs for Custom Resources",
		Long:    createClientOpenAPILong,
		Example: createClientOpenAPIExample,

		Run: func(cmd *cobra.Command, args []string) {
			o.Cmd = cmd
			o.Args = args
			err := o.Run()
			CheckErr(err)
		},
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Warnf("Error getting working directory for %v\n", err)
	}

	openAPIDependencies := []string{
		"k8s.io/apimachinery:pkg/apis:meta:v1",
		"k8s.io/apimachinery:pkg/api:resource:",
		"k8s.io/apimachinery:pkg/util:intstr:",
		"k8s.io/api::batch:v1",
		"k8s.io/api::core:v1",
		"k8s.io/api::rbac:v1",
	}

	moduleName := strings.TrimPrefix(strings.TrimPrefix(wd, filepath.Join(build.Default.GOPATH, "src")), "/")

	defaultVersion := os.Getenv("VERSION")
	cmd.Flags().StringVarP(&o.OutputBase, "output-base", "", wd,
		"Output base directory, by default the current working directory")
	cmd.Flags().StringVarP(&o.BoilerplateFile, optionBoilerplateFile, "", "custom-boilerplate.go.txt",
		"Custom boilerplate to add to all files if the file is missing it will be ignored")
	cmd.Flags().StringVarP(&o.InputBase, optionInputBase, "", wd,
		"Input base (the root of module the OpenAPI is being generated for), by default the current working directory")
	cmd.Flags().StringVarP(&o.InputPackage, optionInputPackage, "i", "", "Input package (relative to input base), "+
		"must specify")
	cmd.Flags().StringVarP(&o.OutputPackage, optionOutputPackage, "o", "", "Output package, must specify")
	cmd.Flags().BoolVarP(&o.Verbose, optionVerbose, "v", false, "Enables verbose logging")
	cmd.Flags().StringVarP(&o.Title, "title", "", "Jenkins X", "Title for OpenAPI, JSON Schema and HTML docs")
	cmd.Flags().StringVarP(&o.Version, "version", "", defaultVersion, "Version for OpenAPI, JSON Schema and HTML docs")
	cmd.Flags().StringArrayVarP(&o.OpenAPIDependencies, "open-api-dependency", "", openAPIDependencies,
		"Add <path:package:group:apiVersion> dependencies for OpenAPI generation")
	cmd.Flags().StringVarP(&o.OpenAPIGenVersion, "openapi-generator-version", "", "ced9eb3070a5f1c548ef46e8dfe2a97c208d9f03",
		"Version (really a commit-ish) of github.com/kubernetes/kube-openapi")
	cmd.Flags().StringVarP(&o.OpenAPIOutputDir, "openapi-output-directory", "",
		"docs/apidocs", "Output directory for the OpenAPI specs, "+
			"relative to the output-base unless absolute. "+
			"OpenAPI spec JSON and YAML files are placed in openapi-spec sub directory.")
	cmd.Flags().StringArrayVarP(&o.GroupsWithVersions, optionGroupWithVersion, "g", make([]string, 0),
		"group name:version (e.g. jenkins.io:v1) to generate, must specify at least once")
	cmd.Flags().StringVarP(&o.ModuleName, optionModuleName, "", moduleName,
		"module name (e.g. github.com/jenkins-x/jx)")
	return cmd
}

// Run implements this command
func (o *CreateClientOpenAPIOptions) Run() error {
	var err error
	o.BoilerplateFile, err = kube.GetBoilerplateFile(o.BoilerplateFile, o.Verbose)
	if err != nil {
		return errors.Wrapf(err, "reading file %s specified by %s", o.BoilerplateFile, optionBoilerplateFile)
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
	if len(o.GroupsWithVersions) < 1 {
		return util.InvalidOptionf(optionGroupWithVersion, o.GroupsWithVersions, "must specify at least once")
	}

	err = kube.InstallOpenApiGen(o.OpenAPIGenVersion, o.Git())
	if err != nil {
		return errors.Wrapf(err, "installing kubernetes openapi tools")
	}

	if !filepath.IsAbs(o.OpenAPIOutputDir) {
		o.OpenAPIOutputDir = filepath.Join(o.OutputBase, o.OpenAPIOutputDir)
	}

	log.Infof("Generating Go code to %s in package %s from package %s\n", o.OutputBase, o.GoPathOutputPackage, o.GoPathInputPackage)
	err = kube.GenerateOpenApi(o.GroupsWithVersions, o.GoPathInputPackage, o.GoPathOutputPackage,
		filepath.Join(build.Default.GOPATH, "src"), o.OpenAPIDependencies, o.InputBase, o.ModuleName, o.Git(),
		o.BoilerplateFile, o.Verbose)
	if err != nil {
		return errors.Wrapf(err, "generating openapi structs to %s", o.GoPathOutputPackage)
	}

	log.Infof("Generating OpenAPI spec files to %s from package %s\n", o.OpenAPIOutputDir, filepath.Join(o.InputBase,
		o.InputPackage))
	err = kube.GenerateSchema(o.OpenAPIOutputDir, o.OutputPackage, o.InputBase, o.Title, o.Version)
	if err != nil {
		return errors.Wrapf(err, "generating schema to %s", o.OpenAPIOutputDir)
	}
	return nil
}
