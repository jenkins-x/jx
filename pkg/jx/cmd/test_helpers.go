package cmd

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	v1fake "github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx/pkg/gits"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/jenkins-x/jx/pkg/util"
	corev1 "k8s.io/api/core/v1"
	apifake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// ConfigureTestOptions lets configure the options for use in tests
// using fake APIs to k8s cluster
func ConfigureTestOptions(o *CommonOptions) {
	ConfigureTestOptionsWithResources(o, nil, nil)
}

// ConfigureTestOptions lets configure the options for use in tests
// using fake APIs to k8s cluster
func ConfigureTestOptionsWithResources(o *CommonOptions, k8sObjects []runtime.Object, jxObjects []runtime.Object) {
	o.Out = tests.Output()
	o.BatchMode = true
	o.Factory = cmdutil.NewFactory()
	o.currentNamespace = "jx"

	namespacesRequired := []string{o.currentNamespace}
	namespaceMap := map[string]*corev1.Namespace{}

	for _, ro := range k8sObjects {
		ns, ok := ro.(*corev1.Namespace)
		if ok {
			namespaceMap[ns.Name] = ns
		}
	}
	hasDev := false
	for _, ro := range jxObjects {
		env, ok := ro.(*v1.Environment)
		if ok {
			ns := env.Spec.Namespace
			if ns != "" && util.StringArrayIndex(namespacesRequired, ns) < 0 {
				namespacesRequired = append(namespacesRequired, ns)
			}
			if env.Name == "dev" {
				hasDev = true
			}
		}
	}

	// ensure we've the dev nenvironment
	if !hasDev {
		devEnv := kube.NewPermanentEnvironment("dev")
		devEnv.Spec.Namespace = o.currentNamespace
		devEnv.Spec.Kind = v1.EnvironmentKindTypeDevelopment

		jxObjects = append(jxObjects, devEnv)
	}

	// add any missing namespaces
	for _, ns := range namespacesRequired {
		if namespaceMap[ns] == nil {
			k8sObjects = append(k8sObjects, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
					Labels: map[string]string{
						"tag": "",
					},
				},
			})
		}
	}

	o.kubeClient = fake.NewSimpleClientset(k8sObjects...)
	o.jxClient = v1fake.NewSimpleClientset(jxObjects...)
	o.apiExtensionsClient = apifake.NewSimpleClientset()
	o.git = gits.NewGitCLI()
}
