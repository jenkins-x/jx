package kube

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	v1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"github.com/jenkins-x/jx/pkg/apis/jx"
)

func RegisterEnvironmentCRD(apiClient *apiextensionsclientset.Clientset) error {
	name := "environments.jenkins.io"
	version := "v1"

	c, err := apiClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(name, metav1.GetOptions{})
	if err == nil {
		fmt.Printf("Found CRD %s with %#v\n", c.Name, c)
		return nil
	}

	crd := &v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group: jx.GroupName,
			Version: version,
			Scope: v1beta1.NamespaceScoped,
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind: "Environment",
				ListKind: "EnvironmentList",
				Plural: "environments",
				Singular: "environment",
				ShortNames: []string{"env"},
			},
		},
	}

	// lets create the CRD
	_, err = apiClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	return err
}
