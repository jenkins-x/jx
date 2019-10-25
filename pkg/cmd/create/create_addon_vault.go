package create

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/config"

	"github.com/jenkins-x/jx/pkg/cmd/helper"

	"github.com/jenkins-x/jx/pkg/helm"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	defaultVaultNamesapce       = "jx"
	defaultVaultOperatorVersion = ""
)

var (
	CreateAddonVaultLong = templates.LongDesc(`
		Creates the Vault operator addon

		This addon will install an operator for HashiCorp Vault.""
`)

	CreateAddonVaultExample = templates.Examples(`
		# Create the vault-operator addon
		jx create addon vault-operator
	`)
)

// CreateAddonVaultptions the options for the create addon vault-operator
type CreateAddonVaultOptions struct {
	CreateAddonOptions
}

// NewCmdCreateAddonVault creates a command object for the "create addon vault-opeator" command
func NewCmdCreateAddonVault(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &CreateAddonVaultOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: commonOpts,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "vault-operator",
		Short:   "Create an vault-operator addon for Hashicorp Vault",
		Long:    CreateAddonVaultLong,
		Example: CreateAddonVaultExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	options.addFlags(cmd, defaultVaultNamesapce, kube.DefaultVaultOperatorReleaseName, defaultVaultOperatorVersion)
	return cmd
}

// Run implements the command
func (o *CreateAddonVaultOptions) Run() error {
	return InstallVaultOperator(o.CommonOptions, o.Namespace, nil)
}

// InstallVaultOperator installs a vault operator in the namespace provided
func InstallVaultOperator(o *opts.CommonOptions, namespace string, vs *config.VersionStreamConfig) error {
	err := o.EnsureHelm()
	if err != nil {
		return errors.Wrap(err, "checking if helm is installed")
	}

	_, err = o.AddHelmBinaryRepoIfMissing(kube.DefaultChartMuseumURL, kube.DefaultChartMuseumJxRepoName, "", "")
	if err != nil {
		return errors.Wrapf(err, "adding '%s' helm charts repository", kube.DefaultChartMuseumURL)
	}

	releaseName := o.ReleaseName
	if releaseName == "" {
		releaseName = kube.DefaultVaultOperatorReleaseName
	}
	log.Logger().Infof("Installing %s...", util.ColorInfo(releaseName))

	values := strings.Split(o.SetValues, ",")
	helmOptions := helm.InstallChartOptions{
		Chart:       kube.ChartVaultOperator,
		ReleaseName: releaseName,
		Version:     o.Version,
		Ns:          namespace,
		SetValues:   values,
	}

	if vs != nil {
		helmOptions.VersionsGitURL = vs.URL
		helmOptions.VersionsGitRef = vs.Ref
	}

	err = o.InstallChartWithOptions(helmOptions)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("installing %s chart", releaseName))
	}

	log.Logger().Infof("%s addon successfully installed.", util.ColorInfo(releaseName))
	return nil
}
