package helm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// LabelReleaseName stores the chart release name
	LabelReleaseName = "jenkins.io/chart-release"

	// LabelReleaseChartVersion stores the version of a chart installation in a label
	LabelReleaseChartVersion = "jenkins.io/version"
)

// HelmTemplate implements common helm actions but purely as client side operations
// delegating a separate Helmer such as HelmCLI for the client side operations
type HelmTemplate struct {
	Client          *HelmCLI
	OutputDir       string
	CWD             string
	Binary          string
	Runner          *util.Command
	KubectlValidate bool
	KubeClient      kubernetes.Interface
}

// NewHelmTemplate creates a new HelmTemplate instance configured to the given client side Helmer
func NewHelmTemplate(client *HelmCLI, workDir string, kubeClient kubernetes.Interface) *HelmTemplate {
	cli := &HelmTemplate{
		Client:          client,
		OutputDir:       workDir,
		Runner:          client.Runner,
		Binary:          "kubectl",
		CWD:             client.CWD,
		KubectlValidate: false,
		KubeClient:      kubeClient,
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
	chartDir, err := h.chartNameToFolder(chart)
	if err != nil {
		return err
	}
	err = h.Client.Template(chartDir, releaseName, ns, h.OutputDir, false, values, valueFiles)
	if err != nil {
		return err
	}

	_, versionText, err := h.getChartNameAndVersion(chartDir, version)
	if err != nil {
		return err
	}

	err = h.addLabelsToFiles(releaseName, versionText)
	if err != nil {
		return err
	}

	wait := true

	err = h.kubectlApply(ns, chart, releaseName, false, true)
	if err != nil {
		return err
	}
	return h.deleteOldResources(ns, releaseName, versionText, wait)
}

// UpgradeChart upgrades a helm chart according with given helm flags
func (h *HelmTemplate) UpgradeChart(chart string, releaseName string, ns string, version *string, install bool,
	timeout *int, force bool, wait bool, values []string, valueFiles []string) error {

	err := h.clearOutputDir()
	if err != nil {
		return err
	}
	chartDir, err := h.chartNameToFolder(chart)
	if err != nil {
		return err
	}
	err = h.Client.Template(chartDir, releaseName, ns, h.OutputDir, false, values, valueFiles)
	if err != nil {
		return err
	}

	_, versionText, err := h.getChartNameAndVersion(chartDir, version)
	if err != nil {
		return err
	}

	err = h.addLabelsToFiles(releaseName, versionText)
	if err != nil {
		return err
	}

	err = h.kubectlApply(ns, chart, releaseName, wait, false)
	if err != nil {
		return err
	}

	return h.deleteOldResources(ns, releaseName, versionText, wait)
}

func (h *HelmTemplate) kubectlApply(ns string, chart string, releaseName string, wait bool, create bool) error {
	log.Infof("Applying generated chart %s YAML via kubectl in dir: %s\n", chart, h.OutputDir)

	command := "apply"
	if create {
		command = "create"
	}
	args := []string{command, "--recursive", "-f", h.OutputDir, "-l", LabelReleaseName + "=" + releaseName}
	if ns != "" {
		args = append(args, "--namespace", ns)
	}
	if wait && !create {
		args = append(args, "--wait")
	}
	if !h.KubectlValidate {
		args = append(args, "--validate=false")
	}
	return h.runKubectl(args...)
}

func (h *HelmTemplate) deleteOldResources(ns string, releaseName string, versionText string, wait bool) error {
	selector := LabelReleaseName + "=" + releaseName + "," + LabelReleaseChartVersion + "!=" + versionText

	log.Infof("Removing Kubernetes resources from older releases using selector: %s\n", util.ColorInfo(selector))

	return h.deleteResourcesBySelector(ns, selector, wait)
}

func (h *HelmTemplate) deleteResourcesBySelector(ns string, selector string, wait bool) error {
	args := []string{"delete", "all", "--ignore-not-found", "--namespace", ns, "-l", selector}
	if wait {
		args = append(args, "--wait")
	}
	err := h.runKubectl(args...)
	if err != nil {
		return err
	}

	// now lets delete resource CRDs
	args = []string{"delete", "release", "--ignore-not-found", "--namespace", ns, "-l", selector}
	if wait {
		args = append(args, "--wait")
	}
	// lets ignore failures - probably due to CRD not yet existing
	h.runKubectl(args...)
	return nil
}

// DeleteRelease removes the given release
func (h *HelmTemplate) DeleteRelease(ns string, releaseName string, purge bool) error {
	selector := LabelReleaseName + "=" + releaseName

	log.Infof("Removing release %s using selector: %s\n", util.ColorInfo(releaseName), util.ColorInfo(selector))

	return h.deleteResourcesBySelector(ns, selector, true)
}

// StatusRelease returns the output of the helm status command for a given release
func (h *HelmTemplate) StatusRelease(ns string, releaseName string) error {
	// TODO
	return nil
}

// StatusReleases returns the status of all installed releases
func (h *HelmTemplate) StatusReleases(ns string) (map[string]string, error) {
	statusMap := map[string]string{}
	if h.KubeClient == nil {
		return statusMap, fmt.Errorf("No KubeClient configured!")
	}
	deployList, err := h.KubeClient.AppsV1beta1().Deployments(ns).List(metav1.ListOptions{})
	if err != nil {
		return statusMap, errors.Wrapf(err, "Failed to list Deployments in namespace %s", ns)
	}
	for _, deploy := range deployList.Items {
		labels := deploy.Labels
		if labels != nil {
			release := labels[LabelReleaseName]
			if release != "" {
				statusMap[release] = "DEPLOYED"
			}
		}
	}
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

func (h *HelmTemplate) chartNameToFolder(chart string) (string, error) {
	exists, err := util.FileExists(chart)
	if err != nil {
		return "", err
	}
	if exists {
		return chart, nil
	}
	safeChartName := strings.Replace(chart, string(os.PathSeparator), "-", -1)
	dir, err := ioutil.TempDir("", safeChartName)
	if err != nil {
		return "", err
	}

	err = h.Client.runHelm("fetch", "-d", dir, "--untar", chart)
	if err != nil {
		return "", err
	}
	answer := dir
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, f := range files {
		if f.IsDir() {
			answer = filepath.Join(dir, f.Name())
			break
		}
	}
	log.Infof("Fetched chart %s to dir %s\n", chart, answer)
	return answer, nil
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
			helmHook := getYamlValueString(&m, "metadata", "annotations", "helm.sh/hook")
			if strings.HasSuffix(helmHook, "-delete") {
				// lets remove any pre/post delete hooks...
				err = os.Remove(path)
				if err != nil {
					log.Warnf("Failed to remove helm hook template %s: %s", path, err)
				} else {
					log.Infof("Ignored helm delete hook file %s\n", path)
				}
				return nil
			}

			err = setYamlValue(&m, chart, "metadata", "labels", LabelReleaseName)
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

func getYamlValueString(mapSlice *yaml.MapSlice, keys ...string) string {
	value := getYamlValue(mapSlice, keys...)
	answer, ok := value.(string)
	if ok {
		return answer
	}
	return ""
}

func getYamlValue(mapSlice *yaml.MapSlice, keys ...string) interface{} {
	if mapSlice == nil {
		return nil
	}
	if mapSlice == nil {
		return fmt.Errorf("No map input!")
	}
	m := mapSlice
	lastIdx := len(keys) - 1
	for idx, k := range keys {
		last := idx >= lastIdx
		found := false
		for _, mi := range *m {
			if mi.Key == k {
				found = true
				if last {
					return mi.Value
				} else {
					value := mi.Value
					if value == nil {
						return nil
					} else {
						v, ok := value.(yaml.MapSlice)
						if ok {
							m = &v
						} else {
							v2, ok := value.(*yaml.MapSlice)
							if ok {
								m = v2
							} else {
								return nil
							}
						}
					}
				}
			}
		}
		if !found {
			return nil
		}
	}
	return nil

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
func (h *HelmTemplate) getChartNameAndVersion(chartDir string, version *string) (string, string, error) {
	versionText := ""
	file := filepath.Join(chartDir, "Chart.yaml")
	if !filepath.IsAbs(chartDir) {
		file = filepath.Join(h.Runner.Dir, file)
	}
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
