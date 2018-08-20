package prow

import (
	"github.com/stretchr/testify/assert"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"testing"

	"github.com/ghodss/yaml"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/plugins"
)

type TestOptions struct {
	prowOptions
}

func (o *TestOptions) Setup() {
	o.prowOptions = prowOptions{
		kubeClient: testclient.NewSimpleClientset(),
		repos:      []string{"test/repo"},
		ns:         "test",
	}
}

func TestProwConfig(t *testing.T) {
	t.Parallel()
	o := TestOptions{}
	o.Setup()

	err := o.addProwConfig()
	assert.NoError(t, err)
}

func TestProwPlugins(t *testing.T) {
	t.Parallel()
	o := TestOptions{}
	o.Setup()

	err := o.addProwPlugins()
	assert.NoError(t, err)
}

func TestMergeProwConfig(t *testing.T) {
	t.Parallel()
	o := TestOptions{}
	o.Setup()

	prowConfig := &config.Config{}
	prowConfig.LogLevel = "debug"

	c, err := yaml.Marshal(prowConfig)
	assert.NoError(t, err)

	data := make(map[string]string)
	data["config.yaml"] = string(c)

	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "config",
		},
		Data: data,
	}

	_, err = o.kubeClient.CoreV1().ConfigMaps(o.ns).Create(cm)
	assert.NoError(t, err)

	err = o.addProwConfig()
	assert.NoError(t, err)

	cm, err = o.kubeClient.CoreV1().ConfigMaps(o.ns).Get("config", metav1.GetOptions{})
	assert.NoError(t, err)

	yaml.Unmarshal([]byte(cm.Data["config.yaml"]), &prowConfig)
	assert.Equal(t, "debug", prowConfig.LogLevel)
	assert.NotEmpty(t, prowConfig.Presubmits["test/repo"])

}

func TestMergeProwPlugin(t *testing.T) {
	t.Parallel()
	o := TestOptions{}
	o.Setup()

	pluginConfig := &plugins.Configuration{}
	pluginConfig.Welcome = plugins.Welcome{MessageTemplate: "okey dokey"}

	c, err := yaml.Marshal(pluginConfig)
	assert.NoError(t, err)

	data := make(map[string]string)
	data["plugins.yaml"] = string(c)

	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "plugins",
		},
		Data: data,
	}

	_, err = o.kubeClient.CoreV1().ConfigMaps(o.ns).Create(cm)
	assert.NoError(t, err)

	err = o.addProwPlugins()
	assert.NoError(t, err)

	cm, err = o.kubeClient.CoreV1().ConfigMaps(o.ns).Get("plugins", metav1.GetOptions{})
	assert.NoError(t, err)

	yaml.Unmarshal([]byte(cm.Data["plugins.yaml"]), &pluginConfig)
	assert.Equal(t, "okey dokey", pluginConfig.Welcome.MessageTemplate)
	assert.Equal(t, "test/repo", pluginConfig.Approve[0].Repos[0])

}

func TestAddProwPlugin(t *testing.T) {
	t.Parallel()
	o := TestOptions{}
	o.Setup()

	o.repos = append(o.repos, "test/repo2")

	err := o.addProwPlugins()
	assert.NoError(t, err)

	cm, err := o.kubeClient.CoreV1().ConfigMaps(o.ns).Get("plugins", metav1.GetOptions{})
	assert.NoError(t, err)

	pluginConfig := &plugins.Configuration{}
	yaml.Unmarshal([]byte(cm.Data["plugins.yaml"]), &pluginConfig)

	assert.Equal(t, "test/repo", pluginConfig.Approve[0].Repos[0])
	assert.Equal(t, "test/repo2", pluginConfig.Approve[1].Repos[0])

}

func TestAddProwConfig(t *testing.T) {
	t.Parallel()
	o := TestOptions{}
	o.Setup()

	o.repos = append(o.repos, "test/repo2")

	err := o.addProwConfig()
	assert.NoError(t, err)

	cm, err := o.kubeClient.CoreV1().ConfigMaps(o.ns).Get("config", metav1.GetOptions{})
	assert.NoError(t, err)

	prowConfig := &config.Config{}

	yaml.Unmarshal([]byte(cm.Data["config.yaml"]), &prowConfig)

	assert.NotEmpty(t, prowConfig.Presubmits["test/repo"])
	assert.NotEmpty(t, prowConfig.Presubmits["test/repo2"])
}

// make sure that rerunning addProwConfig replaces any modified changes in the configmap
func TestReplaceProwConfig(t *testing.T) {
	t.Parallel()
	o := TestOptions{}
	o.Setup()

	err := o.addProwConfig()
	assert.NoError(t, err)

	// now modify the cm
	cm, err := o.kubeClient.CoreV1().ConfigMaps(o.ns).Get("config", metav1.GetOptions{})
	assert.NoError(t, err)

	prowConfig := &config.Config{}

	yaml.Unmarshal([]byte(cm.Data["config.yaml"]), &prowConfig)

	p := prowConfig.Presubmits["test/repo"]
	p[0].Agent = "foo"

	configYAML, err := yaml.Marshal(&prowConfig)
	assert.NoError(t, err)

	data := make(map[string]string)
	data["config.yaml"] = string(configYAML)
	cm = &v1.ConfigMap{
		Data: data,
		ObjectMeta: metav1.ObjectMeta{
			Name: "config",
		},
	}

	_, err = o.kubeClient.CoreV1().ConfigMaps(o.ns).Update(cm)

	// ensure the value was modified
	cm, err = o.kubeClient.CoreV1().ConfigMaps(o.ns).Get("config", metav1.GetOptions{})
	assert.NoError(t, err)

	prowConfig = &config.Config{}

	yaml.Unmarshal([]byte(cm.Data["config.yaml"]), &prowConfig)

	p = prowConfig.Presubmits["test/repo"]
	assert.Equal(t, "foo", p[0].Agent)

	// generate the prow config again
	err = o.addProwConfig()
	assert.NoError(t, err)

	// assert value is reset
	cm, err = o.kubeClient.CoreV1().ConfigMaps(o.ns).Get("config", metav1.GetOptions{})
	assert.NoError(t, err)

	prowConfig = &config.Config{}

	yaml.Unmarshal([]byte(cm.Data["config.yaml"]), &prowConfig)

	p = prowConfig.Presubmits["test/repo"]
	assert.Equal(t, "kubernetes", p[0].Agent)
}
