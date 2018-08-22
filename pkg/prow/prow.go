package prow

import (
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/kube"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/plugins"
)

const (
	Hook                          = "hook"
	DefaultProwReleaseName        = "jx-prow"
	DefaultKnativeBuilReleaseName = "jx-knative-build"
	ProwVersion                   = "0.0.11"
	KnativeBuildVersion           = "0.0.2"
	ChartProw                     = "jenkins-x/prow"
	ChartKnativeBuild             = "jenkins-x/knative-build"
)

// Options for prow
type Options struct {
	KubeClient kubernetes.Interface
	Repos      []string
	NS         string
}

func AddRepo(kubeClient kubernetes.Interface, repos []string, ns string) error {

	if len(repos) == 0 {
		return fmt.Errorf("no repo defined")
	}
	o := Options{
		KubeClient: kubeClient,
		Repos:      repos,
		NS:         ns,
	}

	err := o.AddProwConfig()
	if err != nil {
		return err
	}

	return o.AddProwPlugins()
}

// create git repo?
// get config and update / overwrite repos?
// should we get the existing CM and do a diff?
// should we just be using git for config and use prow to auto update via gitops?

func (o *Options) createPreSubmit() config.Presubmit {
	ps := config.Presubmit{}

	ps.Name = "promotion-gate"
	ps.AlwaysRun = true
	ps.SkipReport = false
	ps.Context = "promotion-gate"
	ps.Agent = "kubernetes"

	spec := &v1.PodSpec{
		Containers: []v1.Container{
			{
				Image: "jenkinsxio/builder-base:latest",
				Args:  []string{"jx", "step", "helm", "build"},
				Env: []v1.EnvVar{
					{Name: "DEPLOY_NAMESPACE", Value: "jx-staging"},
					{Name: "CHART_REPOSITORY", Value: "http://jenkins-x-chartmuseum:8080"},
					{Name: "XDG_CONFIG_HOME", Value: "/home/jenkins"},
					{Name: "GIT_COMMITTER_EMAIL", Value: "jenkins-x@googlegroups.com"},
					{Name: "GIT_AUTHOR_EMAIL", Value: "jenkins-x@googlegroups.com"},
					{Name: "GIT_AUTHOR_NAME", Value: "jenkins-x-bot"},
					{Name: "GIT_COMMITTER_NAME", Value: "jenkins-x-bot"},
				},
			},
		},
		ServiceAccountName: "jenkins",
	}

	ps.Spec = spec
	ps.RerunCommand = "/test this"
	ps.Trigger = "(?m)^/test( all| this),?(\\s+|$)"

	return ps
}
func (o *Options) createPostSubmit() config.Postsubmit {
	ps := config.Postsubmit{}
	ps.Name = "test-postsubmits"

	ps.Agent = "kubernetes"

	spec := &v1.PodSpec{
		Containers: []v1.Container{
			{
				Image: "jenkinsxio/builder-base:latest",
				Args:  []string{"jx", "step", "helm", "apply"},
				Env: []v1.EnvVar{
					{Name: "DEPLOY_NAMESPACE", Value: "jx-staging"},
					{Name: "CHART_REPOSITORY", Value: "http://jenkins-x-chartmuseum:8080"},
					{Name: "XDG_CONFIG_HOME", Value: "/home/jenkins"},
					{Name: "GIT_COMMITTER_EMAIL", Value: "jenkins-x@googlegroups.com"},
					{Name: "GIT_AUTHOR_EMAIL", Value: "jenkins-x@googlegroups.com"},
					{Name: "GIT_AUTHOR_NAME", Value: "jenkins-x-bot"},
					{Name: "GIT_COMMITTER_NAME", Value: "jenkins-x-bot"},
				},
			},
		},
		ServiceAccountName: "jenkins",
	}

	ps.Spec = spec

	return ps
}
func (o *Options) createTide() config.Tide {
	// todo get the real URL, though we need to handle the multi cluster usecase where dev namespace may be another cluster, so pass it in as an arg?
	t := config.Tide{
		TargetURL: "https://tide.foo.bar",
	}

	var qs []config.TideQuery

	for _, r := range o.Repos {
		q := config.TideQuery{
			Repos:         []string{r},
			Labels:        []string{"lgtm", "approved"},
			MissingLabels: []string{"do-not-merge", "do-not-merge/hold", "do-not-merge/work-in-progress", "needs-ok-to-test", "needs-rebase"},
		}
		qs = append(qs, q)
	}

	queries := qs

	t.Queries = queries

	// todo JR not sure if we need the contexts if we add the branch protection plugin
	//orgPolicies := make(map[string]config.TideOrgContextPolicy)
	//repoPolicies := make(map[string]config.TideRepoContextPolicy)
	//
	//ctxPolicy := config.TideContextPolicy{}
	//
	//repoPolicy := config.TideRepoContextPolicy{}
	//
	//repoPolicies[""] = repoPolicy
	//orgPolicy := config.TideOrgContextPolicy{
	//	TideContextPolicy: ctxPolicy,
	//	Repos:             repoPolicies,
	//}
	//
	//orgPolicies[""] = orgPolicy

	myTrue := true
	t.ContextOptions = config.TideContextPolicyOptions{
		TideContextPolicy: config.TideContextPolicy{
			FromBranchProtection: &myTrue,
			SkipUnknownContexts:  &myTrue,
		},
		//Orgs: orgPolicies,
	}

	return t
}

// AddProwConfig adds config to prow
func (o *Options) AddProwConfig() error {

	preSubmit := o.createPreSubmit()
	postSubmit := o.createPostSubmit()
	tide := o.createTide()

	cm, err := o.KubeClient.CoreV1().ConfigMaps(o.NS).Get("config", metav1.GetOptions{})
	create := true
	prowConfig := &config.Config{}
	if err != nil {
		prowConfig.Presubmits = make(map[string][]config.Presubmit)
		prowConfig.Postsubmits = make(map[string][]config.Postsubmit)

	} else {
		create = false
		err = yaml.Unmarshal([]byte(cm.Data["config.yaml"]), &prowConfig)
		if err != nil {
			return err
		}
		if len(prowConfig.Presubmits) == 0 {
			prowConfig.Presubmits = make(map[string][]config.Presubmit)
		}
		if len(prowConfig.Postsubmits) == 0 {
			prowConfig.Postsubmits = make(map[string][]config.Postsubmit)
		}
	}

	for _, r := range o.Repos {
		prowConfig.Presubmits[r] = []config.Presubmit{preSubmit}
		prowConfig.Postsubmits[r] = []config.Postsubmit{postSubmit}
	}

	prowConfig.Tide = tide

	configYAML, err := yaml.Marshal(prowConfig)
	if err != nil {
		return err
	}

	data := make(map[string]string)
	data["config.yaml"] = string(configYAML)
	cm = &v1.ConfigMap{
		Data: data,
		ObjectMeta: metav1.ObjectMeta{
			Name: "config",
		},
	}

	if create {
		_, err = o.KubeClient.CoreV1().ConfigMaps(o.NS).Create(cm)
	} else {
		_, err = o.KubeClient.CoreV1().ConfigMaps(o.NS).Update(cm)
	}

	return err
}

// AddProwPlugins adds plugins to prow
func (o *Options) AddProwPlugins() error {

	pluginsList := []string{"approve", "assign", "blunderbuss", "help", "hold", "lgtm", "lifecycle", "size", "trigger", "wip"}

	cm, err := o.KubeClient.CoreV1().ConfigMaps(o.NS).Get("plugins", metav1.GetOptions{})
	create := true
	pluginConfig := &plugins.Configuration{}
	if err != nil {
		pluginConfig.Plugins = make(map[string][]string)
		pluginConfig.Approve = []plugins.Approve{}

	} else {
		create = false
		err = yaml.Unmarshal([]byte(cm.Data["plugins.yaml"]), &pluginConfig)
		if err != nil {
			return err
		}
		if pluginConfig == nil {
			pluginConfig = &plugins.Configuration{}
		}
		if len(pluginConfig.Plugins) == 0 {
			pluginConfig.Plugins = make(map[string][]string)
		}
		if len(pluginConfig.Approve) == 0 {
			pluginConfig.Approve = []plugins.Approve{}
		}

	}

	// add or overwrite
	for _, r := range o.Repos {
		pluginConfig.Plugins[r] = pluginsList

		a := plugins.Approve{
			Repos:               []string{r},
			ReviewActsAsApprove: true,
			LgtmActsAsApprove:   true,
		}
		pluginConfig.Approve = append(pluginConfig.Approve, a)
	}

	pluginYAML, err := yaml.Marshal(pluginConfig)
	if err != nil {
		return err
	}

	data := make(map[string]string)
	data["plugins.yaml"] = string(pluginYAML)
	cm = &v1.ConfigMap{
		Data: data,
		ObjectMeta: metav1.ObjectMeta{
			Name: "plugins",
		},
	}
	if create {
		_, err = o.KubeClient.CoreV1().ConfigMaps(o.NS).Create(cm)
	} else {
		_, err = o.KubeClient.CoreV1().ConfigMaps(o.NS).Update(cm)
	}

	return err
}

func IsProwInstalled(kubeClient kubernetes.Interface, ns string) (bool, error) {

	podCount, err := kube.DeploymentPodCount(kubeClient, Hook, ns)
	if err != nil {
		return false, fmt.Errorf("failed when looking for hook deployment: %v", err)
	}
	if podCount == 0 {
		return false, nil
	}
	return true, nil
}
