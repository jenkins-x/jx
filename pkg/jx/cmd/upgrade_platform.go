package cmd

import (
	"github.com/jenkins-x/jx/pkg/kube"
	"io"
	"strings"

	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	core_v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	upgrade_platform_long = templates.LongDesc(`
		Upgrades the Jenkins X platform if there is a newer release
`)

	upgrade_platform_example = templates.Examples(`
		# Upgrades the Jenkins X platform 
		jx upgrade platform
	`)
)

// UpgradePlatformOptions the options for the create spring command
type UpgradePlatformOptions struct {
	InstallOptions

	Version       string
	ReleaseName   string
	Chart         string
	Namespace     string
	Set           string
	AlwaysUpgrade bool

	InstallFlags InstallFlags
}

// NewCmdUpgradePlatform defines the command
func NewCmdUpgradePlatform(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &UpgradePlatformOptions{
		InstallOptions: InstallOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "platform",
		Short:   "Upgrades the Jenkins X platform if there is a new release available",
		Aliases: []string{"install"},
		Long:    upgrade_platform_long,
		Example: upgrade_platform_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "", "", "The Namespace to promote to")
	cmd.Flags().StringVarP(&options.ReleaseName, "name", "n", "jenkins-x", "The release name")
	cmd.Flags().StringVarP(&options.Chart, "chart", "c", "jenkins-x/jenkins-x-platform", "The Chart to upgrade")
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The specific platform version to upgrade to")
	cmd.Flags().StringVarP(&options.Set, "set", "s", "", "The helm parameters to pass in while upgrading")
	cmd.Flags().BoolVarP(&options.AlwaysUpgrade, "always-upgrade", "", false, "If set to true, jx will upgrade platform Helm chart even if requested version is already installed.")

	options.addCommonFlags(cmd)
	options.InstallFlags.addCloudEnvOptions(cmd)

	return cmd
}

// Run implements the command
func (o *UpgradePlatformOptions) Run() error {
	targetVersion := o.Version
	err := o.Helm().UpdateRepo()
	if err != nil {
		return err
	}
	apisClient, err := o.CreateApiExtensionsClient()
	if err != nil {
		return errors.Wrap(err, "failed to create the API extensions client")
	}
	kube.RegisterAllCRDs(apisClient)
	if err != nil {
		return err
	}
	ns := o.Namespace
	if ns == "" {
		_, ns, err = o.JXClientAndDevNamespace()
		if err != nil {
			return err
		}
	}
	if targetVersion == "" {
		io := &InstallOptions{}
		io.CommonOptions = o.CommonOptions
		io.Flags = o.InstallFlags
		wrkDir, err := io.cloneJXCloudEnvironmentsRepo()
		if err != nil {
			return err
		}
		targetVersion, err = LoadVersionFromCloudEnvironmentsDir(wrkDir)
		if err != nil {
			return err
		}
	}

	// Current version
	var currentVersion string
	output, err := o.Helm().ListCharts()
	if err != nil {
		log.Warnf("Failed to find helm installs: %s\n", err)
		return err
	} else {
		for _, line := range strings.Split(output, "\n") {
			fields := strings.Split(line, "\t")
			if len(fields) > 4 && strings.TrimSpace(fields[0]) == "jenkins-x" {
				for _, f := range fields[4:] {
					f = strings.TrimSpace(f)
					if strings.HasPrefix(f, jxChartPrefix) {
						currentVersion = strings.TrimPrefix(f, jxChartPrefix)
					}
				}
			}
		}
	}
	if currentVersion == "" {
		return errors.New("Jenkins X platform helm chart is not installed.")
	}

	settings, err := o.TeamSettings()
	if err != nil {
		return err
	}
	log.Infof("Using provider %s from team settings\n", util.ColorInfo(settings.KubeProvider))

	// get secrets to use in helm install
	secrets, err := o.getGitSecrets()
	if err != nil {
		return errors.Wrap(err, "failed to read the git secrets from configuration")
	}

	err = o.AdminSecretsService.NewAdminSecretsConfig()
	if err != nil {
		return errors.Wrap(err, "failed to create the admin secret config service")
	}

	adminSecrets, err := o.AdminSecretsService.Secrets.String()
	if err != nil {
		return errors.Wrap(err, "failed to read the admin secrets")
	}

	helmConfig := &o.CreateEnvOptions.HelmValuesConfig
	exposeController := helmConfig.ExposeController
	if exposeController != nil && exposeController.Config.Domain == "" {
		helmConfig.ExposeController.Config.Domain = o.InitOptions.Flags.Domain
	}

	config, err := helmConfig.String()
	if err != nil {
		return errors.Wrap(err, "failed to get the helm config")
	}

	// clone the environments repo
	wrkDir, err := o.cloneJXCloudEnvironmentsRepo()
	if err != nil {
		return errors.Wrap(err, "failed to clone the jx cloud environments reppo")
	}

	makefileDir := filepath.Join(wrkDir, fmt.Sprintf("env-%s", strings.ToLower(settings.KubeProvider)))
	if _, err := os.Stat(wrkDir); os.IsNotExist(err) {
		return fmt.Errorf("cloud environment dir %s not found", makefileDir)
	}

	// create a temporary file that's used to pass current git creds to helm in order to create a secret for pipelines to tag releases
	dir, err := util.ConfigDir()
	if err != nil {
		return errors.Wrap(err, "failed to create a temporary config dir for Git credentials")
	}

	secretsFileName := filepath.Join(dir, GitSecretsFile)
	err = ioutil.WriteFile(secretsFileName, []byte(secrets), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write the git secrets in the secrets file")
	}

	adminSecretsFileName := filepath.Join(dir, AdminSecretsFile)
	err = ioutil.WriteFile(adminSecretsFileName, []byte(adminSecrets), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write the admin secrets in the secrets file")
	}

	configFileName := filepath.Join(dir, ExtraValuesFile)
	err = ioutil.WriteFile(configFileName, []byte(config), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write the config file")
	}

	data := make(map[string][]byte)
	data[ExtraValuesFile] = []byte(config)
	data[AdminSecretsFile] = []byte(adminSecrets)
	data[GitSecretsFile] = []byte(secrets)

	jxSecrets := &core_v1.Secret{
		Data: data,
		ObjectMeta: metav1.ObjectMeta{
			Name: JXInstallConfig,
		},
	}
	secretResources := o.KubeClientCached.CoreV1().Secrets(ns)
	oldSecret, err := secretResources.Get(JXInstallConfig, metav1.GetOptions{})
	if oldSecret == nil || err != nil {
		_, err = secretResources.Create(jxSecrets)
		if err != nil {
			return errors.Wrap(err, "failed to create the jx secret resource")
		}
	} else {
		oldSecret.Data = jxSecrets.Data
		_, err = secretResources.Update(oldSecret)
		if err != nil {
			return errors.Wrap(err, "failed to update the jx secret resource")
		}
	}

	if targetVersion != currentVersion {
		log.Infof("Upgrading platform from version %s to version %s\n", util.ColorInfo(currentVersion), util.ColorInfo(targetVersion))
	} else if o.AlwaysUpgrade {
		log.Infof("Rerunning platform version %s\n", util.ColorInfo(targetVersion))
	} else {
		log.Infof("Already installed platform version %s. Skipping upgrade process.\n", util.ColorInfo(targetVersion))
		return nil
	}

	cloudEnvironmentValuesLocation := filepath.Join(makefileDir, CloudEnvValuesFile)
	cloudEnvironmentSecretsLocation := filepath.Join(makefileDir, CloudEnvSecretsFile)
	valueFiles := []string{cloudEnvironmentValuesLocation, cloudEnvironmentSecretsLocation, secretsFileName, adminSecretsFileName, configFileName}
	valueFiles, err = helm.AppendMyValues(valueFiles)
	if err != nil {
		return errors.Wrap(err, "failed to append the myvalues.yaml file")
	}

	values := []string{}
	if o.Set != "" {
		values = append(values, o.Set)
	}
	return o.Helm().UpgradeChart(o.Chart, o.ReleaseName, ns, &targetVersion, false, nil, false, false, values, valueFiles)
}
