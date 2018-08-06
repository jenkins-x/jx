package kube

import (
	"sort"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetUsers returns the pending users with the sorted order of names
func GetUsers(jxClient versioned.Interface, ns string) (map[string]*v1.User, []string, error) {
	m := map[string]*v1.User{}

	names := []string{}
	userList, err := jxClient.JenkinsV1().Users(ns).List(metav1.ListOptions{})
	if err != nil {
		return m, names, err
	}
	for _, user := range userList.Items {
		n := user.Name
		copy := user
		m[n] = &copy
		if n != "" {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return m, names, nil
}

// CreateUser creates a new default User
func CreateUser(ns string, login string, name string, email string) *v1.User {
	user := &v1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ToValidName(login),
			Namespace: ns,
		},
		Spec: v1.UserDetails{
			Login: login,
			Name:  name,
			Email: email,
		},
	}
	return user
}

// DeleteUser deletes the user resource but does not uninstall the underlying namespaces
func DeleteUser(jxClient versioned.Interface, ns string, userName string) error {
	userInterface := jxClient.JenkinsV1().Users(ns)
	_, err := userInterface.Get(userName, metav1.GetOptions{})
	if err == nil {
		err = userInterface.Delete(userName, nil)
	}
	return err
}
