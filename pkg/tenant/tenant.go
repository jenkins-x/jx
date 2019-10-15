package tenant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/cenkalti/backoff"
	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	basePath                     = "/api/v1"
	tenantServiceTokenSecretName = "tenant-service-token"
	tenantServiceTokenSecretKey  = "token"
)

var (
	allowedDomainRegex = regexp.MustCompile("^[a-z0-9]+([_\\-\\.]{1}[a-z0-9]+)*\\.[a-z]{2,6}$")
)

type tenantClient struct {
	httpClient *http.Client
}

type Option func(*tenantClient)

// SetHTTPClient used to set the HTTP Client
func SetHTTPClient(httpClient *http.Client) Option {
	return func(tCli *tenantClient) {
		tCli.httpClient = httpClient
	}
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

func (tCli *tenantClient) GetInstallationID(tenantServiceURL string, tenantServiceAuth string, gitHubOrg string) (string, error) {
	requestUrl := fmt.Sprintf("%s%s/installation-id", tenantServiceURL, basePath)

	params := url.Values{}
	params.Set("org", gitHubOrg)

	respBody, err := util.CallWithExponentialBackOff(requestUrl, tenantServiceAuth, "GET", []byte{}, params)
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

func (tCli *tenantClient) GetAndStoreTenantToken(tenantServiceURL string, tenantServiceAuth string, project string, tempToken string, namespace string, kubeClient kubernetes.Interface) error {
	if project != "" {
		token, err := tCli.GetTenantToken(tenantServiceURL, tenantServiceAuth, project, tempToken)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("getting tenant-service token for %s project", project))
		}
		if "" != token {
			err := writeKubernetesSecret([]byte(token), namespace, kubeClient)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("writing kubernetes secret in namespace: %s", namespace))
			}
			response, err := tCli.DeleteTempTenantToken(tenantServiceURL, tenantServiceAuth, project, tempToken)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("deleting temporary tenant-service token for %s project", project))
			}
			log.Logger().Infof("temporary tenant-service token response: %s", response)
		} else {
			return errors.Errorf("tenant token is empty")
		}
	} else {
		return errors.Errorf("project is empty")
	}
	return nil
}

func (tCli *tenantClient) GetTenantToken(tenantServiceURL string, tenantServiceAuth string, project string, tempToken string) (string, error) {
	var url, token = "", ""
	if project != "" {
		url = fmt.Sprintf("%s%s/rockets/token/tmp/%s", tenantServiceURL, basePath, project)
		reqBody := []byte(tempToken)
		respBody, err := tCli.callWithExponentialBackOff(url, tenantServiceAuth, "POST", reqBody)
		if err != nil {
			return "", errors.Wrapf(err, "error getting tenant token via %s", url)
		}
		token = string(respBody)
	} else {
		return "", errors.Errorf("project is empty")
	}
	return token, nil
}

func (tCli *tenantClient) DeleteTempTenantToken(tenantServiceURL string, tenantServiceAuth string, project string, tempToken string) (string, error) {
	var url, token = "", ""
	if project != "" {
		url = fmt.Sprintf("%s%s/rockets/token/tmp/%s", tenantServiceURL, basePath, project)
		reqBody := []byte(tempToken)
		respBody, err := tCli.callWithExponentialBackOff(url, tenantServiceAuth, "DELETE", reqBody)
		if err != nil {
			return "", errors.Wrapf(err, "error getting tenant token via %s", url)
		}
		token = string(respBody)
	} else {
		return "", errors.Errorf("project is empty")
	}
	return token, nil
}

func (tCli *tenantClient) GetTenantSubDomain(tenantServiceURL string, tenantServiceAuth string, projectID string, cluster string, gcloud gke.GClouder) (string, error) {
	url := fmt.Sprintf("%s%s/domain", tenantServiceURL, basePath)
	var domainName, reqBody, userEmail = "", []byte{}, ""

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
	if projectID != "" {
		respBody, err := tCli.callWithExponentialBackOff(url, tenantServiceAuth, "POST", reqBody)
		if err != nil {
			return "", errors.Wrapf(err, "error getting tenant sub-domain via %s", url)
		}
		var d Domain
		err = json.Unmarshal(respBody, &d)
		if err != nil {
			return "", errors.Wrap(err, "unmarshalling json message")
		}
		domainName = d.Data.Subdomain
	} else {
		return "", errors.Errorf("projectID is empty")
	}

	err = ValidateDomainName(domainName)
	if err != nil {
		return "", errors.Wrap(err, "domain name failed validation")
	}

	// Checking whether dns api is enabled
	err = gcloud.EnableAPIs(projectID, "dns")
	if err != nil {
		return "", errors.Wrap(err, "enabling the dns api")
	}

	// Create domain if it doesn't exist and return name servers list
	managedZone, nameServers, err := gcloud.CreateDNSZone(projectID, domainName)
	if err != nil {
		return "", errors.Wrap(err, "while trying to create the tenants subdomain zone")
	}

	log.Logger().Infof("%s domain is operating on the following nameservers %v", domainName, nameServers)
	err = tCli.PostTenantZoneNameServers(tenantServiceURL, tenantServiceAuth, projectID, domainName, managedZone, nameServers)
	if err != nil {
		return "", errors.Wrap(err, "posting the name service list to the tenant service")
	}
	return domainName, nil
}

func (tCli *tenantClient) PostTenantZoneNameServers(tenantServiceURL string, tenantServiceAuth string, projectID string, domain string, zone string, nameServers []string) error {
	url := fmt.Sprintf("%s%s/nameservers", tenantServiceURL, basePath)
	reqStruct := nsRequest{
		Project:     projectID,
		Domain:      domain,
		Zone:        zone,
		Nameservers: nameServers,
	}
	reqBody, respBody := []byte{}, []byte{}
	reqBody, err := json.Marshal(reqStruct)
	if err != nil {
		return errors.Wrap(err, "error marshalling struct into json")
	}

	if projectID != "" && zone != "" && len(nameServers) > 0 {
		respBody, err = tCli.callWithExponentialBackOff(url, tenantServiceAuth, "POST", reqBody)
		if err != nil {
			return errors.Wrapf(err, "error posting tenant sub-domain nameservers via %s", url)
		}
		var r Result
		err = json.Unmarshal(respBody, &r)
		if err != nil {
			return errors.Wrap(err, "unmarshalling json message")
		}
	} else {
		return errors.Errorf("projectID/zone/nameServers is empty")
	}
	return nil
}

// NewTenantClient creates a new tenant client
func NewTenantClient(options ...Option) *tenantClient {
	tCli := tenantClient{
		httpClient: &http.Client{},
	}

	for option := range options {
		options[option](&tCli)
	}
	return &tCli
}

func (tCli *tenantClient) callWithExponentialBackOff(url string, auth string, httpMethod string, reqBody []byte) ([]byte, error) {
	log.Logger().Debugf("%sing %s to %s", httpMethod, reqBody, url)
	resp, respBody := &http.Response{}, []byte{}
	if url != "" && httpMethod != "" {
		f := func() error {
			req, err := http.NewRequest(httpMethod, url, bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			if !strings.Contains(url, "localhost") || !strings.Contains(url, "127.0.0.1") {
				if strings.Count(auth, ":") == 1 {
					jxBasicAuthUser, jxBasicAuthPass := getBasicAuthUserAndPassword(auth)
					req.SetBasicAuth(jxBasicAuthUser, jxBasicAuthPass)
				}
			}

			resp, err = tCli.httpClient.Do(req)
			if err != nil {
				return backoff.Permanent(err)
			}
			if resp.StatusCode < 200 && resp.StatusCode >= 300 {
				return errors.Errorf("%s not available, error was %d %s", url, resp.StatusCode, resp.Status)
			}
			respBody, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return backoff.Permanent(errors.Wrap(err, "parsing response body"))
			}
			resp.Body.Close()
			return nil
		}
		exponentialBackOff := backoff.NewExponentialBackOff()
		timeout := 1 * time.Minute
		exponentialBackOff.MaxElapsedTime = timeout
		exponentialBackOff.Reset()
		err := backoff.Retry(f, exponentialBackOff)
		if err != nil {
			return []byte{}, errors.Wrapf(err, "error getting tenant sub-domain via %s", url)
		}
	}
	return respBody, nil
}

func getBasicAuthUserAndPassword(auth string) (string, string) {
	if auth != "" {
		creds := strings.Fields(strings.Replace(auth, ":", " ", -1))
		return creds[0], creds[1]
	}
	return "", ""
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

func writeKubernetesSecret(token []byte, namespace string, client kubernetes.Interface) error {
	var err error
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: tenantServiceTokenSecretName,
		},
		Data: map[string][]byte{
			tenantServiceTokenSecretKey: token,
		},
	}
	secrets := client.CoreV1().Secrets(namespace)
	_, err = secrets.Get(tenantServiceTokenSecretName, metav1.GetOptions{})
	if err != nil {
		_, err = secrets.Create(secret)
	} else {
		_, err = secrets.Update(secret)
	}
	return nil
}
