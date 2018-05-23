package cmd

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
)

func (o *CommonOptions) registerLocalHelmRepo(repoName, ns string) error {
	if repoName == "" {
		repoName = kube.LocalHelmRepoName
	}
	// TODO we should use the auth package to keep a list of server login/pwds
	// TODO we have a chartmuseumAuth.yaml now but sure yet if that's the best thing to do
	username := "admin"
	password := "admin"

	// lets check if we have a local helm repository
	client, _, err := o.Factory.CreateClient()
	if err != nil {
		return err
	}
	u, err := kube.FindServiceURL(client, ns, kube.ServiceChartMuseum)
	if err != nil {
		return err
	}
	u2, err := url.Parse(u)
	if err != nil {
		return err
	}
	if u2.User == nil {
		u2.User = url.UserPassword(username, password)
	}
	helmUrl := u2.String()
	// lets check if we already have the helm repo installed or if we need to add it or remove + add it
	text, err := o.getCommandOutput("", "helm", "repo", "list")
	if err != nil {
		return err
	}
	lines := strings.Split(text, "\n")
	remove := false
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t != "" {
			fields := strings.Fields(t)
			if len(fields) > 1 {
				if fields[0] == repoName {
					if fields[1] == helmUrl {
						return nil
					} else {
						remove = true
					}
				}
			}
		}
	}
	if remove {
		err = o.runCommand("helm", "repo", "remove", repoName)
		if err != nil {
			return err
		}
	}
	return o.runCommand("helm", "repo", "add", repoName, helmUrl)
}

// addHelmRepoIfMissing adds the given helm repo if its not already added
func (o *CommonOptions) addHelmRepoIfMissing(helmUrl string, repoName string) error {
	missing, err := o.isHelmRepoMissing(helmUrl)
	if err != nil {
		return err
	}
	if missing {
		fmt.Fprintf(o.Out, "Helm repository %s (%s) not found. Adding...\n", repoName, helmUrl)
		err = o.runCommand("helm", "repo", "add", repoName, helmUrl)
		if err == nil {
			fmt.Fprintf(o.Out, "Succesfully added Helm repository %s.\n", repoName)
		}
		return err
	}
	return nil
}

// installChart installs the given chart
func (o *CommonOptions) installChart(releaseName string, chart string, version string, ns string, helmUpdate bool, setValues []string) error {
	if helmUpdate {
		fmt.Fprintf(o.Out, "Updating Helm repository...\n")
		err := o.runCommand("helm", "repo", "update")
		if err != nil {
			return err
		}
		fmt.Fprintf(o.Out, "Helm repository update done.\n")
	}
	timeout := fmt.Sprintf("--timeout=%s", defaultInstallTimeout)
	args := []string{"upgrade", "--install", timeout}
	if version != "" {
		args = append(args, "--version", version)
	}
	if ns != "" {
		kubeClient, _, err := o.KubeClient()
		if err != nil {
			return err
		}
		annotations := map[string]string{"jenkins-x.io/created-by": "Jenkins X"}
		kube.EnsureNamespaceCreated(kubeClient, ns, nil, annotations)
		args = append(args, "--namespace", ns)
	}
	for _, value := range setValues {
		args = append(args, "--set", value)
		o.Printf("Set chart value: --set %s\n", util.ColorInfo(value))
	}
	args = append(args, releaseName, chart)
	return o.runCommandVerbose("helm", args...)
}

// deleteChart deletes the given chart
func (o *CommonOptions) deleteChart(releaseName string, purge bool) error {
	args := []string{"delete"}
	if purge {
		args = append(args, "--purge")
	}
	args = append(args, releaseName)
	return o.runCommandVerbose("helm", args...)
}

func (*CommonOptions) FindHelmChart() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// lets try find the chart file
	chartFile := filepath.Join(dir, "Chart.yaml")
	exists, err := util.FileExists(chartFile)
	if err != nil {
		return "", err
	}
	if !exists {
		// lets try find all the chart files
		files, err := filepath.Glob("*/Chart.yaml")
		if err != nil {
			return "", err
		}
		if len(files) > 0 {
			chartFile = files[0]
		} else {
			files, err = filepath.Glob("*/*/Chart.yaml")
			if err != nil {
				return "", err
			}
			if len(files) > 0 {
				chartFile = files[0]
				return chartFile, nil
			}
		}
	}
	return "", nil
}
func (o *CommonOptions) isHelmRepoMissing(helmUrlString string) (bool, error) {
	// lets check if we already have the helm repo installed or if we need to add it or remove + add it
	text, err := o.getCommandOutput("", "helm", "repo", "list")
	if err != nil {
		return false, err
	}
	helmUrl, err := url.Parse(helmUrlString)
	if err != nil {
		return false, err
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t != "" {
			fields := strings.Fields(t)
			if len(fields) > 1 {
				localURL, err := url.Parse(fields[1])
				if err != nil {
					return false, err
				}
				if localURL.Host == helmUrl.Host {
					return false, nil
				}
			}
		}
	}
	return true, nil
}
