package bdd

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// BddClusters contains an array of clusters
type BddClusters struct {
	Clusters []*BddCluster `yaml:"clusters,omitempty"`
}

// BddCluster defines how to create a cluster
type BddCluster struct {
	Name string   `yaml:"name,omitempty"`
	Args []string `yaml:"args,omitempty"`
}


// LoadBddClusters loads the cluster configuration from the given file
func LoadBddClusters(fileName string) (*BddClusters, error) {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
	  return nil, errors.Wrapf(err, "failed to load file %s" , fileName)
	}
	answer := &BddClusters{}
	err = yaml.Unmarshal(data, answer)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal YAML in file %s" , fileName)
	}
	return answer, nil
}