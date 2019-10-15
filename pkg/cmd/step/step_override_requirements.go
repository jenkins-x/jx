package step

import (
	"os"

	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
)

// StepOverrideRequirementsOptions contains the command line flags
type StepOverrideRequirementsOptions struct {
	*opts.CommonOptions
	Dir string
}

// NewCmdStepOverrideRequirements creates the `jx step verify pod` command
func NewCmdStepOverrideRequirements(commonOpts *opts.CommonOptions) *cobra.Command {

	options := &StepOverrideRequirementsOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:   "override-requirements",
		Short: "Overrides requirements with environment variables to be persisted in the `jx-requirements.yml`",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Dir, "dir", "d", ".", "the directory to look for the install requirements file")

	return cmd
}

// Run implements this command
func (o *StepOverrideRequirementsOptions) Run() error {
	requirements, requirementsFileName, err := config.LoadRequirementsConfig(o.Dir)
	if err != nil {
		return err
	}

	requirements, err = o.overrideRequirements(requirements, requirementsFileName)
	if err != nil {
		return err
	}

	return nil
}

// gatherRequirements gathers cluster requirements and connects to the cluster if required
func (o *StepOverrideRequirementsOptions) overrideRequirements(requirements *config.RequirementsConfig, requirementsFileName string) (*config.RequirementsConfig, error) {
	log.Logger().Debug("Overriding Requirements...")

	if "" != os.Getenv(config.RequirementClusterName) {
		requirements.Cluster.ClusterName = os.Getenv(config.RequirementClusterName)
	}
	if "" != os.Getenv(config.RequirementProject) {
		requirements.Cluster.ProjectID = os.Getenv(config.RequirementProject)
	}
	if "" != os.Getenv(config.RequirementZone) {
		requirements.Cluster.Zone = os.Getenv(config.RequirementZone)
	}
	if "" != os.Getenv(config.RequirementEnvGitOwner) {
		requirements.Cluster.EnvironmentGitOwner = os.Getenv(config.RequirementEnvGitOwner)
	}
	publicEnvRepo, found := os.LookupEnv(config.RequirementEnvGitPublic)
	if found {
		if publicEnvRepo == "true" {
			requirements.Cluster.EnvironmentGitPublic = true
		} else {
			requirements.Cluster.EnvironmentGitPublic = false
		}
	}
	publicAppRepo, found := os.LookupEnv(config.RequirementGitPublic)
	if found {
		if publicAppRepo == "true" {
			requirements.Cluster.GitPublic = true
		} else {
			requirements.Cluster.GitPublic = false
		}
	}
	if "" != os.Getenv(config.RequirementExternalDNSServiceAccountName) {
		requirements.Cluster.ExternalDNSSAName = os.Getenv(config.RequirementExternalDNSServiceAccountName)
	}

	if "" != os.Getenv(config.RequirementVaultName) {
		requirements.Vault.Name = os.Getenv(config.RequirementVaultName)
	}
	if "" != os.Getenv(config.RequirementVaultServiceAccountName) {
		requirements.Vault.ServiceAccount = os.Getenv(config.RequirementVaultServiceAccountName)
	}
	if "" != os.Getenv(config.RequirementVaultKeyringName) {
		requirements.Vault.Keyring = os.Getenv(config.RequirementVaultKeyringName)
	}
	if "" != os.Getenv(config.RequirementVaultKeyName) {
		requirements.Vault.Key = os.Getenv(config.RequirementVaultKeyName)
	}
	if "" != os.Getenv(config.RequirementVaultBucketName) {
		requirements.Vault.Bucket = os.Getenv(config.RequirementVaultBucketName)
	}
	if "" != os.Getenv(config.RequirementVaultRecreateBucket) {
		recreate := os.Getenv(config.RequirementVaultRecreateBucket)
		if recreate == "true" {
			requirements.Vault.RecreateBucket = true
		} else {
			requirements.Vault.RecreateBucket = false
		}
	}
	if "" != os.Getenv(config.RequirementVaultDisableURLDiscovery) {
		disable := os.Getenv(config.RequirementVaultDisableURLDiscovery)
		if disable == "true" {
			requirements.Vault.DisableURLDiscovery = true
		} else {
			requirements.Vault.DisableURLDiscovery = false
		}
	}
	if "" != os.Getenv(config.RequirementSecretStorageType) {
		requirements.SecretStorage = config.SecretStorageType(os.Getenv(config.RequirementSecretStorageType))
	}
	if "" != os.Getenv(config.RequirementKanikoServiceAccountName) {
		requirements.Cluster.KanikoSAName = os.Getenv(config.RequirementKanikoServiceAccountName)
	}
	if "" != os.Getenv(config.RequirementDomainIssuerURL) {
		requirements.Ingress.DomainIssuerURL = os.Getenv(config.RequirementDomainIssuerURL)
	}
	if "" != os.Getenv(config.RequirementIngressTLSProduction) {
		useProduction := os.Getenv(config.RequirementIngressTLSProduction)
		if useProduction == "yes" {
			requirements.Ingress.TLS.Production = true
		} else {
			requirements.Ingress.TLS.Production = false
		}
	}
	if "" != os.Getenv(config.RequirementKaniko) {
		kaniko := os.Getenv(config.RequirementKaniko)
		if kaniko == "true" {
			requirements.Kaniko = true
		}
	}
	if "" != os.Getenv(config.RequirementWebhook) {
		webhookString := os.Getenv(config.RequirementWebhook)
		requirements.Webhook = config.WebhookType(webhookString)
	}
	if "" != os.Getenv(config.RequirementStorageBackupEnabled) {
		storageBackup := os.Getenv(config.RequirementStorageBackupEnabled)
		if storageBackup == "true" && "" != os.Getenv(config.RequirementStorageBackupURL) {
			requirements.Storage.Backup.Enabled = true
			requirements.Storage.Backup.URL = os.Getenv(config.RequirementStorageBackupURL)
		}
	}
	if "" != os.Getenv(config.RequirementStorageLogsEnabled) {
		storageLogs := os.Getenv(config.RequirementStorageLogsEnabled)
		if storageLogs == "true" && "" != os.Getenv(config.RequirementStorageLogsURL) {
			requirements.Storage.Logs.Enabled = true
			requirements.Storage.Logs.URL = os.Getenv(config.RequirementStorageLogsURL)
		}
	}
	if "" != os.Getenv(config.RequirementStorageReportsEnabled) {
		storageReports := os.Getenv(config.RequirementStorageReportsEnabled)
		if storageReports == "true" && "" != os.Getenv(config.RequirementStorageReportsURL) {
			requirements.Storage.Reports.Enabled = true
			requirements.Storage.Reports.URL = os.Getenv(config.RequirementStorageReportsURL)
		}
	}
	if "" != os.Getenv(config.RequirementStorageRepositoryEnabled) {
		storageRepository := os.Getenv(config.RequirementStorageRepositoryEnabled)
		if storageRepository == "true" && "" != os.Getenv(config.RequirementStorageRepositoryURL) {
			requirements.Storage.Repository.Enabled = true
			requirements.Storage.Repository.URL = os.Getenv(config.RequirementStorageRepositoryURL)
		}
	}
	githubApp, found := os.LookupEnv(config.RequirementGitApp)
	if found {
		if githubApp == "true" {
			requirements.Cluster.GithubApp = true
		} else {
			requirements.Cluster.GithubApp = false
		}
	}

	log.Logger().Debugf("saving %s", requirementsFileName)
	requirements.SaveConfig(requirementsFileName)

	return requirements, nil
}
