package opts

import (
	"fmt"

	jxv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	v1fake "github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apifake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/kube"
)

// SetFakeKubeClient creates a fake KubeClient for CommonOptions
// Use this in case there is no active cluster that can be used
// to retrieve configuration information.
func (o *CommonOptions) SetFakeKubeClient() error {
	currentNamespace := "jx"
	k8sObjects := []runtime.Object{}
	jxObjects := []runtime.Object{}
	devEnv := kube.NewPermanentEnvironment("dev")
	devEnv.Spec.Namespace = currentNamespace
	devEnv.Spec.Kind = jxv1.EnvironmentKindTypeDevelopment

	k8sObjects = append(k8sObjects, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: currentNamespace,
			Labels: map[string]string{
				"tag": "",
			},
		},
	})

	client := fake.NewSimpleClientset(k8sObjects...)
	o.SetKubeClient(client)
	jxObjects = append(jxObjects, devEnv)
	o.SetJxClient(v1fake.NewSimpleClientset(jxObjects...))
	o.SetAPIExtensionsClient(apifake.NewSimpleClientset())
	return nil
}

// CreateGitAuthConfigService creates git auth config service
func (o *CommonOptions) CreateGitAuthConfigService() (auth.ConfigService, error) {
	_, namespace, err := o.KubeClientAndDevNamespace()
	if err != nil {
		// not returning an error here as we get this when attempting
		// to create a cluster when still connected to an old cluster
		log.Logger().Warnf("failed to find development namespace - %s", err)
	}
	return o.factory.CreateAuthConfigService(auth.GitAuthConfigFile, namespace, kube.ValueKindGit, "")
}

// CreateChatAuthConfigService creates a new chat auth service
func (o *CommonOptions) CreateChatAuthConfigService() (auth.ConfigService, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	_, namespace, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find development namespace")
	}
	return o.factory.CreateChatAuthConfigService(namespace)
}

// PickPipelineUserAuth returns the user auth for pipeline user
func (o *CommonOptions) PickPipelineUserAuth(config *auth.AuthConfig, server *auth.ServerAuth) (*auth.UserAuth, error) {
	var userName string
	if userName != "" {
		userAuth := config.GetOrCreateUserAuth(server.URL, userName)
		if userAuth != nil {
			return userAuth, nil
		}
	}
	var userAuth *auth.UserAuth
	var err error
	url := server.URL
	userAuths := config.FindUserAuths(url)
	if len(userAuths) > 1 {
		userAuth, err = config.PickServerUserAuth(server, "user name for the Pipeline", o.BatchMode, "", o.In, o.Out, o.Err)
		if err != nil {
			return userAuth, err
		}
	}
	if userAuth == nil {
		// lets create an empty one for now
		userAuth = &auth.UserAuth{}
	}
	return userAuth, nil
}

// GetDefaultAdminPassword returns the default admin password from dev namespace
func (o *CommonOptions) GetDefaultAdminPassword(devNamespace string) (string, error) {
	client, err := o.KubeClient() // cache may not have been created yet...
	if err != nil {
		return "", fmt.Errorf("cannot obtain k8s client %v", err)
	}
	basicAuth, err := client.CoreV1().Secrets(devNamespace).Get(JXInstallConfig, v1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("cannot find secret %s in namespace %s: %v", kube.SecretBasicAuth, devNamespace, err)
	}
	adminSecrets := basicAuth.Data[AdminSecretsFile]
	adminConfig := config.AdminSecretsConfig{}

	err = yaml.Unmarshal(adminSecrets, &adminConfig)
	if err != nil {
		return "", err
	}
	return adminConfig.Jenkins.JenkinsSecret.Password, nil
}

// AddonAuthConfigService creates the addon auth config service
func (o *CommonOptions) CreateAddonAuthConfigService() (auth.ConfigService, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	_, namespace, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find development namespace")
	}
	return o.factory.CreateAddonAuthConfigService(namespace)
}

// JenkinsAuthConfigService creates the jenkins auth config service
func (o *CommonOptions) CreateJenkinsAuthConfigService(namespace string) (auth.ConfigService, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	_, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return nil, err
	}
	if namespace == "" {
		namespace = ns
	}
	return o.factory.CreateJenkinsAuthConfigService(namespace)
}

// ChartmuseumAuthConfigService creates the chart museum auth config service
func (o *CommonOptions) CreateChartmuseumAuthConfigService() (auth.ConfigService, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	_, namespace, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find development namespace")
	}
	return o.factory.CreateChartmuseumAuthConfigService(namespace)
}

// AuthConfigService creates the auth config service for a given service
func (o *CommonOptions) CreateAuthConfigService(file string, serverKind string, serviceKind string) (auth.ConfigService, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	_, namespace, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find development namespace")
	}
	return o.factory.CreateAuthConfigService(file, namespace, serverKind, serviceKind)
}
