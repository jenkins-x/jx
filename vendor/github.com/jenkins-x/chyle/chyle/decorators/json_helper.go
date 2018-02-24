package decorators

import (
	"bytes"
	"net/http"

	"github.com/tidwall/gjson"

	"github.com/antham/chyle/chyle/apih"
)

// jSONResponse extracts datas from a JSON api using defined keys
// and add it to final commitMap data structure
type jSONResponse struct {
	client  *http.Client
	request *http.Request
	pairs   map[string]struct {
		DESTKEY string
		FIELD   string
	}
}

func (j jSONResponse) Decorate(commitMap *map[string]interface{}) (*map[string]interface{}, error) {
	statusCode, body, err := apih.SendRequest(j.client, j.request)

	if statusCode == 404 {
		return commitMap, nil
	}

	if err != nil {
		return commitMap, err
	}

	buf := bytes.NewBuffer(body)

	for _, pair := range j.pairs {
		if gjson.Get(buf.String(), pair.FIELD).Exists() {
			(*commitMap)[pair.DESTKEY] = gjson.Get(buf.String(), pair.FIELD).Value()
		}
	}

	return commitMap, nil
}
