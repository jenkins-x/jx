package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/helm"
	configio "github.com/jenkins-x/jx/pkg/io"
	"github.com/jenkins-x/jx/pkg/io/secrets"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
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
			StepOptions: StepOptions{
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
			CheckErr(err)
		},
	}
	options.addStepHelmFlags(cmd)

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "", "", "The Kubernetes namespace to apply the helm chart to")
	cmd.Flags().StringVarP(&options.ReleaseName, "name", "n", "", "The name of the release")
	cmd.Flags().BoolVarP(&options.Wait, "wait", "", true, "Wait for Kubernetes readiness probe to confirm deployment")
	cmd.Flags().BoolVarP(&options.Force, "force", "f", true, "Whether to to pass '--force' to helm to help deal with upgrading if a previous promote failed")
	cmd.Flags().BoolVar(&options.DisableHelmVersion, "no-helm-version", false, "Don't set Chart version before applying")
	cmd.Flags().BoolVarP(&options.Vault, "vault", "", false, "Helm secrets are stored in vault")

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
				StepOptions: StepOptions{
					CommonOptions: &opts.CommonOptions{},
				},
			},
		}).Run()
	}
	_, err = o.HelmInitDependencyBuild(dir, o.DefaultReleaseCharts())
	if err != nil {
		return err
	}

	helmBinary, noTiller, helmTemplate, err := o.TeamHelmBin()
	if err != nil {
		return err
	}

	ns := o.Namespace
	if ns == "" {
		ns = os.Getenv("DEPLOY_NAMESPACE")
	}
	kubeClient, curNs, err := o.KubeClientAndNamespace()
	if err != nil {
		return err
	}
	if ns == "" {
		ns = curNs
		log.Infof("No --namespace option specified or $DEPLOY_NAMESPACE environment variable available so defaulting to using namespace %s\n", ns)
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
			releaseName = JenkinsXPlatformRelease
		} else {
			releaseName = ns

			if helmBinary != "helm" || noTiller || helmTemplate {
				releaseName = "jx"
			}
		}
	}

	info := util.ColorInfo
	log.Infof("Applying helm chart at %s as release name %s to namespace %s\n", info(dir), info(releaseName), info(ns))

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
				log.Infof("adding secret file %s\n", sf)
				valueFiles = append(valueFiles, sf)
			}
		}
		defer func() {
			for _, secretsFile := range secretsFiles {
				err := util.DestroyFile(secretsFile)
				if err != nil {
					log.Warnf("Failed to cleanup the secrets files (%s): %v",
						strings.Join(secretsFiles, ", "), err)
				}
			}
		}()
	}

	chartValues, err := helm.GenerateValues(dir, nil, true)
	if err != nil {
		return errors.Wrapf(err, "generating values.yaml for tree from %s", dir)
	}
	chartValuesFile := filepath.Join(dir, helm.ValuesFileName)
	err = ioutil.WriteFile(chartValuesFile, chartValues, 0755)
	if err != nil {
		return errors.Wrapf(err, "writing values.yaml for tree to %s", chartValuesFile)
	}
	log.Infof("Wrote chart values.yaml %s generated from directory tree\n", chartValuesFile)

	data, err := ioutil.ReadFile(chartValuesFile)
	if err != nil {
		log.Warnf("failed to load file %s: %s\n", chartValuesFile, err.Error())
	} else {
		log.Infof("generated helm %s\n", chartValuesFile)
		log.Infof("\n%s\n\n", util.ColorStatus(string(data)))
	}

	log.Infof("Using values files: %s\n", strings.Join(valueFiles, ", "))

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
	log.Infof("Applying chart overrides\n")
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
						log.Infof("Exploding chart %s\n", chartArchives[0])
						archiver.Unarchive(chartArchives[0], depChartsDir)
						// Remove the unexploded chart
						os.Remove(chartArchives[0])
					}
				}
				overrideDst := filepath.Join(depChartDir, "templates", templateName)
				log.Infof("Copying chart override %s\n", overrideSrc)
				err = ioutil.WriteFile(overrideDst, data, util.DefaultWritePermissions)
				if err != nil {
					log.Warnf("Error copying template %s to %s\n", overrideSrc, overrideDst)
				}

			}
		}
	}
	return err
}

func (o *StepHelmApplyOptions) fetchSecretFilesFromVault(dir string, store configio.ConfigStore) ([]string, error) {
	log.Infof("Fetching secrets from vault into directory %q\n", dir)
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
		secretFile := filepath.Join(dir, secretPath)
		err = store.Write(secretFile, []byte(secret))
		if err != nil {
			return files, errors.Wrapf(err, "saving the secret file %q", secretFile)
		}
		log.Infof("Saved secrets file %s\n", util.ColorInfo(secretFile))
		files = append(files, secretFile)
	}
	return files, nil
}
