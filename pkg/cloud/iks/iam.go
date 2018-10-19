package iks

import (
	"encoding/base64"
	"fmt"

	bluemix "github.com/IBM-Cloud/bluemix-go"
	"github.com/IBM-Cloud/bluemix-go/bmxerror"
	"github.com/IBM-Cloud/bluemix-go/rest"
)

const (
	//ErrCodeInvalidToken  ...
	ErrCodeInvalidToken = "InvalidToken"
)

//IAMError ...
type IAMError struct {
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
	ErrorDetails string `json:"errorDetails"`
}

//Description ...
func (e IAMError) Description() string {
	if e.ErrorDetails != "" {
		return e.ErrorDetails
	}
	return e.ErrorMessage
}

//IAMTokenResponse ...
type IAMTokenResponse struct {
	AccessToken     string `json:"access_token"`
	RefreshToken    string `json:"refresh_token"`
	UAAAccessToken  string `json:"uaa_token"`
	UAARefreshToken string `json:"uaa_refresh_token"`
	TokenType       string `json:"token_type"`
}

//IAMAuthRepository ...
type IAMAuthRepository struct {
	config   *bluemix.Config
	client   *rest.Client
	endpoint string
}

//NewIAMAuthRepository ...
func NewIAMAuthRepository(config *bluemix.Config, client *rest.Client) (*IAMAuthRepository, error) {
	var endpoint string

	if config.TokenProviderEndpoint != nil {
		endpoint = *config.TokenProviderEndpoint
	} else {
		var err error
		endpoint, err = config.EndpointLocator.IAMEndpoint()
		if err != nil {
			return nil, err
		}
	}

	return &IAMAuthRepository{
		config:   config,
		client:   client,
		endpoint: endpoint,
	}, nil
}

//AuthenticatePassword ...
func (auth *IAMAuthRepository) AuthenticatePassword(username string, password string) error {
	data := make(map[string]string, 1)
	data["grant_type"] = "password"
	data["username"] = username
	data["password"] = password
	return auth.getToken(data)
}

//AuthenticateAPIKey ...
func (auth *IAMAuthRepository) AuthenticateAPIKey(apiKey string) error {
	data := make(map[string]string, 1)
	data["grant_type"] = "urn:ibm:params:oauth:grant-type:apikey"
	data["apikey"] = apiKey
	return auth.getToken(data)
}

//AuthenticateSSO ...
func (auth *IAMAuthRepository) AuthenticateSSO(passcode string) error {
	data := make(map[string]string, 1)
	data["grant_type"] = "urn:ibm:params:oauth:grant-type:passcode"
	data["passcode"] = passcode
	return auth.getToken(data)
}

//RefreshToken ...
func (auth *IAMAuthRepository) RefreshToken() (string, error) {
	data := make(map[string]string, 1)
	data["grant_type"] = "refresh_token"
	data["refresh_token"] = auth.config.IAMRefreshToken

	err := auth.getToken(data)
	if err != nil {
		return "", err
	}

	return auth.config.IAMAccessToken, nil
}

func (auth *IAMAuthRepository) getToken(data map[string]string) error {
	request := rest.PostRequest(auth.endpoint+"/oidc/token").
		Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("bx:bx"))).
		Field("response_type", "cloud_iam,uaa").
		Field("uaa_client_id", "cf").
		Field("uaa_client_secret", "")

	for k, v := range data {
		request.Field(k, v)
	}

	var tokens IAMTokenResponse
	var apiErr IAMError

	resp, err := auth.client.Do(request, &tokens, &apiErr)
	if err != nil {
		return err
	}

	if apiErr.ErrorCode != "" {
		if apiErr.ErrorCode == "BXNIM0407E" {
			return bmxerror.New(ErrCodeInvalidToken, apiErr.Description())
		}
		return bmxerror.NewRequestFailure(apiErr.ErrorCode, apiErr.Description(), resp.StatusCode)
	}

	auth.config.IAMAccessToken = fmt.Sprintf("%s %s", tokens.TokenType, tokens.AccessToken)
	auth.config.IAMRefreshToken = tokens.RefreshToken

	auth.config.UAAAccessToken = fmt.Sprintf("%s %s", tokens.TokenType, tokens.UAAAccessToken)
	auth.config.UAARefreshToken = tokens.UAARefreshToken

	return nil
}

func (auth *IAMAuthRepository) RefreshTokenToLinkAccounts(account *Account) error {
	data := make(map[string]string, 1)
	data["grant_type"] = "refresh_token"
	data["refresh_token"] = auth.config.IAMRefreshToken
	data["bss_account"] = account.GUID
	data["ims_account"] = account.IMSAccountID

	err := auth.getToken(data)
	if err != nil {
		return err
	}

	return nil
}
