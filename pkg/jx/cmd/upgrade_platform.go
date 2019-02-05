package cmd

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	configio "github.com/jenkins-x/jx/pkg/io"

	"io"
	"io/ioutil"
	"strings"

	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/jenkins-x/jx/pkg/kube"

	"fmt"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
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
	UpdateSecrets bool

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
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "", "", "The Namespace to promote to.")
	cmd.Flags().StringVarP(&options.ReleaseName, "name", "n", "jenkins-x", "The release name.")
	cmd.Flags().StringVarP(&options.Chart, "chart", "c", "jenkins-x/jenkins-x-platform", "The Chart to upgrade.")
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The specific platform version to upgrade to.")
	cmd.Flags().StringVarP(&options.Set, "set", "s", "", "The helm parameters to pass in while upgrading, separated by comma, e.g. key1=val1,key2=val2.")
	cmd.Flags().BoolVarP(&options.AlwaysUpgrade, "always-upgrade", "", false, "If set to true, jx will upgrade platform Helm chart even if requested version is already installed.")
	cmd.Flags().BoolVarP(&options.Flags.CleanupTempFiles, "cleanup-temp-files", "", true, "Cleans up any temporary values.yaml used by helm install [default true].")
	cmd.Flags().BoolVarP(&options.UpdateSecrets, "update-secrets", "", false, "Regenerate adminSecrets.yaml on upgrade")

	options.addCommonFlags(cmd)
	options.InstallFlags.addCloudEnvOptions(cmd)

	return cmd
}

// Run implements the command
func (o *UpgradePlatformOptions) Run() error {
	configStore := configio.NewFileStore()
	targetVersion := o.Version
	err := o.Helm().UpdateRepo()
	if err != nil {
		return err
	}
	apisClient, err := o.ApiExtensionsClient()
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

	settings, err := o.TeamSettings()
	if err != nil {
		return err
	}

	if "" == settings.KubeProvider {
		log.Warnf("Unable to determine provider from team settings")

		surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)

		provider := ""

		prompt := &survey.Select{
			Message: "Select the kube provider:",
			Options: KUBERNETES_PROVIDERS,
			Default: "",
		}
		survey.AskOne(prompt, &provider, nil, surveyOpts)

		err = o.ModifyDevEnvironment(func(env *v1.Environment) error {
			settings = &env.Spec.TeamSettings
			settings.KubeProvider = provider
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "failed to create the API extensions client")
		}
	}

	log.Infof("Using provider '%s' from team settings\n", util.ColorInfo(settings.KubeProvider))

	wrkDir := ""

	if targetVersion == "" {
		io := &InstallOptions{}
		io.CommonOptions = o.CommonOptions
		io.Flags = o.InstallFlags
		versionsDir, err := io.cloneJXVersionsRepo(o.Flags.VersionsRepository)
		if err != nil {
			return err
		}
		targetVersion, err = LoadVersionFromCloudEnvironmentsDir(versionsDir, configStore)
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
		o.Debugf("Installed helm charts\n%s\n", output)
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

	helmConfig := &o.CreateEnvOptions.HelmValuesConfig
	exposeController := helmConfig.ExposeController
	if exposeController != nil && exposeController.Config.Domain == "" {
		helmConfig.ExposeController.Config.Domain = o.InitOptions.Flags.Domain
	}

	// clone the environments repo
	if wrkDir == "" {
		wrkDir, err = o.cloneJXCloudEnvironmentsRepo()
		if err != nil {
			return errors.Wrap(err, "failed to clone the jx cloud environments repo")
		}
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

	// file locations
	adminSecretsFileName := filepath.Join(dir, AdminSecretsFile)
	configFileName := filepath.Join(dir, ExtraValuesFile)

	cloudEnvironmentValuesLocation := filepath.Join(makefileDir, CloudEnvValuesFile)
	cloudEnvironmentSecretsLocation := filepath.Join(makefileDir, CloudEnvSecretsFile)
	cloudEnvironmentSopsLocation := filepath.Join(makefileDir, CloudEnvSopsConfigFile)

	client, err := o.KubeClient()
	if err != nil {
		return err
	}
	secretResources := client.CoreV1().Secrets(ns)
	oldSecret, err := secretResources.Get(JXInstallConfig, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get the jx-install-config secret")
	}

	if oldSecret == nil {
		return errors.Wrap(err, "secret jx-install-config doesn't exist, aborting")
	}

	if targetVersion != currentVersion {
		log.Infof("Upgrading platform from version %s to version %s\n", util.ColorInfo(currentVersion), util.ColorInfo(targetVersion))
	} else if o.AlwaysUpgrade {
		log.Infof("Rerunning platform version %s\n", util.ColorInfo(targetVersion))
	} else {
		log.Infof("Already installed platform version %s. Skipping upgrade process.\n", util.ColorInfo(targetVersion))
		return nil
	}

	err = o.removeFileIfExists(adminSecretsFileName)
	if err != nil {
		return errors.Wrapf(err, "unable to remove %s if exist", adminSecretsFileName)
	}

	err = o.removeFileIfExists(configFileName)
	if err != nil {
		return errors.Wrapf(err, "unable to remove %s if exist", configFileName)
	}

	log.Infof("Creating %s from %s\n", util.ColorInfo(adminSecretsFileName), util.ColorInfo(JXInstallConfig))
	err = ioutil.WriteFile(adminSecretsFileName, oldSecret.Data[AdminSecretsFile], 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to write the config file %s", adminSecretsFileName)
	}

	o.Debugf("%s from %s is %s\n", AdminSecretsFile, JXInstallConfig, oldSecret.Data[AdminSecretsFile])

	if o.UpdateSecrets {
		// load admin secrets service from adminSecretsFileName
		err = o.AdminSecretsService.NewAdminSecretsConfigFromSecret(adminSecretsFileName)
		if err != nil {
			return errors.Wrap(err, "failed to create the admin secret config service from the secrets file")
		}

		o.AdminSecretsService.NewMavenSettingsXML()
		adminSecrets := &o.AdminSecretsService.Secrets


		o.Debugf("Rewriting secrets file to %s\n", util.ColorInfo(adminSecretsFileName))
		err = configStore.WriteObject(adminSecretsFileName, adminSecrets)
		if err != nil {
			return errors.Wrapf(err, "writing the admin secrets in the secrets file '%s'", adminSecretsFileName)
		}
	}

	log.Infof("Creating %s from %s\n", util.ColorInfo(configFileName), util.ColorInfo(JXInstallConfig))
	err = ioutil.WriteFile(configFileName, oldSecret.Data[ExtraValuesFile], 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to write the config file %s", configFileName)
	}

	sopsFileExists, err := util.FileExists(cloudEnvironmentSopsLocation)
	if err != nil {
		return errors.Wrap(err, "failed to look for "+cloudEnvironmentSopsLocation)
	}

	if sopsFileExists {
		log.Infof("Attempting to decrypt secrets file %s\n", util.ColorInfo(cloudEnvironmentSecretsLocation))
		// need to decrypt secrets now
		err = o.Helm().DecryptSecrets(cloudEnvironmentSecretsLocation)
		if err != nil {
			return errors.Wrap(err, "failed to decrypt "+cloudEnvironmentSecretsLocation)
		}

		cloudEnvironmentSecretsDecryptedLocation := filepath.Join(makefileDir, CloudEnvSecretsFile+".dec")
		decryptedSecretsFile, err := util.FileExists(cloudEnvironmentSecretsDecryptedLocation)
		if err != nil {
			return errors.Wrap(err, "failed to look for "+cloudEnvironmentSecretsDecryptedLocation)
		}

		if decryptedSecretsFile {
			log.Infof("Successfully decrypted %s\n", util.ColorInfo(cloudEnvironmentSecretsDecryptedLocation))
			cloudEnvironmentSecretsLocation = cloudEnvironmentSecretsDecryptedLocation
		}
	}

	valueFiles := []string{cloudEnvironmentValuesLocation, configFileName, adminSecretsFileName, cloudEnvironmentSecretsLocation}
	valueFiles, err = helm.AppendMyValues(valueFiles)
	if err != nil {
		return errors.Wrap(err, "failed to append the myvalues.yaml file")
	}

	values := []string{}
	if o.Set != "" {
		sets := strings.Split(o.Set, ",")
		values = append(values, sets...)
	}

	for _, v := range valueFiles {
		o.Debugf("Adding values file %s\n", util.ColorInfo(v))
	}

	err = o.Helm().UpgradeChart(o.Chart, o.ReleaseName, ns, targetVersion, false, -1, false, false, values,
		valueFiles, "", "", "")
	if err != nil {
		return errors.Wrap(err, "unable to upgrade helm chart")
	}

	if o.Flags.CleanupTempFiles {
		err = o.removeFileIfExists(configFileName)
		if err != nil {
			return errors.Wrap(err, "failed to cleanup the config file")
		}

		err = o.removeFileIfExists(adminSecretsFileName)
		if err != nil {
			return errors.Wrap(err, "failed to cleanup the admin secrets file")
		}
	}

	return nil
}

func (o *UpgradePlatformOptions) removeFileIfExists(fileName string) error {
	fileNameExists, err := util.FileExists(fileName)
	if err != nil {
		return errors.Wrapf(err, "unable to determine if %s exist", fileName)
	}
	if fileNameExists {
		o.Debugf("Removing values file %s\n", util.ColorInfo(fileName))
		err = os.Remove(fileName)
		if err != nil {
			return errors.Wrapf(err, "failed to remove %s", fileName)
		}
	}
	return nil
}
