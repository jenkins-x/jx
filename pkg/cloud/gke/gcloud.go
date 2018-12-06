package gke

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/kube"

	"time"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

const KmsLocation = "global"

var (
	REQUIRED_SERVICE_ACCOUNT_ROLES = []string{"roles/compute.instanceAdmin.v1",
		"roles/iam.serviceAccountActor",
		"roles/container.clusterAdmin",
		"roles/container.admin",
		"roles/container.developer",
		"roles/storage.objectAdmin",
		"roles/editor"}
)

// ClusterName gets the cluster name from the current context
// Note that this just reads the ClusterName from the local kube config, which can be renamed (but is unlikely to happen)
func ClusterName(kuber kube.Kuber) (string, error) {
	config, _, err := kuber.LoadConfig()
	if err != nil {
		return "", err
	}

	context := kube.CurrentContext(config)
	if context == nil {
		return "", errors.New("kube context was nil")
	}
	// context.Cluster will likely be in the form gke_<accountName>_<region>_<clustername>
	// Trim off the crud from the beginning context.Cluster
	return GetSimplifiedClusterName(context.Cluster), nil
}

// ShortClusterName returns a short clusters name. Eg, if ClusterName would return tweetypie-jenkinsx-dev, ShortClusterName
// would return tweetypie. This is needed because GCP has character limits on things like service accounts (6-30 chars)
// and combining a long cluster name and a long vault name exceeds this limit
func ShortClusterName(kuber kube.Kuber) (string, error) {
	clusterName, err := ClusterName(kuber)
	return strings.Split(clusterName, "-")[0], err
}

// GetSimplifiedClusterName get the simplified cluster name from the long-winded context cluster name that gets generated
// GKE cluster names as defined in the kube config are of the form gke_<projectname>_<region>_<clustername>
// This method will return <clustername> in the above
func GetSimplifiedClusterName(complexClusterName string) string {
	split := strings.Split(complexClusterName, "_")
	return split[len(split)-1]
}

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

func FindBucket(bucketName string) bool {
	fullBucketName := fmt.Sprintf("gs://%s", bucketName)
	args := []string{"list", "-b", fullBucketName}

	cmd := util.Command{
		Name: "gsutil",
		Args: args,
	}
	_, err := cmd.RunWithoutRetry()
	if err != nil {
		return false
	}
	return true
}

func DeleteAllObjectsInBucket(bucketName string) error {
	found := FindBucket(bucketName)
	if !found {
		return nil // nothing to delete
	}
	fullBucketName := fmt.Sprintf("gs://%s", bucketName)
	args := []string{"-m", "rm", "-r", fullBucketName}

	cmd := util.Command{
		Name: "gsutil",
		Args: args,
	}
	_, err := cmd.RunWithoutRetry()
	if err != nil {
		return err
	}
	return nil
}

func DeleteBucket(bucketName string) error {
	found := FindBucket(bucketName)
	if !found {
		return nil // nothing to delete
	}
	fullBucketName := fmt.Sprintf("gs://%s", bucketName)
	args := []string{"rb", fullBucketName}

	cmd := util.Command{
		Name: "gsutil",
		Args: args,
	}
	_, err := cmd.RunWithoutRetry()
	if err != nil {
		return err
	}
	return nil
}

func GetRegionFromZone(zone string) string {
	return zone[0 : len(zone)-2]
}

func FindServiceAccount(serviceAccount string, projectId string) bool {
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
		return false
	}

	if output == "Listed 0 items." {
		return false
	}
	return true
}

func GetOrCreateServiceAccount(serviceAccount string, projectId string, clusterConfigDir string, roles []string) (string, error) {
	if projectId == "" {
		return "", errors.New("cannot get/create a service account without a projectId")
	}

	found := FindServiceAccount(serviceAccount, projectId)
	if !found {
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
		args := []string{"iam",
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
		for _, role := range roles {
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
		err := CreateServiceAccountKey(serviceAccount, projectId, keyPath)
		if err != nil {
			log.Infof("Exceeds the maximum number of keys on service account %s\n",
				util.ColorInfo(serviceAccount))
			err := CleanupServiceAccountKeys(serviceAccount, projectId)
			if err != nil {
				return "", errors.Wrap(err, "cleaning up the service account keys")
			}
			err = CreateServiceAccountKey(serviceAccount, projectId, keyPath)
			if err != nil {
				return "", errors.Wrap(err, "creating service account key")
			}
		}
	} else {
		log.Info("Key already exists")
	}

	return keyPath, nil
}

// CreateServiceAccountKey creates a new service account key and downloads into the given file
func CreateServiceAccountKey(serviceAccount string, projectId string, keyPath string) error {
	args := []string{"iam",
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
		return errors.Wrap(err, "creating a new service account key")
	}
	return nil
}

// GetServiceAccountKeys returns all keys of a service account
func GetServiceAccountKeys(serviceAccount string, projectId string) ([]string, error) {
	keys := []string{}
	account := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", serviceAccount, projectId)
	args := []string{"iam",
		"service-accounts",
		"keys",
		"list",
		"--iam-account",
		account,
		"--project",
		projectId}
	cmd := util.Command{
		Name: "gcloud",
		Args: args,
	}
	output, err := cmd.RunWithoutRetry()
	if err != nil {
		return keys, errors.Wrapf(err, "listing the keys of the service account '%s'", account)
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	// Skip the first line with the header information
	scanner.Scan()
	for scanner.Scan() {
		keyFields := strings.Fields(scanner.Text())
		if len(keyFields) > 0 {
			keys = append(keys, keyFields[0])
		}
	}
	return keys, nil
}

// DeleteServiceAccountKey deletes a service account key
func DeleteServiceAccountKey(serviceAccount string, projectId string, key string) error {
	account := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", serviceAccount, projectId)
	args := []string{"iam",
		"service-accounts",
		"keys",
		"delete",
		key,
		"--iam-account",
		account,
		"--project",
		projectId,
		"--quiet"}
	cmd := util.Command{
		Name: "gcloud",
		Args: args,
	}
	_, err := cmd.RunWithoutRetry()
	if err != nil {
		return errors.Wrapf(err, "deleting the key '%s'from service account '%s'", key, account)
	}
	return nil
}

// CleanupServiceAccountKeys remove all keys from given service account
func CleanupServiceAccountKeys(serviceAccount string, projectId string) error {
	keys, err := GetServiceAccountKeys(serviceAccount, projectId)
	if err != nil {
		return errors.Wrap(err, "retrieving the service account keys")
	}

	log.Infof("Cleaning up the keys of the service account %s\n", util.ColorInfo(serviceAccount))

	for _, key := range keys {
		err := DeleteServiceAccountKey(serviceAccount, projectId, key)
		if err != nil {
			log.Infof("Cannot delete the key %s from service account %s: %v\n",
				util.ColorWarning(key), util.ColorInfo(serviceAccount), err)
		} else {
			log.Infof("Key %s was removed form service account %s\n",
				util.ColorInfo(key), util.ColorInfo(serviceAccount))
		}
	}
	return nil
}

func DeleteServiceAccount(serviceAccount string, projectId string, roles []string) error {
	found := FindServiceAccount(serviceAccount, projectId)
	if !found {
		return nil // nothing to delete
	}
	// remove roles to service account
	for _, role := range roles {
		log.Infof("Removing role %s\n", role)
		args := []string{"projects",
			"remove-iam-policy-binding",
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
			return err
		}
	}
	args := []string{"iam",
		"service-accounts",
		"delete",
		fmt.Sprintf("%s@%s.iam.gserviceaccount.com", serviceAccount, projectId),
		"--project",
		projectId}

	cmd := util.Command{
		Name: "gcloud",
		Args: args,
	}
	_, err := cmd.RunWithoutRetry()
	if err != nil {
		return err
	}
	return nil
}

func GetEnabledApis(projectId string) ([]string, error) {
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

// CreateKmsKeyring creates a new KMS keyring
func CreateKmsKeyring(keyringName string, projectId string) error {
	if keyringName == "" {
		return errors.New("provided keyring name is empty")
	}

	if IsKmsKeyringAvailable(keyringName, projectId) {
		return nil
	}

	args := []string{"kms",
		"keyrings",
		"create",
		keyringName,
		"--location",
		KmsLocation,
		"--project",
		projectId,
	}

	cmd := util.Command{
		Name: "gcloud",
		Args: args,
	}
	_, err := cmd.RunWithoutRetry()
	if err != nil {
		return errors.Wrap(err, "creating kms keyring")
	}
	return nil
}

// IsKmsKeyringAvailable checks if the KMS keyring is already available
func IsKmsKeyringAvailable(keyringName string, projectId string) bool {
	args := []string{"kms",
		"keyrings",
		"describe",
		keyringName,
		"--location",
		KmsLocation,
		"--project",
		projectId,
	}

	cmd := util.Command{
		Name: "gcloud",
		Args: args,
	}
	_, err := cmd.RunWithoutRetry()
	if err != nil {
		return false
	}
	return true
}

// CreateKmsKey creates a new KMS key in the given keyring
func CreateKmsKey(keyName string, keyringName string, projectId string) error {
	if IsKmsKeyAvailable(keyName, keyringName, projectId) {
		return nil
	}
	args := []string{"kms",
		"keys",
		"create",
		keyName,
		"--location",
		KmsLocation,
		"--keyring",
		keyringName,
		"--purpose",
		"encryption",
		"--project",
		projectId,
	}
	cmd := util.Command{
		Name: "gcloud",
		Args: args,
	}
	_, err := cmd.RunWithoutRetry()
	if err != nil {
		return errors.Wrapf(err, "creating kms key '%s' into keyring '%s'", keyName, keyringName)
	}
	return nil
}

// IsKmsKeyAvailable cheks if the KMS key is already available
func IsKmsKeyAvailable(keyName string, keyringName string, projectId string) bool {
	args := []string{"kms",
		"keys",
		"describe",
		keyName,
		"--location",
		KmsLocation,
		"--keyring",
		keyringName,
		"--project",
		projectId,
	}

	cmd := util.Command{
		Name: "gcloud",
		Args: args,
	}
	_, err := cmd.RunWithoutRetry()
	if err != nil {
		return false
	}
	return true
}
