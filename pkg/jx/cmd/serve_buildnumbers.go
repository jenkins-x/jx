package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/buildnum"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

const (
	command    = "buildnumbers"
	optionPort = "port"
	optionBind = "bind"
)

// ServeBuildNumbersOptions holds the options for the build number service.
type ServeBuildNumbersOptions struct {
	CommonOptions
	BindAddress string
	Port        int
}

var (
	serveBuildNumbersLong = templates.LongDesc(`Runs the build number controller that serves sequential build 
		numbers over an HTTP interface.`)

	serveBuildNumbersExample = templates.Examples("jx " + command)
)

// NewCmdSControllerBuildNumbers builds a new command to serving build numbers over an HTTP interface.
func NewCmdSControllerBuildNumbers(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := ServeBuildNumbersOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     command,
		Short:   "Runs the service to generate build numbers.",
		Long:    serveBuildNumbersLong,
		Example: serveBuildNumbersExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().IntVarP(&options.Port, optionPort, "", 8080, "The TCP port to listen on.")
	cmd.Flags().StringVarP(&options.BindAddress, optionBind, "", "",
		"The interface address to bind to (by default, will listen on all interfaces/addresses).")
	return cmd
}

// Run will execute this command, starting the HTTP build number generation service with the specified options.
func (o *ServeBuildNumbersOptions) Run() error {
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	buildNumGen := buildnum.NewCRDBuildNumGen(jxClient, ns)

	httpBuildNumServer := buildnum.NewHTTPBuildNumberServer(o.BindAddress, o.Port, buildNumGen)
	return httpBuildNumServer.Start()
}
