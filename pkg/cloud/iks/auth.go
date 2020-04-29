package iks

import (
	"encoding/json"
	"io/ioutil"
	gohttp "net/http"
	"os"
	"os/user"

	ibmcloud "github.com/IBM-Cloud/bluemix-go"
	"github.com/IBM-Cloud/bluemix-go/http"
	"github.com/IBM-Cloud/bluemix-go/rest"
)

type ConfigJSON struct {
	APIEndpoint     string `json:"APIEndpoint"`
	ConsoleEndpoint string `json:"ConsoleEndpoint"`
	Region          string `json:"Region"`
	RegionID        string `json:"RegionID"`
	RegionType      string `json:"RegionType"`
	IAMEndpoint     string `json:"IAMEndpoint"`
	IAMToken        string `json:"IAMToken"`
	IAMRefreshToken string `json:"IAMRefreshToken"`
	Account         struct {
		GUID  string `json:"GUID"`
		Name  string `json:"Name"`
		Owner string `json:"Owner"`
	} `json:"Account"`
	ResourceGroup struct {
		GUID    string `json:"GUID"`
		Name    string `json:"Name"`
		State   string `json:"State"`
		Default bool   `json:"Default"`
		QuotaID string `json:"QuotaID"`
	} `json:"ResourceGroup"`
	CFEETargeted bool   `json:"CFEETargeted"`
	CFEEEnvID    string `json:"CFEEEnvID"`
	PluginRepos  []struct {
		Name string `json:"Name"`
		URL  string `json:"URL"`
	} `json:"PluginRepos"`
	SSLDisabled                bool   `json:"SSLDisabled"`
	Locale                     string `json:"Locale"`
	Trace                      string `json:"Trace"`
	ColorEnabled               string `json:"ColorEnabled"`
	HTTPTimeout                int    `json:"HTTPTimeout"`
	CLIInfoEndpoint            string `json:"CLIInfoEndpoint"`
	CheckCLIVersionDisabled    bool   `json:"CheckCLIVersionDisabled"`
	UsageStatsDisabled         bool   `json:"UsageStatsDisabled"`
	SDKVersion                 string `json:"SDKVersion"`
	UpdateCheckInterval        int    `json:"UpdateCheckInterval"`
	UpdateRetryCheckInterval   int    `json:"UpdateRetryCheckInterval"`
	UpdateNotificationInterval int    `json:"UpdateNotificationInterval"`
}

func ConfigFromJSON(config *ibmcloud.Config) (accountID string, err error) {
	configjson := new(ConfigJSON)
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	jsonFile, err := os.Open(usr.HomeDir + "/.bluemix/config.json")
	if err != nil {
		return "", err
	}

	defer jsonFile.Close() //nolint:errcheck
	byteValue, _ := ioutil.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, configjson)
	if err != nil {
		return "", err
	}

	config.Region = configjson.Region
	config.IAMAccessToken = configjson.IAMToken
	config.IAMRefreshToken = configjson.IAMRefreshToken
	config.SSLDisable = configjson.SSLDisabled
	config.Region = configjson.Region
	config.BluemixAPIKey = "fake"
	config.IBMID = "fake"
	config.IBMIDPassword = "fake"
	accountID = configjson.Account.GUID

	return accountID, nil
}

func getIAMAuthRepository(config *ibmcloud.Config) (*IAMAuthRepository, error) {
	if config.HTTPClient == nil {
		config.HTTPClient = http.NewHTTPClient(config)
	}
	return NewIAMAuthRepository(config, &rest.Client{
		DefaultHeader: gohttp.Header{
			"User-Agent": []string{http.UserAgent()},
		},
		HTTPClient: config.HTTPClient,
	})
}

func AuthenticateSSO(passcode string, config *ibmcloud.Config) error {
	config.IBMIDPassword = passcode
	config.IBMID = passcode
	config.Endpoint = nil
	iamauthrepo, err := getIAMAuthRepository(config)
	if err != nil {
		return nil
	}
	return iamauthrepo.AuthenticateSSO(passcode)
}

func AuthenticatePassword(username string, password string, config *ibmcloud.Config) error {
	config.IBMIDPassword = password
	config.IBMID = username
	config.Endpoint = nil
	iamauthrepo, err := getIAMAuthRepository(config)
	if err != nil {
		return nil
	}
	return iamauthrepo.AuthenticatePassword(username, password)
}

func AuthenticateAPIKey(apikey string, config *ibmcloud.Config) error {
	config.BluemixAPIKey = apikey
	config.Endpoint = nil
	iamauthrepo, err := getIAMAuthRepository(config)
	if err != nil {
		return nil
	}
	return iamauthrepo.AuthenticateAPIKey(apikey)
}

func RefreshTokenToLinkAccounts(account *Account, config *ibmcloud.Config) error {
	config.Endpoint = nil
	iamauthrepo, err := getIAMAuthRepository(config)
	if err != nil {
		return nil
	}
	return iamauthrepo.RefreshTokenToLinkAccounts(account)
}
