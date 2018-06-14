package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

const (
	defaultIstioNamespace   = "istio-system"
	defaultIstioReleaseName = "istio"
	defaultIstioPassword    = "istio"
	defaultIstioConfigDir   = "/istio_service_dir"
)

var (
	createAddonIstioLong = templates.LongDesc(`
		Creates the istio addon for service mesh on kubernetes
`)

	createAddonIstioExample = templates.Examples(`
		# Create the istio addon 
		jx create addon istio

		# Create the istio addon in a custom namespace
		jx create addon istio -n mynamespace
	`)
)

// CreateAddonIstioOptions the options for the create spring command
type CreateAddonIstioOptions struct {
	CreateAddonOptions

	Chart             string
	Password          string
	ConfigDir         string
	NoInjectorWebhook bool
	Dir               string
}

// NewCmdCreateAddonIstio creates a command object for the "create" command
func NewCmdCreateAddonIstio(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateAddonIstioOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "istio",
		Short:   "Create the Istio addon for service mesh",
		Aliases: []string{"env"},
		Long:    createAddonIstioLong,
		Example: createAddonIstioExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd, defaultIstioNamespace, defaultIstioReleaseName)

	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The version of the Istio chart to use")
	cmd.Flags().StringVarP(&options.Password, "password", "p", defaultIstioPassword, "The default password to use for Istio")
	cmd.Flags().StringVarP(&options.ConfigDir, "config-dir", "d", defaultIstioConfigDir, "The config directory to use")
	cmd.Flags().StringVarP(&options.Chart, optionChart, "c", "", "The name of the chart to use")
	cmd.Flags().BoolVarP(&options.NoInjectorWebhook, "no-injector-webhook", "", false, "Disables the injector webhook")
	return cmd
}

// Run implements the command
func (o *CreateAddonIstioOptions) Run() error {
	if o.Chart == "" {
		// lets git clone the source to find the istio charts as they are not published anywhere yet
		dir, err := o.getIstioChartsFromGitHub()
		if err != nil {
			return err
		}
		chartDir := filepath.Join(dir, kube.ChartIstio)
		exists, err := util.FileExists(chartDir)
		if !exists {
			return fmt.Errorf("Could not find folder %s inside istio clone at %s", kube.ChartIstio, dir)
		}
		o.Dir = dir
		o.Chart = kube.ChartIstio
	}
	if o.ReleaseName == "" {
		return util.MissingOption(optionRelease)
	}
	if o.Chart == "" {
		return util.MissingOption(optionChart)
	}
	_, _, err := o.KubeClient()
	if err != nil {
		return err
	}

	devNamespace, _, err := kube.GetDevNamespace(o.kubeClient, o.currentNamespace)
	if err != nil {
		return fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
	}

	log.Infof("found dev namespace %s\n", devNamespace)

	values := []string{}
	if o.NoInjectorWebhook {
		values = append(values, "sidecarInjectorWebhook.enabled=false")
	}
	err = o.installChartAt(o.Dir, o.ReleaseName, o.Chart, o.Version, o.Namespace, true, values)
	if err != nil {
		return fmt.Errorf("istio deployment failed: %v", err)
	}
	return nil
}

func (o *CreateAddonIstioOptions) getIstioChartsFromGitHub() (string, error) {
	answer, err := ioutil.TempDir("", "istio")
	if err != nil {
		return answer, err
	}
	gitRepo := "https://github.com/istio/istio.git"
	o.Printf("Cloning %s to %s\n", util.ColorInfo(gitRepo), util.ColorInfo(answer))
	err = gits.GitClone(gitRepo, answer)
	if err != nil {
		return answer, err
	}
	return answer, nil
}
