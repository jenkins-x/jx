package users

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/kube"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type UserDetailService struct {
	jxClient  versioned.Interface
	namespace string
}

func NewUserDetailService(jxClient versioned.Interface, namespace string) UserDetailService {
	return UserDetailService{
		jxClient:  jxClient,
		namespace: namespace,
	}
}


func (this *UserDetailService) CreateOrUpdateUser(u *v1.UserDetails) error {
	if u == nil || u.Login == "" {
		return fmt.Errorf("Unable to get or create user, nil or missing login")
	}

	log.Infof("CreateOrUpdateUser: %s <%s>\n", u.Login, u.Email)

	id := kube.ToValidName(u.Login)

	// check for an existing user by email
	user, err := this.jxClient.JenkinsV1().Users(this.namespace).Get(id, metav1.GetOptions{})
	if err != nil {
		// we get an error when not found
		log.Info("Unable to find user: " + id + " -- " + err.Error() + "\n")
	}

	if user != nil && err == nil {
		changed := false

		existing := &user.User

		if existing.Email != u.Email {
			existing.Email = u.Email
			changed = true
		}

		if existing.AvatarURL != u.AvatarURL {
			existing.AvatarURL = u.AvatarURL
			changed = true
		}

		if existing.URL != u.URL {
			existing.URL = u.URL
			changed = true
		}

		if existing.Name != u.Name {
			existing.Name = u.Name
			changed = true
		}

		if existing.Login != u.Login {
			existing.Login = u.Login
			changed = true
		}

		if changed {
			log.Info("Updating modified user: " + existing.Email + "\n")
			_, err = this.jxClient.JenkinsV1().Users(this.namespace).Update(user)
			if err != nil {
				return err
			}
		} else {
			log.Info("Existing user found: " + existing.Email + "\n")
		}
	} else {
		user = &v1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: id,
			},
			Spec: *u,
		}

		log.Info("Adding missing user: " + id + "\n")
		_, err = this.jxClient.JenkinsV1().Users(this.namespace).Create(user)
		if err != nil {
			return err
		}
	}

	return nil
}
