package cmd

import (
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/tests"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	// TODO
	// apifake "k8s.io/apiextensions-apiserver/pkg/client/clientset/versioned/fake"
)

func configureTestOptions(o *CommonOptions) {
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

	// TODO
	//o.apiExtensionsClient = apifake.NewSimpleClientset()
}
