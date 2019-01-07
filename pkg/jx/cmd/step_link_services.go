package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
	corev1 "k8s.io/api/core/v1"
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
func NewCmdStepLinkServices(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepLinkServicesOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
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
	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.FromNamespace, fromNamespace, "f", "", "The source namespace from which the linking would happen")
	cmd.Flags().StringVarP(&options.ToNamespace, toNamespace, "t", "", "The destination namespace to which the linking would happen")
	cmd.Flags().StringArrayVarP(&options.Includes, includes, "i", []string{}, "What services from source namespace to include in the linking process")
	cmd.Flags().StringArrayVarP(&options.Excludes, excludes, "e", []string{}, "What services from the source namespace to exclude from the linking process")
	return cmd
}

// Run implements this command
func (o *StepLinkServicesOptions) Run() error {
	fromNs := o.FromNamespace
	if fromNs == "" {
		return util.MissingOption(fromNamespace)
	}
	kubeClient, currentNs, err := o.KubeClientAndNamespace()
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
		serviceList, err := kubeClient.CoreV1().Services(fromNs).List(metav1.ListOptions{})
		if err != nil {
			return err
		} else {
			for _, service := range serviceList.Items {
				name := service.GetName()
				if util.StringMatchesAny(name, o.Includes, o.Excludes) {
					targetService, err := kubeClient.CoreV1().Services(targetNamespace).Get(name, metav1.GetOptions{})
					create := false
					if err != nil {
						copy := corev1.Service{}
						targetService = &copy
						create = true
					}
					targetService.Name = name
					targetService.Namespace = targetNamespace
					targetService.Annotations = service.Annotations
					targetService.Labels = service.Labels
					targetService.Spec = corev1.ServiceSpec{
						Type:         corev1.ServiceTypeExternalName,
						ExternalName: fmt.Sprintf("%s.%s.svc.cluster.local", name, fromNs),
					}

					if create {
						_, err := kubeClient.CoreV1().Services(targetNamespace).Create(targetService)
						if err != nil {
							log.Warnf("Failed to create the service '%s' in target namespace '%s'. Error: %s",
								name, targetNamespace, err)
						}
					} else {
						_, err := kubeClient.CoreV1().Services(targetNamespace).Update(targetService)
						if err != nil {
							log.Warnf("Failed to update the service '%s' in target namespace '%s'. Error: %s",
								name, targetNamespace, err)
						}
					}
				}
			}
		}
	}
	return nil
}
