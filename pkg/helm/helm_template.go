package helm

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/table"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// AnnotationChartName stores the chart name
	AnnotationChartName = "jenkins.io/chart"
	// AnnotationAppVersion stores the chart's app version
	AnnotationAppVersion = "jenkins.io/chart-app-version"

	// LabelReleaseName stores the chart release name
	LabelReleaseName = "jenkins.io/chart-release"

	// LabelReleaseChartVersion stores the version of a chart installation in a label
	LabelReleaseChartVersion = "jenkins.io/version"

	hookFailed    = "hook-failed"
	hookSucceeded = "hook-succeeded"

	// resourcesSeparator is used to separate multiple objects stored in the same YAML file
	resourcesSeparator = "---"
)

// HelmTemplate implements common helm actions but purely as client side operations
// delegating a separate Helmer such as HelmCLI for the client side operations
type HelmTemplate struct {
	Client          *HelmCLI
	WorkDir         string
	CWD             string
	Binary          string
	Runner          util.Commander
	KubectlValidate bool
	KubeClient      kubernetes.Interface
	Namespace       string
}

type ChartListing struct {
	Chart         string
	Revision      string
	Updated       string
	Status        string
	ChartFullName string
	ChartVersion  string
	ReleaseName   string
	AppVersion    string
	Namespace     string
}

// NewHelmTemplate creates a new HelmTemplate instance configured to the given client side Helmer
func NewHelmTemplate(client *HelmCLI, workDir string, kubeClient kubernetes.Interface, ns string) *HelmTemplate {
	cli := &HelmTemplate{
		Client:          client,
		WorkDir:         workDir,
		Runner:          client.Runner,
		Binary:          "kubectl",
		CWD:             client.CWD,
		KubectlValidate: false,
		KubeClient:      kubeClient,
		Namespace:       ns,
	}
	return cli
}

type HelmHook struct {
	Kind               string
	Name               string
	File               string
	Hooks              []string
	HookDeletePolicies []string
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
func (h *HelmTemplate) AddRepo(repo, URL, username, password string) error {
	return h.Client.AddRepo(repo, URL, username, password)
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
	ns := h.Namespace
	list, _ := h.KubeClient.AppsV1beta1().Deployments(ns).List(metav1.ListOptions{})
	var buffer bytes.Buffer
	writer := bufio.NewWriter(&buffer)
	t := table.CreateTable(writer)
	t.Separator = "\t"
	t.AddRow("NAME", "REVISION", "UPDATED", "STATUS", "CHART", "APP VERSION", "NAMESPACE")
	if list != nil {
		keys := []string{}
		charts := map[string]*ChartListing{}
		for _, deploy := range list.Items {
			labels := deploy.Labels
			ann := deploy.Annotations
			if labels != nil && ann != nil {
				status := "ERROR"
				if deploy.Status.Replicas > 0 {
					if deploy.Status.UnavailableReplicas > 0 {
						status = "PENDING"
					} else {
						status = "DEPLOYED"
					}
				}
				updated := deploy.CreationTimestamp.Format("Mon Jan 2 15:04:05 2006")
				chartName := ann[AnnotationChartName]
				chartVersion := labels[LabelReleaseChartVersion]
				info := &ChartListing{
					Chart:         chartName,
					ChartFullName: chartName + "-" + chartVersion,
					Revision:      strconv.FormatInt(deploy.Generation, 10),
					Updated:       updated,
					Status:        status,
					ChartVersion:  chartVersion,
					ReleaseName:   labels[LabelReleaseName],
					AppVersion:    ann[AnnotationAppVersion],
					Namespace:     ns,
				}
				key := info.ReleaseName
				if charts[key] == nil {
					charts[key] = info
					keys = append(keys, key)
				}
			}
		}
		sort.Strings(keys)
		for _, key := range keys {
			info := charts[key]
			if info != nil {
				t.AddRow(key, info.Revision, info.Updated, info.Status, info.ChartFullName, info.AppVersion, info.Namespace)
			}
		}
	}
	t.Render()
	writer.Flush()
	return buffer.String(), nil
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
func (h *HelmTemplate) InstallChart(chart string, releaseName string, ns string, version string, timeout int,
	values []string, valueFiles []string, repo string, username string, password string) error {

	err := h.clearOutputDir(releaseName)
	if err != nil {
		return err
	}
	outputDir, _, chartsDir, err := h.getDirectories(releaseName)

	chartDir, err := h.fetchChart(chart, version, chartsDir, repo, username, password)
	if err != nil {
		return err
	}
	err = h.Client.Template(chartDir, releaseName, ns, outputDir, false, values, valueFiles)
	if err != nil {
		return err
	}

	metadata, versionText, err := h.getChart(chartDir, version)
	if err != nil {
		return err
	}

	helmHooks, err := h.addLabelsToFiles(chart, releaseName, versionText, metadata)
	if err != nil {
		return err
	}
	helmCrdPhase := "crd-install"
	helmPrePhase := "pre-install"
	helmPostPhase := "post-install"
	wait := true
	create := true

	err = h.runHooks(helmHooks, helmCrdPhase, ns, chart, releaseName, wait, create)
	if err != nil {
		return err
	}

	err = h.runHooks(helmHooks, helmPrePhase, ns, chart, releaseName, wait, create)
	if err != nil {
		return err
	}
	err = h.kubectlApply(ns, chart, releaseName, wait, create, outputDir)
	if err != nil {
		h.deleteHooks(helmHooks, helmPrePhase, hookFailed, ns)
		return err
	}
	log.Info("\n")
	h.deleteHooks(helmHooks, helmPrePhase, hookSucceeded, ns)

	err = h.runHooks(helmHooks, helmPostPhase, ns, chart, releaseName, wait, create)
	if err != nil {
		h.deleteHooks(helmHooks, helmPostPhase, hookFailed, ns)
		return err
	}

	err = h.deleteHooks(helmHooks, helmPostPhase, hookSucceeded, ns)
	err2 := h.deleteOldResources(ns, releaseName, versionText, wait)
	log.Info("\n")

	return util.CombineErrors(err, err2)
}

// FetchChart fetches a Helm Chart
func (h *HelmTemplate) FetchChart(chart string, version string, untar bool, untardir string, repo string,
	username string, password string) error {
	_, err := h.fetchChart(chart, version, untardir, repo, username, password)
	return err
}

// UpgradeChart upgrades a helm chart according with given helm flags
func (h *HelmTemplate) UpgradeChart(chart string, releaseName string, ns string, version string, install bool,
	timeout int, force bool, wait bool, values []string, valueFiles []string, repo string, username string,
	password string) error {

	err := h.clearOutputDir(releaseName)
	if err != nil {
		return err
	}
	outputDir, _, chartsDir, err := h.getDirectories(releaseName)

	chartDir, err := h.fetchChart(chart, version, chartsDir, repo, username, password)
	if err != nil {
		return err
	}
	err = h.Client.Template(chartDir, releaseName, ns, outputDir, false, values, valueFiles)
	if err != nil {
		return err
	}

	metadata, versionText, err := h.getChart(chartDir, version)
	if err != nil {
		return err
	}

	helmHooks, err := h.addLabelsToFiles(chart, releaseName, versionText, metadata)
	if err != nil {
		return err
	}

	helmCrdPhase := "crd-install"
	helmPrePhase := "pre-upgrade"
	helmPostPhase := "post-upgrade"
	create := false

	err = h.runHooks(helmHooks, helmCrdPhase, ns, chart, releaseName, wait, create)
	if err != nil {
		return err
	}

	err = h.runHooks(helmHooks, helmPrePhase, ns, chart, releaseName, wait, create)
	if err != nil {
		return err
	}

	err = h.kubectlApply(ns, chart, releaseName, wait, create, outputDir)
	if err != nil {
		h.deleteHooks(helmHooks, helmPrePhase, hookFailed, ns)
		return err
	}
	h.deleteHooks(helmHooks, helmPrePhase, hookSucceeded, ns)

	err = h.runHooks(helmHooks, helmPostPhase, ns, chart, releaseName, wait, create)
	if err != nil {
		h.deleteHooks(helmHooks, helmPostPhase, hookFailed, ns)
		return err
	}

	err = h.deleteHooks(helmHooks, helmPostPhase, hookSucceeded, ns)
	err2 := h.deleteOldResources(ns, releaseName, versionText, wait)

	return util.CombineErrors(err, err2)
}

func (h *HelmTemplate) DecryptSecrets(location string) error {
	return h.Client.DecryptSecrets(location)
}

func (h *HelmTemplate) kubectlApply(ns string, chart string, releaseName string, wait bool, create bool, dir string) error {
	log.Infof("Applying generated chart %s YAML via kubectl in dir: %s\n", chart, dir)

	command := "apply"
	if create {
		command = "create"
	}
	args := []string{command, "--recursive", "-f", dir, "-l", LabelReleaseName + "=" + releaseName}
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

func (h *HelmTemplate) kubectlApplyFile(ns string, helmHook string, wait bool, create bool, file string) error {
	log.Infof("Applying Helm hook %s YAML via kubectl in file: %s\n", helmHook, file)

	command := "apply"
	if create {
		command = "create"
	}
	args := []string{command, "-f", file}
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

func (h *HelmTemplate) kubectlDeleteFile(ns string, file string) error {
	log.Infof("Deleting helm hook sources from file: %s\n", file)
	return h.runKubectl("delete", "-f", file, "--namespace", ns, "--wait")
}

func (h *HelmTemplate) deleteOldResources(ns string, releaseName string, versionText string, wait bool) error {
	selector := LabelReleaseName + "=" + releaseName + "," + LabelReleaseChartVersion + "!=" + versionText

	log.Infof("Removing Kubernetes resources from older releases using selector: %s\n", util.ColorInfo(selector))

	return h.deleteResourcesBySelector(ns, selector, wait)
}

func (h *HelmTemplate) deleteResourcesBySelector(ns string, selector string, wait bool) error {
	kinds := []string{"all", "pvc", "configmap", "release"}
	for _, kind := range kinds {
		args := []string{"delete", kind, "--ignore-not-found", "--namespace", ns, "-l", selector}
		if wait {
			args = append(args, "--wait")
		}
		output, err := h.runKubectlWithOutput(args...)
		if err != nil {
			return err
		}
		output = strings.TrimSpace(output)
		if output != "No resources found" {
			log.Info(output + "\n")
		}
	}
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
func (h *HelmTemplate) StatusReleases(ns string) (map[string]Release, error) {
	statusMap := map[string]Release{}
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
			releaseName := labels[LabelReleaseName]
			release := Release{
				Release: releaseName,
				Status:  "DEPLOYED",
				Version: "",
			}

			if releaseName != "" {
				statusMap[releaseName] = release
			}
		}
	}
	return statusMap, nil
}

func (h *HelmTemplate) getDirectories(releaseName string) (string, string, string, error) {
	if releaseName == "" {
		return "", "", "", fmt.Errorf("No release name specified!")
	}
	if h.WorkDir == "" {
		var err error
		h.WorkDir, err = ioutil.TempDir("", "helm-template-workdir-")
		if err != nil {
			return "", "", "", errors.Wrap(err, "Failed to create temporary directory for helm template workdir")
		}
	}
	workDir := h.WorkDir
	outDir := filepath.Join(workDir, releaseName, "output")
	helmHookDir := filepath.Join(workDir, releaseName, "helmHooks")
	chartsDir := filepath.Join(workDir, releaseName, "chartFiles")

	dirs := []string{outDir, helmHookDir, chartsDir}
	for _, d := range dirs {
		err := os.MkdirAll(d, util.DefaultWritePermissions)
		if err != nil {
			return "", "", "", err
		}
	}
	return outDir, helmHookDir, chartsDir, nil
}

// clearOutputDir removes all files in the helm output dir
func (h *HelmTemplate) clearOutputDir(releaseName string) error {
	dir, helmDir, chartsDir, err := h.getDirectories(releaseName)
	if err != nil {
		return err
	}
	return util.RecreateDirs(dir, helmDir, chartsDir)
}

func (h *HelmTemplate) fetchChart(chart string, version string, dir string, repo string, username string,
	password string) (string, error) {
	exists, err := util.FileExists(chart)
	if err != nil {
		return "", err
	}
	if exists {
		return chart, nil
	}
	if dir == "" {
		return "", fmt.Errorf("must specify dir for chart %s", chart)
	}
	args := []string{
		"fetch", "-d", dir, "--untar", chart,
	}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	if version != "" {
		args = append(args, "--version", version)
	}
	if username != "" {
		args = append(args, "--username", username)
	}
	if password != "" {
		args = append(args, "--password", password)
	}
	err = h.Client.runHelm(args...)
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

func (h *HelmTemplate) addLabelsToFiles(chart string, releaseName string, version string, metadata *chart.Metadata) ([]*HelmHook, error) {
	dir, helmHookDir, _, err := h.getDirectories(releaseName)
	if err != nil {
		return nil, err
	}
	return addLabelsToChartYaml(dir, helmHookDir, chart, releaseName, version, metadata)
}

func splitObjectsInFiles(file string) ([]string, error) {
	result := make([]string, 0)
	f, err := os.Open(file)
	if err != nil {
		return result, errors.Wrapf(err, "opening file %q", file)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var buf bytes.Buffer
	dir := filepath.Dir(file)
	fileName := filepath.Base(file)
	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == resourcesSeparator && buf.Len() > 0 {
			objFile, err := writeObjectInFile(&buf, dir, fileName, count)
			if err != nil {
				return result, errors.Wrapf(err, "saving object")
			}
			result = append(result, objFile)
			buf.Reset()
			count += count + 1
		} else {
			_, err := buf.WriteString(line)
			if err != nil {
				return result, errors.Wrapf(err, "writing line from file %q into a buffer", file)
			}
			_, err = buf.WriteString("\n")
			if err != nil {
				return result, errors.Wrapf(err, "writing a new line in the buffer")
			}
		}
	}
	if buf.Len() > 0 {
		if count > 0 {
			objFile, err := writeObjectInFile(&buf, dir, fileName, count)
			if err != nil {
				return result, errors.Wrapf(err, "saving object")
			}
			result = append(result, objFile)
		} else {
			result = append(result, file)
		}
	}

	return result, nil
}

func writeObjectInFile(buf *bytes.Buffer, dir string, fileName string, count int) (string, error) {
	const filePrefix = "part"
	partFile := fmt.Sprintf("%s%d-%s", filePrefix, count, fileName)
	absFile := filepath.Join(dir, partFile)
	file, err := os.Create(absFile)
	if err != nil {
		return "", errors.Wrapf(err, "creating file %q", absFile)
	}
	defer file.Close()
	_, err = buf.WriteTo(file)
	if err != nil {
		return "", errors.Wrapf(err, "writing object to file %q", absFile)
	}
	return absFile, nil
}

func addLabelsToChartYaml(dir string, hooksDir string, chart string, releaseName string, version string, metadata *chart.Metadata) ([]*HelmHook, error) {
	helmHooks := []*HelmHook{}

	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		ext := filepath.Ext(path)
		if ext == ".yaml" {
			file := path
			objFiles, err := splitObjectsInFiles(file)
			if err != nil {
				return errors.Wrapf(err, "spliting objects from file %q", file)
			}
			for _, file := range objFiles {
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
				if helmHook != "" {
					// lets move any helm hooks to the new path
					relPath, err := filepath.Rel(dir, path)
					if err != nil {
						return err
					}
					if relPath == "" {
						return fmt.Errorf("Failed to find relative path of dir %s and path %s", dir, path)
					}
					newPath := filepath.Join(hooksDir, relPath)
					newDir, _ := filepath.Split(newPath)
					err = os.MkdirAll(newDir, util.DefaultWritePermissions)
					if err != nil {
						return err
					}
					err = os.Rename(path, newPath)
					if err != nil {
						log.Warnf("Failed to move helm hook template %s to %s: %s", path, newPath, err)
						return err
					}
					name := getYamlValueString(&m, "metadata", "name")
					kind := getYamlValueString(&m, "kind")
					helmDeletePolicy := getYamlValueString(&m, "metadata", "annotations", "helm.sh/hook-delete-policy")
					helmHooks = append(helmHooks, NewHelmHook(kind, name, newPath, helmHook, helmDeletePolicy))
					return nil
				}
				err = setYamlValue(&m, releaseName, "metadata", "labels", LabelReleaseName)
				if err != nil {
					return errors.Wrapf(err, "Failed to modify YAML of file %s", file)
				}
				err = setYamlValue(&m, version, "metadata", "labels", LabelReleaseChartVersion)
				if err != nil {
					return errors.Wrapf(err, "Failed to modify YAML of file %s", file)
				}
				chartName := ""

				if metadata != nil {
					chartName = metadata.GetName()
					appVersion := metadata.GetAppVersion()
					if appVersion != "" {
						err = setYamlValue(&m, appVersion, "metadata", "annotations", AnnotationAppVersion)
						if err != nil {
							return errors.Wrapf(err, "Failed to modify YAML of file %s", file)
						}
					}
				}
				if chartName == "" {
					chartName = chart
				}
				err = setYamlValue(&m, chartName, "metadata", "annotations", AnnotationChartName)
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
		}
		return nil
	})
	return helmHooks, err
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
				} else if i < len(*m) {
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
	h.Runner.SetDir(h.CWD)
	h.Runner.SetName(h.Binary)
	h.Runner.SetArgs(args)
	output, err := h.Runner.RunWithoutRetry()
	log.Info(output + "\n")
	return err
}

func (h *HelmTemplate) runKubectlWithOutput(args ...string) (string, error) {
	h.Runner.SetDir(h.CWD)
	h.Runner.SetName(h.Binary)
	h.Runner.SetArgs(args)
	return h.Runner.RunWithoutRetry()
}

// getChartNameAndVersion returns the chart name and version for the current chart folder
func (h *HelmTemplate) getChartNameAndVersion(chartDir string, version *string) (string, string, error) {
	versionText := ""
	if version != nil && *version != "" {
		versionText = *version
	}
	file := filepath.Join(chartDir, ChartFileName)
	if !filepath.IsAbs(chartDir) {
		file = filepath.Join(h.Runner.CurrentDir(), file)
	}
	exists, err := util.FileExists(file)
	if err != nil {
		return "", versionText, err
	}
	if !exists {
		return "", versionText, fmt.Errorf("No file %s found!", file)
	}
	chartName, versionText, err := LoadChartNameAndVersion(file)
	return chartName, versionText, err
}

// getChart returns the chart metadata for the given dir
func (h *HelmTemplate) getChart(chartDir string, version string) (*chart.Metadata, string, error) {
	file := filepath.Join(chartDir, ChartFileName)
	if !filepath.IsAbs(chartDir) {
		file = filepath.Join(h.Runner.CurrentDir(), file)
	}
	exists, err := util.FileExists(file)
	if err != nil {
		return nil, version, err
	}
	if !exists {
		return nil, version, fmt.Errorf("no file %s found!", file)
	}
	metadata, err := chartutil.LoadChartfile(file)
	if version == "" && metadata != nil {
		version = metadata.GetVersion()
	}
	return metadata, version, err
}

func (h *HelmTemplate) runHooks(hooks []*HelmHook, hookPhase string, ns string, chart string, releaseName string, wait bool, create bool) error {
	matchingHooks := MatchingHooks(hooks, hookPhase, "")
	for _, hook := range matchingHooks {
		err := h.kubectlApplyFile(ns, hookPhase, wait, create, hook.File)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *HelmTemplate) deleteHooks(hooks []*HelmHook, hookPhase string, hookDeletePolicy string, ns string) error {
	matchingHooks := MatchingHooks(hooks, hookPhase, hookDeletePolicy)
	for _, hook := range matchingHooks {
		kind := hook.Kind
		name := hook.Name
		if kind == "Job" && name != "" {
			log.Infof("Waiting for helm %s hook Job %s to complete before removing it\n", hookPhase, name)
			err := kube.WaitForJobToTerminate(h.KubeClient, ns, name, time.Minute*10)
			if err != nil {
				log.Warnf("Job %s has not yet terminated for helm hook phase %s due to: %s so removing it anyway\n", name, hookPhase, err)
			}
		} else {
			log.Warnf("Could not wait for hook resource to complete as it is kind %s and name %s for phase %s\n", kind, name, hookPhase)
		}
		// TODO wait for job to be complete
		err := h.kubectlDeleteFile(ns, hook.File)
		if err != nil {
			return err
		}
	}
	return nil
}

// NewHelmHook returns a newly created HelmHook
func NewHelmHook(kind string, name string, file string, hook string, hookDeletePolicy string) *HelmHook {
	return &HelmHook{
		Kind:               kind,
		Name:               name,
		File:               file,
		Hooks:              strings.Split(hook, ","),
		HookDeletePolicies: strings.Split(hookDeletePolicy, ","),
	}
}

// MatchingHooks returns the matching files which have the given hook name and if hookPolicy is not blank the hook policy too
func MatchingHooks(hooks []*HelmHook, hook string, hookDeletePolicy string) []*HelmHook {
	answer := []*HelmHook{}
	for _, h := range hooks {
		if util.StringArrayIndex(h.Hooks, hook) >= 0 &&
			(hookDeletePolicy == "" || util.StringArrayIndex(h.HookDeletePolicies, hookDeletePolicy) >= 0) {
			answer = append(answer, h)
		}
	}
	return answer
}

func asText(text *string) string {
	if text != nil {
		return *text
	}
	return ""
}
