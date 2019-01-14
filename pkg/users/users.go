package users

import (
	"sort"

	"github.com/jenkins-x/jx/pkg/kube"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetUsers returns the pending users with the sorted order of names
func GetUsers(jxClient versioned.Interface, ns string) (map[string]*jenkinsv1.User, []string, error) {
	m := map[string]*jenkinsv1.User{}

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
func CreateUser(ns string, login string, name string, email string) *jenkinsv1.User {
	id := login
	if email != "" {
		id = kube.EmailToK8sID(email)
	}
	user := &jenkinsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      id,
			Namespace: ns,
		},
		Spec: jenkinsv1.UserDetails{
			Login: login,
			Name:  name,
			Email: email,
		},
	}
	return user
}

// AddAccountReference adds an account reference and label (created from gitProviderKey and id) to the user
func AddAccountReference(user *jenkinsv1.User, gitProviderKey string, id string) *jenkinsv1.User {
	if user.Spec.Accounts == nil {
		user.Spec.Accounts = make([]jenkinsv1.AccountReference, 0)
	}
	user.Spec.Accounts = append(user.Spec.Accounts, jenkinsv1.AccountReference{
		Provider: gitProviderKey,
		ID:       id,
	})
	if user.ObjectMeta.Labels == nil {
		user.ObjectMeta.Labels = make(map[string]string)
	}
	user.ObjectMeta.Labels[gitProviderKey] = id
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
