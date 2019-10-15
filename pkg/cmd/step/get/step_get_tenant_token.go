package get

import (
	"fmt"
	"net/url"
	"os"

	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/tenant"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

var (
	getTenantTokenExampleLong = templates.LongDesc(`
		Gets a token from the tenant service 
`)

	getTenantTokenExample = templates.Examples(`
		# perform a token exchange with the tenant service
		jx step get tenant-token
	
			`)
)

// StepGetTenantTokenOptions contains the command line flags
type StepGetTenantTokenOptions struct {
	step.StepOptions

	Dir string
}

// StepGetTenantTokenResults stores the generated results
type StepGetTenantTokenResults struct {
	Pipeline    *pipelineapi.Pipeline
	Task        *pipelineapi.Task
	PipelineRun *pipelineapi.PipelineRun
}

// NewCmdStepGetTenantToken Creates a new Command object
func NewCmdStepGetTenantToken(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepGetSubdomainOptions{
		StepOptions: step.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "tenant-token",
		Short:   "Gets a tenant-token from the tenant service",
		Long:    getTenantTokenExampleLong,
		Example: getTenantTokenExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Dir, "dir", "d", ".", "the directory to look for the values.yaml file")
	return cmd
}

// Run implements this command
func (o *StepGetTenantTokenOptions) Run() error {
	var err error
	if o.Dir == "" {
		o.Dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	requirements, requirementsFileName, err := config.LoadRequirementsConfig(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to load Jenkins X requirements")
	}

	if requirements.Cluster.Provider == "" {
		log.Logger().Warnf("No provider configured")
	}

	if requirements.Ingress.Domain == "" {
		err = o.getTenantToken(requirements, requirementsFileName)
		if err != nil {
			return errors.Wrapf(err, "failed to discover the Ingress domain")
		}
	}

	return requirements.SaveConfig(requirementsFileName)
}

func (o *StepGetTenantTokenOptions) getTenantToken(requirements *config.RequirementsConfig, requirementsFileName string) error {
	var tempToken string

	if "" != os.Getenv("TENANT_TEMP_TOKEN") {
		tempToken = os.Getenv("TENANT_TEMP_TOKEN")
	}

	if requirements.Ingress.DomainIssuerURL != "" && tempToken != "" {
		err := o.performTokenExchange(requirements.Ingress.DomainIssuerURL, requirements.Cluster.ProjectID, tempToken)
		if err != nil {
			return errors.Wrap(err, "exchanging temp tenant token")
		}
	}

	return nil
}

func (o *StepGetTenantTokenOptions) performTokenExchange(domainIssuerURL, project string, tempToken string) error {

	_, err := url.ParseRequestURI(domainIssuerURL)
	if err != nil {
		return errors.Wrapf(err, "parsing Domain Issuer URL %s", domainIssuerURL)
	}
	username := os.Getenv(config.RequirementDomainIssuerUsername)
	password := os.Getenv(config.RequirementDomainIssuerPassword)
	namespace := os.Getenv(config.BootDeployNamespace)

	if username == "" {
		return errors.Errorf("no %s environment variable found", config.RequirementDomainIssuerUsername)
	}
	if password == "" {
		return errors.Errorf("no %s environment variable found", config.RequirementDomainIssuerPassword)
	}

	kubeClient, err := o.KubeClient()
	if err != nil {
		return errors.Errorf("unable to create kubernetes client")
	}

	tenantServiceAuth := fmt.Sprintf("%s:%s", username, password)
	tCli := tenant.NewTenantClient()

	return tCli.GetAndStoreTenantToken(domainIssuerURL, tenantServiceAuth, project, tempToken, namespace, kubeClient)
}
