package gke

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

func GetGoogleZones() []string {
	var zones []string
	out, err := exec.Command("gcloud", "compute", "zones", "list").Output()
	if err != nil {
        fmt.Println("error occured")
        fmt.Printf("%s", err)
	}

	for _, item := range strings.Split(string(out), "\n") {
		zone := strings.Split(item, " ")[0]
		if strings.Contains(zone, "-") {
			zones = append(zones, zone)
		}
		sort.Strings(zones)
	}
	return zones
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
