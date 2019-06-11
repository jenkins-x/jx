package get

import (
	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

// GetAWSInfoOptions containers the CLI options
type GetAWSInfoOptions struct {
	GetOptions
}

var (
	getAWSInfoLong = templates.LongDesc(`
		Display the AWS information for the current user
`)

	getAWSInfoExample = templates.Examples(`
		# Get the AWS account information
		jx get aws info
	`)
)

// NewCmdGetAWSInfo creates the new command for: jx get env
func NewCmdGetAWSInfo(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &GetAWSInfoOptions{
		GetOptions: GetOptions{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "aws info",
		Short:   "Displays AWS account information",
		Aliases: []string{"aws"},
		Long:    getAWSInfoLong,
		Example: getAWSInfoExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	options.AddGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetAWSInfoOptions) Run() error {
	id, region, err := amazon.GetAccountIDAndRegion("", "")
	if err != nil {
		return err
	}
	log.Logger().Infof("AWS Account ID: %s", util.ColorInfo(id))
	log.Logger().Infof("AWS Region:     %s", util.ColorInfo(region))
	return nil
}
