package cmd

import (
	"io"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	complianceDeleteLong = templates.LongDesc(`
		Deletes the Kubernetes resources allocated by the compliance tests
	`)

	complianceDeleteExample = templates.Examples(`
		# Delete the Kubernetes resources allocated by the compliance test
		jx compliance delete
	`)
)

// ComplianceDeleteOptions options for "compliance delete" command
type ComplianceDeleteOptions struct {
	CommonOptions
}

// NewCmdComplianceDeletecreates a command object for the "compliance delete" action, which
// delete the Kubernetes resources allocated by the compliance tests
func NewCmdComplianceDelete(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ComplianceDeleteOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "Deletes the Kubernetes resources allocated by the compliance tests",
		Long:    complianceDeleteLong,
		Example: complianceDeleteExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	return cmd
}

// Run implements the "compliance delete" command
func (o *ComplianceDeleteOptions) Run() error {
	cc, err := o.Factory.CreateComplianceClient()
	if err != nil {
		return errors.Wrap(err, "could not create the compliance client")
	}
	deleteOpts := &client.DeleteConfig{
		Namespace:  complianceNamespace,
		EnableRBAC: false,
		DeleteAll:  true,
	}
	return cc.Delete(deleteOpts)
}
