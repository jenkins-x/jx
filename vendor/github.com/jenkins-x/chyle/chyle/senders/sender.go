package senders

import (
	"github.com/antham/chyle/chyle/types"
)

// Sender defines where the changelog produced must be sent
type Sender interface {
	Send(changelog *types.Changelog) error
}

// Send forwards changelog to senders
func Send(senders *[]Sender, changelog *types.Changelog) error {
	for _, sender := range *senders {
		err := sender.Send(changelog)

		if err != nil {
			return err
		}
	}

	return nil
}

// Create builds senders from a config
func Create(features Features, senders Config) *[]Sender {
	results := []Sender{}

	if !features.ENABLED {
		return &results
	}

	if features.GITHUBRELEASE {
		results = append(results, newGithubRelease(senders.GITHUBRELEASE))
	}

	if features.CUSTOMAPI {
		results = append(results, newCustomAPI(senders.CUSTOMAPI))
	}

	if features.STDOUT {
		results = append(results, newStdout(senders.STDOUT))
	}

	return &results
}
