package cmd

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/helm"
	configio "github.com/jenkins-x/jx/pkg/io"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/vault"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// StepHelmApplyOptions contains the command line flags
type StepHelmApplyOptions struct {
	StepHelmOptions

	Namespace          string
	ReleaseName        string
	Wait               bool
	Force              bool
	DisableHelmVersion bool
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

	defaultValueFileNames = []string{"values.yaml", "myvalues.yaml", helm.SecretsFileName}
)

func NewCmdStepHelmApply(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := StepHelmApplyOptions{
		StepHelmOptions: StepHelmOptions{
			StepOptions: StepOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					In:      in,
					Out:     out,
					Err:     errOut,
				},
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
		(&StepHelmVersionOptions{}).Run()
	}
	_, err = o.helmInitDependencyBuild(dir, o.defaultReleaseCharts())
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

	if releaseName == "" {
		releaseName = ns
		if helmBinary != "helm" || noTiller || helmTemplate {
			releaseName = "jx"
		}
	}

	info := util.ColorInfo
	log.Infof("Applying helm chart at %s as release name %s to namespace %s\n", info(dir), info(releaseName), info(ns))

	o.Helm().SetCWD(dir)

	if o.UseVault() {
		store := configio.NewFileStore()
		secretsFiles, err := o.fetchSecretFilesFromVault(dir, store)
		if err != nil {
			return errors.Wrap(err, "fetching secrets files from vault")
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

	valueFiles := []string{}
	for _, name := range defaultValueFileNames {
		file := filepath.Join(dir, name)
		exists, err := util.FileExists(file)
		if exists && err == nil {
			valueFiles = append(valueFiles, file)
		}
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

	log.Infof("Using values files: %s\n", strings.Join(valueFiles, ", "))

	if o.Wait {
		timeout := 600
		err = o.Helm().UpgradeChart(chartName, releaseName, ns, "", true, timeout, o.Force, true, nil, valueFiles,
			"", "", "")
	} else {
		err = o.Helm().UpgradeChart(chartName, releaseName, ns, "", true, -1, o.Force, false, nil, valueFiles, "",
			"", "")
	}
	if err != nil {
		return errors.Wrapf(err, "upgrading helm chart '%s'", chartName)
	}
	return nil
}

func (o *StepHelmApplyOptions) fetchSecretFilesFromVault(dir string, store configio.ConfigStore) ([]string, error) {
	files := []string{}
	client, err := o.CreateSystemVaultClient()
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
		secret, err := client.Read(gitopsSecretPath)
		if err != nil {
			return files, errors.Wrapf(err, "retrieving the secret '%s' from Vault", secretPath)
		}
		secretFile := filepath.Join(dir, secretPath)
		err = store.WriteObject(secretFile, secret)
		if err != nil {
			return files, errors.Wrapf(err, "saving the secret file '%s'", secretFile)
		}
		files = append(files, secretFile)
	}
	return files, nil
}
