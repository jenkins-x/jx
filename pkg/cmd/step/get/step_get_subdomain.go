package get

import (
	"fmt"
	"net/url"
	"os"

	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/tenant"
	"github.com/jenkins-x/jx/pkg/util"

	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

var (
	getSubdomainExampleLong = templates.LongDesc(`
		Gets a subdomain from the tenant service 
`)

	getSubdomainExample = templates.Examples(`
		# populate the cluster/values.yaml file
		jx step get subdomain
	
			`)
)

// StepGetSubdomainOptions contains the command line flags
type StepGetSubdomainOptions struct {
	step.StepOptions

	Dir string
}

// StepGetSubdomainResults stores the generated results
type StepGetSubdomainResults struct {
	Pipeline    *pipelineapi.Pipeline
	Task        *pipelineapi.Task
	PipelineRun *pipelineapi.PipelineRun
}

// NewCmdStepGetSubdomain Creates a new Command object
func NewCmdStepGetSubdomain(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepGetSubdomainOptions{
		StepOptions: step.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "subdomain",
		Short:   "Gets a subdomain from the tenant service",
		Long:    getSubdomainExampleLong,
		Example: getSubdomainExample,
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
func (o *StepGetSubdomainOptions) Run() error {
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
		err = o.discoverIngressDomain(requirements, requirementsFileName)
		if err != nil {
			return errors.Wrapf(err, "failed to discover the Ingress domain")
		}
	}

	return requirements.SaveConfig(requirementsFileName)
}

func (o *StepGetSubdomainOptions) discoverIngressDomain(requirements *config.RequirementsConfig, requirementsFileName string) error {
	var domain string
	var err error

	// domain is already configured, do nothing
	if requirements.Ingress.Domain != "" {
		return nil
	}

	if requirements.Ingress.DomainIssuerURL != "" {
		domain, err = o.getDomainFromIssuer(requirements.Ingress.DomainIssuerURL, requirements.Cluster.ProjectID, requirements.Cluster.ClusterName)
		if err != nil {
			return errors.Wrap(err, "issuing domain")
		}
	}

	if domain == "" {
		return fmt.Errorf("failed to discover domain")
	}

	requirements.Ingress.Domain = domain
	err = requirements.SaveConfig(requirementsFileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save changes to file: %s", requirementsFileName)
	}
	log.Logger().Infof("defaulting the domain to %s and modified %s\n", util.ColorInfo(domain), util.ColorInfo(requirementsFileName))
	return nil
}

func (o *StepGetSubdomainOptions) getDomainFromIssuer(domainIssuerURL, projectID string, cluster string) (string, error) {

	_, err := url.ParseRequestURI(domainIssuerURL)
	if err != nil {
		return "", errors.Wrapf(err, "parsing Domain Issuer URL %s", domainIssuerURL)
	}
	username := os.Getenv(config.RequirementDomainIssuerUsername)
	password := os.Getenv(config.RequirementDomainIssuerPassword)

	if username == "" {
		return "", errors.Errorf("no %s environment variable found", config.RequirementDomainIssuerUsername)
	}
	if password == "" {
		return "", errors.Errorf("no %s environment variable found", config.RequirementDomainIssuerPassword)
	}

	tenantServiceAuth := fmt.Sprintf("%s:%s", username, password)
	tCli := tenant.NewTenantClient()
	return tCli.GetTenantSubDomain(domainIssuerURL, tenantServiceAuth, projectID, cluster, o.GCloud())
}
