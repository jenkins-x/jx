package users

import (
	"fmt"
	"sort"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/jenkins-x/jx/pkg/gits"

	"github.com/jenkins-x/jx/pkg/kube/naming"
	"github.com/jenkins-x/jx/pkg/log"

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

// Resolve does the heavy lifting for user resolution.
// This function is not normally called directly but by a dedicated user resolver (e.g. GitUserResolver)
// * checking the user custom resources to see if the user is present there
// * calling selectUsers to try to identify the user
// as often user info is not complete in a git response
func Resolve(gitUser *gits.GitUser, providerKey string, jxClient versioned.Interface,
	namespace string) (*jenkinsv1.User, *jenkinsv1.UserList, error) {

	id := naming.ToValidValue(gitUser.Login)

	if id != "" {

		labelSelector := fmt.Sprintf("%s=%s", providerKey, id)

		// First try to find by label - this is much faster as it uses an index
		users, err := jxClient.JenkinsV1().Users(namespace).List(metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return nil, nil, err
		}
		if len(users.Items) > 1 {
			return nil, nil, fmt.Errorf("more than one user found in users.jenkins.io with label %s, found %v", labelSelector,
				users.Items)
		} else if len(users.Items) == 1 {
			return &users.Items[0], users, nil
		}
	}

	// Next try without the label - this might occur if someone manually updated the list
	users, err := jxClient.JenkinsV1().Users(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}

	if id == "" {
		return nil, users, nil
	} else if id != "" {
		possibles := make([]jenkinsv1.User, 0)
		for _, u := range users.Items {
			for _, a := range u.Spec.Accounts {
				if a.Provider == providerKey && a.ID == id {
					possibles = append(possibles, u)
				}
			}
		}
		if len(possibles) > 1 {
			possibleUsers := make([]string, 0)
			for _, p := range possibles {
				possibleUsers = append(possibleUsers, p.Name)
			}
			return nil, nil, fmt.Errorf("more than one user found in users.jenkins.io with login %s for provider %s, "+
				"found %s", id, providerKey, possibleUsers)
		} else if len(possibles) == 0 {
			return nil, users, nil
		}

		// Add the label for next time
		found := &possibles[0]

		if found.Labels == nil {
			found.Labels = make(map[string]string)
		}

		found.Labels[providerKey] = id
		found, err := jxClient.JenkinsV1().Users(namespace).Update(found)
		if err != nil {
			return nil, nil, err
		}

		log.Logger().Infof("Adding label %s=%s to user %s in users.jenkins.io", providerKey, id, found.Name)
		return found, users, nil
	}

	return nil, users, nil
}

func findOrCreateJXUser(jxUser *jenkinsv1.User, err error, users *jenkinsv1.UserList, gitUser *gits.GitUser, jxClient versioned.Interface, namespace string, providerKey string) (*jenkinsv1.User, error) {
	id, possibles, jxUser, err := matchPossibleUsers(jxUser, users.Items, gitUser, jxClient, namespace)
	if err != nil {
		return nil, err
	}
	if len(possibles) > 1 {
		possibleStrings := make([]string, 0)
		for _, p := range possibles {
			possibleStrings = append(possibleStrings, p.Name)
		}
		return nil, fmt.Errorf("selected more than one user from users.jenkins.io: %v", possibleStrings)
	} else if len(possibles) == 1 {
		found := &possibles[0]
		if id != "" {
			// Add the git id to the user
			if found.Spec.Accounts == nil {
				found.Spec.Accounts = make([]jenkinsv1.AccountReference, 0)
			}
			found.Spec.Accounts = append(found.Spec.Accounts, jenkinsv1.AccountReference{
				ID:       id,
				Provider: providerKey,
			})
			// Add the label as well
			if found.Labels == nil {
				found.Labels = make(map[string]string)
			}
			found.Labels[providerKey] = id
			log.Logger().Infof("Associating user %s in users.jenkins.io with email %s to git GitProvider user with login %s as "+
				"emails match", found.Name, found.Spec.Email, id)
			log.Logger().Infof("Adding label %s=%s to user %s in users.jenkins.io", providerKey, id, found.Name)
			_, err := jxClient.JenkinsV1().Users(namespace).Update(found)
			if err != nil {
				return nil, err
			}
		}
		return found, nil
	} else {
		// Otherwise, create a new user
		return jxClient.JenkinsV1().Users(namespace).Create(jxUser)
	}
}

func matchPossibleUsers(user *jenkinsv1.User, users []jenkinsv1.User, gitUser *gits.GitUser, jxClient versioned.Interface,
	namespace string) (string, []jenkinsv1.User,
	*jenkinsv1.User, error) {
	possibles := make([]jenkinsv1.User, 0)
	if gitUser.Email != "" {
		// Don't do this if email is empty as otherwise we risk matching any users who have empty emails!
		for _, u := range users {
			if u.Spec.Email == gitUser.Email {
				possibles = append(possibles, u)
			}
		}
	}

	id := gitUser.Login

	// Check if the user id is available, if not append "-<n>" where <n> is some integer
	for i := 0; true; i++ {
		_, err := jxClient.JenkinsV1().Users(namespace).Get(id, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			break
		}
		id = fmt.Sprintf("%s-%d", gitUser.Login, i)
	}

	user.Name = naming.ToValidName(id)

	return id, possibles, user, nil
}
