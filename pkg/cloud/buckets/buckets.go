package buckets

import (
	"fmt"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
)

// CreateBucketURL creates a go-cloud URL to a bucket
func CreateBucketURL(name string, kind string, settings *jenkinsv1.TeamSettings) (string, error) {
	if kind == "" {
		provider := settings.KubeProvider
		if provider == "" {
			return "", fmt.Errorf("No bucket kind provided nor is a kubernetes provider configured for this team so it could not be defaulted")
		}
		kind = kubeProviderToBucketKind(provider)
		if kind == "" {
			return "", fmt.Errorf("No bucket kind is associated with kubernetes provider %s", provider)
		}
	}
	return kind + "://" + name, nil
}

func kubeProviderToBucketKind(provider string) string {
	switch provider {
	case "gke":
		return "gs"
	case "aws", "eks":
		return "s3"
	case "aks", "azure":
		return "azblob"
	default:
		return ""
	}
}
