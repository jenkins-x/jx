package prow

import (
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	build "github.com/knative/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/plugins"
)

const (
	Hook             = "hook"
	BuilderBaseImage = "jenkinsxio/builder-base:0.0.604"

	Application Kind = "APPLICATION"
	Environment Kind = "ENVIRONMENT"
	Compliance  Kind = "COMPLIANCE"

	ServerlessJenkins = "serverless-jenkins"
	ComplianceCheck   = "compliance-check"
	PromotionBuild    = "promotion-build"

	KnativeBuildAgent = "knative-build"
	KubernetesAgent   = "kubernetes"

	// TODO latest is the wrong thing to do here
	JXImage = "jenkinsxio/jx"
)

type Kind string

const ProwConfigMapName = "config"

// Options for prow
type Options struct {
	KubeClient           kubernetes.Interface
	Repos                []string
	NS                   string
	Kind                 Kind
	DraftPack            string
	EnvironmentNamespace string
}

func add(kubeClient kubernetes.Interface, repos []string, ns string, kind Kind, draftPack, environmentNamespace string) error {

	if len(repos) == 0 {
		return fmt.Errorf("no repo defined")
	}
	o := Options{
		KubeClient:           kubeClient,
		Repos:                repos,
		NS:                   ns,
		Kind:                 kind,
		DraftPack:            draftPack,
		EnvironmentNamespace: environmentNamespace,
	}

	err := o.AddProwConfig()
	if err != nil {
		return err
	}

	return o.AddProwPlugins()
}

func AddEnvironment(kubeClient kubernetes.Interface, repos []string, ns, environmentNamespace string) error {
	return add(kubeClient, repos, ns, Environment, "", environmentNamespace)
}

func AddApplication(kubeClient kubernetes.Interface, repos []string, ns, draftPack string) error {
	return add(kubeClient, repos, ns, Application, draftPack, "")
}

func AddCompliance(kubeClient kubernetes.Interface, repos []string, ns string) error {
	return add(kubeClient, repos, ns, Compliance, "", "")
}

// create Git repo?
// get config and update / overwrite repos?
// should we get the existing CM and do a diff?
// should we just be using git for config and use prow to auto update via gitops?

func (o *Options) createPreSubmitEnvironment() config.Presubmit {
	ps := config.Presubmit{}

	ps.Name = PromotionBuild
	ps.AlwaysRun = true
	ps.SkipReport = false
	ps.Context = PromotionBuild
	ps.Agent = "knative-build"

	spec := &build.BuildSpec{
		Steps: []corev1.Container{
			{
				Image:      BuilderBaseImage,
				Args:       []string{"jx", "step", "helm", "build"},
				WorkingDir: "/workspace/env",
				Env: []corev1.EnvVar{
					{Name: "DEPLOY_NAMESPACE", Value: o.EnvironmentNamespace},
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

	ps.BuildSpec = spec
	ps.RerunCommand = "/test this"
	ps.Trigger = "(?m)^/test( all| this),?(\\s+|$)"

	return ps
}

func (o *Options) createPostSubmitEnvironment() config.Postsubmit {
	ps := config.Postsubmit{}
	ps.Name = "promotion"
	ps.Agent = "knative-build"
	ps.Branches = []string{"master"}

	spec := &build.BuildSpec{
		Steps: []corev1.Container{
			{
				Image:      BuilderBaseImage,
				Args:       []string{"jx", "step", "helm", "apply"},
				WorkingDir: "/workspace/env",
				Env: []corev1.EnvVar{
					{Name: "DEPLOY_NAMESPACE", Value: o.EnvironmentNamespace},
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
	ps.BuildSpec = spec
	return ps
}

func (o *Options) createPostSubmitApplication() config.Postsubmit {
	ps := config.Postsubmit{}
	ps.Branches = []string{"master"}
	ps.Name = "release"
	ps.Agent = "knative-build"

	templateName := fmt.Sprintf("jenkins-%s", o.DraftPack)
	log.Infof("generating prow config, using Knative BuildTemplate %s\n", templateName)

	spec := &build.BuildSpec{
		Template: &build.TemplateInstantiationSpec{
			Name: templateName,
		},
	}

	ps.BuildSpec = spec
	return ps
}

func (o *Options) createPreSubmitApplication() config.Presubmit {
	ps := config.Presubmit{}

	ps.Context = ServerlessJenkins
	ps.Name = ServerlessJenkins
	ps.RerunCommand = "/test this"
	ps.Trigger = "(?m)^/test( all| this),?(\\s+|$)"
	ps.AlwaysRun = false
	ps.SkipReport = false
	ps.Agent = KnativeBuildAgent

	templateName := fmt.Sprintf("jenkins-%s", o.DraftPack)
	log.Infof("generating prow config, using Knative BuildTemplate %s\n", templateName)

	spec := &build.BuildSpec{
		Template: &build.TemplateInstantiationSpec{
			Name: templateName,
		},
	}

	ps.BuildSpec = spec
	ps.RerunCommand = "/test this"
	ps.Trigger = "(?m)^/test( all| this),?(\\s+|$)"

	return ps
}

func (o *Options) createPreSubmitCompliance() config.Presubmit {
	ps := config.Presubmit{}

	ps.Context = ComplianceCheck
	ps.Name = ComplianceCheck
	ps.RerunCommand = "/test compliance"
	ps.Trigger = "(?m)^/test( compliance),?(\\s+|$)"
	ps.AlwaysRun = true
	ps.SkipReport = false
	ps.Agent = KubernetesAgent
	ps.Spec = &corev1.PodSpec{
		Containers: []corev1.Container{
			corev1.Container{
				Image: fmt.Sprintf("%s:%s", JXImage, "1.3.418"),
				Command: []string{
					"jx",
				},
				Args: []string{
					"step",
					"pre",
					"check",
					"compliance",
				},
			},
		},
		ServiceAccountName: "jenkins",
	}
	return ps
}

func (o *Options) createContextPolicyCompliance() config.ContextPolicy {
	cp := config.ContextPolicy{
		Contexts: []string{
			ComplianceCheck,
		},
	}
	return cp
}

func (o *Options) addRepoToTideConfig(t *config.Tide, repo string, kind Kind) error {
	switch o.Kind {
	case Application:
		found := false
		for index, q := range t.Queries {
			if util.Contains(q.Labels, "approved") {
				found = true
				repos := t.Queries[index].Repos
				if !util.Contains(repos, repo) {
					repos = append(repos, repo)
					t.Queries[index].Repos = repos
				}
			}
		}

		if !found {
			log.Infof("Failed to find 'application' tide config, adding...\n")
			t.Queries = append(t.Queries, o.createApplicationTideQuery())
		}
	case Environment:
		found := false
		for index, q := range t.Queries {
			if !util.Contains(q.Labels, "approved") {
				found = true
				repos := t.Queries[index].Repos
				if !util.Contains(repos, repo) {
					repos = append(repos, repo)
					t.Queries[index].Repos = repos
				}
			}
		}

		if !found {
			log.Infof("Failed to find 'environment' tide config, adding...\n")
			t.Queries = append(t.Queries, o.createEnvironmentTideQuery())
		}
	case Compliance:
		// No Tide config needed for Compliance
	default:
		return fmt.Errorf("unknown prow config kind %s", o.Kind)
	}
	return nil
}

func (o *Options) addRepoToBranchProtection(bp *config.BranchProtection, repoSpec string, kind Kind) error {
	bp.ProtectTested = true
	if bp.Orgs == nil {
		bp.Orgs = make(map[string]config.Org, 0)
	}
	s := strings.Split(repoSpec, "/")
	if len(s) != 2 {
		return fmt.Errorf("%s is not of the format org/repo", repoSpec)
	}
	requiredOrg := s[0]
	requiredRepo := s[1]
	if _, ok := bp.Orgs[requiredOrg]; !ok {
		bp.Orgs[requiredOrg] = config.Org{
			Repos: make(map[string]config.Repo, 0),
		}
	}
	if _, ok := bp.Orgs[requiredOrg].Repos[requiredRepo]; !ok {
		bp.Orgs[requiredOrg].Repos[requiredRepo] = config.Repo{
			Policy: config.Policy{
				RequiredStatusChecks: &config.ContextPolicy{},
			},
		}

	}
	if bp.Orgs[requiredOrg].Repos[requiredRepo].Policy.RequiredStatusChecks.Contexts == nil {
		bp.Orgs[requiredOrg].Repos[requiredRepo].Policy.RequiredStatusChecks.Contexts = make([]string, 0)
	}
	contexts := bp.Orgs[requiredOrg].Repos[requiredRepo].Policy.RequiredStatusChecks.Contexts
	switch o.Kind {
	case Application:
		if !util.Contains(contexts, ServerlessJenkins) {
			contexts = append(contexts, ServerlessJenkins)
		}
	case Environment:
		if !util.Contains(contexts, PromotionBuild) {
			contexts = append(contexts, PromotionBuild)
		}
	case Compliance:
		if !util.Contains(contexts, ComplianceCheck) {
			contexts = append(contexts, ComplianceCheck)
		}
	default:
		return fmt.Errorf("unknown prow config kind %s", o.Kind)
	}
	bp.Orgs[requiredOrg].Repos[requiredRepo].Policy.RequiredStatusChecks.Contexts = contexts
	return nil
}

func (o *Options) createApplicationTideQuery() config.TideQuery {
	return config.TideQuery{
		Repos:         []string{"jenkins-x/dummy"},
		Labels:        []string{"approved"},
		MissingLabels: []string{"do-not-merge", "do-not-merge/hold", "do-not-merge/work-in-progress", "needs-ok-to-test", "needs-rebase"},
	}
}

func (o *Options) createEnvironmentTideQuery() config.TideQuery {
	return config.TideQuery{
		Repos:         []string{"jenkins-x/dummy-environment"},
		Labels:        []string{},
		MissingLabels: []string{"do-not-merge", "do-not-merge/hold", "do-not-merge/work-in-progress", "needs-ok-to-test", "needs-rebase"},
	}
}

func (o *Options) createTide() config.Tide {
	// todo get the real URL, though we need to handle the multi cluster use case where dev namespace may be another cluster, so pass it in as an arg?
	t := config.Tide{
		TargetURL: "https://tide.foo.bar",
	}

	var qs []config.TideQuery
	qs = append(qs, o.createApplicationTideQuery())
	qs = append(qs, o.createEnvironmentTideQuery())
	t.Queries = qs

	myTrue := true
	myFalse := false
	t.ContextOptions = config.TideContextPolicyOptions{
		TideContextPolicy: config.TideContextPolicy{
			FromBranchProtection: &myTrue,
			SkipUnknownContexts:  &myFalse,
		},
		//Orgs: orgPolicies,
	}

	return t
}

// AddProwConfig adds config to prow
func (o *Options) AddProwConfig() error {
	var preSubmit config.Presubmit
	var postSubmit config.Postsubmit

	switch o.Kind {
	case Application:
		preSubmit = o.createPreSubmitApplication()
		postSubmit = o.createPostSubmitApplication()
	case Environment:
		preSubmit = o.createPreSubmitEnvironment()
		postSubmit = o.createPostSubmitEnvironment()
	case Compliance:
		preSubmit = o.createPreSubmitCompliance()
	default:
		return fmt.Errorf("unknown prow config kind %s", o.Kind)
	}

	cm, err := o.KubeClient.CoreV1().ConfigMaps(o.NS).Get(ProwConfigMapName, metav1.GetOptions{})
	create := true
	prowConfig := &config.Config{}
	// config doesn't exist, creating
	if err != nil {
		prowConfig.Presubmits = make(map[string][]config.Presubmit)
		prowConfig.Postsubmits = make(map[string][]config.Postsubmit)
		prowConfig.BranchProtection = config.BranchProtection{}
		prowConfig.Tide = o.createTide()
	} else {
		// config exists, updating
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
	prowConfig.PodNamespace = o.NS
	prowConfig.ProwJobNamespace = o.NS

	for _, r := range o.Repos {
		err = o.addRepoToTideConfig(&prowConfig.Tide, r, o.Kind)
		if err != nil {
			return err
		}
		err = o.addRepoToBranchProtection(&prowConfig.BranchProtection, r, o.Kind)
		if err != nil {
			return err
		}
	}

	for _, r := range o.Repos {
		if prowConfig.Presubmits[r] == nil {
			prowConfig.Presubmits[r] = make([]config.Presubmit, 0)
		}
		if prowConfig.Postsubmits[r] == nil {
			prowConfig.Postsubmits[r] = make([]config.Postsubmit, 0)
		}
		if preSubmit.Name != "" {
			found := false
			for i, j := range prowConfig.Presubmits[r] {
				if j.Name == preSubmit.Name {
					found = true
					prowConfig.Presubmits[r][i] = preSubmit
					break
				}
			}
			if !found {
				prowConfig.Presubmits[r] = append(prowConfig.Presubmits[r], preSubmit)
			}
		}
		if postSubmit.Name != "" {
			found := false
			for i, j := range prowConfig.Postsubmits[r] {
				if j.Name == postSubmit.Name {
					found = true
					prowConfig.Postsubmits[r][i] = postSubmit
					break
				}
			}
			if !found {
				prowConfig.Postsubmits[r] = append(prowConfig.Postsubmits[r], postSubmit)
			}
		}
	}

	configYAML, err := yaml.Marshal(prowConfig)
	if err != nil {
		return err
	}

	data := make(map[string]string)
	data["config.yaml"] = string(configYAML)
	cm = &corev1.ConfigMap{
		Data: data,
		ObjectMeta: metav1.ObjectMeta{
			Name: ProwConfigMapName,
		},
	}

	if create {
		// replace with git repository version of a configmap
		_, err = o.KubeClient.CoreV1().ConfigMaps(o.NS).Create(cm)
	} else {
		_, err = o.KubeClient.CoreV1().ConfigMaps(o.NS).Update(cm)
	}

	return err

}

// AddProwPlugins adds plugins to prow
func (o *Options) AddProwPlugins() error {

	pluginsList := []string{"config-updater", "approve", "assign", "blunderbuss", "help", "hold", "lgtm", "lifecycle", "size", "trigger", "wip", "heart", "cat", "override"}

	cm, err := o.KubeClient.CoreV1().ConfigMaps(o.NS).Get("plugins", metav1.GetOptions{})
	create := true
	pluginConfig := &plugins.Configuration{}
	if err != nil {
		pluginConfig.Plugins = make(map[string][]string)
		pluginConfig.Approve = []plugins.Approve{}

		pluginConfig.ConfigUpdater.Maps = make(map[string]plugins.ConfigMapSpec)
		pluginConfig.ConfigUpdater.Maps["prow/config.yaml"] = plugins.ConfigMapSpec{Name: ProwConfigMapName}
		pluginConfig.ConfigUpdater.Maps["prow/plugins.yaml"] = plugins.ConfigMapSpec{Name: "plugins"}

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
	cm = &corev1.ConfigMap{
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
