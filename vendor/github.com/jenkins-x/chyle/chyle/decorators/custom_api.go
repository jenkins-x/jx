package decorators

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/antham/chyle/chyle/apih"
)

type customAPIConfig struct {
	CREDENTIALS struct {
		TOKEN string
	}
	ENDPOINT struct {
		URL string
	}
	KEYS map[string]struct {
		DESTKEY string
		FIELD   string
	}
}

// customAPI fetch data using a provided custom HTTP api
type customAPI struct {
	client http.Client
	config customAPIConfig
}

func (c customAPI) Decorate(commitMap *map[string]interface{}) (*map[string]interface{}, error) {
	var ID string

	switch v := (*commitMap)["customApiId"].(type) {
	case string:
		ID = v
	case int64:
		ID = fmt.Sprintf("%d", v)
	default:
		return commitMap, nil
	}

	req, err := http.NewRequest("GET", regexp.MustCompile(`{{\s*ID\s*}}`).ReplaceAllString(c.config.ENDPOINT.URL, ID), nil)

	apih.SetHeaders(req, map[string]string{
		"Authorization": "token " + c.config.CREDENTIALS.TOKEN,
		"Content-Type":  "application/json",
	})

	if err != nil {
		return commitMap, err
	}

	return jSONResponse{&c.client, req, c.config.KEYS}.Decorate(commitMap)
}

func newCustomAPI(config customAPIConfig) Decorater {
	return customAPI{http.Client{}, config}
}
