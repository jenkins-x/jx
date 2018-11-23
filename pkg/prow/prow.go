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
	Hook = "hook"

	Application Kind = "APPLICATION"
	Environment Kind = "ENVIRONMENT"
	Protection  Kind = "PROTECTION"

	ServerlessJenkins = "serverless-jenkins"
	ComplianceCheck   = "compliance-check"
	PromotionBuild    = "promotion-build"

	KnativeBuildAgent = "knative-build"
	KubernetesAgent   = "kubernetes"

	applyTemplate = "environment-apply"
	buildTemplate = "environment-build"

	serviceAccountApply = "helm"
	serviceAccountBuild = "knative-build-bot"
)

type Kind string

const (
	ProwConfigMapName        = "config"
	ProwPluginsConfigMapName = "plugins"
	ProwConfigFilename       = "config.yaml"
	ProwPluginsFilename      = "plugins.yaml"
)

// Options for Prow
type Options struct {
	KubeClient           kubernetes.Interface
	Repos                []string
	NS                   string
	Kind                 Kind
	DraftPack            string
	EnvironmentNamespace string
	Context              string
}

func add(kubeClient kubernetes.Interface, repos []string, ns string, kind Kind, draftPack, environmentNamespace string, context string) error {

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
		Context:              context,
	}

	err := o.AddProwConfig()
	if err != nil {
		return err
	}

	return o.AddProwPlugins()
}

func AddEnvironment(kubeClient kubernetes.Interface, repos []string, ns, environmentNamespace string) error {
	return add(kubeClient, repos, ns, Environment, "", environmentNamespace, "")
}

func AddApplication(kubeClient kubernetes.Interface, repos []string, ns, draftPack string) error {
	return add(kubeClient, repos, ns, Application, draftPack, "", "")
}

func AddProtection(kubeClient kubernetes.Interface, repos []string, context string, ns string) error {
	return add(kubeClient, repos, ns, Protection, "", "", context)
}

// create Git repo?
// get config and update / overwrite repos?
// should we get the existing CM and do a diff?
// should we just be using git for config and use Prow to auto update via gitops?

func (o *Options) createPreSubmitEnvironment() config.Presubmit {
	ps := config.Presubmit{}

	ps.Name = PromotionBuild
	ps.AlwaysRun = true
	ps.SkipReport = false
	ps.Context = PromotionBuild
	ps.Agent = KnativeBuildAgent

	spec := &build.BuildSpec{
		ServiceAccountName: serviceAccountBuild,
		Template: &build.TemplateInstantiationSpec{
			Name: buildTemplate,
		},
	}

	ps.BuildSpec = spec
	ps.RerunCommand = "/test this"
	ps.Trigger = "(?m)^/test( all| this),?(\\s+|$)"

	return ps
}

func (o *Options) createPostSubmitEnvironment() config.Postsubmit {
	ps := config.Postsubmit{}
	ps.Name = "promotion"
	ps.Agent = KnativeBuildAgent
	ps.Branches = []string{"master"}

	spec := &build.BuildSpec{
		ServiceAccountName: serviceAccountApply,
		Template: &build.TemplateInstantiationSpec{
			Name: applyTemplate,
			Env: []corev1.EnvVar{
				{Name: "DEPLOY_NAMESPACE", Value: o.EnvironmentNamespace},
			},
		},
	}
	ps.BuildSpec = spec
	return ps
}

func (o *Options) createPostSubmitApplication() config.Postsubmit {
	ps := config.Postsubmit{}
	ps.Branches = []string{"master"}
	ps.Name = "release"
	ps.Agent = KnativeBuildAgent

	templateName := fmt.Sprintf("jenkins-%s", o.DraftPack)
	log.Infof("generating Prow config, using Knative BuildTemplate %s\n", templateName)

	spec := &build.BuildSpec{
		ServiceAccountName: serviceAccountBuild,
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
	log.Infof("generating Prow config, using Knative BuildTemplate %s\n", templateName)

	spec := &build.BuildSpec{
		ServiceAccountName: serviceAccountBuild,
		Template: &build.TemplateInstantiationSpec{
			Name: templateName,
		},
	}

	ps.BuildSpec = spec
	ps.RerunCommand = "/test this"
	ps.Trigger = "(?m)^/test( all| this),?(\\s+|$)"

	return ps
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
	case Protection:
		// No Tide config needed for Protection
	default:
		return fmt.Errorf("unknown Prow config kind %s", o.Kind)
	}
	return nil
}

func (o *Options) addRepoToBranchProtection(bp *config.BranchProtection, repoSpec string, context string, kind Kind) error {
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
	case Protection:
		if !util.Contains(contexts, ComplianceCheck) {
			contexts = append(contexts, context)
		}
	default:
		return fmt.Errorf("unknown Prow config kind %s", o.Kind)
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

// AddProwConfig adds config to Prow
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
	case Protection:
		// Nothing needed
	default:
		return fmt.Errorf("unknown Prow config kind %s", o.Kind)
	}

	prowConfig, create, err := o.GetProwConfig()
	if err != nil {
		return err
	}

	prowConfig.PodNamespace = o.NS
	prowConfig.ProwJobNamespace = o.NS

	for _, r := range o.Repos {
		err = o.addRepoToTideConfig(&prowConfig.Tide, r, o.Kind)
		if err != nil {
			return err
		}
		err = o.addRepoToBranchProtection(&prowConfig.BranchProtection, r, o.Context, o.Kind)
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
	data[ProwConfigFilename] = string(configYAML)
	cm := &corev1.ConfigMap{
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

func (o *Options) GetProwConfig() (*config.Config, bool, error) {
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
		err = yaml.Unmarshal([]byte(cm.Data[ProwConfigFilename]), &prowConfig)
		if err != nil {
			return prowConfig, create, err
		}
		if len(prowConfig.Presubmits) == 0 {
			prowConfig.Presubmits = make(map[string][]config.Presubmit)
		}
		if len(prowConfig.Postsubmits) == 0 {
			prowConfig.Postsubmits = make(map[string][]config.Postsubmit)
		}
		if prowConfig.BranchProtection.Orgs == nil {
			prowConfig.BranchProtection.Orgs = make(map[string]config.Org, 0)
		}
	}
	return prowConfig, create, nil
}

func (o *Options) GetAllBranchProtectionContexts(org string, repo string) ([]string, error) {
	result := make([]string, 0)
	prowConfig, _, err := o.GetProwConfig()
	if err != nil {
		return result, err
	}
	prowOrg, ok := prowConfig.BranchProtection.Orgs[org]
	if !ok {
		prowOrg = config.Org{}
	}
	if prowOrg.Repos == nil {
		prowOrg.Repos = make(map[string]config.Repo, 0)
	}
	prowRepo, ok := prowOrg.Repos[repo]
	if !ok {
		prowRepo = config.Repo{}
	}
	if prowRepo.RequiredStatusChecks == nil {
		prowRepo.RequiredStatusChecks = &config.ContextPolicy{}
	}
	return prowRepo.RequiredStatusChecks.Contexts, nil
}

func (o *Options) GetBranchProtectionContexts(org string, repo string) ([]string, error) {
	result := make([]string, 0)
	contexts, err := o.GetAllBranchProtectionContexts(org, repo)
	if err != nil {
		return result, err
	}
	for _, c := range contexts {
		if c != ServerlessJenkins && c != PromotionBuild {
			result = append(result, c)
		}
	}
	return result, nil
}

// AddProwPlugins adds plugins to prow
func (o *Options) AddProwPlugins() error {

	pluginsList := []string{"config-updater", "approve", "assign", "blunderbuss", "help", "hold", "lgtm", "lifecycle", "size", "trigger", "wip", "heart", "cat", "override"}

	cm, err := o.KubeClient.CoreV1().ConfigMaps(o.NS).Get(ProwPluginsConfigMapName, metav1.GetOptions{})
	create := true
	pluginConfig := &plugins.Configuration{}
	if err != nil {
		pluginConfig.Plugins = make(map[string][]string)
		pluginConfig.Approve = []plugins.Approve{}

		pluginConfig.ConfigUpdater.Maps = make(map[string]plugins.ConfigMapSpec)
		pluginConfig.ConfigUpdater.Maps["prow/config.yaml"] = plugins.ConfigMapSpec{Name: ProwConfigMapName}
		pluginConfig.ConfigUpdater.Maps["prow/plugins.yaml"] = plugins.ConfigMapSpec{Name: ProwPluginsConfigMapName}

	} else {
		create = false
		err = yaml.Unmarshal([]byte(cm.Data[ProwPluginsFilename]), &pluginConfig)
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
	data[ProwPluginsFilename] = string(pluginYAML)
	cm = &corev1.ConfigMap{
		Data: data,
		ObjectMeta: metav1.ObjectMeta{
			Name: ProwPluginsConfigMapName,
		},
	}
	if create {
		_, err = o.KubeClient.CoreV1().ConfigMaps(o.NS).Create(cm)
	} else {
		_, err = o.KubeClient.CoreV1().ConfigMaps(o.NS).Update(cm)
	}

	return err
}

func (o *Options) GetReleaseJobs() ([]string, error) {
	cm, err := o.KubeClient.CoreV1().ConfigMaps(o.NS).Get(ProwConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	prowConfig := &config.Config{}
	err = yaml.Unmarshal([]byte(cm.Data[ProwConfigFilename]), &prowConfig)
	if err != nil {
		return nil, err
	}
	var jobs []string

	for repo, p := range prowConfig.Postsubmits {
		for _, q := range p {
			for _, b := range q.Branches {
				repo = strings.Replace(repo, ":", "", -1)
				jobName := fmt.Sprintf("%s/%s", repo, b)
				jobs = append(jobs, jobName)
			}
		}
	}
	return jobs, nil
}

func (o *Options) GetBuildSpec(org, repo, branch string) (*build.BuildSpec, error) {

	cm, err := o.KubeClient.CoreV1().ConfigMaps(o.NS).Get(ProwConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	prowConfig := &config.Config{}
	err = yaml.Unmarshal([]byte(cm.Data[ProwConfigFilename]), &prowConfig)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s/%s", org, repo)
	for _, p := range prowConfig.Postsubmits[key] {

		for _, a := range p.Branches {
			if a == branch {
				return p.BuildSpec, nil
			}
		}
	}
	return nil, fmt.Errorf("no prow config build spec found for %s/%s/%s", org, repo, branch)
}
