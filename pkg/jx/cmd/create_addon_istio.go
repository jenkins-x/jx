package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jenkins-x/jx/pkg/binaries"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

const (
	defaultIstioNamespace   = "istio-system"
	defaultIstioReleaseName = "istio"
	defaultIstioPassword    = "istio"
	defaultIstioConfigDir   = "/istio_service_dir"
	defaultIstioVersion     = ""
)

var (
	createAddonIstioLong = templates.LongDesc(`
		Creates the istio addon for service mesh on Kubernetes
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
func NewCmdCreateAddonIstio(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateAddonIstioOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
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
		Use:     "istio",
		Short:   "Create the Istio addon for service mesh",
		Aliases: []string{"env"},
		Long:    createAddonIstioLong,
		Example: createAddonIstioExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd, defaultIstioNamespace, defaultIstioReleaseName, defaultIstioVersion)

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
	err := o.ensureHelm()
	if err != nil {
		return errors.Wrap(err, "failed to ensure that Helm is present")
	}
	client, err := o.KubeClient()
	if err != nil {
		return err
	}

	err = o.generateSecrets()
	if err != nil {
		return err
	}

	devNamespace, _, err := kube.GetDevNamespace(client, o.currentNamespace)
	if err != nil {
		return fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
	}

	log.Infof("found dev namespace %s\n", devNamespace)

	values := []string{}
	if o.NoInjectorWebhook {
		values = append(values, "sidecarInjectorWebhook.enabled=false")
	}
	setValues := strings.Split(o.SetValues, ",")
	values = append(values, setValues...)
	err = o.installChartAt(o.Dir, o.ReleaseName, o.Chart, o.Version, o.Namespace, true, values, nil, "")
	if err != nil {
		return fmt.Errorf("istio deployment failed: %v", err)
	}
	return nil
}

// getIstioChartsFromGitHub lets download the latest release of istio
func (o *CreateAddonIstioOptions) getIstioChartsFromGitHub() (string, error) {
	answer := ""
	latestVersion, err := util.GetLatestVersionFromGitHub("istio", "istio")
	if err != nil {
		return answer, fmt.Errorf("unable to get latest version for github.com/%s/%s %v", "istio", "istio", err)
	}

	binDir, err := util.JXBinLocation()
	if err != nil {
		return answer, err
	}

	binaryFile := "istioctl"
	extension := ""
	switch runtime.GOOS {
	case "windows":
		extension = "win.zip"
		binaryFile += ".exe"
	case "darwin":
		extension = "osx.tar.gz"
	default:
		extension = "linux.tar.gz"
	}

	clientURL := fmt.Sprintf("https://github.com/istio/istio/releases/download/%s/istio-%s-%s", latestVersion, latestVersion, extension)

	outputDir := filepath.Join(binDir, "istio-"+latestVersion.String())
	os.RemoveAll(outputDir)

	answer = outputDir

	err = os.MkdirAll(outputDir, util.DefaultWritePermissions)
	if err != nil {
		return answer, err
	}

	tarPath := filepath.Join(binDir, "istio-"+extension)
	os.Remove(tarPath)
	err = binaries.DownloadFile(clientURL, tarPath)
	if err != nil {
		return answer, err
	}

	defer os.Remove(tarPath)

	if strings.HasSuffix(extension, ".zip") {
		err = util.Unzip(tarPath, binDir)
		if err != nil {
			return answer, err
		}
	} else {
		err = util.UnTargzAll(tarPath, binDir)
		if err != nil {
			return answer, err
		}
	}
	f := filepath.Join(outputDir, "bin", binaryFile)
	exists, err := util.FileExists(f)
	if err != nil {
		return answer, err
	}
	if exists {
		binaryDest := filepath.Join(binDir, binaryFile)
		os.Remove(binaryDest)
		err = os.Rename(f, binaryDest)
		if err != nil {
			return answer, err
		}
	}
	return answer, nil
}

func (o *CreateAddonIstioOptions) generateSecrets() error {
	// generate secret for kiali && grafana
	return nil
}
