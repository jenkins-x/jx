package cmd

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	fromNamespace = "from-namespace"
	toNamespace   = "to-namespace"
	includes      = "includes"
	excludes      = "excludes"
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
func NewCmdStepLinkServices(commonOpts *CommonOptions) *cobra.Command {
	options := &StepLinkServicesOptions{
		StepOptions: StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "link services",
		Short:   "achieve service linking in preview environments",
		Long:    StepLinkServicesLong,
		Example: StepLinkServicesExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.FromNamespace, fromNamespace, "f", "", "The source namespace from which the linking would happen")
	cmd.Flags().StringVarP(&options.ToNamespace, toNamespace, "t", "", "The destination namespace to which the linking would happen")
	cmd.Flags().StringArrayVarP(&options.Includes, includes, "i", []string{}, "What services from source namespace to include in the linking process")
	cmd.Flags().StringArrayVarP(&options.Excludes, excludes, "e", []string{}, "What services from the source namespace to exclude from the linking process")
	return cmd
}

// Run implements this command
func (o *StepLinkServicesOptions) Run() error {
	if o.FromNamespace == "" {
		return util.MissingOption(fromNamespace)
	}
	kubeClient, currentNs, err := o.KubeClient()
	if err != nil {
		return err
	}
	targetNamespace := o.ToNamespace
	if targetNamespace == "" {
		//to-namespace wasn't supplied, let's assume it is current namespace
		targetNamespace = currentNs
	}
	if targetNamespace == "" {
		//We don't want to continue if we still can't derive to-namespace
		return util.MissingOption(toNamespace)
	} else {
		serviceList, err := kubeClient.CoreV1().Services(o.FromNamespace).List(metav1.ListOptions{})
		if err != nil {
			return err
		} else {
			for _, service := range serviceList.Items {
				if util.StringMatchesAny(service.Name, o.Includes, o.Excludes) {
					targetService, err := kubeClient.CoreV1().Services(targetNamespace).Get(service.GetName(), metav1.GetOptions{})
					if targetService == nil {
						copy := service
						targetService = &copy
					}
					targetService.Namespace = targetNamespace
					// Reset the cluster IP, because this is dynamically allocated
					targetService.Spec.ClusterIP = ""
					targetService.ResourceVersion = ""
					//Change the namespace in the service to target namespace
					// We would create a new service if it doesn't already exist OR update if it already exists
					if err == nil {
						_, err := kubeClient.CoreV1().Services(targetNamespace).Update(targetService)
						if err != nil {
							log.Warnf("Failed to update the service '%s' in target namespace '%s'. Error: %s",
								service.GetName(), targetNamespace, err)
						}
					} else {
						_, err := kubeClient.CoreV1().Services(targetNamespace).Create(targetService)
						if err != nil {
							log.Warnf("Failed to create the service '%s' in target namespace '%s'. Error: %s",
								service.GetName(), targetNamespace, err)
						}
					}
				}
			}
		}
	}
	return nil
}
