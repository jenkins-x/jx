package tenant

import (
	"encoding/json"
	"fmt"
	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/util"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"
)

const (
	basePath = "/api/v1"
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

func (tCli *tenantClient) GetInstallationId(tenantServiceURL string, tenantServiceAuth string, gitHubOrg string) (string, error) {
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
		respBody, err := util.CallWithExponentialBackOff(url, tenantServiceAuth, "POST", reqBody, nil)
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
		respBody, err = util.CallWithExponentialBackOff(url, tenantServiceAuth, "POST", reqBody, nil)
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
		httpClient: util.GetClient(),
	}

	for option := range options {
		options[option](&tCli)
	}
	return &tCli
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
