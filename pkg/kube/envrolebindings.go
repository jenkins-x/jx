package kube

import (
	v1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/v2/pkg/util"
)

// EnvironmentMatches returns true if the environment matches the given filter
func EnvironmentMatches(env *v1.Environment, filter *v1.EnvironmentFilter) bool {
	kind := filter.Kind
	if string(kind) != "" {
		if env.Spec.Kind != kind {
			return false
		}
	}
	includes := filter.Includes
	excludes := filter.Excludes
	return util.StringMatchesAny(env.Name, includes, excludes)
}

// EnvironmentMatchesAny returns true if the list of filters is empty or one of the filters matches
// the given environment
func EnvironmentMatchesAny(env *v1.Environment, filters []v1.EnvironmentFilter) bool {
	for _, f := range filters {
		filter := f
		if EnvironmentMatches(env, &filter) {
			return true
		}
	}
	return len(filters) == 0
}
