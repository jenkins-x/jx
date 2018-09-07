package kube

import (
	"fmt"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io"
	"github.com/jenkins-x/jx/pkg/jx/cmd/certmanager"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	CertmanagerCertificateProd    = "letsencrypt-prod"
	CertmanagerCertificateStaging = "letsencrypt-staging"
	CertmanagerIssuerProd         = "letsencrypt-prod"
	CertmanagerIssuerStaging      = "letsencrypt-staging"
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
	columns := []v1beta1.CustomResourceColumnDefinition{
		{
			Name:        "Namespace",
			Type:        "string",
			Description: "The namespace used for the environment",
			JSONPath:    ".spec.namespace",
		},
		{
			Name:        "Kind",
			Type:        "string",
			Description: "The kind of environment",
			JSONPath:    ".spec.kind",
		},
		{
			Name:        "Promotion Strategy",
			Type:        "string",
			Description: "The strategy used for promoting to this environment",
			JSONPath:    ".spec.oromotionStrategy",
		},
		{
			Name:        "Order",
			Type:        "integer",
			Description: "The order in which environments are automatically promoted",
			JSONPath:    ".spec.order",
		},
		{
			Name:        "Git URL",
			Type:        "string",
			Description: "The git repository URL for the source of the environment configuration",
			JSONPath:    ".spec.source.url",
		},
		{
			Name:        "Git Branch",
			Type:        "string",
			Description: "The git branch for the source of the environment configuration",
			JSONPath:    ".spec.source.ref",
		},
	}
	return registerCRD(apiClient, name, names, columns)
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
	columns := []v1beta1.CustomResourceColumnDefinition{}
	return registerCRD(apiClient, name, names, columns)
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
	columns := []v1beta1.CustomResourceColumnDefinition{}
	return registerCRD(apiClient, name, names, columns)
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
	columns := []v1beta1.CustomResourceColumnDefinition{
		{
			Name:        "Git URL",
			Type:        "string",
			Description: "The URL of the git repository",
			JSONPath:    ".spec.gitUrl",
		},
		{
			Name:        "Status",
			Type:        "string",
			Description: "The status of the pipeline",
			JSONPath:    ".spec.status",
		},
	}
	return registerCRD(apiClient, name, names, columns)
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
	columns := []v1beta1.CustomResourceColumnDefinition{
		{
			Name:        "Name",
			Type:        "string",
			Description: "The name of the Release",
			JSONPath:    ".spec.name",
		},
		{
			Name:        "Version",
			Type:        "string",
			Description: "The version number of the Release",
			JSONPath:    ".spec.version",
		},
		{
			Name:        "Git URL",
			Type:        "string",
			Description: "The URL of the git repository",
			JSONPath:    ".spec.gitHttpUrl",
		},
	}
	return registerCRD(apiClient, name, names, columns)
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
	columns := []v1beta1.CustomResourceColumnDefinition{
		{
			Name:        "Name",
			Type:        "string",
			Description: "The name of the user",
			JSONPath:    ".spec.name",
		},
		{
			Name:        "Email",
			Type:        "string",
			Description: "The email address of the user",
			JSONPath:    ".spec.email",
		},
	}
	return registerCRD(apiClient, name, names, columns)
}

// RegisterTeamCRD ensures that the CRD is registered for Team
func RegisterTeamCRD(apiClient apiextensionsclientset.Interface) error {
	name := "teams." + jenkinsio.GroupName
	names := &v1beta1.CustomResourceDefinitionNames{
		Kind:       "Team",
		ListKind:   "TeamList",
		Plural:     "teams",
		Singular:   "team",
		ShortNames: []string{"tm"},
	}
	columns := []v1beta1.CustomResourceColumnDefinition{
		{
			Name:        "Kind",
			Type:        "string",
			Description: "The kind of Team",
			JSONPath:    ".spec.kind",
		},
		{
			Name:        "Status",
			Type:        "string",
			Description: "The provision status of the Team",
			JSONPath:    ".status.provisionStatus",
		},
	}
	return registerCRD(apiClient, name, names, columns)
}

// RegisterWorkflowCRD ensures that the CRD is registered for Environments
func RegisterWorkflowCRD(apiClient apiextensionsclientset.Interface) error {
	name := "workflows." + jenkinsio.GroupName
	names := &v1beta1.CustomResourceDefinitionNames{
		Kind:       "Workflow",
		ListKind:   "WorkflowList",
		Plural:     "workflows",
		Singular:   "workflow",
		ShortNames: []string{"flow"},
	}
	columns := []v1beta1.CustomResourceColumnDefinition{}
	return registerCRD(apiClient, name, names, columns)
}

func registerCRD(apiClient apiextensionsclientset.Interface, name string, names *v1beta1.CustomResourceDefinitionNames, columns []v1beta1.CustomResourceColumnDefinition) error {
	crd := &v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   jenkinsio.GroupName,
			Version: jenkinsio.Version,
			Scope:   v1beta1.NamespaceScoped,
			Names:   *names,
			AdditionalPrinterColumns: columns,
		},
	}

	return register(apiClient, name, crd)
}

func register(apiClient apiextensionsclientset.Interface, name string, crd *v1beta1.CustomResourceDefinition ) error {
	crdResources := apiClient.ApiextensionsV1beta1().CustomResourceDefinitions()
	old, err := crdResources.Get(name, metav1.GetOptions{})
	if err == nil {
		if !reflect.DeepEqual(&crd.Spec, old.Spec) {
			old.Spec = crd.Spec
			_, err = crdResources.Update(old)
			return err
		}
		return nil
	}

	_, err = crdResources.Create(crd)
	return err
}

func CleanCertmanagerResources(c kubernetes.Interface, ns string, config IngressConfig) error {

	if config.Issuer == CertmanagerIssuerProd {
		_, err := c.CoreV1().RESTClient().Get().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/issuers", ns)).Name(CertmanagerIssuerProd).DoRaw()
		if err == nil {
			// existing clusterissuers found, recreate
			_, err = c.CoreV1().RESTClient().Delete().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/issuers", ns)).Name(CertmanagerIssuerProd).DoRaw()
			if err != nil {
				return fmt.Errorf("failed to delete issuer %s %v", "letsencrypt-prod", err)
			}
		}

		if config.TLS {
			issuerProd := fmt.Sprintf(certmanager.Cert_manager_issuer_prod, config.Email)
			json, err := yaml.YAMLToJSON([]byte(issuerProd))

			resp, err := c.CoreV1().RESTClient().Post().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/issuers", ns)).Body(json).DoRaw()
			if err != nil {
				return fmt.Errorf("failed to create issuer %v: %s", err, string(resp))
			}
		}

	} else {
		_, err := c.CoreV1().RESTClient().Get().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/issuers", ns)).Name(CertmanagerIssuerStaging).DoRaw()
		if err == nil {
			// existing clusterissuers found, recreate
			resp, err := c.CoreV1().RESTClient().Delete().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/issuers", ns)).Name(CertmanagerIssuerStaging).DoRaw()
			if err != nil {
				return fmt.Errorf("failed to delete issuer %v: %s", err, string(resp))
			}
		}

		if config.TLS {
			issuerStage := fmt.Sprintf(certmanager.Cert_manager_issuer_stage, config.Email)
			json, err := yaml.YAMLToJSON([]byte(issuerStage))

			resp, err := c.CoreV1().RESTClient().Post().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/issuers", ns)).Body(json).DoRaw()
			if err != nil {
				return fmt.Errorf("failed to create issuer %v: %s", err, string(resp))
			}
		}
	}

	// lets not error if they dont exist
	c.CoreV1().RESTClient().Delete().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/certificates", ns)).Name(CertmanagerCertificateStaging).DoRaw()
	c.CoreV1().RESTClient().Delete().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/certificates", ns)).Name(CertmanagerCertificateProd).DoRaw()

	// dont think we need this as we use a shim from ingress annotations to dynamically create the certificates
	//if config.TLS {
	//	cert := fmt.Sprintf(certmanager.Cert_manager_certificate, config.Issuer, config.Issuer, config.Domain, config.Domain)
	//	json, err := yaml.YAMLToJSON([]byte(cert))
	//	if err != nil {
	//		return fmt.Errorf("unable to convert YAML %s to JSON: %v", cert, err)
	//	}
	//	_, err = c.CoreV1().RESTClient().Post().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/certificates", ns)).Body(json).DoRaw()
	//	if err != nil {
	//		return fmt.Errorf("failed to create certificate %v", err)
	//	}
	//}

	return nil
}
