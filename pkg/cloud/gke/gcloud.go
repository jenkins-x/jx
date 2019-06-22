package gke

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"time"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	osUser "os/user"
	"sigs.k8s.io/yaml"
)

// KmsLocation indicates the location used by the Google KMS service
const KmsLocation = "global"

var (
	// RequiredServiceAccountRoles the roles required to create a cluster with terraform
	RequiredServiceAccountRoles = []string{"roles/owner"}

	// KanikoServiceAccountRoles the roles required to run kaniko with GCS
	KanikoServiceAccountRoles = []string{"roles/storage.admin",
		"roles/storage.objectAdmin",
		"roles/storage.objectCreator"}
)

// getManagedZoneName constructs and returns a managed zone name using the domain value
func getManagedZoneName(domain string) string {

	var managedZoneName string

	if domain != "" {
		managedZoneName = strings.Replace(domain, ".", "-", -1)
		return fmt.Sprintf("%s-zone", managedZoneName)
	}
	return ""
}

// managedZoneExists checks for a given domain zone within the specified project
func managedZoneExists(projectID string, domain string) (bool, error) {
	args := []string{"dns",
		"managed-zones",
		fmt.Sprintf("--project=%s", projectID),
		"list",
		fmt.Sprintf("--filter=%s.", domain),
		"--format=json",
	}

	cmd := util.Command{
		Name: "gcloud",
		Args: args,
	}

	output, err := cmd.RunWithoutRetry()
	if err != nil {
		return true, errors.Wrap(err, "executing gcloud dns managed-zones list command ")
	}

	type managedZone struct {
		Name string `json:"name"`
	}

	var managedZones []managedZone

	err = yaml.Unmarshal([]byte(output), &managedZones)
	if err != nil {
		return true, errors.Wrap(err, "unmarshalling gcloud response")
	}

	if len(managedZones) > 0 {
		return true, nil
	}

	return false, nil
}

// CreateManagedZone creates a managed zone for the given domain in the specified project
func CreateManagedZone(projectID string, domain string) error {
	zoneExists, err := managedZoneExists(projectID, domain)
	if err != nil {
		return errors.Wrap(err, "unable to determine whether managed zone exists")
	}
	if !zoneExists {
		log.Logger().Infof("Managed Zone doesn't exist for %s domain, creating...", domain)
		managedZoneName := getManagedZoneName(domain)
		args := []string{"dns",
			"managed-zones",
			fmt.Sprintf("--project=%s", projectID),
			"create",
			managedZoneName,
			"--dns-name",
			fmt.Sprintf("%s.", domain),
			"--description=managed-zone utilised by jx",
		}

		cmd := util.Command{
			Name: "gcloud",
			Args: args,
		}

		_, err := cmd.RunWithoutRetry()
		if err != nil {
			return errors.Wrap(err, "executing gcloud dns managed-zones list command ")
		}
	} else {
		log.Logger().Infof("Managed Zone exists for %s domain.", domain)
	}
	return nil
}

// GetManagedZoneNameServers retrieves a list of name servers associated with a zone
func GetManagedZoneNameServers(projectID string, domain string) (string, []string, error) {
	var managedZoneName, nameServers = "", []string{}
	zoneExists, err := managedZoneExists(projectID, domain)
	if err != nil {
		return "", []string{}, errors.Wrap(err, "unable to determine whether managed zone exists")
	}
	if zoneExists {
		log.Logger().Infof("Getting nameservers for %s domain", domain)
		managedZoneName = getManagedZoneName(domain)
		args := []string{"dns",
			"managed-zones",
			fmt.Sprintf("--project=%s", projectID),
			"describe",
			managedZoneName,
			"--format=json",
		}

		cmd := util.Command{
			Name: "gcloud",
			Args: args,
		}

		type mz struct {
			Name        string   `json:"name"`
			NameServers []string `json:"nameServers"`
		}

		var managedZone mz

		output, err := cmd.RunWithoutRetry()
		if err != nil {
			return "", []string{}, errors.Wrap(err, "executing gcloud dns managed-zones list command ")
		}

		json.Unmarshal([]byte(output), &managedZone)
		if err != nil {
			return "", []string{}, errors.Wrap(err, "unmarshalling gcloud response when returning managed-zone nameservers")
		}
		nameServers = managedZone.NameServers
	} else {
		log.Logger().Infof("Managed Zone doesn't exist for %s domain.", domain)
	}
	return managedZoneName, nameServers, nil
}

// ClusterZone retrives the zone of GKE cluster description
func ClusterZone(cluster string) (string, error) {
	args := []string{"container",
		"clusters",
		"describe",
		cluster}

	cmd := util.Command{
		Name: "gcloud",
		Args: args,
	}
	output, err := cmd.RunWithoutRetry()
	if err != nil {
		return "", err
	}

	zone, err := parseClusterZone(output)
	if err != nil {
		return "", err
	}
	return zone, nil
}

func parseClusterZone(clusterInfo string) (string, error) {
	ci := struct {
		Zone string `json:"zone"`
	}{}

	err := yaml.Unmarshal([]byte(clusterInfo), &ci)
	if err != nil {
		return "", errors.Wrap(err, "extracting cluster zone from cluster info")
	}
	return ci.Zone, nil
}

type nodeConfig struct {
	OauthScopes []string `json:"oauthScopes"`
}

func parseScopes(clusterInfo string) ([]string, error) {

	ci := struct {
		NodeConfig nodeConfig `json:"nodeConfig"`
	}{}

	err := yaml.Unmarshal([]byte(clusterInfo), &ci)
	if err != nil {
		return nil, errors.Wrap(err, "extracting cluster oauthScopes from cluster info")
	}
	return ci.NodeConfig.OauthScopes, nil
}

// BucketExists checks if a Google Storage bucket exists
func BucketExists(projectID string, bucketName string) (bool, error) {
	fullBucketName := fmt.Sprintf("gs://%s", bucketName)
	args := []string{"ls"}

	if projectID != "" {
		args = append(args, "-p")
		args = append(args, projectID)
	}

	cmd := util.Command{
		Name: "gsutil",
		Args: args,
	}
	output, err := cmd.RunWithoutRetry()
	if err != nil {
		log.Logger().Infof("Error checking bucket exists: %s, %s", output, err)
		return false, err
	}
	return strings.Contains(output, fullBucketName), nil
}

// CreateBucket creates a new Google Storage bucket
func CreateBucket(projectID string, bucketName string, location string) error {
	fullBucketName := fmt.Sprintf("gs://%s", bucketName)
	args := []string{"mb", "-l", location}

	if projectID != "" {
		args = append(args, "-p")
		args = append(args, projectID)
	}

	args = append(args, fullBucketName)

	cmd := util.Command{
		Name: "gsutil",
		Args: args,
	}
	output, err := cmd.RunWithoutRetry()
	if err != nil {
		log.Logger().Infof("Error creating bucket: %s, %s", output, err)
		return err
	}
	return nil
}

//AddBucketLabel adds a label to a Google Storage bucket
func AddBucketLabel(bucketName string, label string) {
	found := FindBucket(bucketName)
	if found && label != "" {
		fullBucketName := fmt.Sprintf("gs://%s", bucketName)
		args := []string{"label", "ch", "-l", label}

		args = append(args, fullBucketName)

		cmd := util.Command{
			Name: "gsutil",
			Args: args,
		}
		output, err := cmd.RunWithoutRetry()
		if err != nil {
			log.Logger().Infof("Error adding bucket label: %s, %s", output, err)
		}
	}
}

// FindBucket finds a Google Storage bucket
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

// DeleteAllObjectsInBucket deletes all objects in a Google Storage bucket
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

// DeleteBucket deletes a Google storage bucket
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

// GetRegionFromZone parses the region from a GCP zone name
func GetRegionFromZone(zone string) string {
	return zone[0 : len(zone)-2]
}

// FindServiceAccount checks if a service account exists
func FindServiceAccount(serviceAccount string, projectID string) bool {
	args := []string{"iam",
		"service-accounts",
		"list",
		"--filter",
		serviceAccount,
		"--project",
		projectID}

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

// GetOrCreateServiceAccount retrieves or creates a GCP service account. It will return the path to the file where the service
// account token is stored
func GetOrCreateServiceAccount(serviceAccount string, projectID string, clusterConfigDir string, roles []string) (string, error) {
	if projectID == "" {
		return "", errors.New("cannot get/create a service account without a projectId")
	}

	found := FindServiceAccount(serviceAccount, projectID)
	if !found {
		log.Logger().Infof("Unable to find service account %s, checking if we have enough permission to create", util.ColorInfo(serviceAccount))

		// if it doesn't check to see if we have permissions to create (assign roles) to a service account
		hasPerm, err := CheckPermission("resourcemanager.projects.setIamPolicy", projectID)
		if err != nil {
			return "", err
		}

		if !hasPerm {
			return "", errors.New("User does not have the required role 'resourcemanager.projects.setIamPolicy' to configure a service account")
		}

		// create service
		log.Logger().Infof("Creating service account %s", util.ColorInfo(serviceAccount))
		args := []string{"iam",
			"service-accounts",
			"create",
			serviceAccount,
			"--project",
			projectID,
			"--display-name",
			serviceAccount}

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
			log.Logger().Infof("Assigning role %s", role)
			args = []string{"projects",
				"add-iam-policy-binding",
				projectID,
				"--member",
				fmt.Sprintf("serviceAccount:%s@%s.iam.gserviceaccount.com", serviceAccount, projectID),
				"--role",
				role,
				"--project",
				projectID}

			cmd := util.Command{
				Name: "gcloud",
				Args: args,
			}
			_, err := cmd.Run()
			if err != nil {
				return "", err
			}
		}

	} else {
		log.Logger().Info("Service Account exists")
	}

	os.MkdirAll(clusterConfigDir, os.ModePerm)
	keyPath := filepath.Join(clusterConfigDir, fmt.Sprintf("%s.key.json", serviceAccount))

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		log.Logger().Info("Downloading service account key")
		err := CreateServiceAccountKey(serviceAccount, projectID, keyPath)
		if err != nil {
			log.Logger().Infof("Exceeds the maximum number of keys on service account %s",
				util.ColorInfo(serviceAccount))
			err := CleanupServiceAccountKeys(serviceAccount, projectID)
			if err != nil {
				return "", errors.Wrap(err, "cleaning up the service account keys")
			}
			err = CreateServiceAccountKey(serviceAccount, projectID, keyPath)
			if err != nil {
				return "", errors.Wrap(err, "creating service account key")
			}
		}
	} else {
		log.Logger().Info("Key already exists")
	}

	return keyPath, nil
}

// CreateServiceAccountKey creates a new service account key and downloads into the given file
func CreateServiceAccountKey(serviceAccount string, projectID string, keyPath string) error {
	args := []string{"iam",
		"service-accounts",
		"keys",
		"create",
		keyPath,
		"--iam-account",
		fmt.Sprintf("%s@%s.iam.gserviceaccount.com", serviceAccount, projectID),
		"--project",
		projectID}

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
func GetServiceAccountKeys(serviceAccount string, projectID string) ([]string, error) {
	keys := []string{}
	account := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", serviceAccount, projectID)
	args := []string{"iam",
		"service-accounts",
		"keys",
		"list",
		"--iam-account",
		account,
		"--project",
		projectID}
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
func DeleteServiceAccountKey(serviceAccount string, projectID string, key string) error {
	account := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", serviceAccount, projectID)
	args := []string{"iam",
		"service-accounts",
		"keys",
		"delete",
		key,
		"--iam-account",
		account,
		"--project",
		projectID,
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
func CleanupServiceAccountKeys(serviceAccount string, projectID string) error {
	keys, err := GetServiceAccountKeys(serviceAccount, projectID)
	if err != nil {
		return errors.Wrap(err, "retrieving the service account keys")
	}

	log.Logger().Infof("Cleaning up the keys of the service account %s", util.ColorInfo(serviceAccount))

	for _, key := range keys {
		err := DeleteServiceAccountKey(serviceAccount, projectID, key)
		if err != nil {
			log.Logger().Infof("Cannot delete the key %s from service account %s: %v",
				util.ColorWarning(key), util.ColorInfo(serviceAccount), err)
		} else {
			log.Logger().Infof("Key %s was removed form service account %s",
				util.ColorInfo(key), util.ColorInfo(serviceAccount))
		}
	}
	return nil
}

// DeleteServiceAccount deletes a service account and its role bindings
func DeleteServiceAccount(serviceAccount string, projectID string, roles []string) error {
	found := FindServiceAccount(serviceAccount, projectID)
	if !found {
		return nil // nothing to delete
	}
	// remove roles to service account
	for _, role := range roles {
		log.Logger().Infof("Removing role %s", role)
		args := []string{"projects",
			"remove-iam-policy-binding",
			projectID,
			"--member",
			fmt.Sprintf("serviceAccount:%s@%s.iam.gserviceaccount.com", serviceAccount, projectID),
			"--role",
			role,
			"--project",
			projectID}

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
		fmt.Sprintf("%s@%s.iam.gserviceaccount.com", serviceAccount, projectID),
		"--project",
		projectID}

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

// GetEnabledApis returns which services have the API enabled
func GetEnabledApis(projectID string) ([]string, error) {
	args := []string{"services", "list", "--enabled"}

	if projectID != "" {
		args = append(args, "--project")
		args = append(args, projectID)
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

	lines := strings.Split(out, "\n")
	for _, l := range lines {
		if strings.Contains(l, "NAME") {
			continue
		}
		fields := strings.Fields(l)
		apis = append(apis, fields[0])
	}

	return apis, nil
}

// EnableAPIs enables APIs for the given services
func EnableAPIs(projectID string, apis ...string) error {
	enabledApis, err := GetEnabledApis(projectID)
	if err != nil {
		return err
	}

	toEnableArray := []string{}

	for _, toEnable := range apis {
		fullName := fmt.Sprintf("%s.googleapis.com", toEnable)
		if !util.Contains(enabledApis, fullName) {
			toEnableArray = append(toEnableArray, fullName)
		}
	}

	if len(toEnableArray) == 0 {
		log.Logger().Debugf("No apis need to be enable as they are already enabled: %s", util.ColorInfo(strings.Join(apis, " ")))
		return nil
	}

	args := []string{"services", "enable"}
	args = append(args, toEnableArray...)

	if projectID != "" {
		args = append(args, "--project")
		args = append(args, projectID)
	}

	log.Logger().Debugf("Lets ensure we have %s enabled on your project via: %s", toEnableArray, util.ColorInfo("gcloud "+strings.Join(args, " ")))

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

// Login login an user into Google account. It skips the interactive login using the
// browser when the skipLogin flag is active
func Login(serviceAccountKeyPath string, skipLogin bool) error {
	if serviceAccountKeyPath != "" {
		log.Logger().Infof("Activating service account %s", util.ColorInfo(serviceAccountKeyPath))

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
			log.Logger().Infof("Checking for readiness...")

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

// CheckPermission checks permission on the given project
func CheckPermission(perm string, projectID string) (bool, error) {
	if projectID == "" {
		return false, errors.New("cannot check permission without a projectId")
	}
	// if it doesn't check to see if we have permissions to create (assign roles) to a service account
	args := []string{"iam",
		"list-testable-permissions",
		fmt.Sprintf("//cloudresourcemanager.googleapis.com/projects/%s", projectID),
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
func CreateKmsKeyring(keyringName string, projectID string) error {
	if keyringName == "" {
		return errors.New("provided keyring name is empty")
	}

	if IsKmsKeyringAvailable(keyringName, projectID) {
		return nil
	}

	args := []string{"kms",
		"keyrings",
		"create",
		keyringName,
		"--location",
		KmsLocation,
		"--project",
		projectID,
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
func IsKmsKeyringAvailable(keyringName string, projectID string) bool {
	args := []string{"kms",
		"keyrings",
		"describe",
		keyringName,
		"--location",
		KmsLocation,
		"--project",
		projectID,
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
func CreateKmsKey(keyName string, keyringName string, projectID string) error {
	if IsKmsKeyAvailable(keyName, keyringName, projectID) {
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
		projectID,
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

// IsKmsKeyAvailable checks if the KMS key is already available
func IsKmsKeyAvailable(keyName string, keyringName string, projectID string) bool {
	args := []string{"kms",
		"keys",
		"describe",
		keyName,
		"--location",
		KmsLocation,
		"--keyring",
		keyringName,
		"--project",
		projectID,
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

// IsGCSWriteRoleEnabled will check if the devstorage.full_control scope is enabled in the cluster in order to use GCS
func IsGCSWriteRoleEnabled(cluster string, zone string) (bool, error) {
	args := []string{"container",
		"clusters",
		"describe",
		cluster,
		"--zone",
		zone}

	cmd := util.Command{
		Name: "gcloud",
		Args: args,
	}
	output, err := cmd.RunWithoutRetry()
	if err != nil {
		return false, err
	}

	oauthScopes, err := parseScopes(output)
	if err != nil {
		return false, err
	}

	for _, s := range oauthScopes {
		if strings.Contains(s, "devstorage.full_control") {
			return true, nil
		}
	}
	return false, nil
}

// UserLabel returns a string identifying current user that can be used as a label
func UserLabel() string {
	user, err := osUser.Current()
	if err == nil && user != nil && user.Username != "" {
		return fmt.Sprintf("created-by:%s", user.Username)
	}
	return ""
}
