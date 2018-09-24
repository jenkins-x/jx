package helm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
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

	// TODO now add labels via kustomize?

	log.Infof("Generated chart %s to dir %s\n", chart, h.OutputDir)

	args := []string{"apply", "--recursive", "-f", h.OutputDir, "--wait"}
	if ns != "" {
		args = append(args, "--namespace", ns)
	}

	err = h.runKubectl(args...)
	return err
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

	// TODO now add labels via kustomize?

	log.Infof("Generated chart %s to dir %s\n", chart, h.OutputDir)

	args := []string{"apply", "--recursive", "-f", h.OutputDir}
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

	// TODO delete old versions without the current version label

	return nil
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

// clearOutputDir removes all files in the helm output dir
func (h *HelmTemplate) clearOutputDir() error {
	if h.OutputDir == "" {
		d, err := ioutil.TempDir("", "helm-template-output-")
		if err != nil {
			return errors.Wrap(err, "Failed to create temporary directory for helm template output")
		}
		h.OutputDir = d
	}
	dir := h.OutputDir
	if dir == "" {
		return fmt.Errorf("No OutputDir specifeid for HelmTemplate")
	}
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
