package senders

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/antham/chyle/chyle/apih"
	"github.com/antham/chyle/chyle/errh"
	"github.com/antham/chyle/chyle/tmplh"
	"github.com/antham/chyle/chyle/types"
)

type githubReleaseConfig struct {
	REPOSITORY struct {
		NAME string
	}
	CREDENTIALS struct {
		OAUTHTOKEN string
		OWNER      string
	}
	RELEASE struct {
		DRAFT           bool
		UPDATE          bool
		PRERELEASE      bool
		NAME            string
		TAGNAME         string
		TARGETCOMMITISH string
		TEMPLATE        string
	}
}

// codebeat:disable[TOO_MANY_IVARS]

// githubReleasePayload follows https://developer.github.com/v3/repos/releases/#create-a-release
type githubReleasePayload struct {
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish,omitempty"`
	Name            string `json:"name,omitempty"`
	Body            string `json:"body,omitempty"`
	Draft           bool   `json:"draft,omitempty"`
	PreRelease      bool   `json:"prerelease,omitempty"`
}

// codebeat:enable[TOO_MANY_IVARS]

func newGithubRelease(config githubReleaseConfig) Sender {
	return githubRelease{&http.Client{}, config}
}

// githubRelease fetch data using jira issue api
type githubRelease struct {
	client *http.Client
	config githubReleaseConfig
}

// buildBody creates a request body from changelog
func (g githubRelease) buildBody(changelog *types.Changelog) ([]byte, error) {
	body, err := tmplh.Build("github-release-template", g.config.RELEASE.TEMPLATE, changelog)

	if err != nil {
		return []byte{}, err
	}

	r := githubReleasePayload{
		g.config.RELEASE.TAGNAME,
		g.config.RELEASE.TARGETCOMMITISH,
		g.config.RELEASE.NAME,
		body,
		g.config.RELEASE.DRAFT,
		g.config.RELEASE.PRERELEASE,
	}

	return json.Marshal(r)
}

func (g githubRelease) createRelease(body []byte) error {
	URL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", g.config.CREDENTIALS.OWNER, g.config.REPOSITORY.NAME)

	req, err := http.NewRequest("POST", URL, bytes.NewBuffer(body))

	if err != nil {
		return err
	}

	apih.SetHeaders(req, map[string]string{
		"Authorization": "token " + g.config.CREDENTIALS.OAUTHTOKEN,
		"Content-Type":  "application/json",
		"Accept":        "application/vnd.github.v3+json",
	})

	_, _, err = apih.SendRequest(g.client, req)

	return errh.AddCustomMessageToError("can't create github release", err)
}

// getReleaseID retrieves github release ID from a given tag name
func (g githubRelease) getReleaseID() (int, error) {
	type s struct {
		ID int `json:"id"`
	}

	release := s{}

	errMsg := fmt.Sprintf("can't retrieve github release %s", g.config.RELEASE.TAGNAME)
	URL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", g.config.CREDENTIALS.OWNER, g.config.REPOSITORY.NAME, g.config.RELEASE.TAGNAME)

	req, err := http.NewRequest("GET", URL, nil)

	if err != nil {
		return 0, err
	}

	apih.SetHeaders(req, map[string]string{
		"Authorization": "token " + g.config.CREDENTIALS.OAUTHTOKEN,
		"Content-Type":  "application/json",
		"Accept":        "application/vnd.github.v3+json",
	})

	_, body, err := apih.SendRequest(g.client, req)

	if err != nil {
		return 0, errh.AddCustomMessageToError(errMsg, err)
	}

	err = json.Unmarshal(body, &release)

	if err != nil {
		return 0, errh.AddCustomMessageToError(errMsg, err)
	}

	return release.ID, nil
}

// updateRelease updates an existing release from a tag name
func (g githubRelease) updateRelease(body []byte) error {
	ID, err := g.getReleaseID()

	if err != nil {
		return err
	}

	URL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/%d", g.config.CREDENTIALS.OWNER, g.config.REPOSITORY.NAME, ID)

	req, err := http.NewRequest("PATCH", URL, bytes.NewBuffer(body))

	if err != nil {
		return err
	}

	apih.SetHeaders(req, map[string]string{
		"Authorization": "token " + g.config.CREDENTIALS.OAUTHTOKEN,
		"Content-Type":  "application/json",
		"Accept":        "application/vnd.github.v3+json",
	})

	_, _, err = apih.SendRequest(g.client, req)

	return errh.AddCustomMessageToError(fmt.Sprintf("can't update github release %s", g.config.RELEASE.TAGNAME), err)
}

func (g githubRelease) Send(changelog *types.Changelog) error {
	body, err := g.buildBody(changelog)

	if err != nil {
		return err
	}

	if g.config.RELEASE.UPDATE {
		return g.updateRelease(body)
	}

	return g.createRelease(body)
}
