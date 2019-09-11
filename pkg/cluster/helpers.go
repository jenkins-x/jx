package cluster

import (
	"github.com/jenkins-x/jx/pkg/kube/naming"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// NewLabelValue returns a cluster safe unique label we can use for locking
func NewLabelValue() (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return naming.ToValidName(id.String()), nil
}

// LockCluster tries to use the given label and value to lock the cluster.
// Return nil if there are no clusters available
func LockCluster(client Client, lockLabel string, labelValue string, filterLabels map[string]string) (*Cluster, error) {
	clusters, err := client.ListFilter(filterLabels)
	if err != nil {
		return nil, err
	}

	for _, c := range clusters {
		if c.Labels == nil || c.Labels[lockLabel] == "" {
			// lets try to update label
			err = client.LabelCluster(c.Name, map[string]string{
				lockLabel: labelValue,
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to label cluster %s with label %s = %s", c.Name, lockLabel, labelValue)
			}

			// now lets requery to verify the label got applied
			copy, err := client.Get(c.Name)
			if err != nil {
				return nil, err
			}
			if copy != nil && copy.Labels != nil && copy.Labels[lockLabel] == labelValue {
				return copy, nil
			}
			if copy == nil {
				log.Logger().Infof("cluster %s no longer exists", c.Name)
			} else {
				log.Logger().Infof("could not label cluster %s with %s = %s it has labels %#v so could be labelled by another process",
					c.Name, lockLabel, labelValue, copy.Labels)
			}
		}
	}
	return nil, nil
}

// GetCluster gets a cluster by listing the clusters
func GetCluster(client Client, name string) (*Cluster, error) {
	clusters, err := client.List()
	if err != nil {
		return nil, err
	}

	for _, c := range clusters {
		if c.Name == name {
			return c, nil
		}
	}
	return nil, nil
}

// ListFilter lists the clusters with a filter
func ListFilter(client Client, labels map[string]string) ([]*Cluster, error) {
	answer := []*Cluster{}
	clusters, err := client.List()
	if err != nil {
		return answer, err
	}
	for _, c := range clusters {
		if LabelsMatch(c.Labels, labels) {
			answer = append(answer, c)
		}
	}
	return answer, nil
}

// LabelsMatch returns true if the filter labels are contained in the label map
func LabelsMatch(labels map[string]string, filter map[string]string) bool {
	for k, v := range filter {
		if labels == nil || labels[k] != v {
			return false
		}
	}
	return true
}
