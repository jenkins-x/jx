package namespace_test

// ToDo (@ankitm123): need to figure out how to test this
// import (
// 	"os"
// 	"testing"

// 	jxv1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
// 	jxfake "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
// 	"github.com/jenkins-x/jx/pkg/cmd/namespace"
// 	"github.com/stretchr/testify/assert"
// 	v1 "k8s.io/api/core/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/client-go/kubernetes/fake"
// )

// func TestNewCmdNamespace(t *testing.T) {
// 	testCases := []struct {
// 		description string
// 		env         string
// 		args        []string
// 		ns          string
// 	}{
// 		{
// 			description: "Test case 1",
// 			env:         "",
// 			args:        []string{""},
// 			ns:          "jx",
// 		},
// 		{
// 			description: "Test case 2",
// 			env:         "dev",
// 			args:        []string{""},
// 			ns:          "jx",
// 		},
// 		{
// 			description: "Test case 3",
// 			env:         "dev",
// 			args:        []string{"jx-staging"},
// 			ns:          "jx",
// 		},
// 		{
// 			description: "Test case 4",
// 			env:         "",
// 			args:        []string{""},
// 			ns:          "jx-test",
// 		},
// 		{
// 			description: "Test case 5",
// 			env:         "",
// 			args:        []string{"default"},
// 			ns:          "jx",
// 		},
// 	}
// 	for k, tt := range testCases {
// 		t.Logf("Running test case #%d: %s", k+1, tt.description)
// 		kubeClient := fake.NewSimpleClientset(&v1.Namespace{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name: "jx",
// 			},
// 		}, &v1.Namespace{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name: "default",
// 			},
// 		})
// 		jxClient := jxfake.NewSimpleClientset(&jxv1.Environment{
// 			TypeMeta: metav1.TypeMeta{},
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:      "dev",
// 				Namespace: tt.ns,
// 			},
// 			Spec: jxv1.EnvironmentSpec{
// 				Namespace: tt.ns,
// 			},
// 			Status: jxv1.EnvironmentStatus{},
// 		})
// 		os.Setenv("KUBECONFIG", "testdata/kubeconfig")
// 		_, o := namespace.NewCmdNamespace()
// 		o.KubeClient = kubeClient
// 		o.JXClient = jxClient
// 		o.Env = tt.env
// 		o.Args = tt.args
// 		o.BatchMode = true
// 		err := o.Run()
// 		assert.NoError(t, err)
// 	}
// }
