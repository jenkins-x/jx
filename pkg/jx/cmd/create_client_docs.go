package cmd

import (
	"io"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/kube"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"

	"github.com/jenkins-x/jx/pkg/log"

	"github.com/jenkins-x/jx/pkg/util"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// CreateClientDocsOptions the options for the create client docs command
type CreateClientDocsOptions struct {
	CreateClientOptions
	ReferenceDocsVersion string
}

var (
	createClientDocsLong = templates.LongDesc(`This command code generates clients docs (Swagger,OpenAPI and HTML) for
	the specified custom resources.
 
`)

	createClientDocsExample = templates.Examples(`
		# lets generate client docs
		jx create client docs
		
		# You will normally want to add a target to your Makefile that looks like:

		generate-clients-docs:
			jx create client docs
		
		# and then call:

		make generate-clients-docs
`)
)

const ()

// NewCmdCreateClientDocs creates apidocs for CRDs
func NewCmdCreateClientDocs(f Factory, in terminal.FileReader, out terminal.FileWriter,
	errOut io.Writer) *cobra.Command {
	o := &CreateClientDocsOptions{
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
		Use:     "docs",
		Short:   "Creates client docs for Custom Resources",
		Long:    createClientDocsLong,
		Example: createClientDocsExample,

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

	cmd.Flags().StringVarP(&o.OutputBase, optionOutputBase, "", filepath.Join(wd, "docs/apidocs"),
		"Output base directory, "+
			"by default the <current working directory>/docs/apidocs")
	cmd.Flags().StringVarP(&o.BoilerplateFile, "boilerplate-file", "", "custom-boilerplate.go.txt",
		"Custom boilerplate to add to all files if the file is missing it will be ignored")
	cmd.Flags().BoolVarP(&o.Verbose, optionVerbose, "v", false, "Enables verbose logging")
	cmd.Flags().StringVarP(&o.ReferenceDocsVersion, "reference-docs-version", "",
		"096940c697f8b79873e2cfd2c1c4da1f6df76c40", "Version (really a commit-ish) of https://github.com/kubernetes-incubator/reference-docs")
	return cmd
}

// Run implements this command
func (o *CreateClientDocsOptions) Run() error {
	var err error
	o.BoilerplateFile, err = kube.GetBoilerplateFile(o.BoilerplateFile, o.Verbose)
	if err != nil {
		return err
	}
	if o.OutputBase == "" {
		return util.MissingOption(optionOutputBase)
	}
	log.Infof("Generating docs to %s\n", o.OutputBase)

	referenceDocsRepo, err := kube.InstallGenApiDocs(o.ReferenceDocsVersion, o.Git())
	if err != nil {
		return err
	}
	err = kube.GenerateApiDocs(o.OutputBase, o.Verbose)
	if err != nil {
		return err
	}
	err = kube.AssembleApiDocsStatic(referenceDocsRepo, o.OutputBase)
	if err != nil {
		return err
	}
	err = kube.AssembleApiDocs(o.OutputBase, filepath.Join(o.OutputBase, "site"))
	if err != nil {
		return err
	}
	return nil
}
