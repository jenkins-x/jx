package helm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	// LabelReleaseChartName stores the name of a chart being installed
	LabelReleaseChartName = "jenkins.io/chart"

	// LabelReleaseChartVersion stores the version of a chart installation in a label
	LabelReleaseChartVersion = "jenkins.io/version"
)

// HelmTemplate implements common helm actions but purely as client side operations
// delegating a separate Helmer such as HelmCLI for the client side operations
type HelmTemplate struct {
	Client    *HelmCLI
	OutputDir string
	CWD       string
	Binary    string
	Runner    *util.Command
}

// NewHelmTemplate creates a new HelmTemplate instance configured to the given client side Helmer
func NewHelmTemplate(client *HelmCLI, workDir string) *HelmTemplate {
	cli := &HelmTemplate{
		Client:    client,
		OutputDir: workDir,
		Runner:    client.Runner,
		Binary:    "kubectl",
		CWD:       client.CWD,
	}
	return cli
}

// SetHost is used to point at a locally running tiller
func (h *HelmTemplate) SetHost(tillerAddress string) {
	// NOOP
}

// SetCWD configures the common working directory of helm CLI
func (h *HelmTemplate) SetCWD(dir string) {
	h.Client.SetCWD(dir)
}

// HelmBinary return the configured helm CLI
func (h *HelmTemplate) HelmBinary() string {
	return h.Client.HelmBinary()
}

// SetHelmBinary configure a new helm CLI
func (h *HelmTemplate) SetHelmBinary(binary string) {
	h.Client.SetHelmBinary(binary)
}

// Init executes the helm init command according with the given flags
func (h *HelmTemplate) Init(clientOnly bool, serviceAccount string, tillerNamespace string, upgrade bool) error {
	return h.Client.Init(true, serviceAccount, tillerNamespace, upgrade)
}

// AddRepo adds a new helm repo with the given name and URL
func (h *HelmTemplate) AddRepo(repo string, URL string) error {
	return h.Client.AddRepo(repo, URL)
}

// RemoveRepo removes the given repo from helm
func (h *HelmTemplate) RemoveRepo(repo string) error {
	return h.Client.RemoveRepo(repo)
}

// ListRepos list the installed helm repos together with their URL
func (h *HelmTemplate) ListRepos() (map[string]string, error) {
	return h.Client.ListRepos()
}

// SearchCharts searches for all the charts matching the given filter
func (h *HelmTemplate) SearchCharts(filter string) ([]ChartSummary, error) {
	return h.Client.SearchCharts(filter)
}

// IsRepoMissing checks if the repository with the given URL is missing from helm
func (h *HelmTemplate) IsRepoMissing(URL string) (bool, error) {
	return h.Client.IsRepoMissing(URL)
}

// UpdateRepo updates the helm repositories
func (h *HelmTemplate) UpdateRepo() error {
	return h.Client.UpdateRepo()
}

// RemoveRequirementsLock removes the requirements.lock file from the current working directory
func (h *HelmTemplate) RemoveRequirementsLock() error {
	return h.Client.RemoveRequirementsLock()
}

// BuildDependency builds the helm dependencies of the helm chart from the current working directory
func (h *HelmTemplate) BuildDependency() error {
	return h.Client.BuildDependency()
}

// ListCharts execute the helm list command and returns its output
func (h *HelmTemplate) ListCharts() (string, error) {
	return h.Client.ListCharts()
}

// SearchChartVersions search all version of the given chart
func (h *HelmTemplate) SearchChartVersions(chart string) ([]string, error) {
	return h.Client.SearchChartVersions(chart)
}

// FindChart find a chart in the current working directory, if no chart file is found an error is returned
func (h *HelmTemplate) FindChart() (string, error) {
	return h.Client.FindChart()
}

// Lint lints the helm chart from the current working directory and returns the warnings in the output
func (h *HelmTemplate) Lint() (string, error) {
	return h.Client.Lint()
}

// Env returns the environment variables for the helmer
func (h *HelmTemplate) Env() map[string]string {
	return h.Client.Env()
}

// PackageChart packages the chart from the current working directory
func (h *HelmTemplate) PackageChart() error {
	return h.Client.PackageChart()
}

// Version executes the helm version command and returns its output
func (h *HelmTemplate) Version(tls bool) (string, error) {
	return h.Client.VersionWithArgs(tls, "--client")
}

// Mutation API

// InstallChart installs a helm chart according with the given flags
func (h *HelmTemplate) InstallChart(chart string, releaseName string, ns string, version *string, timeout *int,
	values []string, valueFiles []string) error {

	err := h.clearOutputDir()
	if err != nil {
		return err
	}
	err = h.Client.Template(chart, releaseName, ns, h.OutputDir, false, values, valueFiles)
	if err != nil {
		return err
	}

	chartName, versionText, err := h.getChartNameAndVersion(version)
	if err != nil {
		return err
	}

	err = h.addLabelsToFiles(chartName, versionText)
	if err != nil {
		return err
	}

	log.Infof("Applying generated chart %s YAML in dir %s via kubectl\n", chart, h.OutputDir)

	args := []string{"apply", "--recursive", "-f", h.OutputDir, "--wait", "-l", LabelReleaseChartName + "=" + chartName}
	if ns != "" {
		args = append(args, "--namespace", ns)
	}

	err = h.runKubectl(args...)
	if err != nil {
		return err
	}

	wait := true
	return h.deleteOldResources(ns, chartName, versionText, wait)
}

// UpgradeChart upgrades a helm chart according with given helm flags
func (h *HelmTemplate) UpgradeChart(chart string, releaseName string, ns string, version *string, install bool,
	timeout *int, force bool, wait bool, values []string, valueFiles []string) error {

	err := h.clearOutputDir()
	if err != nil {
		return err
	}
	err = h.Client.Template(chart, releaseName, ns, h.OutputDir, false, values, valueFiles)
	if err != nil {
		return err
	}

	chartName, versionText, err := h.getChartNameAndVersion(version)
	if err != nil {
		return err
	}

	err = h.addLabelsToFiles(chartName, versionText)
	if err != nil {
		return err
	}

	log.Infof("Applying generated chart %s YAML in dir %s via kubectl\n", chart, h.OutputDir)

	args := []string{"apply", "--recursive", "-f", h.OutputDir, "-l", LabelReleaseChartName + "=" + chartName}
	if ns != "" {
		args = append(args, "--namespace", ns)
	}
	if wait {
		args = append(args, "--wait")
	}

	err = h.runKubectl(args...)
	if err != nil {
		return err
	}

	return h.deleteOldResources(ns, chartName, versionText, wait)
}

func (h *HelmTemplate) deleteOldResources(ns string, chartName string, versionText string, wait bool) error {
	args := []string{"delete", "all", "--ignore-not-found", "--namespace", ns, "-l", LabelReleaseChartName + "=" + chartName + "," + LabelReleaseChartVersion + "!=" + versionText}
	if wait {
		args = append(args, "--wait")
	}
	err := h.runKubectl(args...)
	if err != nil {
		return err
	}

	// now lets delete resource CRDs
	args = []string{"delete", "release", "--ignore-not-found", "--namespace", ns, "-l", LabelReleaseChartName + "=" + chartName + "," + LabelReleaseChartVersion + "!=" + versionText}
	if wait {
		args = append(args, "--wait")
	}
	return h.runKubectl(args...)
}

// DeleteRelease removes the given release
func (h *HelmTemplate) DeleteRelease(releaseName string, purge bool) error {
	// TODO delete all resource with the jenkins chart label
	return nil
}

// StatusRelease returns the output of the helm status command for a given release
func (h *HelmTemplate) StatusRelease(releaseName string) error {
	// TODO
	return nil
}

// StatusReleases returns the status of all installed releases
func (h *HelmTemplate) StatusReleases() (map[string]string, error) {
	statusMap := map[string]string{}
	// TODO
	return statusMap, nil
}

func (h *HelmTemplate) getOutputDir() (string, error) {
	if h.OutputDir == "" {
		d, err := ioutil.TempDir("", "helm-template-output-")
		if err != nil {
			return "", errors.Wrap(err, "Failed to create temporary directory for helm template output")
		}
		h.OutputDir = d
	}
	dir := h.OutputDir
	if dir == "" {
		return dir, fmt.Errorf("No OutputDir specifeid for HelmTemplate")
	}
	return dir, nil
}

// clearOutputDir removes all files in the helm output dir
func (h *HelmTemplate) clearOutputDir() error {
	dir, err := h.getOutputDir()
	files, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return err
	}
	for _, file := range files {
		err = os.RemoveAll(file)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *HelmTemplate) addLabelsToFiles(chart string, version string) error {
	dir, err := h.getOutputDir()
	if err != nil {
		return err
	}
	return addLabelsToChartYaml(dir, chart, version)
}

func addLabelsToChartYaml(dir string, chart string, version string) error {
	return filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		ext := filepath.Ext(path)
		if ext == ".yaml" {
			file := path
			data, err := ioutil.ReadFile(file)
			if err != nil {
				return errors.Wrapf(err, "Failed to load file %s", file)
			}
			m := yaml.MapSlice{}
			err = yaml.Unmarshal(data, &m)
			if err != nil {
				return errors.Wrapf(err, "Failed to parse YAML of file %s", file)
			}
			err = setYamlValue(&m, chart, "metadata", "labels", LabelReleaseChartName)
			if err != nil {
				return errors.Wrapf(err, "Failed to modify YAML of file %s", file)
			}
			err = setYamlValue(&m, version, "metadata", "labels", LabelReleaseChartVersion)
			if err != nil {
				return errors.Wrapf(err, "Failed to modify YAML of file %s", file)
			}

			data, err = yaml.Marshal(&m)
			if err != nil {
				return errors.Wrapf(err, "Failed to marshal YAML of file %s", file)
			}
			err = ioutil.WriteFile(file, data, util.DefaultWritePermissions)
			if err != nil {
				return errors.Wrapf(err, "Failed to write YAML file %s", file)
			}
		}
		return nil
	})
}

// setYamlValue navigates through the YAML object structure lazily creating or inserting new values
func setYamlValue(mapSlice *yaml.MapSlice, value string, keys ...string) error {
	if mapSlice == nil {
		return fmt.Errorf("No map input!")
	}
	m := mapSlice
	lastIdx := len(keys) - 1
	for idx, k := range keys {
		last := idx >= lastIdx
		found := false
		for i, mi := range *m {
			if mi.Key == k {
				found = true
				if last {
					(*m)[i].Value = value
				} else {
					value := (*m)[i].Value
					if value == nil {
						v := &yaml.MapSlice{}
						(*m)[i].Value = v
						m = v
					} else {
						v, ok := value.(yaml.MapSlice)
						if ok {
							m2 := &yaml.MapSlice{}
							*m2 = append(*m2, v...)
							(*m)[i].Value = m2
							m = m2
						} else {
							v2, ok := value.(*yaml.MapSlice)
							if ok {
								m2 := &yaml.MapSlice{}
								*m2 = append(*m2, *v2...)
								(*m)[i].Value = m2
								m = m2
							} else {
								return fmt.Errorf("Could not convert key %s value %#v to a yaml.MapSlice", k, value)
							}
						}
					}
				}
			}
		}
		if !found {
			if last {
				*m = append(*m, yaml.MapItem{
					Key:   k,
					Value: value,
				})
			} else {
				m2 := &yaml.MapSlice{}
				*m = append(*m, yaml.MapItem{
					Key:   k,
					Value: m2,
				})
				m = m2
			}
		}
	}
	return nil
}

func (h *HelmTemplate) runKubectl(args ...string) error {
	h.Runner.Name = h.Binary
	h.Runner.Dir = h.CWD
	h.Runner.Args = args
	_, err := h.Runner.RunWithoutRetry()
	return err
}

func (h *HelmTemplate) runKubectlWithOutput(args ...string) (string, error) {
	h.Runner.Dir = h.CWD
	h.Runner.Name = h.Binary
	h.Runner.Args = args
	return h.Runner.RunWithoutRetry()
}

// getChartNameAndVersion returns the chart name and version for the current chart folder
func (h *HelmTemplate) getChartNameAndVersion(version *string) (string, string, error) {
	versionText := ""
	file := filepath.Join(h.Runner.Dir, "Chart.yaml")
	exists, err := util.FileExists(file)
	if err != nil {
		return "", versionText, err
	}
	if !exists {
		return "", versionText, fmt.Errorf("No file %s found!", file)
	}
	chartName, versionText, err := LoadChartNameAndVersion(file)
	if version != nil && *version != "" {
		versionText = *version
	}
	return chartName, versionText, err
}
