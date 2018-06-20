package cmd

import (
	"fmt"
	"testing"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestEnvironmentRoleBinding(t *testing.T) {
	o := &StepEnvRoleBindingOptions{}
	ConfigureTestOptionsWithResources(&o.CommonOptions,
		[]runtime.Object{
			&rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myrole",
					Namespace: "jx",
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"get", "watch", "list"},
						APIGroups: []string{""},
						Resources: []string{"configmaps", "pods", "services"},
					},
				},
			},
		},
		[]runtime.Object{
			kube.NewPermanentEnvironment("staging"),
			kube.NewPermanentEnvironment("production"),
			kube.NewPreviewEnvironment("jstrachan-demo96-pr-1"),
			kube.NewPreviewEnvironment("jstrachan-another-pr-3"),
			&v1.EnvironmentRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "env-role-bindings",
					Namespace: "jx",
				},
				Spec: v1.EnvironmentRoleBindingSpec{
					Subjects: []rbacv1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      "jenkins",
							Namespace: "jx",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "Role",
						Name:     "myrole",
					},
					Environments: []v1.EnvironmentFilter{
						{
							Includes: []string{"*"},
						},
					},
				},
			},
		})

	err := o.Run()
	assert.NoError(t, err)

	if err == nil {
		kubeClient, _, err := o.KubeClient()
		assert.NoError(t, err)
		if err == nil {
			namespaces, err := kubeClient.CoreV1().Namespaces().List(metav1.ListOptions{})
			assert.NoError(t, err)
			if err == nil {
				for _, ns := range namespaces.Items {
					fmt.Printf("Has namespace %s\n", ns.Name)
				}
			}
		}
	}
}
