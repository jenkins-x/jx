package kube

import (
	"fmt"
	"reflect"
	"time"

	"github.com/pkg/errors"

	"github.com/cenkalti/backoff"
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

// RegisterAllCRDs ensures that all Jenkins-X CRDs are registered
func RegisterAllCRDs(apiClient apiextensionsclientset.Interface) error {
	err := RegisterBuildPackCRD(apiClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the Build Pack CRD")
	}
	err = RegisterCommitStatusCRD(apiClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the Commit Status CRD")
	}
	err = RegisterEnvironmentCRD(apiClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the Environment CRD")
	}
	err = RegisterExtensionCRD(apiClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the Extension CRD")
	}
	err = RegisterAppCRD(apiClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the App CRD")
	}
	err = RegisterPluginCRD(apiClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the Plugin CRD")
	}
	err = RegisterEnvironmentRoleBindingCRD(apiClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the Environment Role Binding CRD")
	}
	err = RegisterGitServiceCRD(apiClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the Git Service CRD")
	}
	err = RegisterPipelineActivityCRD(apiClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the Pipeline Activity CRD")
	}
	err = RegisterReleaseCRD(apiClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the Release CRD")
	}
	err = RegisterSourceRepositoryCRD(apiClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the SourceRepository CRD")
	}
	err = RegisterTeamCRD(apiClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the Team CRD")
	}
	err = RegisterUserCRD(apiClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the User CRD")
	}
	err = RegisterWorkflowCRD(apiClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the Workflow CRD")
	}
	return nil
}

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
			Name:        "Promotion",
			Type:        "string",
			Description: "The strategy used for promoting to this environment",
			JSONPath:    ".spec.promotionStrategy",
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
			Description: "The Git repository URL for the source of the environment configuration",
			JSONPath:    ".spec.source.url",
		},
		{
			Name:        "Git Branch",
			Type:        "string",
			Description: "The git branch for the source of the environment configuration",
			JSONPath:    ".spec.source.ref",
		},
	}
	validation := v1beta1.CustomResourceValidation{}
	return RegisterCRD(apiClient, name, names, columns, &validation, jenkinsio.GroupName)
}

// RegisterEnvironmentRoleBindingCRD ensures that the CRD is registered for Environments
func RegisterEnvironmentRoleBindingCRD(apiClient apiextensionsclientset.Interface) error {
	name := "environmentrolebindings." + jenkinsio.GroupName
	names := &v1beta1.CustomResourceDefinitionNames{
		Kind:       "EnvironmentRoleBinding",
		ListKind:   "EnvironmentRoleBindingList",
		Plural:     "environmentrolebindings",
		Singular:   "environmentrolebinding",
		ShortNames: []string{"envrolebindings", "envrolebinding", "envrb"},
	}
	columns := []v1beta1.CustomResourceColumnDefinition{}
	validation := v1beta1.CustomResourceValidation{}
	return RegisterCRD(apiClient, name, names, columns, &validation, jenkinsio.GroupName)
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
	columns := []v1beta1.CustomResourceColumnDefinition{
		{
			Name:        "Git URL",
			Type:        "string",
			Description: "The URL of the Git repository",
			JSONPath:    ".spec.url",
		},
		{
			Name:        "Kind",
			Type:        "string",
			Description: "The kind of the Git provider",
			JSONPath:    ".spec.gitKind",
		},
	}
	validation := v1beta1.CustomResourceValidation{}
	return RegisterCRD(apiClient, name, names, columns, &validation, jenkinsio.GroupName)
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
			Description: "The URL of the Git repository",
			JSONPath:    ".spec.gitUrl",
		},
		{
			Name:        "Status",
			Type:        "string",
			Description: "The status of the pipeline",
			JSONPath:    ".spec.status",
		},
	}
	validation := v1beta1.CustomResourceValidation{}
	return RegisterCRD(apiClient, name, names, columns, &validation, jenkinsio.GroupName)
}

// RegisterExtensionCRD ensures that the CRD is registered for Extension
func RegisterExtensionCRD(apiClient apiextensionsclientset.Interface) error {
	name := "extensions." + jenkinsio.GroupName
	names := &v1beta1.CustomResourceDefinitionNames{
		Kind:       "Extension",
		ListKind:   "ExtensionList",
		Plural:     "extensions",
		Singular:   "extensions",
		ShortNames: []string{"extension", "ext"},
	}
	columns := []v1beta1.CustomResourceColumnDefinition{
		{
			Name:        "Name",
			Type:        "string",
			Description: "The name of the extension",
			JSONPath:    ".spec.name",
		},
		{
			Name:        "Description",
			Type:        "string",
			Description: "A description of the extension",
			JSONPath:    ".spec.description",
		},
	}
	validation := v1beta1.CustomResourceValidation{}
	return RegisterCRD(apiClient, name, names, columns, &validation, jenkinsio.GroupName)
}

// RegisterBuildPackCRD ensures that the CRD is registered for BuildPack
func RegisterBuildPackCRD(apiClient apiextensionsclientset.Interface) error {
	name := "buildpacks." + jenkinsio.GroupName
	names := &v1beta1.CustomResourceDefinitionNames{
		Kind:       "BuildPack",
		ListKind:   "BuildPackList",
		Plural:     "buildpacks",
		Singular:   "buildpack",
		ShortNames: []string{"bp"},
	}
	columns := []v1beta1.CustomResourceColumnDefinition{
		{
			Name:        "LABEL",
			Type:        "string",
			Description: "The label of the BuildPack",
			JSONPath:    ".spec.Label",
		},
		{
			Name:        "GIT URL",
			Type:        "string",
			Description: "The Git URL of the BuildPack",
			JSONPath:    ".spec.gitUrl",
		},
		{
			Name:        "Git Ref",
			Type:        "string",
			Description: "The Git REf of the BuildPack",
			JSONPath:    ".spec.gitRef",
		},
	}
	validation := v1beta1.CustomResourceValidation{}
	return RegisterCRD(apiClient, name, names, columns, &validation, jenkinsio.GroupName)
}

// RegisterAppCRD ensures that the CRD is registered for App
func RegisterAppCRD(apiClient apiextensionsclientset.Interface) error {
	name := "apps." + jenkinsio.GroupName
	names := &v1beta1.CustomResourceDefinitionNames{
		Kind:       "App",
		ListKind:   "AppList",
		Plural:     "apps",
		Singular:   "app",
		ShortNames: []string{"app"},
	}
	columns := []v1beta1.CustomResourceColumnDefinition{}
	validation := v1beta1.CustomResourceValidation{}
	return RegisterCRD(apiClient, name, names, columns, &validation, jenkinsio.GroupName)
}

// RegisterSourceRepositoryCRD ensures that the CRD is registered for Applications
func RegisterSourceRepositoryCRD(apiClient apiextensionsclientset.Interface) error {
	name := "applications." + jenkinsio.GroupName
	names := &v1beta1.CustomResourceDefinitionNames{
		Kind:       "SourceRepository",
		ListKind:   "SourceRepositoryList",
		Plural:     "sourcerepositiories",
		Singular:   "sourcerepository",
		ShortNames: []string{"sourcerepo", "srcrepo", "sr"},
	}
	columns := []v1beta1.CustomResourceColumnDefinition{
		{
			Name:        "Description",
			Type:        "string",
			Description: "A description of the source code repository - non-functional user-data",
			JSONPath:    ".spec.description",
		},
		{
			Name:        "Provider",
			Type:        "string",
			Description: "The source code provider (eg github) that the source repository is hosted in",
			JSONPath:    ".spec.org",
		},
		{
			Name:        "Org",
			Type:        "string",
			Description: "The git organisation that the source repository belongs to",
			JSONPath:    ".spec.org",
		},
		{
			Name:        "Repo",
			Type:        "string",
			Description: "The name of the repository",
			JSONPath:    ".spec.repo",
		},
	}
	validation := v1beta1.CustomResourceValidation{}
	return RegisterCRD(apiClient, name, names, columns, &validation, jenkinsio.GroupName)
}

// RegisterPluginCRD ensures that the CRD is registered for Plugin
func RegisterPluginCRD(apiClient apiextensionsclientset.Interface) error {
	name := "plugins." + jenkinsio.GroupName
	names := &v1beta1.CustomResourceDefinitionNames{
		Kind:     "Plugin",
		ListKind: "PluginList",
		Plural:   "plugins",
		Singular: "plugin",
	}
	columns := []v1beta1.CustomResourceColumnDefinition{
		{
			Name:        "Name",
			Type:        "string",
			Description: "The name of the plugin",
			JSONPath:    ".spec.name",
		},
		{
			Name:        "Description",
			Type:        "string",
			Description: "A description of the plugin",
			JSONPath:    ".spec.description",
		},
	}
	validation := v1beta1.CustomResourceValidation{}
	return RegisterCRD(apiClient, name, names, columns, &validation, jenkinsio.GroupName)
}

// RegisterCommitStatusCRD ensures that the CRD is registered for CommitStatus
func RegisterCommitStatusCRD(apiClient apiextensionsclientset.Interface) error {
	name := "commitstatuses." + jenkinsio.GroupName
	names := &v1beta1.CustomResourceDefinitionNames{
		Kind:       "CommitStatus",
		ListKind:   "CommitStatusList",
		Plural:     "commitstatuses",
		Singular:   "commitstatus",
		ShortNames: []string{"commitstatus"},
	}
	columns := []v1beta1.CustomResourceColumnDefinition{}
	validation := v1beta1.CustomResourceValidation{}
	return RegisterCRD(apiClient, name, names, columns, &validation, jenkinsio.GroupName)
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
			Description: "The URL of the Git repository",
			JSONPath:    ".spec.gitHttpUrl",
		},
	}
	validation := v1beta1.CustomResourceValidation{}
	return RegisterCRD(apiClient, name, names, columns, &validation, jenkinsio.GroupName)
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
	validation := v1beta1.CustomResourceValidation{}
	return RegisterCRD(apiClient, name, names, columns, &validation, jenkinsio.GroupName)
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
	validation := v1beta1.CustomResourceValidation{}
	return RegisterCRD(apiClient, name, names, columns, &validation, jenkinsio.GroupName)
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
	validation := v1beta1.CustomResourceValidation{}
	return RegisterCRD(apiClient, name, names, columns, &validation, jenkinsio.GroupName)
}

// RegisterCRD allows new custom resources to be registered using apiClient under a particular name.
// Various forms of the name are provided using names. In Kubernetes 1.11
// and later a custom display format for kubectl is used, which is specified using columns.
func RegisterCRD(apiClient apiextensionsclientset.Interface, name string,
	names *v1beta1.CustomResourceDefinitionNames, columns []v1beta1.CustomResourceColumnDefinition,
	validation *v1beta1.CustomResourceValidation, groupName string) error {
	crd := &v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:                    groupName,
			Version:                  jenkinsio.Version,
			Scope:                    v1beta1.NamespaceScoped,
			Names:                    *names,
			AdditionalPrinterColumns: columns,
			Validation:               validation,
		},
	}

	return register(apiClient, name, crd)
}

func register(apiClient apiextensionsclientset.Interface, name string, crd *v1beta1.CustomResourceDefinition) error {
	crdResources := apiClient.ApiextensionsV1beta1().CustomResourceDefinitions()

	f := func() error {
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

	exponentialBackOff := backoff.NewExponentialBackOff()
	timeout := 60 * time.Second
	exponentialBackOff.MaxElapsedTime = timeout
	exponentialBackOff.Reset()
	return backoff.Retry(f, exponentialBackOff)
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
