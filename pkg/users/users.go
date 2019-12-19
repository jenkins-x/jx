package users

import (
	"errors"
	"fmt"
	"sort"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/jenkins-x/jx/pkg/gits"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/kube/naming"
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
		n := user.Spec.Login
		if n == "" {
			n = user.Name
		}
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
	id := naming.ToValidName(login)
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
	id := naming.ToValidName(userName)
	userInterface := jxClient.JenkinsV1().Users(ns)
	_, err := userInterface.Get(id, metav1.GetOptions{})
	if err == nil {
		err = userInterface.Delete(id, nil)
	}
	return err
}

// ResolveJXUser checks for a Jenkins X users in the User CRD matching the specified git user.
// If a match is found it is returned. If no match is found or an error occurs nil is returned together with
// the potential error.
func ResolveJXUser(user *gits.GitUser, providerKey string, jxClient versioned.Interface, namespace string) (*jenkinsv1.User, error) {
	if user == nil {
		return nil, errors.New("a git user needs to be specified")
	}

	// need to convert the git login to a valid Kubernetes label. This label is not unique and multiple users
	// can match the same label, hence we need to match the returns Jenkins X users.
	id := naming.ToValidValue(user.Login)

	// First try to find by label - this is much faster as it uses an index
	if id != "" {
		labelSelector := fmt.Sprintf("%s=%s", providerKey, id)
		jxUsers, err := jxClient.JenkinsV1().Users(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			return nil, err
		}
		matchedUser, err := matchGitUser(user, jxUsers, providerKey)
		if err != nil {
			return nil, err
		}

		// return match found within labeled Jenkins X users
		if matchedUser != nil {
			return matchedUser, nil
		}
	}

	// Next try without the label - this might occur if someone manually updated the list
	jxUsers, err := jxClient.JenkinsV1().Users(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	matchedUser, err := matchGitUser(user, jxUsers, providerKey)
	if err != nil {
		return nil, err
	}

	if matchedUser != nil {
		matchedUser, err = addLabelsToJXUser(matchedUser, providerKey, id, namespace, jxClient)
		if err != nil {
			return nil, err
		}
	}
	return matchedUser, nil
}

func addLabelsToJXUser(user *jenkinsv1.User, providerKey string, id string, namespace string, jxClient versioned.Interface) (*jenkinsv1.User, error) {
	if user.Labels == nil {
		user.Labels = make(map[string]string)
	}

	user.Labels[providerKey] = id
	user, err := jxClient.JenkinsV1().Users(namespace).Update(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// createJXUser creates the specified user after creating a unique name for it.
func createJXUser(jxUser *jenkinsv1.User, jxClient versioned.Interface, namespace string) (*jenkinsv1.User, error) {
	if jxUser == nil {
		return nil, errors.New("the user to create can not be nil")
	}

	name := naming.ToValidName(jxUser.Name)
	// Check if the user id is available, if not append "-<n>" where <n> is some integer
	for i := 0; true; i++ {
		_, err := jxClient.JenkinsV1().Users(namespace).Get(name, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			break
		}
		name = fmt.Sprintf("%s-%d", name, i)
	}
	jxUser.Name = name
	return jxClient.JenkinsV1().Users(namespace).Create(jxUser)
}

// matchGitUser tries to match a git user against a list of Jenkins X users.
// If there is a unique match the match is returned. If more than one Jenkins X user match the git user an error is returned.
// If not match is found no user and no error is returned.
func matchGitUser(gitUser *gits.GitUser, users *jenkinsv1.UserList, providerKey string) (*jenkinsv1.User, error) {
	possibles := make([]jenkinsv1.User, 0)
	for _, jxUser := range users.Items {
		for _, a := range jxUser.Spec.Accounts {
			if a.Provider == providerKey {
				// email and provider match - add to possible!
				if gitUser.Email != "" && jxUser.Spec.Email == gitUser.Email {
					possibles = append(possibles, jxUser)
					continue
				}

				// login and provider match - add to possibles!
				if a.ID == gitUser.Login {
					possibles = append(possibles, jxUser)
					continue
				}
			}
		}
	}

	switch len(possibles) {
	case 0:
		return nil, nil
	case 1:
		return &possibles[0], nil
	default:
		var userNames []string
		for _, p := range possibles {
			userNames = append(userNames, p.Name)
		}
		return nil, fmt.Errorf("selected more than one user from users.jenkins.io: %v", userNames)
	}
}
