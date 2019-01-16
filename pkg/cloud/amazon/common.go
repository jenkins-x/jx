package amazon

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"path"
	"runtime"
)

const DefaultRegion = "us-west-2"

const regionDataMapKey = "AWS_REGION"

func NewAwsSession(profileOption string, regionOption string) (*session.Session, error) {
	config := aws.Config{}
	if regionOption != "" {
		config.Region = aws.String(regionOption)
	}
	if _, err := os.Stat(path.Join(UserHomeDir(), ".aws", "credentials")); !os.IsNotExist(err) {
		config.Credentials = credentials.NewChainCredentials(
			[]credentials.Provider{
				&credentials.EnvProvider{},
				&credentials.SharedCredentialsProvider{Filename: "", Profile: profileOption},
			})
	}

	sessionOptions := session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            config,
	}

	if profileOption != "" {
		sessionOptions.Profile = profileOption
	}

	awsSession, err := session.NewSessionWithOptions(sessionOptions)

	if *awsSession.Config.Region == "" {
		awsSession.Config.Region = aws.String(DefaultRegion)
	}

	if err != nil {
		return nil, err
	}
	return awsSession, nil
}

func NewAwsSessionWithoutOptions() (*session.Session, error) {
	return NewAwsSession("", "")
}

func ResolveRegion(profileOption string, regionOption string) (string, error) {
	session, err := NewAwsSession(profileOption, regionOption)
	if err != nil {
		return "", err
	}
	return *session.Config.Region, nil
}

func ResolveRegionWithoutOptions() (string, error) {
	return ResolveRegion("", "")
}

func RememberRegion(kubeClient kubernetes.Interface, namespace string, region string) error {
	_, err := kube.DefaultModifyConfigMap(kubeClient, namespace, kube.ConfigMapNameJXInstallConfig, func(configMap *v1.ConfigMap) error {
		configMap.Data[regionDataMapKey] = region
		return nil
	}, nil)
	if err != nil {
		return errors.Wrapf(err, "saving AWS region in ConfigMap %s", kube.ConfigMapNameJXInstallConfig)
	} else {
		return nil
	}
}

func ReadRegion(kubeClient kubernetes.Interface, namespace string) (string, error) {
	data, err := kube.GetConfigMapData(kubeClient, kube.ConfigMapNameJXInstallConfig, namespace)
	if err != nil {
		return "", err
	}
	return data[regionDataMapKey], nil
}

// UserHomeDir returns the home directory for the user the process is running under.
// This is a copy of shareddefaults.UserHomeDir in the internal AWS package.
// We can't user user.Current().HomeDir as we want to override this during testing. :-|
func UserHomeDir() string {
	if runtime.GOOS == "windows" { // Windows
		return os.Getenv("USERPROFILE")
	}

	// *nix
	return os.Getenv("HOME")
}
