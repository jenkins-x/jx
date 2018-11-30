package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd/storage"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/vault"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	cmd.Flags().StringVarP(&options.ReleaseName, "name", "", "", "The name of the release")
	cmd.Flags().BoolVarP(&options.Wait, "wait", "", true, "Wait for Kubernetes readiness probe to confirm deployment")
	cmd.Flags().BoolVarP(&options.Force, "force", "f", true, "Whether to to pass '--force' to helm to help deal with upgrading if a previous promote failed")
	cmd.Flags().BoolVar(&options.DisableHelmVersion, "no-helm-version", false, "Don't set Chart version before applying")

	return cmd
}

func (o *StepHelmApplyOptions) Run() error {
	var err error
	chartName := o.Dir
	dir := o.Dir
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
	if ns == "" {
		return fmt.Errorf("No --namespace option specified or $DEPLOY_NAMESPACE environment variable available")
	}

	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return err
	}
	err = kube.EnsureNamespaceCreated(kubeClient, ns, nil, nil)
	if err != nil {
		return err
	}

	releaseName := o.ReleaseName
	if releaseName == "" {
		releaseName = ns
		if helmBinary != "helm" || noTiller || helmTemplate {
			releaseName = "jx"
		}
	}

	info := util.ColorInfo
	log.Infof("Applying helm chart at %s as release name %s to namespace %s\n", info(dir), info(releaseName), info(ns))

	o.Helm().SetCWD(dir)

	secretsAlreadyExisted, err := o.ensureHelmSecrets(helm.SecretsFileName)
	if !secretsAlreadyExisted {
		// Make sure we destroy any temporary files created that contain sensitive information.
		defer util.DestroyFile(helm.SecretsFileName)
	}

	// lets discover any local value files
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
		err = o.Helm().UpgradeChart(chartName, releaseName, ns, nil, true, &timeout, o.Force, true, nil, valueFiles, "")
	} else {
		err = o.Helm().UpgradeChart(chartName, releaseName, ns, nil, true, nil, o.Force, false, nil, valueFiles, "")
	}
	if err != nil {
		return err
	}
	return nil
}

// ensureHelmSecrets ensures that the provided filename exists. If it does not, it will automatically create it and
// populate it with secrets from the system vault. If the file exists, it naively assumes it is populated and won't
// do any checks.
// Returns true if the file already existed.
func (o *StepHelmApplyOptions) ensureHelmSecrets(filename string) (bool, error) {
	exists, _ := util.FileExists(filename)
	if !exists {
		// The secrets file does not exist. Populate its values from the system vault
		client, err := o.Factory.GetSystemVault()
		if err != nil {
			return exists, errors.Wrapf(err,
				"Unable to populate helm secrets. No %s file found nor system vault", filename)
		}

		secretNames, err := client.List(vault.InstallSecretsPrefix)
		allSecrets := make(map[string]interface{})

		var wg sync.WaitGroup
		secretsChannel := make(chan map[string]interface{})
		wg.Add(len(secretNames))
		for _, secretName := range secretNames {
			go func(filename string) {
				defer wg.Done()
				secret, err := client.Read(vault.InstallSecretsPrefix + filename)
				if err != nil {
					log.Errorf("Error retrieving secret %s from vault: %v", filename, err)
				}
				secretsChannel <- secret
			}(secretName)
		}
		go func() {
			// Wait for the the waitgroup, then close the channel
			wg.Wait()
			close(secretsChannel)
		}()

		// Merge the secrets together
		for secret := range secretsChannel {
			util.CombineMapTrees(allSecrets, secret)
		}

		// Now save the map as yaml to filename
		s := storage.NewFileStore()
		err = s.WriteObject(filename, allSecrets)
		if err != nil {
			return exists, errors.Wrapf(err, "Unable to save helm secrets to %s", filename)
		}

	}
	return exists, nil
}
