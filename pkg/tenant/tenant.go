package tenant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"
)

const (
	basePath = "/api/v1"
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

// Result type for nameservers response
type Result struct {
	Message string `json:"message"`
}

func (tCli *tenantClient) GetTenantSubDomain(tenantServiceURL string, tenantServiceAuth string, projectID string) (string, error) {
	url := fmt.Sprintf("%s%s/domain", tenantServiceURL, basePath)
	var domainName, reqBody = "", []byte{}
	reqStruct := dRequest{
		Project: projectID,
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
	log.Logger().Debugf("%sing %s to %s\n", httpMethod, reqBody, url)
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
