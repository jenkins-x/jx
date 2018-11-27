package helm_test

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestCombineMapTrees(t *testing.T) {
	t.Parallel()

	assertCombineMapTrees(t,
		map[string]interface{}{
			"foo": "foovalue",
			"bar": "barvalue",
		},
		map[string]interface{}{
			"foo": "foovalue",
		},
		map[string]interface{}{
			"bar": "barvalue",
		},
	)

	assertCombineMapTrees(t,
		map[string]interface{}{
			"child": map[string]interface{}{
				"foo": "foovalue",
				"bar": "barvalue",
			},
			"m1": map[string]interface{}{
				"thingy": "thingyvalue",
			},
			"m2": map[string]interface{}{
				"another": "anothervalue",
			},
		},
		map[string]interface{}{
			"child": map[string]interface{}{
				"foo": "foovalue",
			},
			"m1": map[string]interface{}{
				"thingy": "thingyvalue",
			},
		},
		map[string]interface{}{
			"child": map[string]interface{}{
				"bar": "barvalue",
			},
			"m2": map[string]interface{}{
				"another": "anothervalue",
			},
		},
	)
}

func assertCombineMapTrees(t *testing.T, expected map[string]interface{}, destination map[string]interface{}, input map[string]interface{}) {
	actual := map[string]interface{}{}
	for k, v := range destination {
		actual[k] = v
	}

	util.CombineMapTrees(actual, input)

	assert.Equal(t, actual, expected, "when combine map trees", mapToString(destination), mapToString(input))
}

func mapToString(m map[string]interface{}) string {
	return fmt.Sprintf("%#v", m)
}
