package cmd

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	fromNamespace = "from-namespace"
	toNamespace   = "to-namespace"
	includes      = "includes"
	excludes      = "excludes"
	blankString   = ""
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
	FromNamespace string
	ToNamespace   string
	Includes      []string
	Excludes      []string
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

	cmd.Flags().StringVarP(&options.FromNamespace, fromNamespace, "f", blankString, "The source namespace from which the linking would happen")
	cmd.Flags().StringVarP(&options.ToNamespace, toNamespace, "t", blankString, "The destination namespace to which the linking would happen")
	cmd.Flags().StringArrayVarP(&options.Includes, includes, "i", []string{}, "What services from source namespace to include in the linking process")
	cmd.Flags().StringArrayVarP(&options.Excludes, excludes, "e", []string{}, "What services from the source namespace to exclude from the linking process")
	return cmd
}

// Run implements this command
func (o *StepLinkServicesOptions) Run() error {
	if o.FromNamespace == blankString {
		return util.MissingOption(fromNamespace)
	}
	kubeClient, currentNs, err := o.KubeClient()
	targetNamespace := o.ToNamespace
	if targetNamespace == blankString {
		//to-namespace wasn't supplied, let's assume it is current namespace
		targetNamespace = currentNs
	}
	if targetNamespace == blankString {
		//We don't want to continue if we still can't derive to-namespace
		return util.MissingOption(toNamespace)
	} else {
		serviceList, err := kubeClient.CoreV1().Services(o.FromNamespace).List(metav1.ListOptions{})
		if err != nil {
			return err
		} else {
			for _, service := range serviceList.Items {
				if util.StringMatchesAny(service.Name, o.Includes, o.Excludes) {
					lookedUpServiceFromSourceNamespace, err := kubeClient.CoreV1().Services(o.FromNamespace).Get(service.GetName(), metav1.GetOptions{})
					if err != nil {
						return err
					}
					lookedUpServiceFromTargetNamespace, err := kubeClient.CoreV1().Services(o.currentNamespace).Get(service.GetName(), metav1.GetOptions{})
					// We would create a new service if it doesn't already exist OR update if it already exists
					if lookedUpServiceFromTargetNamespace.Name != blankString {
						kubeClient.CoreV1().Services(targetNamespace).Update(lookedUpServiceFromSourceNamespace)
					} else {
						kubeClient.CoreV1().Services(targetNamespace).Create(lookedUpServiceFromSourceNamespace)
					}
				}
			}
		}
	}
	return nil //TODO change return
}
