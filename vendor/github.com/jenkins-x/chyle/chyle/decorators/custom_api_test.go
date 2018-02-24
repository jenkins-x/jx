package decorators

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v0"
)

func TestCustomAPI(t *testing.T) {
	config := customAPIConfig{}
	config.CREDENTIALS.TOKEN = "d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f"
	config.ENDPOINT.URL = "http://test.com/api/issue/{{ID}}"
	config.KEYS = map[string]struct {
		DESTKEY string
		FIELD   string
	}{
		"KEY": {
			"authorEmail",
			"author.email",
		},
		"WHATEVER": {
			"whatever",
			"whatever",
		},
	}

	defer gock.Off()

	gock.New("http://test.com/api/issue/1").
		MatchHeader("Authorization", "token d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f").
		MatchHeader("Content-Type", "application/json").
		Reply(200).
		JSON(`{"id":"1","author":{"email":"test@test.com","name":"test"}}`)

	gock.New("http://test.com/api/issue/145d5926-2c7b-42c5-b0ff-41cd9b73c56c").
		MatchHeader("Authorization", "token d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f").
		MatchHeader("Content-Type", "application/json").
		Reply(200).
		JSON(`{"id":"145d5926-2c7b-42c5-b0ff-41cd9b73c56c","author":{"email":"test@test.com","name":"test"}}`)

	client := &http.Client{Transport: &http.Transport{}}
	gock.InterceptClient(client)

	c := customAPI{*client, config}

	// request with int id
	result, err := c.Decorate(&map[string]interface{}{"test": "test", "customApiId": int64(1)})

	expected := map[string]interface{}{
		"test":        "test",
		"customApiId": int64(1),
		"authorEmail": "test@test.com",
	}

	assert.NoError(t, err)
	assert.Equal(t, expected, *result)

	// request with string id
	result, err = c.Decorate(&map[string]interface{}{"test": "test", "customApiId": "145d5926-2c7b-42c5-b0ff-41cd9b73c56c"})

	expected = map[string]interface{}{
		"test":        "test",
		"customApiId": "145d5926-2c7b-42c5-b0ff-41cd9b73c56c",
		"authorEmail": "test@test.com",
	}

	assert.NoError(t, err)
	assert.Equal(t, expected, *result)
	assert.True(t, gock.IsDone(), "Must have no pending requests")
}

func TestCustomAPIWithUnvalidResponse(t *testing.T) {
	config := customAPIConfig{}
	config.CREDENTIALS.TOKEN = "d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f"
	config.ENDPOINT.URL = "http://test.com/api/issue/{{ID}}"
	config.KEYS = map[string]struct {
		DESTKEY string
		FIELD   string
	}{
		"WHATEVER": {
			"whatever",
			"whatever",
		},
	}

	defer gock.Off()

	gock.New("http://test.com/api/issue/5b23f37a-7404-49ce-812e-e7b3595ac721").
		MatchHeader("Authorization", "token d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f").
		MatchHeader("Content-Type", "application/json").
		Reply(200).
		BodyString("test")

	client := &http.Client{Transport: &http.Transport{}}
	gock.InterceptClient(client)

	c := customAPI{*client, config}

	result, err := c.Decorate(&map[string]interface{}{"test": "test", "customApiId": "5b23f37a-7404-49ce-812e-e7b3595ac721"})

	expected := map[string]interface{}{
		"test":        "test",
		"customApiId": "5b23f37a-7404-49ce-812e-e7b3595ac721",
	}

	assert.NoError(t, err)
	assert.Equal(t, expected, *result)
	assert.True(t, gock.IsDone(), "Must have no pending requests")
}

func TestCustomAPIWithNoCustomApiIdDefined(t *testing.T) {
	defer gock.Off()

	gock.New("http://test.com/api/issue/5b23f37a-7404-49ce-812e-e7b3595ac721").
		MatchHeader("Authorization", "token d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f").
		MatchHeader("Content-Type", "application/json").
		Reply(200).
		JSON(`{}`)

	client := &http.Client{Transport: &http.Transport{}}
	gock.InterceptClient(client)

	c := customAPI{*client, customAPIConfig{}}

	result, err := c.Decorate(&map[string]interface{}{"test": "test"})

	expected := map[string]interface{}{
		"test": "test",
	}

	assert.NoError(t, err)
	assert.Equal(t, expected, *result)
	assert.False(t, gock.IsDone(), "Must have one pending request")
}

func TestCustomAPIWithAnErrorStatusCode(t *testing.T) {
	config := customAPIConfig{}
	config.CREDENTIALS.TOKEN = "d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f"
	config.ENDPOINT.URL = "http://test.com/api/issue/{{ID}}"
	config.KEYS = map[string]struct {
		DESTKEY string
		FIELD   string
	}{
		"WHATEVER": {
			"whatever",
			"whatever",
		},
	}

	defer gock.Off()

	gock.New("http://test.com/api/issue/5b23f37a-7404-49ce-812e-e7b3595ac721").
		MatchHeader("Authorization", "token d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f").
		MatchHeader("Content-Type", "application/json").
		Reply(401).
		JSON(`{"error":"not found"}`)

	client := &http.Client{Transport: &http.Transport{}}
	gock.InterceptClient(client)

	c := customAPI{*client, config}

	result, err := c.Decorate(&map[string]interface{}{"test": "test", "customApiId": "5b23f37a-7404-49ce-812e-e7b3595ac721"})

	expected := map[string]interface{}{
		"test":        "test",
		"customApiId": "5b23f37a-7404-49ce-812e-e7b3595ac721",
	}

	assert.EqualError(t, err, `an error occurred when contacting remote api through http://test.com/api/issue/5b23f37a-7404-49ce-812e-e7b3595ac721, status code 401, body {"error":"not found"}`)
	assert.Equal(t, expected, *result)
	assert.True(t, gock.IsDone(), "Must have no pending requests")
}

func TestCustomAPIWhenEntryIsNotFound(t *testing.T) {
	config := customAPIConfig{}
	config.CREDENTIALS.TOKEN = "d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f"
	config.ENDPOINT.URL = "http://test.com/api/issue/{{ID}}"
	config.KEYS = map[string]struct {
		DESTKEY string
		FIELD   string
	}{
		"WHATEVER": {
			"whatever",
			"whatever",
		},
	}

	defer gock.Off()

	gock.New("http://test.com/api/issue/5b23f37a-7404-49ce-812e-e7b3595ac721").
		MatchHeader("Authorization", "token d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f").
		MatchHeader("Content-Type", "application/json").
		Reply(404).
		JSON(`{"error":"not found"}`)

	client := &http.Client{Transport: &http.Transport{}}
	gock.InterceptClient(client)

	c := customAPI{*client, config}

	result, err := c.Decorate(&map[string]interface{}{"test": "test", "customApiId": "5b23f37a-7404-49ce-812e-e7b3595ac721"})

	expected := map[string]interface{}{
		"test":        "test",
		"customApiId": "5b23f37a-7404-49ce-812e-e7b3595ac721",
	}

	assert.NoError(t, err)
	assert.Equal(t, expected, *result)
	assert.True(t, gock.IsDone(), "Must have no pending requests")
}
