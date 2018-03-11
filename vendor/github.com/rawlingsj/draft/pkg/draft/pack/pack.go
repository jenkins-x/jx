package pack

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/Azure/draft/pkg/osutil"
)

const (
	// ChartfileName is the default Chart file name.
	ChartfileName = "Chart.yaml"
	// DockerfileName is the name of the Dockerfile.
	DockerfileName = "Dockerfile"
	// ValuesfileName is the default values file name.
	ValuesfileName = "values.yaml"
	// IgnorefileName is the name of the Helm ignore file.
	IgnorefileName = ".helmignore"
	// DeploymentName is the name of the deployment file.
	DeploymentName = "deployment.yaml"
	// ServiceName is the name of the service file.
	ServiceName = "service.yaml"
	// IngressName is the name of the ingress file.
	IngressName = "ingress.yaml"
	// NotesName is the name of the NOTES.txt file.
	NotesName = "NOTES.txt"
	// HelpersName is the name of the helpers file.
	HelpersName = "_helpers.tpl"
	// TemplatesDir is the relative directory name for templates.
	TemplatesDir = "templates"
	// ChartsDir is the directory name for the packaged chart.
	// This also doubles as the directory name for chart dependencies.
	ChartsDir = "charts"
	// HerokuLicenseName is the name of the Neroku License
	HerokuLicenseName = "NOTICE"
	// DockerignoreName is the name of the Docker ignore file
	DockerignoreName = ".dockerignore"
)

// Pack defines a Draft Starter Pack.
type Pack struct {
	// Chart is the Helm chart to be installed with the Pack.
	Chart *chart.Chart
	// Dockerfile is the pre-defined Dockerfile that will be installed with the Pack.
	Dockerfile []byte
}

// SaveDir saves a pack as files in a directory.
func (p *Pack) SaveDir(dest string) error {
	// Create the chart directory
	chartPath := filepath.Join(dest, ChartsDir)
	if err := os.Mkdir(chartPath, 0755); err != nil {
		return fmt.Errorf("Could not create %s: %s", chartPath, err)
	}
	if err := chartutil.SaveDir(p.Chart, chartPath); err != nil {
		return err
	}

	// save Dockerfile
	dockerfilePath := filepath.Join(dest, DockerfileName)
	exists, err := osutil.Exists(dockerfilePath)
	if err != nil {
		return err
	}
	if !exists {
		if err := ioutil.WriteFile(dockerfilePath, p.Dockerfile, 0644); err != nil {
			return err
		}
	}

	return nil
}
