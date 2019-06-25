package helm

import (
	"fmt"

	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/helm"
	configio "github.com/jenkins-x/jx/pkg/io"
	"github.com/jenkins-x/jx/pkg/io/secrets"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/vault"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// StepHelmApplyOptions contains the command line flags
type StepHelmApplyOptions struct {
	StepHelmOptions

	Namespace          string
	ReleaseName        string
	Wait               bool
	Force              bool
	DisableHelmVersion bool
	Vault              bool
	UseTempDir         bool
}

var (
	StepHelmApplyLong = templates.LongDesc(`
		Applies the helm chart in a given directory.

		This step is usually used to apply any GitOps promotion changes into a Staging or Production cluster.
`)

	StepHelmApplyExample = templates.Examples(`
		# apply the chart in the env folder to namespace jx-staging 
		jx step helm apply --dir env --namespace jx-staging

`)

	defaultValueFileNames = []string{"values.yaml", "myvalues.yaml", helm.SecretsFileName, filepath.Join("env", helm.SecretsFileName)}
)

func NewCmdStepHelmApply(commonOpts *opts.CommonOptions) *cobra.Command {
	options := StepHelmApplyOptions{
		StepHelmOptions: StepHelmOptions{
			StepOptions: opts.StepOptions{
				CommonOptions: commonOpts,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "apply",
		Short:   "Applies the helm chart in a given directory",
		Aliases: []string{""},
		Long:    StepHelmApplyLong,
		Example: StepHelmApplyExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	options.addStepHelmFlags(cmd)

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "", "", "The Kubernetes namespace to apply the helm chart to")
	cmd.Flags().StringVarP(&options.ReleaseName, "name", "n", "", "The name of the release")
	cmd.Flags().BoolVarP(&options.Wait, "wait", "", true, "Wait for Kubernetes readiness probe to confirm deployment")
	cmd.Flags().BoolVarP(&options.Force, "force", "f", true, "Whether to to pass '--force' to helm to help deal with upgrading if a previous promote failed")
	cmd.Flags().BoolVar(&options.DisableHelmVersion, "no-helm-version", false, "Don't set Chart version before applying")
	cmd.Flags().BoolVarP(&options.Vault, "vault", "", false, "Helm secrets are stored in vault")
	cmd.Flags().BoolVarP(&options.UseTempDir, "use-temp-dir", "", true, "Whether to build and apply the helm chart from a temporary directory - to avoid updating the local values.yaml file from the generated file as part of the apply which could get accidentally checked into git")

	return cmd
}

func (o *StepHelmApplyOptions) Run() error {
	var err error
	chartName := o.Dir
	dir := o.Dir
	releaseName := o.ReleaseName

	// let allow arguments to be passed in like for `helm install releaseName dir`
	args := o.Args
	if releaseName == "" && len(args) > 0 {
		releaseName = args[0]
	}
	if dir == "" && len(args) > 1 {
		dir = args[1]
	}

	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	if !o.DisableHelmVersion {
		(&StepHelmVersionOptions{
			StepHelmOptions: StepHelmOptions{
				StepOptions: opts.StepOptions{
					CommonOptions: &opts.CommonOptions{},
				},
			},
		}).Run()
	}
	helmBinary, noTiller, helmTemplate, err := o.TeamHelmBin()
	if err != nil {
		return err
	}

	ns, err := o.GetDeployNamespace(o.Namespace)
	if err != nil {
		return err
	}
	kubeClient, err := o.KubeClient()
	if err != nil {
		return err
	}

	err = kube.EnsureNamespaceCreated(kubeClient, ns, nil, nil)
	if err != nil {
		return err
	}

	_, devNs, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}

	if releaseName == "" {
		if devNs == ns {
			releaseName = opts.JenkinsXPlatformRelease
		} else {
			releaseName = ns

			if helmBinary != "helm" || noTiller || helmTemplate {
				releaseName = "jx"
			}
		}
	}
	info := util.ColorInfo

	path, err := filepath.Abs(dir)
	if err != nil {
		return errors.Wrapf(err, "could not find absolute path of dir %s", dir)
	}
	dir = path

	if o.UseTempDir {
		rootTmpDir, err := ioutil.TempDir("", "jx-helm-apply-")
		if err != nil {
			return errors.Wrapf(err, "failed to create a temporary directory to apply the helm chart")
		}
		defer os.RemoveAll(rootTmpDir)

		// lets use the same child dir name as the original as helm is quite particular about the name of the directory it runs from
		_, name := filepath.Split(dir)
		if name == "" {
			return fmt.Errorf("could not find the relative name of the directory %s", dir)
		}
		tmpDir := filepath.Join(rootTmpDir, name)
		log.Logger().Infof("Copying the helm source directory %s to a temporary location for building and applying %s\n", info(dir), info(tmpDir))

		err = util.CopyDir(dir, tmpDir, false)
		if err != nil {
			return errors.Wrapf(err, "failed to copy helm dir %s to temporary dir %s", dir, tmpDir)
		}
		dir = tmpDir
	}
	log.Logger().Infof("Applying helm chart at %s as release name %s to namespace %s", info(dir), info(releaseName), info(ns))

	o.Helm().SetCWD(dir)

	valueFiles := []string{}
	for _, name := range defaultValueFileNames {
		file := filepath.Join(dir, name)
		exists, err := util.FileExists(file)
		if exists && err == nil {
			valueFiles = append(valueFiles, file)
		}
	}

	if (o.GetSecretsLocation() == secrets.VaultLocationKind) || o.Vault {
		store := configio.NewFileStore()
		secretsFiles, err := o.fetchSecretFilesFromVault(dir, store)
		if err != nil {
			return errors.Wrap(err, "fetching secrets files from vault")
		}
		for _, sf := range secretsFiles {
			if util.StringArrayIndex(valueFiles, sf) < 0 {
				log.Logger().Infof("adding secret file %s", sf)
				valueFiles = append(valueFiles, sf)
			}
		}
		defer func() {
			for _, secretsFile := range secretsFiles {
				err := util.DestroyFile(secretsFile)
				if err != nil {
					log.Logger().Warnf("Failed to cleanup the secrets files (%s): %v",
						strings.Join(secretsFiles, ", "), err)
				}
			}
		}()
	}

	secretURLClient, err := o.GetSecretURLClient()
	if err != nil {
		return errors.Wrap(err, "failed to create a Secret RL client")
	}
	chartValues, err := helm.GenerateValues(dir, nil, true, secretURLClient)
	if err != nil {
		return errors.Wrapf(err, "generating values.yaml for tree from %s", dir)
	}
	chartValuesFile := filepath.Join(dir, helm.ValuesFileName)
	err = ioutil.WriteFile(chartValuesFile, chartValues, 0755)
	if err != nil {
		return errors.Wrapf(err, "writing values.yaml for tree to %s", chartValuesFile)
	}
	log.Logger().Infof("Wrote chart values.yaml %s generated from directory tree", chartValuesFile)

	data, err := ioutil.ReadFile(chartValuesFile)
	if err != nil {
		log.Logger().Warnf("failed to load file %s: %s", chartValuesFile, err.Error())
	} else {
		log.Logger().Infof("generated helm %s", chartValuesFile)
		log.Logger().Infof("\n%s\n", util.ColorStatus(string(data)))
	}

	log.Logger().Infof("Using values files: %s", strings.Join(valueFiles, ", "))

	_, err = o.HelmInitDependencyBuild(dir, o.DefaultReleaseCharts(), valueFiles)
	if err != nil {
		return err
	}

	err = o.applyAppsTemplateOverrides(chartName)
	if err != nil {
		return errors.Wrap(err, "applying app chart overrides")
	}
	err = o.applyTemplateOverrides(chartName)
	if err != nil {
		return errors.Wrap(err, "applying chart overrides")
	}

	helmOptions := helm.InstallChartOptions{
		Chart:       chartName,
		ReleaseName: releaseName,
		Ns:          ns,
		NoForce:     !o.Force,
		ValueFiles:  valueFiles,
	}
	if o.Wait {
		helmOptions.Wait = true
		err = o.InstallChartWithOptionsAndTimeout(helmOptions, "600")
	} else {
		err = o.InstallChartWithOptions(helmOptions)
	}
	if err != nil {
		return errors.Wrapf(err, "upgrading helm chart '%s'", chartName)
	}
	return nil
}

func (o *StepHelmApplyOptions) applyTemplateOverrides(chartName string) error {
	log.Logger().Infof("Applying chart overrides")
	templateOverrides, err := filepath.Glob(chartName + "/../*/templates/*.yaml")
	for _, overrideSrc := range templateOverrides {
		if !strings.Contains(overrideSrc, "/env/") {
			data, err := ioutil.ReadFile(overrideSrc)
			if err == nil {
				writeTemplateParts := strings.Split(overrideSrc, string(os.PathSeparator))
				depChartsDir := filepath.Join(chartName, "charts")
				depChartName := writeTemplateParts[len(writeTemplateParts)-3]
				templateName := writeTemplateParts[len(writeTemplateParts)-1]
				depChartDir := filepath.Join(depChartsDir, depChartName)
				// If the chart directory does not exist explode the tgz
				if exists, err := util.DirExists(depChartDir); err == nil && !exists {
					chartArchives, _ := filepath.Glob(filepath.Join(depChartsDir, depChartName+"*.tgz"))
					if len(chartArchives) == 1 {
						log.Logger().Infof("Exploding chart %s", chartArchives[0])
						archiver.Unarchive(chartArchives[0], depChartsDir)
						// Remove the unexploded chart
						os.Remove(chartArchives[0])
					}
				}
				overrideDst := filepath.Join(depChartDir, "templates", templateName)
				log.Logger().Infof("Copying chart override %s", overrideSrc)
				err = ioutil.WriteFile(overrideDst, data, util.DefaultWritePermissions)
				if err != nil {
					log.Logger().Warnf("Error copying template %s to %s", overrideSrc, overrideDst)
				}

			}
		}
	}
	return err
}

func (o *StepHelmApplyOptions) applyAppsTemplateOverrides(chartName string) error {
	log.Logger().Infof("Applying Apps chart overrides")
	templateOverrides, err := filepath.Glob(chartName + "/../*/*/templates/app.yaml")
	for _, overrideSrc := range templateOverrides {
		data, err := ioutil.ReadFile(overrideSrc)
		if err == nil {
			writeTemplateParts := strings.Split(overrideSrc, string(os.PathSeparator))
			depChartsDir := filepath.Join(chartName, "charts")
			depChartName := writeTemplateParts[len(writeTemplateParts)-3]
			templateName := writeTemplateParts[len(writeTemplateParts)-1]
			depChartDir := filepath.Join(depChartsDir, depChartName)
			chartArchives, _ := filepath.Glob(filepath.Join(depChartsDir, depChartName+"*.tgz"))
			if len(chartArchives) == 1 {
				uuid, _ := uuid.NewUUID()
				log.Logger().Infof("Exploding App chart %s", chartArchives[0])
				explodedChartTempDir := filepath.Join(os.TempDir(), uuid.String())
				if err = archiver.Unarchive(chartArchives[0], explodedChartTempDir); err != nil {
					return defineAppsChartOverridingError(chartName, err)
				}
				overrideDst := filepath.Join(explodedChartTempDir, depChartName, "templates", templateName)
				log.Logger().Infof("Copying chart override %s", overrideSrc)
				err = ioutil.WriteFile(overrideDst, data, util.DefaultWritePermissions)
				if err != nil {
					log.Logger().Warnf("Error copying template %s to %s", overrideSrc, overrideDst)
				}
				if err = os.Remove(chartArchives[0]); err != nil {
					return defineAppsChartOverridingError(chartName, err)
				}
				if err = archiver.Archive([]string{filepath.Join(explodedChartTempDir, depChartName)}, chartArchives[0]); err != nil {
					return defineAppsChartOverridingError(chartName, err)
				}
				if err = os.RemoveAll(explodedChartTempDir); err != nil {
					log.Logger().Warnf("There was a problem deleting the temp folder %s", depChartDir)
				}
			}
		}
	}
	return err
}

func defineAppsChartOverridingError(chartName string, err error) error {
	return errors.Wrapf(err, "there was a problem overriding the chart %s", chartName)
}

func (o *StepHelmApplyOptions) fetchSecretFilesFromVault(dir string, store configio.ConfigStore) ([]string, error) {
	log.Logger().Infof("Fetching secrets from vault into directory %q", dir)
	files := []string{}
	client, err := o.SystemVaultClient(kube.DefaultNamespace)
	if err != nil {
		return files, errors.Wrap(err, "retrieving the system Vault")
	}
	secretNames, err := client.List(vault.GitOpsSecretsPath)
	if err != nil {
		return files, errors.Wrap(err, "listing the GitOps secrets in Vault")
	}
	secretPaths := []string{}
	for _, secretName := range secretNames {
		if secretName == vault.GitOpsTemplatesPath {
			templatesPath := vault.GitOpsSecretPath(vault.GitOpsTemplatesPath)
			templatesSecretNames, err := client.List(templatesPath)
			if err == nil {
				for _, templatesSecretName := range templatesSecretNames {
					templateSecretPath := vault.GitOpsTemplatesPath + templatesSecretName
					secretPaths = append(secretPaths, templateSecretPath)
				}
			}
		} else {
			secretPaths = append(secretPaths, secretName)
		}
	}

	for _, secretPath := range secretPaths {
		gitopsSecretPath := vault.GitOpsSecretPath(secretPath)
		secret, err := client.ReadYaml(gitopsSecretPath)
		if err != nil {
			return files, errors.Wrapf(err, "retrieving the secret %q from Vault", secretPath)
		}
		if secret == "" {
			return files, fmt.Errorf("secret %q is empty", secretPath)
		}
		secretFile := filepath.Join(dir, secretPath)
		err = store.Write(secretFile, []byte(secret))
		if err != nil {
			return files, errors.Wrapf(err, "saving the secret file %q", secretFile)
		}
		log.Logger().Infof("Saved secrets file %s", util.ColorInfo(secretFile))
		files = append(files, secretFile)
	}
	return files, nil
}
