package tenant

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"github.com/jenkins-x/jx/pkg/cloud/gke"

	"github.com/jenkins-x/jx/pkg/kube"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	basePath                     = "/api/v1"
	tenantServiceTokenSecretName = "tenant-service-token"
	tenantServiceTokenSecretKey  = "token"
	tenantSignatureHeader        = "X-JenkinsX-Signature"
)

var (
	allowedDomainRegex = regexp.MustCompile("^[a-z0-9]+([_\\-\\.]{1}[a-z0-9]+)*\\.[a-z]{2,6}$")
)

// Tenant type for all tenant interactions
type Tenant struct {
	HttpClient *http.Client
	Gcloud     gke.GClouder
	Kube       kubernetes.Interface
	Namespace  string
}

type dRequest struct {
	Project string
	Cluster string
	User    string
}

type nsRequest struct {
	Project     string
	Domain      string
	Zone        string
	Nameservers []string
}

// Domain type for domain response
type Domain struct {
	Data struct {
		Subdomain string `json:"subdomain"`
	} `json:"data"`
}

type Installation struct {
	InstallationId string `json:installation-id`
	Organisation   string `json:org`
}

// Result type for nameservers response
type Result struct {
	Message string `json:"message"`
}

// Option allows configuration to be applied to the client
type Option func(*Tenant)

// NewTenantClient creates a new tenant
func NewTenantClient(options ...Option) *Tenant {
	t := Tenant{
		HttpClient: &http.Client{},
	}

	for option := range options {
		options[option](&t)
	}
	return &t
}

// GetInstallationID returns the GitHub InstallationID from the tenant-service
func (t *Tenant) GetInstallationID(tenantServiceURL string, tenantServiceAuth string, gitHubOrg string) (string, error) {
	requestUrl := fmt.Sprintf("%s%s/installation-id", tenantServiceURL, basePath)

	params := url.Values{}
	params.Set("org", gitHubOrg)

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	token := t.getTenantSignatureToken(tenantServiceTokenSecretName, tenantServiceTokenSecretKey)
	if token != "" {
		headers.Set(tenantSignatureHeader, token)
	}

	httpUtils := &util.HttpUtils{
		Client:     t.HttpClient,
		HTTPMethod: http.MethodGet,
		URL:        requestUrl,
		Auth:       tenantServiceAuth,
		Headers:    &headers,
		ReqParams:  &params,
	}

	respBody, err := httpUtils.CallWithExponentialBackOff()
	if err != nil {
		return "", errors.Wrapf(err, "error getting installation id via %s", requestUrl)
	}

	var installation Installation
	err = json.Unmarshal(respBody, &installation)
	if err != nil {
		return "", errors.Wrap(err, "unmarshalling json message")
	}

	return installation.InstallationId, nil
}

// GetAndStoreTenantToken retrieves the real tenant token and stores its value within a kubernetes secret
func (t *Tenant) GetAndStoreTenantToken(tenantServiceURL string, tenantServiceAuth string, project string, tempToken string) error {
	if project == "" {
		return errors.Errorf("project is empty")
	}
	token, err := t.getTenantToken(tenantServiceURL, tenantServiceAuth, project, tempToken)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("getting tenant-service token for %s project", project))
	}

	if token == "" {
		return errors.Errorf("tenant token is empty")
	}
	err = t.writeKubernetesSecret(tenantServiceTokenSecretName, tenantServiceTokenSecretKey, []byte(token))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("writing kubernetes secret in namespace: %s", t.Namespace))
	}
	response, err := t.deleteTempTenantToken(tenantServiceURL, tenantServiceAuth, project, tempToken)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("deleting temporary tenant-service token for %s project", project))
	}
	log.Logger().Infof("temporary tenant-service token response: %s", response)
	return nil
}

func (t *Tenant) getTenantToken(tenantServiceURL string, tenantServiceAuth string, project string, tempToken string) (string, error) {
	var url, token = "", ""
	if project != "" {
		url = fmt.Sprintf("%s%s/rockets/token/tmp/%s", tenantServiceURL, basePath, project)
		reqBody := []byte(tempToken)

		headers := http.Header{}
		headers.Set("Content-Type", "application/json")

		httpUtils := &util.HttpUtils{
			Client:     t.HttpClient,
			URL:        url,
			Auth:       tenantServiceAuth,
			ReqBody:    reqBody,
			Headers:    &headers,
			HTTPMethod: http.MethodPost,
			ReqParams:  nil,
		}

		respBody, err := httpUtils.CallWithExponentialBackOff()
		if err != nil {
			return "", errors.Wrapf(err, "error getting tenant token via %s", url)
		}
		token = string(respBody)
	} else {
		return "", errors.Errorf("project is empty")
	}
	return token, nil
}

func (t *Tenant) deleteTempTenantToken(tenantServiceURL string, tenantServiceAuth string, project string, tempToken string) (string, error) {
	var url = ""
	if project != "" {
		url = fmt.Sprintf("%s%s/rockets/token/tmp/%s", tenantServiceURL, basePath, project)
		reqBody := []byte(tempToken)

		headers := http.Header{}
		headers.Set("Content-Type", "application/json")
		token := t.getTenantSignatureToken(tenantServiceTokenSecretName, tenantServiceTokenSecretKey)
		if token != "" {
			headers.Set(tenantSignatureHeader, token)
		}

		httpUtils := &util.HttpUtils{
			Client:     t.HttpClient,
			URL:        tenantServiceURL,
			Auth:       tenantServiceAuth,
			HTTPMethod: http.MethodDelete,
			Headers:    &headers,
			ReqBody:    reqBody,
		}

		respBody, err := httpUtils.CallWithExponentialBackOff()
		if err != nil {
			return "", errors.Wrapf(err, "error getting tenant token via %s", url)
		}
		return string(respBody), nil
	} else {
		return "", errors.Errorf("project is empty")
	}
}

// GetTenantSubDomain requests a subdomain for a given projectID
func (t *Tenant) GetTenantSubDomain(tenantServiceURL string, tenantServiceAuth string, projectID string, cluster string) (string, error) {
	var domainName, reqBody, userEmail = "", []byte{}, ""
	if projectID == "" {
		return "", errors.Errorf("projectID is empty")
	}

	url := fmt.Sprintf("%s%s/domain", tenantServiceURL, basePath)
	// temporary change, this will be refactored into a step
	if "" != os.Getenv("USER_EMAIL") {
		userEmail = os.Getenv("USER_EMAIL")
	}

	reqStruct := dRequest{
		Project: projectID,
		Cluster: cluster,
		User:    userEmail,
	}
	reqBody, err := json.Marshal(reqStruct)
	if err != nil {
		return "", errors.Wrap(err, "error marshalling struct into json")
	}

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	token := t.getTenantSignatureToken(tenantServiceTokenSecretName, tenantServiceTokenSecretKey)
	if token != "" {
		headers.Set(tenantSignatureHeader, token)
	}

	httpUtils := &util.HttpUtils{
		Client:     t.HttpClient,
		URL:        tenantServiceURL,
		Auth:       tenantServiceAuth,
		HTTPMethod: http.MethodPost,
		Headers:    &headers,
		ReqBody:    reqBody,
	}

	respBody, err := httpUtils.CallWithExponentialBackOff()
	if err != nil {
		return "", errors.Wrapf(err, "error getting tenant sub-domain via %s", url)
	}
	var d Domain
	err = json.Unmarshal(respBody, &d)
	if err != nil {
		return "", errors.Wrap(err, "unmarshalling json message")
	}
	domainName = d.Data.Subdomain

	err = ValidateDomainName(domainName)
	if err != nil {
		return "", errors.Wrap(err, "domain name failed validation")
	}

	// Checking whether dns api is enabled
	err = t.Gcloud.EnableAPIs(projectID, "dns")
	if err != nil {
		return "", errors.Wrap(err, "enabling the dns api")
	}

	// Create domain if it doesn't exist and return name servers list
	managedZone, nameServers, err := t.Gcloud.CreateDNSZone(projectID, domainName)
	if err != nil {
		return "", errors.Wrap(err, "while trying to create the tenants subdomain zone")
	}

	log.Logger().Infof("%s domain is operating on the following nameservers %v", domainName, nameServers)
	err = t.PostTenantZoneNameServers(tenantServiceURL, tenantServiceAuth, projectID, domainName, managedZone, nameServers)
	if err != nil {
		return "", errors.Wrap(err, "posting the name service list to the tenant service")
	}
	return domainName, nil
}

// PostTenantZoneNameServers registers a tenants managed domain nameservers in order to delegate to the subdomain from the parent domain
func (t *Tenant) PostTenantZoneNameServers(tenantServiceURL string, tenantServiceAuth string, projectID string, domain string, zone string, nameServers []string) error {
	url := fmt.Sprintf("%s%s/nameservers", tenantServiceURL, basePath)
	if projectID != "" && zone != "" && len(nameServers) > 0 {
		reqStruct := nsRequest{
			Project:     projectID,
			Domain:      domain,
			Zone:        zone,
			Nameservers: nameServers,
		}
		respBody := []byte{}
		reqBody, err := json.Marshal(reqStruct)
		if err != nil {
			return errors.Wrap(err, "error marshalling struct into json")
		}

		headers := http.Header{}
		headers.Set("Content-Type", "application/json")
		token := t.getTenantSignatureToken(tenantServiceTokenSecretName, tenantServiceTokenSecretKey)
		if token != "" {
			headers.Set(tenantSignatureHeader, token)
		}

		httpUtils := &util.HttpUtils{
			Client:     t.HttpClient,
			URL:        tenantServiceURL,
			Auth:       tenantServiceAuth,
			HTTPMethod: http.MethodPost,
			Headers:    &headers,
			ReqBody:    reqBody,
		}

		respBody, err = httpUtils.CallWithExponentialBackOff()
		if err != nil {
			return errors.Wrapf(err, "error posting tenant sub-domain nameservers via %s", url)
		}
		var r Result
		err = json.Unmarshal(respBody, &r)
		if err != nil {
			return errors.Wrap(err, "unmarshalling json message")
		}
		return nil
	}
	return errors.Errorf("projectID/zone/nameServers is empty")
}

// ValidateDomainName checks for compliance in a supplied domain name
func ValidateDomainName(domain string) error {
	// Check whether the domain is greater than 3 and fewer than 63 characters in length
	if len(domain) < 3 || len(domain) > 63 {
		err := fmt.Errorf("domain name %v has fewer than 3 or greater than 63 characters", domain)
		return err
	}
	// Ensure each part of the domain name only contains lower/upper case characters, numbers and dashes
	if !allowedDomainRegex.MatchString(domain) {
		err := fmt.Errorf("domain name %v contains invalid characters", domain)
		return err
	}
	return nil
}

func (t *Tenant) writeKubernetesSecret(secretName string, secretKey string, token []byte) error {
	var err error
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Data: map[string][]byte{
			secretKey: token,
		},
	}
	secrets := t.Kube.CoreV1().Secrets(t.Namespace)
	_, err = secrets.Get(secretName, metav1.GetOptions{})
	if err != nil {
		_, err = secrets.Create(secret)
	} else {
		_, err = secrets.Update(secret)
	}
	return nil
}

func (t *Tenant) getTenantSignatureToken(secretName string, secretKey string) string {
	secret, err := kube.GetSecret(t.Kube, t.Namespace, secretName, secretKey)
	if err != nil {
		log.Logger().Infof("%e: '%s' secret not found in '%s' namespace", err, secretName, t.Namespace)
		return ""
	}
	return string(secret.Data[secretKey])
}
