package gke

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"time"
)

var (
	REQUIRED_SERVICE_ACCOUNT_ROLES = []string{"roles/compute.instanceAdmin.v1",
		"roles/iam.serviceAccountActor",
		"roles/container.clusterAdmin",
		"roles/container.admin",
		"roles/container.developer",
		"roles/storage.objectAdmin",
		"roles/editor"}
)

func BucketExists(projectId string, bucketName string) (bool, error) {
	fullBucketName := fmt.Sprintf("gs://%s", bucketName)
	args := []string{"ls"}

	if projectId != "" {
		args = append(args, "-p")
		args = append(args, projectId)
	}

	cmd := util.Command{
		Name: "gsutil",
		Args: args,
	}
	output, err := cmd.RunWithoutRetry()
	if err != nil {
		log.Infof("Error checking bucket exists: %s, %s\n", output, err)
		return false, err
	}
	return strings.Contains(output, fullBucketName), nil
}

func CreateBucket(projectId string, bucketName string, location string) error {
	fullBucketName := fmt.Sprintf("gs://%s", bucketName)
	args := []string{"mb", "-l", location}

	if projectId != "" {
		args = append(args, "-p")
		args = append(args, projectId)
	}

	args = append(args, fullBucketName)

	cmd := util.Command{
		Name: "gsutil",
		Args: args,
	}
	output, err := cmd.RunWithoutRetry()
	if err != nil {
		log.Infof("Error creating bucket: %s, %s\n", output, err)
		return err
	}
	return nil
}

func GetRegionFromZone(zone string) string {
	return zone[0 : len(zone)-2]
}

func GetOrCreateServiceAccount(serviceAccount string, projectId string, clusterConfigDir string) (string, error) {
	if projectId == "" {
		return "", errors.New("cannot get/create a service account without a projectId")
	}

	args := []string{"iam",
		"service-accounts",
		"list",
		"--filter",
		serviceAccount,
		"--project",
		projectId}

	cmd := util.Command{
		Name: "gcloud",
		Args: args,
	}
	output, err := cmd.RunWithoutRetry()
	if err != nil {
		return "", err
	}

	if output == "Listed 0 items." {
		log.Infof("Unable to find service account %s, checking if we have enough permission to create\n", util.ColorInfo(serviceAccount))

		// if it doesn't check to see if we have permissions to create (assign roles) to a service account
		hasPerm, err := CheckPermission("resourcemanager.projects.setIamPolicy", projectId)
		if err != nil {
			return "", err
		}

		if !hasPerm {
			return "", errors.New("User does not have the required role 'resourcemanager.projects.setIamPolicy' to configure a service account")
		}

		// create service
		log.Infof("Creating service account %s\n", util.ColorInfo(serviceAccount))
		args = []string{"iam",
			"service-accounts",
			"create",
			serviceAccount,
			"--project",
			projectId}

		cmd := util.Command{
			Name: "gcloud",
			Args: args,
		}
		_, err = cmd.RunWithoutRetry()
		if err != nil {
			return "", err
		}

		// assign roles to service account
		for _, role := range REQUIRED_SERVICE_ACCOUNT_ROLES {
			log.Infof("Assigning role %s\n", role)
			args = []string{"projects",
				"add-iam-policy-binding",
				projectId,
				"--member",
				fmt.Sprintf("serviceAccount:%s@%s.iam.gserviceaccount.com", serviceAccount, projectId),
				"--role",
				role,
				"--project",
				projectId}

			cmd := util.Command{
				Name: "gcloud",
				Args: args,
			}
			_, err := cmd.RunWithoutRetry()
			if err != nil {
				return "", err
			}
		}

	} else {
		log.Info("Service Account exists\n")
	}

	os.MkdirAll(clusterConfigDir, os.ModePerm)
	keyPath := filepath.Join(clusterConfigDir, fmt.Sprintf("%s.key.json", serviceAccount))

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		log.Info("Downloading service account key\n")
		args = []string{"iam",
			"service-accounts",
			"keys",
			"create",
			keyPath,
			"--iam-account",
			fmt.Sprintf("%s@%s.iam.gserviceaccount.com", serviceAccount, projectId),
			"--project",
			projectId}

		cmd := util.Command{
			Name: "gcloud",
			Args: args,
		}
		_, err := cmd.RunWithoutRetry()
		if err != nil {
			return "", err
		}
	} else {
		log.Info("Key already exists")
	}

	return keyPath, nil
}

func GetEnabledApis(projectId string) ([]string,error) {
	args := []string{"services", "list", "--enabled"}

	if projectId != "" {
		args = append(args, "--project")
		args = append(args, projectId)
	}

	apis := []string{}

	cmd := util.Command{
		Name: "gcloud",
		Args: args,
	}

	out, err := cmd.RunWithoutRetry()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		if strings.Contains(l, "NAME") {
			continue
		}
		fields := strings.Fields(l)
		apis = append(apis, fields[0])
	}

	return apis, nil
}

func EnableApis(projectId string, apis ...string) error {
	enabledApis, err := GetEnabledApis(projectId)
	if err != nil {
		return err
	}

	toEnableArray := []string{}

	for _, toEnable := range apis {
		fullName := fmt.Sprintf("%s.googleapis.com", toEnable)
		if !util.Contains(enabledApis, fullName) {
			toEnableArray = append(toEnableArray, toEnable)
		}
	}

	if len(toEnableArray) == 0 {
		log.Infof("No apis to enable\n")
		return nil
	}
	args := []string{"services", "enable"}
	args = append(args, toEnableArray...)

	if projectId != "" {
		args = append(args, "--project")
		args = append(args, projectId)
	}

	log.Infof("Lets ensure we have container and compute enabled on your project via: %s\n", util.ColorInfo("gcloud "+strings.Join(args, " ")))

	cmd := util.Command{
		Name: "gcloud",
		Args: args,
	}
	_, err = cmd.RunWithoutRetry()
	if err != nil {
		return err
	}
	return nil
}

func Login(serviceAccountKeyPath string, skipLogin bool) error {
	if serviceAccountKeyPath != "" {
		log.Infof("Activating service account %s\n", util.ColorInfo(serviceAccountKeyPath))

		if _, err := os.Stat(serviceAccountKeyPath); os.IsNotExist(err) {
			return errors.New("Unable to locate service account " + serviceAccountKeyPath)
		}

		cmd := util.Command{
			Name: "gcloud",
			Args: []string{"auth", "activate-service-account", "--key-file", serviceAccountKeyPath},
		}
		_, err := cmd.RunWithoutRetry()
		if err != nil {
			return err
		}

		// GCP IAM changes can take up to 80 seconds to propagate
		retry(10, 10*time.Second, func() error {
			log.Infof("Checking for readiness...\n")

			projects, err := GetGoogleProjects()
			if err != nil {
				return err
			}

			if len(projects) == 0 {
				return errors.New("service account not ready yet")
			}

			return nil
		})

	} else if !skipLogin {
		cmd := util.Command{
			Name: "gcloud",
			Args: []string{"auth", "login", "--brief"},
		}
		_, err := cmd.RunWithoutRetry()
		if err != nil {
			return err
		}
	}
	return nil
}

func retry(attempts int, sleep time.Duration, fn func() error) error {
	if err := fn(); err != nil {
		if s, ok := err.(stop); ok {
			// Return the original error for later checking
			return s.error
		}

		if attempts--; attempts > 0 {
			time.Sleep(sleep)
			return retry(attempts, 2*sleep, fn)
		}
		return err
	}
	return nil
}

type stop struct {
	error
}

func CheckPermission(perm string, projectId string) (bool, error) {
	if projectId == "" {
		return false, errors.New("cannot check permission without a projectId")
	}
	// if it doesn't check to see if we have permissions to create (assign roles) to a service account
	args := []string{"iam",
		"list-testable-permissions",
		fmt.Sprintf("//cloudresourcemanager.googleapis.com/projects/%s", projectId),
		"--filter",
		perm}

	cmd := util.Command{
		Name: "gcloud",
		Args: args,
	}
	output, err := cmd.RunWithoutRetry()
	if err != nil {
		return false, err
	}

	return strings.Contains(output, perm), nil
}
