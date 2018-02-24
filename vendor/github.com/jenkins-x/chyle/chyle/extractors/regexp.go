package extractors

import (
	"regexp"

	"github.com/antham/chyle/chyle/convh"
)

// regex uses a regexp to extract data
type regex struct {
	index      string
	identifier string
	re         *regexp.Regexp
}

func (r regex) Extract(commitMap *map[string]interface{}) *map[string]interface{} {
	var mapValue interface{}
	var ok bool

	if mapValue, ok = (*commitMap)[r.index]; !ok {
		return commitMap
	}

	var value string

	value, ok = mapValue.(string)

	if !ok {
		return commitMap
	}

	var result string

	results := r.re.FindStringSubmatch(value)

	if len(results) > 1 {
		result = results[1]
	}

	(*commitMap)[r.identifier] = convh.GuessPrimitiveType(result)

	return commitMap
}
