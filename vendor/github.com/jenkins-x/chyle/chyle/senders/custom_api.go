package senders

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/antham/chyle/chyle/apih"
	"github.com/antham/chyle/chyle/errh"
	"github.com/antham/chyle/chyle/types"
)

type customAPIConfig struct {
	CREDENTIALS struct {
		TOKEN string
	}
	ENDPOINT struct {
		URL string
	}
}

// customAPI fetch data using a provided custom HTTP api
type customAPI struct {
	client *http.Client
	config customAPIConfig
}

func (c customAPI) createRequest(changelog *types.Changelog) (*http.Request, error) {
	payload, err := json.Marshal(changelog)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.config.ENDPOINT.URL, bytes.NewBuffer(payload))

	if err != nil {
		return nil, err
	}

	apih.SetHeaders(req, map[string]string{
		"Authorization": "token " + c.config.CREDENTIALS.TOKEN,
		"Content-Type":  "application/json",
	})

	return req, nil
}

func (c customAPI) Send(changelog *types.Changelog) error {
	errMsg := "can't call custom api to send release"

	req, err := c.createRequest(changelog)

	if err != nil {
		return errh.AddCustomMessageToError(errMsg, err)
	}

	_, _, err = apih.SendRequest(c.client, req)

	return errh.AddCustomMessageToError(errMsg, err)
}

func newCustomAPI(config customAPIConfig) Sender {
	return customAPI{&http.Client{}, config}
}
