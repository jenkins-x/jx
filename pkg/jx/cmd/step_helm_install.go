package cmd

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/helm"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/sirupsen/logrus"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

// StepHelmInstallOptions contains the command line flags
type StepHelmInstallOptions struct {
	StepHelmOptions

	Name        string
	Namespace   string
	Version     string
	Values      []string
	ValuesFiles []string
}

var (
	StepHelmInstallLong = templates.LongDesc(`
		Installs the given chart
`)

	StepHelmInstallExample = templates.Examples(`
		# installs a helm chart
		jx step helm install foo/bar

`)
)

func NewCmdStepHelmInstall(commonOpts *opts.CommonOptions) *cobra.Command {
	options := StepHelmInstallOptions{
		StepHelmOptions: StepHelmOptions{
			StepOptions: StepOptions{
				CommonOptions: commonOpts,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "install",
		Short:   "Installs the given chart",
		Aliases: []string{""},
		Long:    StepHelmInstallLong,
		Example: StepHelmInstallExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addStepHelmFlags(cmd)

	cmd.Flags().StringVarP(&options.Name, "name", "n", "", "The name of the release to install")
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The version to install. Defaults to the latest")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "", "", "The namespace to install into. Defaults to the current namespace")
	cmd.Flags().StringArrayVarP(&options.Values, "set", "", []string{}, "The values to override in the helm chart")
	cmd.Flags().StringArrayVarP(&options.ValuesFiles, "set-file", "", []string{}, "The values files to override values in the helm chart")

	return cmd
}

func (o *StepHelmInstallOptions) Run() error {
	args := o.Args
	if len(args) == 0 {
		return fmt.Errorf("Missing chart argument")
	}
	err := o.RegisterReleaseCRD()
	if err != nil {
		return err
	}
	chart := args[0]
	releaseName := o.Name
	ns := o.Namespace
	if ns == "" {
		_, ns, err = o.KubeClientAndNamespace()
		if err != nil {
			return err
		}
	}

	version := o.Version
	if o.Version == "" {
		version = ""
	}
	helmOptions := helm.InstallChartOptions{
		Chart:       chart,
		ReleaseName: releaseName,
		Version:     version,
		Ns:          ns,
		SetValues:   o.Values,
		ValueFiles:  o.ValuesFiles,
	}
	err = o.InstallChartWithOptions(helmOptions)
	if err != nil {
		return err
	}
	logrus.Infof("Installed chart %s with name %s into namespace %s\n", util.ColorInfo(chart), util.ColorInfo(releaseName), util.ColorInfo(ns))
	return nil
}
