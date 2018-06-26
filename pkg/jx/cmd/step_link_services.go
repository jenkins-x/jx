package cmd

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
	"io"
)

var (
	StepLinkServicesLong = templates.LongDesc(`
		This pipeline step helps to link microservices from different namespaces like staging/production onto a preview environment
`)

	StepLinkServicesExample = templates.Examples(`
	#Link services from jx-staging namespace to the current namespace
	jx step link services --from-namespace jx-staging 

	#Link services from jx-staging namespace to the jx-prod namespace
	jx step link services --from-namespace jx-staging --to-namespace jx-prod
	
	#Link services from jx-staging namespace to the jx-prod namespace including all but the ones starting with  the characters 'cheese'
	jx step link services --from-namespace jx-staging --to-namespace jx-prod --includes * --excludes cheese*
`)
)

// StepLinkServicesOptions contains the command line flags
type StepLinkServicesOptions struct {
	StepOptions
	FromNameSpace string
	ToNameSpace   string
	Includes      string
	Excludes      string
}

// NewCmdStepLinkServices Creates a new Command object
func NewCmdStepLinkServices(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &StepLinkServicesOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "service linking",
		Short:   "achieve service linking in preview environments",
		Long:    StepLinkServicesLong,
		Example: StepLinkServicesExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.FromNameSpace, "from-namespace", "f", "", "The source namespace from which the linking would happen")
	cmd.Flags().StringVarP(&options.ToNameSpace, "to-namespace", "t", "", "The destination namespace to which the linking would happen")
	cmd.Flags().StringVarP(&options.Includes, "includes", "i", "", "What services from source namespace to include in the linking process")
	cmd.Flags().StringVarP(&options.Excludes, "excludes", "e", "", "What services from the source namespace to exclude from the linking process")
	return cmd
}

// Run implements this command
func (o *StepLinkServicesOptions) Run() error {
	return o.Cmd.Help()
}
