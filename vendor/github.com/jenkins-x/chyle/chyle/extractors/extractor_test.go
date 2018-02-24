package extractors

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/antham/chyle/chyle/types"
)

func TestExtract(t *testing.T) {
	extractors := []Extracter{
		regex{
			"id",
			"serviceId",
			regexp.MustCompile(`(\#\d+)`),
		},
		regex{
			"id",
			"booleanValue",
			regexp.MustCompile(`(true|false)`),
		},
		regex{
			"id",
			"intValue",
			regexp.MustCompile(` (\d+)`),
		},
		regex{
			"id",
			"floatValue",
			regexp.MustCompile(`(\d+\.\d+)`),
		},
		regex{
			"secondIdentifier",
			"secondServiceId",
			regexp.MustCompile(`(#\d+)`),
		},
	}

	commitMaps := []map[string]interface{}{
		{
			"id":               "Whatever #30 whatever true 12345 whatever 12345.12",
			"secondIdentifier": "test #12345",
		},
		{
			"id":               "Whatever #40 whatever false whatever 78910 whatever 78910.12",
			"secondIdentifier": "test #45678",
		},
		{
			"id": "Whatever whatever whatever",
		},
	}

	results := Extract(&extractors, &commitMaps)

	expected := types.Changelog{
		Datas: []map[string]interface{}{
			{
				"id":               "Whatever #30 whatever true 12345 whatever 12345.12",
				"secondIdentifier": "test #12345",
				"serviceId":        "#30",
				"secondServiceId":  "#12345",
				"booleanValue":     true,
				"intValue":         int64(12345),
				"floatValue":       12345.12,
			},
			{
				"id":               "Whatever #40 whatever false whatever 78910 whatever 78910.12",
				"secondIdentifier": "test #45678",
				"serviceId":        "#40",
				"secondServiceId":  "#45678",
				"booleanValue":     false,
				"intValue":         int64(78910),
				"floatValue":       78910.12,
			},
			{
				"id":           "Whatever whatever whatever",
				"serviceId":    "",
				"booleanValue": "",
				"intValue":     "",
				"floatValue":   "",
			},
		},
		Metadatas: map[string]interface{}{},
	}

	assert.Equal(t, expected, *results)
}

func TestCreate(t *testing.T) {
	extractors := Config{
		"ID": {
			"id",
			"test",
			regexp.MustCompile(".*"),
		},
		"AUTHORNAME": {
			"authorName",
			"test2",
			regexp.MustCompile(".*"),
		},
	}

	e := Create(Features{ENABLED: true}, extractors)

	assert.Len(t, *e, 2)

	expected := map[string]map[string]string{
		"id": {
			"index":      "id",
			"identifier": "test",
			"regexp":     ".*",
		},
		"authorName": {
			"index":      "authorName",
			"identifier": "test2",
			"regexp":     ".*",
		},
	}

	for i := 0; i < 2; i++ {
		index := (*e)[0].(regex).index

		v, ok := expected[index]

		if !ok {
			assert.Fail(t, "Index must exists in expected")
		}

		assert.Equal(t, (*e)[0].(regex).index, v["index"])
		assert.Equal(t, (*e)[0].(regex).identifier, v["identifier"])
		assert.Equal(t, (*e)[0].(regex).re, regexp.MustCompile(v["regexp"]))
	}
}

func TestCreateWithFeatureDisabled(t *testing.T) {
	e := Create(Features{}, Config{
		"ID": {
			"id",
			"test",
			regexp.MustCompile(".*"),
		},
	})

	assert.Len(t, *e, 0)
}
