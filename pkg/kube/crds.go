package kube

import (
	jenkinsio "github.com/jenkins-x/jx/pkg/apis/jenkins.io"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RegisterEnvironmentCRD ensures that the CRD is registered for Environments
func RegisterEnvironmentCRD(apiClient apiextensionsclientset.Interface) error {
	name := "environments." + jenkinsio.GroupName
	names := &v1beta1.CustomResourceDefinitionNames{
		Kind:       "Environment",
		ListKind:   "EnvironmentList",
		Plural:     "environments",
		Singular:   "environment",
		ShortNames: []string{"env"},
	}

	return registerCRD(apiClient, name, names)
}

// RegisterEnvironmentRoleBindingCRD ensures that the CRD is registered for Environments
func RegisterEnvironmentRoleBindingCRD(apiClient apiextensionsclientset.Interface) error {
	name := "environmentrolebindings." + jenkinsio.GroupName
	names := &v1beta1.CustomResourceDefinitionNames{
		Kind:       "EnvironmentRoleBinding",
		ListKind:   "EnvironmentRoleBindingList",
		Plural:     "environmentrolebindings",
		Singular:   "environmentrolebinding",
		ShortNames: []string{"envrolebindings", "envrb"},
	}

	return registerCRD(apiClient, name, names)
}

// RegisterGitServiceCRD ensures that the CRD is registered for GitServices
func RegisterGitServiceCRD(apiClient apiextensionsclientset.Interface) error {
	name := "gitservices." + jenkinsio.GroupName
	names := &v1beta1.CustomResourceDefinitionNames{
		Kind:       "GitService",
		ListKind:   "GitServiceList",
		Plural:     "gitservices",
		Singular:   "gitservice",
		ShortNames: []string{"gits"},
	}

	return registerCRD(apiClient, name, names)
}

// RegisterPipelineActivityCRD ensures that the CRD is registered for PipelineActivity
func RegisterPipelineActivityCRD(apiClient apiextensionsclientset.Interface) error {
	name := "pipelineactivities." + jenkinsio.GroupName
	names := &v1beta1.CustomResourceDefinitionNames{
		Kind:       "PipelineActivity",
		ListKind:   "PipelineActivityList",
		Plural:     "pipelineactivities",
		Singular:   "pipelineactivity",
		ShortNames: []string{"activity", "act"},
	}

	return registerCRD(apiClient, name, names)
}

// RegisterReleaseCRD ensures that the CRD is registered for Release
func RegisterReleaseCRD(apiClient apiextensionsclientset.Interface) error {
	name := "releases." + jenkinsio.GroupName
	names := &v1beta1.CustomResourceDefinitionNames{
		Kind:       "Release",
		ListKind:   "ReleaseList",
		Plural:     "releases",
		Singular:   "release",
		ShortNames: []string{"rel"},
	}

	return registerCRD(apiClient, name, names)
}

// RegisterUserCRD ensures that the CRD is registered for User
func RegisterUserCRD(apiClient apiextensionsclientset.Interface) error {
	name := "users." + jenkinsio.GroupName
	names := &v1beta1.CustomResourceDefinitionNames{
		Kind:       "User",
		ListKind:   "UserList",
		Plural:     "users",
		Singular:   "user",
		ShortNames: []string{"usr"},
	}

	return registerCRD(apiClient, name, names)
}

func registerCRD(apiClient apiextensionsclientset.Interface, name string, names *v1beta1.CustomResourceDefinitionNames) error {
	_, err := apiClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(name, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	crd := &v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   jenkinsio.GroupName,
			Version: jenkinsio.Version,
			Scope:   v1beta1.NamespaceScoped,
			Names:   *names,
		},
	}
	_, err = apiClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	return err
}
