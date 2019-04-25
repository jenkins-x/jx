package gke

import (
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

	for _, item := range strings.Split(out, "\n") {
		zone := strings.Split(item, " ")[0]
		if strings.Contains(zone, "-") {
			zones = append(zones, zone)
		}
		sort.Strings(zones)
	}
	return zones, nil
}

func GetGoogleRegions(project string) ([]string, error) {
	var regions []string
	args := []string{"compute", "regions", "list"}

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

	regions = append(regions, "none")
	for _, item := range strings.Split(out, "\n") {
		region := strings.Split(item, " ")[0]
		if strings.Contains(region, "-") {
			regions = append(regions, region)
		}
		sort.Strings(regions)
	}
	return regions, nil
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

	lines := strings.Split(out, "\n")
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

// CreateGCPServiceAccount creates a service account in GCP for a service using the account roles specified
func CreateGCPServiceAccount(kubeClient kubernetes.Interface, serviceName, namespace, clusterName, projectID string, serviceAccountRoles []string, serviceAccountSecretKey string) (string, error) {
	serviceAccountDir, err := ioutil.TempDir("", "gke")
	if err != nil {
		return "", errors.Wrap(err, "creating a temporary folder where the service account will be stored")
	}
	defer os.RemoveAll(serviceAccountDir)

	serviceAccountName := ServiceAccountName(serviceName)
	if err != nil {
		return "", err
	}
	serviceAccountPath, err := GetOrCreateServiceAccount(serviceAccountName, projectID, serviceAccountDir, serviceAccountRoles)
	if err != nil {
		return "", errors.Wrap(err, "creating the service account")
	}

	secretName, err := storeGCPServiceAccountIntoSecret(kubeClient, serviceAccountPath, serviceName, namespace, serviceAccountSecretKey)
	if err != nil {
		return "", errors.Wrap(err, "storing the service account into a secret")
	}
	return secretName, nil
}

func storeGCPServiceAccountIntoSecret(client kubernetes.Interface, serviceAccountPath, serviceName, namespace string, serviceAccountSecretKey string) (string, error) {
	serviceAccount, err := ioutil.ReadFile(serviceAccountPath)
	if err != nil {
		return "", errors.Wrapf(err, "reading the service account from file '%s'", serviceAccountPath)
	}

	secretName := GcpServiceAccountSecretName(serviceName)
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Data: map[string][]byte{
			serviceAccountSecretKey: serviceAccount,
		},
	}

	secrets := client.CoreV1().Secrets(namespace)
	_, err = secrets.Get(secretName, metav1.GetOptions{})
	if err != nil {
		_, err = secrets.Create(secret)
	} else {
		_, err = secrets.Update(secret)
	}
	return secretName, nil
}
