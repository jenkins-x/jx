package matchers

import (
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

// Matcher describes a way of applying a matcher against a commit
type Matcher interface {
	Match(*object.Commit) bool
}

// Filter filters commits that don't fit any matchers
func Filter(matchers *[]Matcher, commits *[]object.Commit) *[]map[string]interface{} {
	results := []object.Commit{}

	for _, commit := range *commits {
		add := true
		for _, matcher := range *matchers {
			if !matcher.Match(&commit) {
				add = false
			}
		}

		if add {
			results = append(results, commit)
		}
	}

	return transformCommitsToMap(&results)
}

// transformCommitsToMap extract useful commits data in hash map table
func transformCommitsToMap(commits *[]object.Commit) *[]map[string]interface{} {
	var commitMap map[string]interface{}
	commitMaps := []map[string]interface{}{}

	for _, c := range *commits {
		commitMap = map[string]interface{}{
			"id":             c.ID().String(),
			"authorName":     c.Author.Name,
			"authorEmail":    c.Author.Email,
			"authorDate":     c.Author.When.String(),
			"committerName":  c.Committer.Name,
			"committerEmail": c.Committer.Email,
			"committerDate":  c.Committer.When.String(),
			"message":        removePGPKey(c.Message),
			"type":           solveType(&c),
		}

		commitMaps = append(commitMaps, commitMap)
	}

	return &commitMaps
}

// Create builds matchers from a config
func Create(features Features, matchers Config) *[]Matcher {
	results := []Matcher{}

	if !features.ENABLED {
		return &results
	}

	if features.AUTHOR {
		results = append(results, newAuthor(matchers.AUTHOR))
	}

	if features.COMMITTER {
		results = append(results, newCommitter(matchers.COMMITTER))
	}

	if features.MESSAGE {
		results = append(results, newMessage(matchers.MESSAGE))
	}

	if features.TYPE {
		results = append(results, newType(matchers.TYPE))
	}

	return &results
}
