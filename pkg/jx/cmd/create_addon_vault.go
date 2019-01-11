package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

const (
	defaultVaultNamesapce        = "jx"
	jxRepoName                   = "jenkinsxio"
	jxRepoURL                    = "https://chartmuseum.jx.cd.jenkins-x.io"
	vaultOperatorImageRepository = "banzaicloud/vault-operator"
	vaultOperatorImageTag        = "0.3.17"
	defaultVaultOperatorVersion  = ""
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
func NewCmdCreateAddonVault(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	commonOptions := CommonOptions{
		Factory: f,
		In:      in,
		Out:     out,
		Err:     errOut,
	}
	options := &CreateAddonVaultOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: commonOptions,
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
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd, defaultVaultNamesapce, kube.DefaultVaultOperatorReleaseName, defaultVaultOperatorVersion)
	return cmd
}

// Run implements the command
func (o *CreateAddonVaultOptions) Run() error {
	return InstallVaultOperator(&o.CommonOptions, o.Namespace)
}

// InstallVaultOperator installs a vault operator in the namespace provided
func InstallVaultOperator(o *CommonOptions, namespace string) error {
	err := o.ensureHelm()
	if err != nil {
		return errors.Wrap(err, "checking if helm is installed")
	}

	err = o.addHelmRepoIfMissing(jxRepoURL, jxRepoName, "", "")
	if err != nil {
		return errors.Wrapf(err, "adding '%s' helm charts repository", jxRepoURL)
	}

	releaseName := o.ReleaseName
	if releaseName == "" {
		releaseName = kube.DefaultVaultOperatorReleaseName
	}
	log.Infof("Installing %s...\n", util.ColorInfo(releaseName))

	values := []string{
		"image.repository=" + vaultOperatorImageRepository,
		"image.tag=" + vaultOperatorImageTag,
	}
	setValues := strings.Split(o.SetValues, ",")
	values = append(values, setValues...)
	err = o.installChart(releaseName, kube.ChartVaultOperator, o.Version, namespace, true, values, nil, "")
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("installing %s chart", releaseName))
	}

	log.Infof("%s addon succesfully installed.\n", util.ColorInfo(releaseName))
	return nil
}
