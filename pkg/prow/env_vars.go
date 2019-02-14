package prow

import (
	"fmt"
	"strings"

	"github.com/iancoleman/orderedmap"
)

// ParsePullRefs parses the Prow PULL_REFS env var formatted string and converts to a map of branch:sha
func ParsePullRefs(pullRefs string) (*orderedmap.OrderedMap, error) {
	kvs := strings.Split(pullRefs, ",")
	answer := orderedmap.New()
	for _, kv := range kvs {
		s := strings.Split(kv, ":")
		if len(s) != 2 {
			return nil, fmt.Errorf("incorrect format for branch:sha %s", kv)
		}
		answer.Set(s[0], s[1])
	}
	return answer, nil
}
