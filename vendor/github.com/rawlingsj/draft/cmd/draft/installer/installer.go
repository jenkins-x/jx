package installer

import (
	"errors"
	"fmt"
	"path"

	"google.golang.org/grpc"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"

	draftconfig "github.com/Azure/draft/cmd/draft/installer/config"
	"github.com/Azure/draft/pkg/version"
)

// Installer is the client used to install draftd into the kubernetes cluster via helm.
type Installer struct {
	client    helm.Interface
	config    *draftconfig.DraftConfig
	namespace string
}

// Interface defines the installer interface.
type Interface interface {
	Install() error
	Upgrade() error
}

// New creates a new Installer
func New(client helm.Interface, config *draftconfig.DraftConfig, namespace string) *Installer {
	return &Installer{
		client:    client,
		config:    config,
		namespace: namespace,
	}
}

// ReleaseName is the name of the release used when installing/uninstalling draft via helm.
const ReleaseName = "draft"

// DefaultChartFiles represent the default chart files relevant to a Draft chart installation
var DefaultChartFiles = []*chartutil.BufferedFile{
	{
		Name: chartutil.ChartfileName,
		Data: []byte(fmt.Sprintf(draftChart, version.Release)),
	},
	{
		Name: chartutil.ValuesfileName,
		Data: []byte(fmt.Sprintf(draftValues, version.Release)),
	},
	{
		Name: chartutil.IgnorefileName,
		Data: []byte(draftIgnore),
	},
	{
		Name: path.Join(chartutil.TemplatesDir, chartutil.DeploymentName),
		Data: []byte(draftDeployment),
	},
	{
		Name: path.Join(chartutil.TemplatesDir, chartutil.ServiceName),
		Data: []byte(draftService),
	},
	{
		Name: path.Join(chartutil.TemplatesDir, chartutil.NotesName),
		Data: []byte(draftNotes),
	},
	{
		Name: path.Join(chartutil.TemplatesDir, chartutil.HelpersName),
		Data: []byte(draftHelpers),
	},
}

// Install uses the helm client to install Draftd with the given config.
//
// Returns an error if the command failed.
func (in *Installer) Install() error {
	if in.config.WithTLS() {
		DefaultChartFiles = append(DefaultChartFiles, &chartutil.BufferedFile{
			// TODO: Add chartutil.SecretName to k8s.io/helm/pkg/chartutil/create.go
			Name: path.Join(chartutil.TemplatesDir, "secret.yaml"),
			Data: []byte(draftSecret),
		})
	}

	chart, err := chartutil.LoadFiles(DefaultChartFiles)
	if err != nil {
		return err
	}
	_, err = in.client.InstallReleaseFromChart(
		chart,
		in.namespace,
		helm.ReleaseName(ReleaseName),
		helm.ValueOverrides([]byte(in.config.String())),
	)
	return prettyError(err)
}

//
// Upgrade uses the helm client to upgrade Draftd using the given config.
//
// Returns an error if the command failed.
func (in *Installer) Upgrade() error {
	chart, err := chartutil.LoadFiles(DefaultChartFiles)
	if err != nil {
		return err
	}
	_, err = in.client.UpdateReleaseFromChart(
		ReleaseName,
		chart,
		helm.UpdateValueOverrides([]byte(in.config.String())),
	)
	return prettyError(err)
}

// prettyError unwraps grpc error descriptions to make them more user-friendly.
func prettyError(err error) error {
	if err == nil {
		return nil
	}

	return errors.New(grpc.ErrorDesc(err))
}
