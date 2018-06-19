package cmd

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	v1fake "github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/tests"
	corev1 "k8s.io/api/core/v1"
	apifake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// ConfigureTestOptions lets configure the options for use in tests
// using fake APIs to k8s cluster
func ConfigureTestOptions(o *CommonOptions) {
	o.Out = tests.Output()
	o.BatchMode = true
	o.Factory = cmdutil.NewFactory()

	// use fake k8s API
	o.kubeClient = fake.NewSimpleClientset(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "jx",
			Labels: map[string]string{
				"tag": "",
			},
		},
	})
	o.currentNamespace = "jx"

	o.jxClient = v1fake.NewSimpleClientset(&v1.Environment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dev",
			Labels: map[string]string{
				"tag": "",
			},
		},
	})

	o.apiExtensionsClient = apifake.NewSimpleClientset()
}
