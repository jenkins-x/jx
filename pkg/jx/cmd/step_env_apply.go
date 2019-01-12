package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/extensions"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// StepEnvApplyOptions contains the command line flags
type StepEnvApplyOptions struct {
	StepEnvOptions

	Namespace          string
	Dir                string
	ReleaseName        string
	Wait               bool
	Force              bool
	DisableHelmVersion bool
}

var (

	// stepEnvApplyLong long description
	stepEnvApplyLong = templates.LongDesc(`
		Applies the GitOps source code (by default in the current directory) to the Environment.

		This command will lazily create an environment, setup Helm and build and apply any helm charts defined in the env/Chart.yaml
`)

	// StepEnvApplyExample example
	stepEnvApplyExample = templates.Examples(`
		# setup and/or update the helm charts for the environment
		jx step env apply --namespace jx-staging
`)
)

// NewCmdStepEnvApply registers the command
func NewCmdStepEnvApply(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := StepEnvApplyOptions{
		StepEnvOptions: StepEnvOptions{
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
		Short:   "Applies the GitOps source code to an environment",
		Aliases: []string{""},
		Long:    stepEnvApplyLong,
		Example: stepEnvApplyExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "The Kubernetes namespace to apply the helm charts to")
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "The directory to look for the environment chart")

	// step helm apply flags
	cmd.Flags().BoolVarP(&options.Wait, "wait", "", true, "Wait for Kubernetes readiness probe to confirm deployment")
	cmd.Flags().BoolVarP(&options.Force, "force", "f", true, "Whether to to pass '--force' to helm to help deal with upgrading if a previous promote failed")
	cmd.Flags().BoolVar(&options.DisableHelmVersion, "no-helm-version", false, "Don't set Chart version before applying")

	return cmd
}

// Run performs the comamand
func (o *StepEnvApplyOptions) Run() error {

	var err error
	dir := o.Dir
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	ns := o.Namespace
	if ns == "" {
		ns = os.Getenv("DEPLOY_NAMESPACE")
	}
	if ns == "" {
		return fmt.Errorf("No --namespace option specified or $DEPLOY_NAMESPACE environment variable available")
	}

	o.SetDevNamespace(ns)
	kubeClient, err := o.KubeClient()
	if err != nil {
		return errors.Wrapf(err, "Could not connect to the kubernetes cluster!")
	}

	apisClient, err := o.ApiExtensionsClient()
	if err != nil {
		return errors.Wrap(err, "failed to create the API extensions client")
	}
	kube.RegisterAllCRDs(apisClient)
	if err != nil {
		return err
	}

	// now lets find the dev environment to know what kind of helmer to use
	chartFile := filepath.Join(dir, helm.ChartFileName)
	exists, err := util.FileExists(chartFile)
	if err != nil {
		return err
	}
	if !exists {
		envDir := filepath.Join(dir, "env")
		chartFile2 := filepath.Join(envDir, helm.ChartFileName)
		exists2, err := util.FileExists(chartFile2)
		if exists2 && err == nil {
			dir = envDir
		} else {
			return fmt.Errorf("There is no Environment chart file at %s or %s\nPlease try specify the directory containing the Chart.yaml or env/Chart.yaml with --dir", chartFile, chartFile2)
		}
	}
	devEnvFile := filepath.Join(dir, "templates", "dev-env.yaml")
	exists, err = util.FileExists(chartFile)
	if exists && err == nil {
		// lets setup the Helmer based on the current settings
		log.Infof("Loading the latest Dev Environment configuration from %s\n", devEnvFile)

		env := v1.Environment{}
		data, err := ioutil.ReadFile(devEnvFile)
		if err != nil {
			return errors.Wrapf(err, "Could not load file %s", devEnvFile)
		}
		err = yaml.Unmarshal(data, &env)
		if err != nil {
			return errors.Wrapf(err, "Failed to unmarshall YAML file %s", devEnvFile)
		}

		teamSettings := &env.Spec.TeamSettings

		// disable the modify of the Dev Environment lazily...
		o.modifyDevEnvironmentFn = func(callback func(env *v1.Environment) error) error {
			callback(&env)
			return nil
		}

		o.helm = o.CreateHelm(false, teamSettings.HelmBinary, teamSettings.NoTiller, teamSettings.HelmTemplate)

		// ensure there's a development namespace setup
		err = kube.EnsureDevNamespaceCreatedWithoutEnvironment(kubeClient, ns)
		if err != nil {
			return errors.Wrapf(err, "Failed to create Namespace %s for Development Environment", ns)
		}

		if o.ReleaseName == "" {
			o.ReleaseName = "jenkins-x"
		}
	} else {
		// ensure there's a development namespace setup
		err = kube.EnsureNamespaceCreated(kubeClient, ns, nil, nil)
		if err != nil {
			return errors.Wrapf(err, "Failed to create Namespace %s for Environment", ns)
		}
	}

	stepHelmBuild := &StepHelmBuildOptions{
		StepHelmOptions: StepHelmOptions{
			StepOptions: StepOptions{
				CommonOptions: o.CommonOptions,
			},
			Dir: dir,
		},
	}
	err = stepHelmBuild.Run()
	if err != nil {
		return errors.Wrapf(err, "Failed to build helm chart in dir %s", dir)
	}

	stepApply := &StepHelmApplyOptions{
		StepHelmOptions:    stepHelmBuild.StepHelmOptions,
		Namespace:          ns,
		ReleaseName:        o.ReleaseName,
		Wait:               o.Wait,
		DisableHelmVersion: o.DisableHelmVersion,
		Force:              o.Force,
	}
	err = stepApply.Run()
	if err != nil {
		return errors.Wrapf(err, "Failed to apply helm chart in dir %s", dir)
	}
	log.Infof("Environment applied in namespace %s\n", util.ColorInfo(ns))
	// Now run any post install actions
	jxClient, _, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	err = extensions.OnApply(jxClient, kubeClient, o.devNamespace, o.Helm(), defaultInstallTimeout)
	if err != nil {
		return err
	}
	return nil
}
