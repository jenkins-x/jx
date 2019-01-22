package bdd

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// CreateClusters contains an array of clusters
type CreateClusters struct {
	Clusters []*CreateCluster `yaml:"clusters,omitempty"`
}

// CreateCluster defines how to create a cluster
type CreateCluster struct {
	Name string   `yaml:"name,omitempty"`
	Args []string `yaml:"args,omitempty"`
}


// LoadBddClusters loads the cluster configuration from the given file
func LoadBddClusters(fileName string) (*CreateClusters, error) {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
	  return nil, errors.Wrapf(err, "failed to load file %s" , fileName)
	}
	answer := &CreateClusters{}
	err = yaml.Unmarshal(data, answer)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal YAML in file %s" , fileName)
	}
	return answer, nil
}