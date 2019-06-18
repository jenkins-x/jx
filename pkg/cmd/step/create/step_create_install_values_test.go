package create_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/step/create"
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	resources_test "github.com/jenkins-x/jx/pkg/kube/resources/mocks"
	"github.com/jenkins-x/jx/pkg/util"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateInstallValues(t *testing.T) {
	testData := path.Join("test_data", "step_create_install_values")
	assert.DirExists(t, testData)

	outputDir, err := ioutil.TempDir("", "test-step-create-install-values-")
	require.NoError(t, err)

	err = util.CopyDir(testData, outputDir, true)
	require.NoError(t, err, "failed to copy test data into temp dir")

	o := &create.StepCreateInstallValuesOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: &opts.CommonOptions{
				In:  os.Stdin,
				Out: os.Stdout,
				Err: os.Stderr,
			},
		},
		Dir:              outputDir,
		Namespace:        "jx",
		IngressNamespace: opts.DefaultIngressNamesapce,
		IngressService:   opts.DefaultIngressServiceName,
	}

	ingressHostName := "1.2.3.4"
	expectedDomain := ingressHostName + ".nip.io"

	runtimeObjects := []runtime.Object{
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      opts.DefaultIngressServiceName,
				Namespace: opts.DefaultIngressNamesapce,
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeLoadBalancer,
			},
			Status: corev1.ServiceStatus{
				LoadBalancer: corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{
							Hostname: ingressHostName,
						},
					},
				},
			},
		},
	}
	testhelpers.ConfigureTestOptionsWithResources(o.CommonOptions,
		runtimeObjects,
		nil,
		gits.NewGitCLI(),
		nil,
		helm.NewHelmCLI("helm", helm.V2, "", true),
		resources_test.NewMockInstaller(),
	)

	err = o.Run()
	require.NoError(t, err, "failed to run step")

	fileName := filepath.Join(outputDir, "values.yaml")

	t.Logf("Generated values file at %s\n", fileName)

	assert.FileExists(t, fileName, "failed to create valid file")

	values, err := helm.LoadValuesFile(fileName)
	require.NoError(t, err, "failed to load file %s", fileName)

	AssertMapPathValueAsString(t, values, "cluster.namespaceSubDomain", ".jx.")
	AssertMapPathValueAsString(t, values, "cluster.domain", expectedDomain)
}

func AssertMapPathValueAsString(t *testing.T, values map[string]interface{}, path string, expected string) {
	actual := util.GetMapValueAsStringViaPath(values, path)
	assert.Equal(t, expected, actual, "invalid helm value for path %s", path)
}
