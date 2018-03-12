package installer

import (
	"k8s.io/helm/pkg/helm"
)

// Uninstall uses the helm client to uninstall Draftd with the given config.
//
// Returns an error if the command failed.
func Uninstall(client *helm.Client) error {
	_, err := client.DeleteRelease(ReleaseName, helm.DeletePurge(true))
	return prettyError(err)
}
