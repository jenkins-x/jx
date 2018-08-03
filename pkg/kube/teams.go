package kube

import (
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetAdminNamespace tries to find the namespace which is annotated as the global admin namespace for the cluster
// or returns the current namespace
func GetAdminNamespace(kubeClient kubernetes.Interface, ns string) (string, error) {
	// TODO find the admin namespace via a label on the current dev namespace - or use current?
	return ns, nil
}

// GetPendingTeams returns the pending teams with the sorted order of names
func GetPendingTeams(jxClient versioned.Interface, ns string) (map[string]*v1.Team, []string, error) {
	m := map[string]*v1.Team{}

	names := []string{}
	teamList, err := jxClient.JenkinsV1().Teams(ns).List(metav1.ListOptions{})
	if err != nil {
		return m, names, err
	}
	for _, team := range teamList.Items {
		n := team.Name
		copy := team
		m[n] = &copy
		if n != "" {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return m, names, nil
}

// CreateTeam creates a new default Team
func CreateTeam(ns string, name string, members []string) *v1.Team {
	kind := v1.TeamKindTypeCD
	team := &v1.Team{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ToValidName(name),
			Namespace: ns,
		},
		Spec: v1.TeamSpec{
			Label:   strings.Title(name),
			Members: members,
			Kind:    kind,
		},
	}
	return team
}

// DeleteTeam deletes the team resource but does not uninstall the underlying namespaces
func DeleteTeam(jxClient versioned.Interface, ns string, teamName string) error {
	teamInterface := jxClient.JenkinsV1().Teams(ns)
	_, err := teamInterface.Get(teamName, metav1.GetOptions{})
	if err == nil {
		err = teamInterface.Delete(teamName, nil)
	}
	return err
}
