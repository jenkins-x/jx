package decorators

import (
	"os"
)

type envConfig map[string]struct {
	DESTKEY string
	VARNAME string
}

// env dumps an environment variable into metadatas
type env struct {
	varName string
	destKey string
}

func (e env) Decorate(metadatas *map[string]interface{}) (*map[string]interface{}, error) {
	(*metadatas)[e.destKey] = os.Getenv(e.varName)

	return metadatas, nil
}

func newEnvs(configs envConfig) []Decorater {
	results := []Decorater{}

	for _, config := range configs {
		results = append(results, env{
			config.VARNAME,
			config.DESTKEY,
		})
	}

	return results
}
