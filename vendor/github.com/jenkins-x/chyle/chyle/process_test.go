package chyle

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/antham/chyle/chyle/config"
	"github.com/antham/chyle/chyle/decorators"
	"github.com/antham/chyle/chyle/extractors"
	"github.com/antham/chyle/chyle/matchers"
	"github.com/antham/chyle/chyle/senders"
	"github.com/antham/chyle/chyle/types"

	"gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/stretchr/testify/assert"
)

func TestBuildProcessWithAnEmptyConfig(t *testing.T) {
	conf := config.CHYLE{}

	p := newProcess(&conf)

	expected := process{
		&[]matchers.Matcher{},
		&[]extractors.Extracter{},
		&map[string][]decorators.Decorater{"metadatas": {}, "datas": {}},
		&[]senders.Sender{},
	}

	assert.EqualValues(t, expected, *p)
}

func TestBuildProcessWithAFullConfig(t *testing.T) {
	conf := config.CHYLE{}

	conf.FEATURES.MATCHERS.ENABLED = true
	conf.FEATURES.MATCHERS.TYPE = true
	conf.MATCHERS = matchers.Config{TYPE: "merge"}

	conf.FEATURES.EXTRACTORS.ENABLED = true
	conf.EXTRACTORS = map[string]struct {
		ORIGKEY string
		DESTKEY string
		REG     *regexp.Regexp
	}{
		"TEST": {
			"TEST",
			"test",
			regexp.MustCompile(".*"),
		},
	}

	conf.FEATURES.DECORATORS.ENABLED = true
	conf.FEATURES.DECORATORS.ENABLED = true
	conf.DECORATORS.ENV = map[string]struct {
		DESTKEY string
		VARNAME string
	}{
		"TEST": {
			"test",
			"TEST",
		},
	}

	conf.FEATURES.SENDERS.ENABLED = true
	conf.FEATURES.SENDERS.STDOUT = true
	conf.SENDERS.STDOUT.FORMAT = "json"

	p := newProcess(&conf)

	assert.Len(t, *(p.matchers), 1)
	assert.Len(t, *(p.extractors), 1)
	assert.Len(t, *(p.decorators), 2)
	assert.Len(t, *(p.senders), 1)
}

type mockDecorator struct {
}

func (m mockDecorator) Decorate(*map[string]interface{}) (*map[string]interface{}, error) {
	return &map[string]interface{}{}, fmt.Errorf("An error occured from mock decorator")
}

type mockSender struct {
}

func (m mockSender) Send(changelog *types.Changelog) error {
	return fmt.Errorf("An error occured from mock sender")
}

func (m mockSender) Decorate(*map[string]interface{}) (*map[string]interface{}, error) {
	return &map[string]interface{}{}, fmt.Errorf("An error occured from mock decorator")
}

func TestBuildProcessWithErrorsFromDecorator(t *testing.T) {
	p := process{
		&[]matchers.Matcher{},
		&[]extractors.Extracter{},
		&map[string][]decorators.Decorater{"metadatas": {}, "datas": {mockDecorator{}}},
		&[]senders.Sender{},
	}

	err := proceed(&p, &[]object.Commit{{}})

	assert.Error(t, err)
	assert.EqualError(t, err, "An error occured from mock decorator")
}

func TestBuildProcessWithErrorsFromSender(t *testing.T) {
	p := process{
		&[]matchers.Matcher{},
		&[]extractors.Extracter{},
		&map[string][]decorators.Decorater{"metadatas": {}, "datas": {}},
		&[]senders.Sender{mockSender{}},
	}

	err := proceed(&p, &[]object.Commit{{}})

	assert.Error(t, err)
	assert.EqualError(t, err, "An error occured from mock sender")
}
