package decorators

import (
	"github.com/antham/chyle/chyle/types"
)

// Decorater extends data from commit hashmap with data picked from third part apps
type Decorater interface {
	Decorate(*map[string]interface{}) (*map[string]interface{}, error)
}

// Decorate process all defined decorator and apply them against datas
func Decorate(decorators *map[string][]Decorater, changelog *types.Changelog) (*types.Changelog, error) {
	var err error

	datas := []map[string]interface{}{}

	for _, d := range changelog.Datas {
		result := &d

		for _, decorator := range (*decorators)["datas"] {
			result, err = decorator.Decorate(&d)

			if err != nil {
				return nil, err
			}
		}

		datas = append(datas, *result)
	}

	changelog.Datas = datas

	metadatas := changelog.Metadatas

	for _, decorator := range (*decorators)["metadatas"] {
		m, err := decorator.Decorate(&metadatas)

		if err != nil {
			return nil, err
		}

		metadatas = *m
	}

	changelog.Metadatas = metadatas

	return changelog, nil
}

// Create builds decorators from a config
func Create(features Features, decorators Config) *map[string][]Decorater {
	results := map[string][]Decorater{"metadatas": {}, "datas": {}}

	if !features.ENABLED {
		return &results
	}

	if features.CUSTOMAPI {
		results["datas"] = append(results["datas"], newCustomAPI(decorators.CUSTOMAPI))
	}

	if features.JIRAISSUE {
		results["datas"] = append(results["datas"], newJiraIssue(decorators.JIRAISSUE))
	}

	if features.GITHUBISSUE {
		results["datas"] = append(results["datas"], newGithubIssue(decorators.GITHUBISSUE))
	}

	if features.SHELL {
		results["datas"] = append(results["datas"], newShell(decorators.SHELL)...)
	}

	if features.ENV {
		results["metadatas"] = append(results["metadatas"], newEnvs(decorators.ENV)...)
	}

	return &results
}
