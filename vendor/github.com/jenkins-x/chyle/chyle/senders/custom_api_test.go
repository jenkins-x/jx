package senders

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v0"

	"github.com/antham/chyle/chyle/types"
)

func TestCustomAPI(t *testing.T) {
	config := customAPIConfig{}
	config.ENDPOINT.URL = "https://test.com/releases"
	config.CREDENTIALS.TOKEN = "d41d8cd98f00b204e9800998ecf8427e"

	defer gock.Off()

	gock.New("https://test.com").
		Post("/releases").
		MatchHeader("Authorization", "token d41d8cd98f00b204e9800998ecf8427e").
		MatchHeader("Content-Type", "application/json").
		JSON(map[string]string{"test": "Hello world !"}).
		Reply(201)

	client := &http.Client{Transport: &http.Transport{}}
	gock.InterceptClient(client)

	s := newCustomAPI(config).(customAPI)
	s.client = client

	c := types.Changelog{
		Datas:     []map[string]interface{}{},
		Metadatas: map[string]interface{}{},
	}

	c.Datas = append(c.Datas, map[string]interface{}{"test": "Hello world !"})

	err := s.Send(&c)

	assert.NoError(t, err)
	assert.True(t, gock.IsDone(), "Must have no pending requests")
}

func TestCustomAPIWithWrongCredentials(t *testing.T) {
	config := customAPIConfig{}
	config.ENDPOINT.URL = "https://test.com/releases"
	config.CREDENTIALS.TOKEN = "d41d8cd98f00b204e9800998ecf8427e"

	defer gock.Off()

	gock.New("https://test.com").
		Post("/releases").
		MatchHeader("Authorization", "token d41d8cd98f00b204e9800998ecf8427e").
		MatchHeader("Content-Type", "application/json").
		ReplyError(fmt.Errorf(`{"error":"You don't have correct credentials"}`))

	client := &http.Client{Transport: &http.Transport{}}
	gock.InterceptClient(client)

	s := newCustomAPI(config).(customAPI)
	s.client = client

	c := types.Changelog{
		Datas:     []map[string]interface{}{},
		Metadatas: map[string]interface{}{},
	}

	c.Datas = append(c.Datas, map[string]interface{}{"test": "Hello world !"})

	err := s.Send(&c)

	assert.EqualError(t, err, `can't call custom api to send release : Post https://test.com/releases: {"error":"You don't have correct credentials"}`)
	assert.True(t, gock.IsDone(), "Must have no pending requests")
}

func TestCustomAPIWithWrongURL(t *testing.T) {
	config := customAPIConfig{}
	config.ENDPOINT.URL = ":test"
	config.CREDENTIALS.TOKEN = "d41d8cd98f00b204e9800998ecf8427e"

	client := &http.Client{Transport: &http.Transport{}}

	s := newCustomAPI(config).(customAPI)
	s.client = client

	c := types.Changelog{
		Datas:     []map[string]interface{}{},
		Metadatas: map[string]interface{}{},
	}

	c.Datas = append(c.Datas, map[string]interface{}{"test": "Hello world !"})

	err := s.Send(&c)

	assert.EqualError(t, err, `can't call custom api to send release : parse :test: missing protocol scheme`)
	assert.True(t, gock.IsDone(), "Must have no pending requests")
}
