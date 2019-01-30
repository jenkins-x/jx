package cmd_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	resources_test "github.com/jenkins-x/jx/pkg/kube/resources/mocks"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	fromNs                   = "from-namespace-powdered-water"
	toNs                     = "to-namespace-journal-entry"
	serviceNameInFromNs      = "service-a-is-for-angry"
	serviceNameDummyInFromNs = "service-p-is-polluted"
	serviceNameInToNs        = "service-b-is-for-berserk"
)

func TestServiceLinking(t *testing.T) {
	t.Parallel()
	o := cmd.StepLinkServicesOptions{
		FromNamespace: fromNs,
		Includes:      []string{serviceNameInFromNs},
		Excludes:      []string{serviceNameDummyInFromNs},
		ToNamespace:   toNs}
	fromNspc := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fromNs,
		},
	}
	svcInFromNs := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceNameInFromNs,
			Namespace: fromNs,
		},
	}
	svcDummyInFromNs := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceNameDummyInFromNs,
			Namespace: fromNs,
		},
	}
	toNspc := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: toNs,
		},
	}
	svcInToNs := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceNameInToNs,
			Namespace: toNs,
		},
	}

	cmd.ConfigureTestOptionsWithResources(&o.CommonOptions,
		[]runtime.Object{fromNspc, toNspc, svcInFromNs, svcInToNs, svcDummyInFromNs},
		nil,
		gits.NewGitCLI(),
		nil,
		helm.NewHelmCLI("helm", helm.V2, "", true),
		resources_test.NewMockInstaller())
	client, err := o.KubeClient()
	assert.NoError(t, err)
	serviceListFromNsBeforeStepLink, _ := client.CoreV1().Services(fromNs).List(metav1.ListOptions{})
	assert.EqualValues(t, len(serviceListFromNsBeforeStepLink.Items), 2)
	serviceListToNsBeforeStepLink, _ := client.CoreV1().Services(toNs).List(metav1.ListOptions{})
	assert.EqualValues(t, len(serviceListToNsBeforeStepLink.Items), 1)
	err = o.Run()
	serviceList, _ := client.CoreV1().Services(toNs).List(metav1.ListOptions{})
	serviceNames := []string{""}
	for _, service := range serviceList.Items {
		serviceNames = append(serviceNames, service.Name)
	}
	serviceListToNsAfterStepLink, _ := client.CoreV1().Services(toNs).List(metav1.ListOptions{})
	assert.EqualValues(t, len(serviceListToNsAfterStepLink.Items), 2)

	assert.Contains(t, serviceNames, serviceNameInFromNs) //Check if service that was in include list got added
	assert.EqualValues(t, len(serviceNames), 3)           //Check if service that was in exclude list didn't get added
	assert.NoError(t, err)
}
