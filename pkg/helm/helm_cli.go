package helm

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

type HelmCLI struct {
	Binary     string
	BinVersion Version
	CWD        string
}

func NewHelmCLI(binary string, version Version, cwd string) *HelmCLI {
	return &HelmCLI{
		Binary:     binary,
		BinVersion: version,
		CWD:        cwd,
	}
}

func (h *HelmCLI) SetCWD(dir string) {
	h.CWD = dir
}

func (h *HelmCLI) runHelm(args ...string) error {
	return util.RunCommand(h.CWD, h.Binary, args...)
}

func (h *HelmCLI) runHelmWithOutput(args ...string) (string, error) {
	return util.RunCommandWithOutput(h.CWD, h.Binary, args...)
}

func (h *HelmCLI) Init(clientOnly bool, serviceAccount string, tillerNamespace string, upgrade bool) error {
	args := []string{}
	args = append(args, "init")
	if clientOnly {
		args = append(args, "--client-only")
	}
	if serviceAccount != "" {
		args = append(args, "--service-account", serviceAccount)
	}
	if tillerNamespace != "" {
		args = append(args, "--tiller-namespace", tillerNamespace)
	}
	if upgrade {
		args = append(args, "--upgrade")
	}
	return h.runHelm(args...)
}

func (h *HelmCLI) AddRepo(repo string, URL string) error {
	return h.runHelm("repo", "add", repo, URL)
}

func (h *HelmCLI) RemoveRepo(repo string) error {
	return h.runHelm("repo", "remove", repo)
}

func (h *HelmCLI) ListRepos() (map[string]string, error) {
	output, err := h.runHelmWithOutput("repo", "list")
	if err != nil {
		return nil, errors.Wrap(err, "failed to list repositories")
	}
	repos := map[string]string{}
	lines := strings.Split(output, "\n")
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)
		if len(fields) > 1 {
			repos[fields[0]] = fields[1]
		} else {
			repos[fields[0]] = ""
		}
	}
	return repos, nil
}

func (h *HelmCLI) IsRepoMissing(URL string) (bool, error) {
	repos, err := h.ListRepos()
	if err != nil {
		return false, errors.Wrap(err, "failed to list the repositories")
	}
	searchedURL, err := url.Parse(URL)
	if err != nil {
		return false, errors.Wrap(err, "provided repo URL is invalid")
	}
	for _, repoURL := range repos {
		if len(repoURL) > 0 {
			url, err := url.Parse(repoURL)
			if err != nil {
				return false, errors.Wrap(err, "failed to parse the repo URL")
			}
			if url.Host == searchedURL.Host {
				return true, nil
			}
		}
	}
	return false, fmt.Errorf("no repository with URL '%s' found", URL)
}

func (h *HelmCLI) UpdateRepo() error {
	return h.runHelm("repo", "update")
}

func (h *HelmCLI) RemoveRequirementsLock() error {
	dir := h.CWD
	path := filepath.Join(dir, "requirements.lock")
	exists, err := util.FileExists(path)
	if err != nil {
		return errors.Wrapf(err, "no requirements.lock file found in directory '%s'", dir)
	}
	if exists {
		err = os.Remove(path)
		if err != nil {
			return errors.Wrap(err, "failed to remove the requirements.lock file")
		}
	}
	return nil
}

func (h *HelmCLI) BuildDependency() error {
	return h.runHelm("dependency", "build")
}

func (h *HelmCLI) InstallChart(chart string, releaseName string, ns string, values []string) error {
	args := []string{}
	args = append(args, "install", "--name", releaseName, "--namespace", ns, chart)
	for _, value := range values {
		args = append(args, "--set", value)
	}
	return h.runHelm(args...)
}

func (h *HelmCLI) UpgradeChart(chart string, releaseName string, ns string, version *string, install bool,
	timeout *int, force bool, wait bool, values []string) error {
	args := []string{}
	args = append(args, "upgrade")
	args = append(args, "--namespace", ns)
	if install {
		args = append(args, "--install")
	}
	if wait {
		args = append(args, "--wait")
	}
	if force {
		args = append(args, "--force")
	}
	if timeout != nil {
		args = append(args, "--timeout", strconv.Itoa(*timeout))
	}
	if version != nil {
		args = append(args, "--version", *version)
	}
	for _, value := range values {
		args = append(args, "--set", value)
	}
	args = append(args, releaseName, chart)
	return h.runHelm(args...)
}

func (h *HelmCLI) DeleteRelease(releaseName string, purge bool) error {
	args := []string{}
	args = append(args, "delete")
	if purge {
		args = append(args, "--purge")
	}
	args = append(args, releaseName)
	return h.runHelm(args...)
}

func (h *HelmCLI) ListCharts() (string, error) {
	return h.runHelmWithOutput("list")
}

func (h *HelmCLI) SearchChartVersions(chart string) ([]string, error) {
	output, err := h.runHelmWithOutput("search", chart, "--versions")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to search chart '%s'", chart)
	}
	versions := []string{}
	for i, line := range strings.Split(output, "\n") {
		if i == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 1 {
			v := fields[1]
			if v != "" {
				versions = append(versions, v)
			}
		}
	}
	return versions, nil
}

func (h *HelmCLI) FindChart() (string, error) {
	dir := h.CWD
	chartFile := filepath.Join(dir, "Chart.yaml")
	exists, err := util.FileExists(chartFile)
	if err != nil {
		return "", errors.Wrapf(err, "no Chart.yaml file found in directory '%s'", dir)
	}
	if !exists {
		files, err := filepath.Glob("*/Chart.yaml")
		if err != nil {
			return "", errors.Wrap(err, "no Chart.yaml file found")
		}
		if len(files) > 0 {
			chartFile = files[0]
		} else {
			files, err = filepath.Glob("*/*/Chart.yaml")
			if err != nil {
				return "", errors.Wrap(err, "no Chart.yaml file found")
			}
			if len(files) > 0 {
				for _, file := range files {
					if !strings.HasSuffix(file, "/preview/Chart.yaml") {
						return file, nil
					}
				}
			}
		}
	}
	return chartFile, nil
}

func (h *HelmCLI) StatusRelease(releaseName string) error {
	return h.runHelm("status", releaseName)
}

func (h *HelmCLI) Lint() (string, error) {
	return h.runHelmWithOutput("lint")
}

func (h *HelmCLI) Version() (string, error) {
	return h.runHelmWithOutput("version")
}
