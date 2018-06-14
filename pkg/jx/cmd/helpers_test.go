package cmd

import (
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/tests"
	"k8s.io/client-go/kubernetes/fake"
)

func configureTestOptions(o *CommonOptions) {
	o.Out = tests.Output()
	o.BatchMode = true
	o.Factory = cmdutil.NewFactory()

	// use fake k8s API
	o.kubeClient = fake.NewSimpleClientset()
}
