package app

import (
	"github.com/jenkins-x/jx/cmd/codegen/util"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"

	"github.com/jenkins-x/jx/pkg/jx/cmd"
)

const (
	optionLogLevel = "log-level"
)

var (
	longHelp = templates.LongDesc(`Generates Go clientsets, OpenAPI spec and API docs for custom resources.

Custom resources are defined using Go structs.

Available generators include:

* openapi - generates OpenAPI specs, required to generate API docs and clients other than Go
* docs -  generates API docs from the OpenAPI specs
* clientset - generates a Go CRUD client directly from custom resources

`)
	logLevel string
)

// Run executes the Cobra root command.
func Run() error {
	rootCommand := &cobra.Command{
		Use:   "codegen",
		Short: "Uses Golang code-generators to generate various application resources and documentation.",
		Long:  longHelp,
		Run:   runHelp,
	}

	commonOpts := &cmd.CommonOptions{
		In:  os.Stdin,
		Out: os.Stdout,
		Err: os.Stderr,
	}

	rootCommand.PersistentFlags().StringVarP(&logLevel, optionLogLevel, "", logrus.InfoLevel.String(), "Sets the logging level (panic, fatal, error, warning, info, debug)")

	rootCommand.AddCommand(NewGenerateClientSetCmd(commonOpts))
	rootCommand.AddCommand(NewCmdCreateClientOpenAPI(commonOpts))
	rootCommand.AddCommand(NewCreateDocsCmd(commonOpts))

	util.SetLevel(logLevel)

	return rootCommand.Execute()
}

func runHelp(cmd *cobra.Command, _args []string) {
	cmd.Help()
}
