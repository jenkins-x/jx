package kube

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func EnsureEnvironmentNamespaceSetup(kubeClient kubernetes.Interface, jxClient versioned.Interface, env *v1.Environment, ns string) error {
	// lets create the namespace if we are on the same cluster
	spec := &env.Spec
	if spec.Cluster == "" && spec.Namespace != "" {
		labels := map[string]string{
			LabelTeam:        ns,
			LabelEnvironment: env.Name,
		}
		annotations := map[string]string{}

		err := EnsureNamespaceCreated(kubeClient, spec.Namespace, labels, annotations)
		if err != nil {
			return err
		}
	}

	// lets annotate the team namespace as being the developer environment
	labels := map[string]string{
		LabelTeam:        ns,
		LabelEnvironment: LabelValueDevEnvironment,
	}
	annotations := map[string]string{}

	// lets check that the current namespace is marked as the dev environment
	err := EnsureNamespaceCreated(kubeClient, ns, labels, annotations)
	if err != nil {
		return err
	}
	_, err = EnsureDevEnvironmentSetup(jxClient, ns)
	return err
}

// EnsureDevEnvironmentSetup ensures that the Environment is created in the given namespace
func EnsureDevEnvironmentSetup(jxClient versioned.Interface, ns string) (*v1.Environment, error) {
	// lets ensure there is a dev Environment setup so that we can easily switch between all the environments
	env, err := jxClient.JenkinsV1().Environments(ns).Get(LabelValueDevEnvironment, metav1.GetOptions{})
	if err != nil {
		// lets create a dev environment
		env = &v1.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name: LabelValueDevEnvironment,
			},
			Spec: v1.EnvironmentSpec{
				Namespace:         ns,
				Label:             "Development",
				PromotionStrategy: v1.PromotionStrategyTypeNever,
				Kind:              v1.EnvironmentKindTypeDevelopment,
				TeamSettings: v1.TeamSettings{
					UseGitOPs:           true,
					AskOnCreate:         false,
					QuickstartLocations: DefaultQuickstartLocations,
				},
			},
		}
		_, err = jxClient.JenkinsV1().Environments(ns).Create(env)
		if err != nil {
			return nil, err
		}
	}
	return env, nil
}

// EnsureEditEnvironmentSetup ensures that the Environment is created in the given namespace
func EnsureEditEnvironmentSetup(kubeClient kubernetes.Interface, jxClient versioned.Interface, ns string, username string) (*v1.Environment, error) {
	// lets ensure there is a dev Environment setup so that we can easily switch between all the environments
	envList, err := jxClient.JenkinsV1().Environments(ns).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	if envList != nil {
		for _, e := range envList.Items {
			if e.Spec.Kind == v1.EnvironmentKindTypeEdit && e.Spec.PreviewGitSpec.User.Username == username {
				return &e, nil
			}
		}
	}

	editNS := ToValidName(ns + "-edit-" + username)
	labels := map[string]string{
		LabelTeam:        ns,
		LabelEnvironment: username,
		LabelKind:        ValueKindEditNamespace,
		LabelUsername:    username,
	}
	annotations := map[string]string{}

	err = EnsureNamespaceCreated(kubeClient, editNS, labels, annotations)
	if err != nil {
		return nil, err
	}

	cm, err := kubeClient.CoreV1().ConfigMaps(ns).Get(ConfigMapExposecontroller, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	oldCm, err := kubeClient.CoreV1().ConfigMaps(editNS).Get(ConfigMapExposecontroller, metav1.GetOptions{})
	if oldCm == nil || err != nil {
		cm.Namespace = editNS
		cm.ResourceVersion = ""
		_, err := kubeClient.CoreV1().ConfigMaps(editNS).Create(cm)
		if err != nil {
			return nil, err
		}
	}

	env := &v1.Environment{
		ObjectMeta: metav1.ObjectMeta{
			Name: username,
		},
		Spec: v1.EnvironmentSpec{
			Namespace:         editNS,
			Label:             username,
			PromotionStrategy: v1.PromotionStrategyTypeNever,
			Kind:              v1.EnvironmentKindTypeEdit,
			PreviewGitSpec: v1.PreviewGitSpec{
				User: v1.UserSpec{
					Username: username,
				},
			},
			Order: 1,
		},
	}
	_, err = jxClient.JenkinsV1().Environments(ns).Create(env)
	if err != nil {
		return nil, err
	}
	return env, nil
}

// Ensure that the namespace exists for the given name
func EnsureNamespaceCreated(kubeClient kubernetes.Interface, name string, labels map[string]string, annotations map[string]string) error {
	n, err := kubeClient.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
	if err == nil {
		// lets check if we have the labels setup
		if n.Annotations == nil {
			n.Annotations = map[string]string{}
		}
		if n.Labels == nil {
			n.Labels = map[string]string{}
		}
		changed := false
		if labels != nil {
			for k, v := range labels {
				if n.Labels[k] != v {
					n.Labels[k] = v
					changed = true
				}
			}
		}
		if annotations != nil {
			for k, v := range annotations {
				if n.Annotations[k] != v {
					n.Annotations[k] = v
					changed = true
				}
			}
		}
		if changed {
			_, err = kubeClient.CoreV1().Namespaces().Update(n)
			if err != nil {
				return fmt.Errorf("Failed to label Namespace %s %s", name, err)
			}
		}
		return nil
	}

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      labels,
			Annotations: annotations,
		},
	}
	_, err = kubeClient.CoreV1().Namespaces().Create(namespace)
	if err != nil {
		return fmt.Errorf("Failed to create Namespace %s %s", name, err)
	}
	return err
}
