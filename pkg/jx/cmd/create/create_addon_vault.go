package create

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"

	"github.com/jenkins-x/jx/pkg/helm"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/kube/vault"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	defaultVaultNamesapce       = "jx"
	jxRepoName                  = "jenkins-x"
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
	return InstallVaultOperator(o.CommonOptions, o.Namespace)
}

// InstallVaultOperator installs a vault operator in the namespace provided
func InstallVaultOperator(o *opts.CommonOptions, namespace string) error {
	err := o.EnsureHelm()
	if err != nil {
		return errors.Wrap(err, "checking if helm is installed")
	}

	_, err = o.AddHelmBinaryRepoIfMissing(kube.DefaultChartMuseumURL, jxRepoName, "", "")
	if err != nil {
		return errors.Wrapf(err, "adding '%s' helm charts repository", kube.DefaultChartMuseumURL)
	}

	releaseName := o.ReleaseName
	if releaseName == "" {
		releaseName = kube.DefaultVaultOperatorReleaseName
	}
	log.Logger().Infof("Installing %s...", util.ColorInfo(releaseName))

	resolver, err := o.CreateVersionResolver(opts.DefaultVersionsURL, "")
	if err != nil {
		return errors.Wrap(err, "creating the docker image version resolver")
	}
	repository, err := resolver.ResolveDockerImage(vault.BankVaultsOperatorImage)
	parts := strings.Split(repository, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid docker image: %s", repository)
	}

	values := []string{
		"image.repository=" + parts[0],
		"image.tag=" + parts[1],
	}

	setValues := strings.Split(o.SetValues, ",")
	values = append(values, setValues...)
	helmOptions := helm.InstallChartOptions{
		Chart:       kube.ChartVaultOperator,
		ReleaseName: releaseName,
		Version:     o.Version,
		Ns:          namespace,
		SetValues:   values,
	}
	err = o.InstallChartWithOptions(helmOptions)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("installing %s chart", releaseName))
	}

	log.Logger().Infof("%s addon succesfully installed.", util.ColorInfo(releaseName))
	return nil
}
