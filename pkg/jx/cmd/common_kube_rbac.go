package cmd

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (o *CommonOptions) ensureServiceAccount(ns string, serviceAccountName string) error {
	client, _, err := o.KubeClient()
	if err != nil {
		return err
	}

	_, err = client.CoreV1().ServiceAccounts(ns).Get(serviceAccountName, meta_v1.GetOptions{})
	if err != nil {
		// lets create a ServiceAccount for tiller
		sa := &corev1.ServiceAccount{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      serviceAccountName,
				Namespace: ns,
			},
		}
		_, err = client.CoreV1().ServiceAccounts(ns).Create(sa)
		if err != nil {
			return fmt.Errorf("Failed to create ServiceAccount %s in namespace %s: %s", serviceAccountName, ns, err)
		}
		log.Infof("Created ServiceAccount %s in namespace %s\n", util.ColorInfo(serviceAccountName), util.ColorInfo(ns))
	}
	return err
}

func (o *CommonOptions) ensureClusterRoleBinding(clusterRoleBindingName string, role string, serviceAccountNamespace string, serviceAccountName string) error {
	client, _, err := o.KubeClient()
	if err != nil {
		return err
	}

	_, err = client.RbacV1().ClusterRoleBindings().Get(clusterRoleBindingName, meta_v1.GetOptions{})
	if err != nil {
		log.Infof("Trying to create ClusterRoleBinding %s for role: %s and ServiceAccount: %s/%s\n",
			clusterRoleBindingName, role, serviceAccountNamespace, serviceAccountName)

		clusterRoleBinding := &rbacv1.ClusterRoleBinding{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: clusterRoleBindingName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      serviceAccountName,
					Namespace: serviceAccountNamespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind:     "ClusterRole",
				Name:     role,
				APIGroup: "rbac.authorization.k8s.io",
			},
		}
		_, err = client.RbacV1().ClusterRoleBindings().Create(clusterRoleBinding)
		if err != nil {
			return fmt.Errorf("Failed to create ClusterRoleBindings %s: %s", clusterRoleBindingName, err)
		}
		log.Infof("Created ClusterRoleBinding %s\n", clusterRoleBindingName)
	}
	return nil
}
