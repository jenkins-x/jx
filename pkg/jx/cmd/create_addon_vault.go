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
	vaultOperatorChartVersion    = ""
	vaultOperatorImageRepository = "banzaicloud/vault-operator"
	vaultOperatorImageTag        = "0.3.4"
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
	options.addFlags(cmd, defaultVaultNamesapce, kube.DefaultVaultOperatorReleaseName)
	return cmd
}

// Run implements the command
func (o *CreateAddonVaultOptions) Run() error {
	_, _, err := o.KubeClient()
	if err != nil {
		return fmt.Errorf("cannot connect to Kubernetes cluster: %v", err)
	}
	err = o.ensureHelm()
	if err != nil {
		return errors.Wrap(err, "checking if helm is installed")
	}

	err = o.addHelmRepoIfMissing(jxRepoURL, jxRepoName)
	if err != nil {
		return errors.Wrapf(err, "adding '%s' helm charts repository", jxRepoURL)
	}

	log.Infof("Installing %s...\n", util.ColorInfo(o.ReleaseName))

	values := []string{
		"image.repository=" + vaultOperatorImageRepository,
		"image.tag=" + vaultOperatorImageTag,
	}
	setValues := strings.Split(o.SetValues, ",")
	values = append(values, setValues...)
	return o.installChart(o.ReleaseName, kube.ChartVaultOperator, vaultOperatorChartVersion, o.Namespace, true, values)
}
