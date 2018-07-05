package cmd

import (
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

const (
	fromNs              = "from-namespace-powdered-water"
	toNs                = "to-namespace-journal-entry"
	serviceNameInFromNs = "service-a-is-for-angry"
	serviceNameInToNs   = "service-b-is-for-berserk"
)

func TestServiceLinking(t *testing.T) {
	o := StepLinkServicesOptions{
		FromNamespace: fromNs,
		Includes:      []string{serviceNameInFromNs},
		ToNamespace:   toNs}

	ConfigureTestOptionsWithResources(&o.CommonOptions,
		nil,
		[]runtime.Object{
			kube.NewPreviewEnvironment(fromNs),
			kube.NewPreviewEnvironment(toNs),
		},
		gits.NewGitCLI())
	svcInFromNs := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceNameInFromNs,
		},
	}
	svcInToNs := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceNameInToNs,
		},
	}

	o.kubeClient.CoreV1().Services(fromNs).Create(&svcInFromNs)
	o.kubeClient.CoreV1().Services(toNs).Create(&svcInToNs)
	err := o.Run()
	serviceList, _ := o.kubeClient.CoreV1().Services(toNs).List(metav1.ListOptions{})
	for _, service := range serviceList.Items {
		if service.Name == serviceNameInFromNs {
			print("registered!")
		}
	}
	assert.NoError(t, err)

}
