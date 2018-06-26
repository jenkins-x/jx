package gke

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

var (
	REQUIRED_SERVICE_ACCOUNT_ROLES = []string{"roles/compute.instanceAdmin.v1",
		"roles/iam.serviceAccountActor",
		"roles/container.clusterAdmin",
		"roles/container.admin",
		"roles/container.developer",
		"roles/storage.objectAdmin"}
)

func GetOrCreateServiceAccount(serviceAccount string, projectId string, clusterConfigDir string) (string, error) {
	args := []string{"iam",
		"service-accounts",
		"list",
		"--filter",
		serviceAccount}

	output, err := util.RunCommandWithOutput("", "gcloud", args...)
	if err != nil {
		return "", err
	}

	if output == "Listed 0 items." {
		log.Infof("Unable to find service account %s, checking if we have enough permission to create\n", serviceAccount)

		// if it doesn't check to see if we have permissions to create (assign roles) to a service account
		hasPerm, err := CheckPermission("resourcemanager.projects.setIamPolicy", projectId)
		if err != nil {
			return "", err
		}

		if !hasPerm {
			return "", errors.New("User does not have the required role 'resourcemanager.projects.setIamPolicy' to configure a service account")
		}

		// create service
		log.Infof("Creating service account %s\n", serviceAccount)
		args = []string{"iam",
			"service-accounts",
			"create",
			serviceAccount}

		err = util.RunCommand("", "gcloud", args...)
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
				role}

			err = util.RunCommand("", "gcloud", args...)
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
			fmt.Sprintf("%s@%s.iam.gserviceaccount.com", serviceAccount, projectId)}

		err = util.RunCommand("", "gcloud", args...)
		if err != nil {
			return "", err
		}
	}

	return keyPath, nil
}

func EnableApis(apis ...string) error {
	args := []string{"services", "enable"}
	args = append(args, apis...)

	log.Infof("Lets ensure we have container and compute enabled on your project via: %s\n", util.ColorInfo("gcloud "+strings.Join(args, " ")))

	err := util.RunCommand("", "gcloud", args...)
	if err != nil {
		return err
	}
	return nil
}

func Login(serviceAccountKeyPath string, skipLogin bool) error {
	if serviceAccountKeyPath != "" {
		if _, err := os.Stat(serviceAccountKeyPath); os.IsNotExist(err) {
			return errors.New("Unable to locate service account " + serviceAccountKeyPath)
		}

		err := util.RunCommand("", "gcloud", "auth", "activate-service-account", "--key-file", serviceAccountKeyPath)
		if err != nil {
			return err
		}
	} else if !skipLogin {
		err := util.RunCommand("", "gcloud", "auth", "login", "--brief")
		if err != nil {
			return err
		}
	}
	return nil
}

func CheckPermission(perm string, projectId string) (bool, error) {
	// if it doesn't check to see if we have permissions to create (assign roles) to a service account
	args := []string{"iam",
		"list-testable-permissions",
		fmt.Sprintf("//cloudresourcemanager.googleapis.com/projects/%s", projectId),
		"--filter",
		perm}

	output, err := util.RunCommandWithOutput("", "gcloud", args...)
	if err != nil {
		return false, err
	}

	return strings.Contains(output, perm), nil
}
