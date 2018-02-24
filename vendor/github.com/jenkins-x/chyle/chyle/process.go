package chyle

import (
	"github.com/antham/chyle/chyle/config"
	"github.com/antham/chyle/chyle/decorators"
	"github.com/antham/chyle/chyle/extractors"
	"github.com/antham/chyle/chyle/matchers"
	"github.com/antham/chyle/chyle/senders"

	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

// process represents all steps executed
// when creating a changelog
type process struct {
	matchers   *[]matchers.Matcher
	extractors *[]extractors.Extracter
	decorators *map[string][]decorators.Decorater
	senders    *[]senders.Sender
}

// newProcess creates process entity from defined configuration
func newProcess(conf *config.CHYLE) *process {
	return &process{
		matchers.Create(conf.FEATURES.MATCHERS, conf.MATCHERS),
		extractors.Create(conf.FEATURES.EXTRACTORS, conf.EXTRACTORS),
		decorators.Create(conf.FEATURES.DECORATORS, conf.DECORATORS),
		senders.Create(conf.FEATURES.SENDERS, conf.SENDERS),
	}
}

// proceed extracts datas from a set of commits
func proceed(process *process, commits *[]object.Commit) error {
	changelog, err := decorators.Decorate(process.decorators,
		extractors.Extract(process.extractors,
			matchers.Filter(process.matchers, commits)))

	if err != nil {
		return err
	}

	return senders.Send(process.senders, changelog)
}
