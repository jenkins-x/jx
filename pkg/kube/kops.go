package kube

import (
	"github.com/Jeffail/gabs"
)

const (
	additionalNodePolicies = `[{"Action":["ecr:InitiateLayerUpload","ecr:UploadLayerPart","ecr:CompleteLayerUpload","ecr:PutImage"],"Effect":"Allow","Resource":["*"],"Sid":"kopsK8sECRwrite"}]`
)

// EnableInsecureRegistry appends the Docker Registry
func EnableInsecureRegistry(iqJson string, dockerRegistry string) (string, error) {
	doc, err := gabs.ParseJSON([]byte(iqJson))
	if err != nil {
		return "", err
	}

	_, err = doc.Set(dockerRegistry, "spec", "docker", "insecureRegistry")
	if err != nil {
		return "", err
	}

	_, err = doc.Set(additionalNodePolicies, "spec", "additionalPolicies", "node")
	if err != nil {
		return "", err
	}
	return doc.String(), nil
}
