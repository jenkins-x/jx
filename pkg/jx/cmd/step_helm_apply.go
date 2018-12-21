package cmd

import (
	"io"
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
	kubeClient, curNs, err := o.KubeClient()
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

	secretsFileExists, err := o.ensureHelmSecrets(dir, helm.SecretsFileName)
	if err != nil {
		return errors.Wrap(err, "ensuring helm secrets file")
	}
	if !secretsFileExists {
		// Destroy the secrets temporary file that contain sensitive information
		defer util.DestroyFile(filepath.Join(dir, helm.SecretsFileName))
	}

	valueFiles := []string{}
	for _, name := range defaultValueFileNames {
		file := filepath.Join(dir, name)
		exists, err := util.FileExists(file)
		if exists && err == nil {
			valueFiles = append(valueFiles, file)
		}
	}

	log.Infof("Using values files: %s\n", strings.Join(valueFiles, ", "))

	if o.Wait {
		timeout := 600
		err = o.Helm().UpgradeChart(chartName, releaseName, ns, nil, true, &timeout, o.Force, true, nil, valueFiles,
			"", "", "")
	} else {
		err = o.Helm().UpgradeChart(chartName, releaseName, ns, nil, true, nil, o.Force, false, nil, valueFiles, "",
			"", "")
	}
	if err != nil {
		return errors.Wrapf(err, "upgrading helm chart '%s'", chartName)
	}
	return nil
}

// ensureHelmSecrets ensures that the provided filename exists. If it does not, it will automatically create it and
// populate it with secrets from the system vault. If the file exists, it naively assumes it is populated and won't
// do any checks.
// Returns true if the file already existed.
func (o *StepHelmApplyOptions) ensureHelmSecrets(dir string, filename string) (bool, error) {
	exists, err := util.FileExists(filename)
	if err != nil {
		return false, errors.Wrapf(err, "checking file '%s' exits", filename)
	}
	// The secrets file does not exist. Populate its values from the system vault
	if !exists {
		if !o.Factory.UseVault() {
			log.Warnf("no Vault found and no file '%s' exists in directory '%s'\n", filename, dir)
			return false, nil
		}
		client, err := o.Factory.GetSystemVaultClient()
		if err != nil {
			return exists, errors.Wrap(err, "retrieving the system Vault")
		}
		secretNames, err := client.List(vault.GitOpsSecretsPath)
		if err != nil {
			return exists, errors.Wrap(err, "listing the install secrets in Vault")
		}

		for _, secretName := range secretNames {
			if secretName == filename {
				secretPath := vault.GitOpsSecretPath(filename)
				secret, err := client.Read(secretPath)
				if err != nil {
					return exists, errors.Wrapf(err, "retrieving the secret '%s' from Vault", secretPath)
				}
				storage := configio.NewFileStore()
				secretsFile := filepath.Join(dir, filename)
				err = storage.WriteObject(secretsFile, secret)
				if err != nil {
					return exists, errors.Wrapf(err, "saving the helm secrets in file '%s'", secretsFile)
				}
				break
			}
		}
	}
	return exists, nil
}
