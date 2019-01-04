package gke

import (
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
)

var PROJECT_LIST_HEADER = "PROJECT_ID"

func GetGoogleZones(project string) ([]string, error) {
	var zones []string
	args := []string{"compute", "zones", "list"}

	if "" != project {
		args = append(args, "--project")
		args = append(args, project)
	}

	cmd := util.Command{
		Name: "gcloud",
		Args: args,
	}

	out, err := cmd.RunWithoutRetry()
	if err != nil {
		return nil, err
	}

	for _, item := range strings.Split(string(out), "\n") {
		zone := strings.Split(item, " ")[0]
		if strings.Contains(zone, "-") {
			zones = append(zones, zone)
		}
		sort.Strings(zones)
	}
	return zones, nil
}

func GetGoogleProjects() ([]string, error) {
	cmd := util.Command{
		Name: "gcloud",
		Args: []string{"projects", "list"},
	}
	out, err := cmd.RunWithoutRetry()
	if err != nil {
		return nil, err
	}

	if out == "Listed 0 items." {
		return []string{}, nil
	}

	lines := strings.Split(string(out), "\n")
	var existingProjects []string
	for _, l := range lines {
		if strings.Contains(l, PROJECT_LIST_HEADER) {
			continue
		}
		fields := strings.Fields(l)
		existingProjects = append(existingProjects, fields[0])
	}
	return existingProjects, nil
}

func GetCurrentProject() (string, error) {
	cmd := util.Command{
		Name: "gcloud",
		Args: []string{"config", "get-value", "project"},
	}
	out, err := cmd.RunWithoutRetry()
	if err != nil {
		return "", err
	}
	return out, nil
}

func GetGoogleMachineTypes() []string {

	return []string{
		"g1-small",
		"n1-standard-1",
		"n1-standard-2",
		"n1-standard-4",
		"n1-standard-8",
		"n1-standard-16",
		"n1-standard-32",
		"n1-standard-64",
		"n1-standard-96",
		"n1-highmem-2",
		"n1-highmem-4",
		"n1-highmem-8",
		"n1-highmem-16",
		"n1-highmem-32",
		"n1-highmem-64",
		"n1-highmem-96",
		"n1-highcpu-2",
		"n1-highcpu-4",
		"n1-highcpu-8",
		"n1-highcpu-16",
		"n1-highcpu-32",
		"n1-highcpu-64",
		"n1-highcpu-96",
	}
}
