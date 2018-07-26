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
	Hook                   = "hook"
	DefaultProwReleaseName = "prow"
	ProwVersion            = "0.0.9"
	ChartProw              = "jenkins-x/prow"
)

type prowOptions struct {
	kubeClient kubernetes.Interface
	repos      []string
	ns         string
}

func AddRepo(kubeClient kubernetes.Interface, repos []string, ns string) error {

	if len(repos) == 0 {
		return fmt.Errorf("no repo defined")
	}
	o := prowOptions{
		kubeClient: kubeClient,
		repos:      repos,
		ns:         ns,
	}

	err := o.addProwConfig()
	if err != nil {
		return err
	}

	return o.addProwPlugins()
}

// create git repo?
// get config and update / overwrite repos?
// should we get the existing CM and do a diff?
// should we just be using git for config and use prow to auto update via gitops?

func (o *prowOptions) createPreSubmit() config.Presubmit {
	ps := config.Presubmit{}

	ps.Name = "promotion-gate"
	ps.AlwaysRun = true
	ps.SkipReport = false
	ps.Context = "promotion-gate"
	ps.Agent = "kubernetes"

	spec := &v1.PodSpec{
		Containers: []v1.Container{
			{
				Image: "rawlingsj/builder-base:dev16",
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
func (o *prowOptions) createPostSubmit() config.Postsubmit {
	ps := config.Postsubmit{}
	ps.Name = "test-postsubmits"

	ps.Agent = "kubernetes"

	spec := &v1.PodSpec{
		Containers: []v1.Container{
			{
				Image: "rawlingsj/builder-base:dev16",
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
func (o *prowOptions) createTide() config.Tide {
	t := config.Tide{
		TargetURL: "https://tide.jx.felix.rawlings.it",
	}

	var qs []config.TideQuery

	for _, r := range o.repos {
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
func (o *prowOptions) addProwConfig() error {
	prowConfig := config.Config{}

	preSubmit := o.createPreSubmit()
	postSubmit := o.createPostSubmit()
	tide := o.createTide()

	preSubmits := make(map[string][]config.Presubmit)
	postSubmits := make(map[string][]config.Postsubmit)

	for _, r := range o.repos {
		preSubmits[r] = []config.Presubmit{preSubmit}
		postSubmits[r] = []config.Postsubmit{postSubmit}
	}

	prowConfig.Presubmits = preSubmits
	prowConfig.Postsubmits = postSubmits
	prowConfig.Tide = tide

	configYAML, err := yaml.Marshal(prowConfig)
	if err != nil {
		return err
	}

	data := make(map[string]string)
	data["config.yaml"] = string(configYAML)
	cm := v1.ConfigMap{
		Data: data,
		ObjectMeta: metav1.ObjectMeta{
			Name: "config",
		},
	}

	_, err = o.kubeClient.CoreV1().ConfigMaps(o.ns).Create(&cm)
	return err
}
func (o *prowOptions) addProwPlugins() error {

	pluginsList := []string{"approve", "assign", "blunderbuss", "help", "hold", "lgtm", "lifecycle", "size", "trigger", "wip"}

	repoPlugins := make(map[string][]string)
	var approves []plugins.Approve

	for _, r := range o.repos {
		repoPlugins[r] = pluginsList

		a := plugins.Approve{
			Repos:               []string{r},
			ReviewActsAsApprove: true,
			LgtmActsAsApprove:   true,
		}
		approves = append(approves, a)
	}

	pluginConfig := plugins.Configuration{
		Plugins: repoPlugins,
		Approve: approves,
	}

	pluginYAML, err := yaml.Marshal(pluginConfig)
	if err != nil {
		return err
	}

	data := make(map[string]string)
	data["plugins.yaml"] = string(pluginYAML)
	cm := v1.ConfigMap{
		Data: data,
		ObjectMeta: metav1.ObjectMeta{
			Name: "config",
		},
	}
	_, err = o.kubeClient.CoreV1().ConfigMaps(o.ns).Create(&cm)
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
