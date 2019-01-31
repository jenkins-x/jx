package prow

import (
	"encoding/json"
	//"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/ghodss/yaml"
	prowconfig "github.com/jenkins-x/jx/pkg/prow/config"
	build "github.com/knative/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/plugins"
)

const (
	Hook = "hook"

	KnativeBuildAgent = "knative-build"
	KubernetesAgent   = "kubernetes"

	applyTemplate = "environment-apply"
	buildTemplate = "environment-build"

	serviceAccountApply = "helm"
	serviceAccountBuild = "knative-build-bot"
)

const (
	ProwConfigMapName           = "config"
	ProwPluginsConfigMapName    = "plugins"
	ProwExternalPluginsFilename = "external-plugins.yaml"
	ProwConfigFilename          = "config.yaml"
	ProwPluginsFilename         = "plugins.yaml"
)

// Options for Prow
type Options struct {
	KubeClient           kubernetes.Interface
	Repos                []string
	NS                   string
	Kind                 prowconfig.Kind
	DraftPack            string
	EnvironmentNamespace string
	Context              string
}

type ExternalPlugins struct {
	Items []plugins.ExternalPlugin
}

func add(kubeClient kubernetes.Interface, repos []string, ns string, kind prowconfig.Kind, draftPack, environmentNamespace string, context string) error {

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

func remove(kubeClient kubernetes.Interface, repos []string, ns string, kind prowconfig.Kind) error {
	if len(repos) == 0 {
		return fmt.Errorf("no repo defined")
	}
	o := Options{
		KubeClient: kubeClient,
		Repos:      repos,
		NS:         ns,
		Kind:       kind,
	}

	return o.RemoveProwConfig()
}

func AddEnvironment(kubeClient kubernetes.Interface, repos []string, ns, environmentNamespace string) error {
	return add(kubeClient, repos, ns, prowconfig.Environment, "", environmentNamespace, "")
}

func AddApplication(kubeClient kubernetes.Interface, repos []string, ns, draftPack string) error {
	return add(kubeClient, repos, ns, prowconfig.Application, draftPack, "", "")
}

// DeleteApplication will delete the Prow configuration for a given set of repositories
func DeleteApplication(kubeClient kubernetes.Interface, repos []string, ns string) error {
	return remove(kubeClient, repos, ns, prowconfig.Application)
}

func AddProtection(kubeClient kubernetes.Interface, repos []string, context string, ns string) error {
	return add(kubeClient, repos, ns, prowconfig.Protection, "", "", context)
}

// AddExternalPlugins adds one or more external plugins to the specified repos. If repos is nil,
// then the external plugins will be added to all repos that have plugins
func AddExternalPlugins(kubeClient kubernetes.Interface, repos []string, ns string,
	add ...plugins.ExternalPlugin) error {
	o := Options{
		KubeClient: kubeClient,
		NS:         ns,
	}
	err := o.AddExternalProwPlugins(add)
	if err != nil {
		return err
	}
	return o.AddProwPlugins()
}

// create Git repo?
// get config and update / overwrite repos?
// should we get the existing CM and do a diff?
// should we just be using git for config and use Prow to auto update via gitops?

func (o *Options) createPreSubmitEnvironment() config.Presubmit {
	ps := config.Presubmit{}

	ps.Name = prowconfig.PromotionBuild
	ps.AlwaysRun = true
	ps.SkipReport = false
	ps.Context = prowconfig.PromotionBuild
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

	ps.Context = prowconfig.ServerlessJenkins
	ps.Name = prowconfig.ServerlessJenkins
	ps.RerunCommand = "/test this"
	ps.Trigger = "(?m)^/test( all| this),?(\\s+|$)"
	ps.AlwaysRun = true
	ps.SkipReport = false
	ps.Agent = KnativeBuildAgent

	templateName := fmt.Sprintf("jenkins-%s", o.DraftPack)

	spec := &build.BuildSpec{
		ServiceAccountName: serviceAccountApply,
		Template: &build.TemplateInstantiationSpec{
			Name: templateName,
		},
	}

	ps.BuildSpec = spec
	ps.RerunCommand = "/test this"
	ps.Trigger = "(?m)^/test( all| this),?(\\s+|$)"

	return ps
}

// AddProwConfig adds config to Prow
func (o *Options) AddProwConfig() error {
	var preSubmit config.Presubmit
	var postSubmit config.Postsubmit

	switch o.Kind {
	case prowconfig.Application:
		preSubmit = o.createPreSubmitApplication()
		postSubmit = o.createPostSubmitApplication()
	case prowconfig.Environment:
		preSubmit = o.createPreSubmitEnvironment()
		postSubmit = o.createPostSubmitEnvironment()
	case prowconfig.Protection:
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
		err = prowconfig.AddRepoToTideConfig(&prowConfig.Tide, r, o.Kind)
		if err != nil {
			return err
		}
		err = prowconfig.AddRepoToBranchProtection(&prowConfig.BranchProtection, r, o.Context, o.Kind)
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

	return o.saveProwConfig(prowConfig, create)
}

// RemoveProwConfig deletes a config (normally a repository integration) from Prow
func (o *Options) RemoveProwConfig() error {
	prowConfig, created, err := o.GetProwConfig()
	if created {
		return errors.New("no existing prow config. Nothing to remove")
	}
	if err != nil {
		return errors.Wrap(err, "getting existing prow config")
	}

	for _, repo := range o.Repos {
		err = prowconfig.RemoveRepoFromTideConfig(&prowConfig.Tide, repo, o.Kind)
		if err != nil {
			return errors.Wrapf(err, "removing repo %s from tide config", repo)
		}
		err = prowconfig.RemoveRepoFromBranchProtection(&prowConfig.BranchProtection, repo)
		if err != nil {
			return errors.Wrapf(err, "removing repo %s from branch protection", repo)
		}

		delete(prowConfig.Presubmits, repo)
		delete(prowConfig.Postsubmits, repo)
	}

	return o.saveProwConfig(prowConfig, created)
}

func (o *Options) saveProwConfig(prowConfig *config.Config, create bool) error {
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
		prowConfig.Tide = prowconfig.CreateTide()
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

func (o *Options) upsertPluginConfig(closure func(pluginConfig *plugins.Configuration,
	externalPlugins *ExternalPlugins) error) error {
	cm, err := o.KubeClient.CoreV1().ConfigMaps(o.NS).Get(ProwPluginsConfigMapName, metav1.GetOptions{})
	create := true
	pluginConfig := &plugins.Configuration{}
	externalPlugins := &ExternalPlugins{}
	if err == nil {
		create = false
		err = yaml.Unmarshal([]byte(cm.Data[ProwPluginsFilename]), pluginConfig)
		if err != nil {
			return err
		}
		err = yaml.Unmarshal([]byte(cm.Data[ProwExternalPluginsFilename]), externalPlugins)
		if err != nil {
			return err
		}
	}

	if pluginConfig == nil {
		pluginConfig = &plugins.Configuration{}
		pluginConfig.ConfigUpdater.Maps = make(map[string]plugins.ConfigMapSpec)
		pluginConfig.ConfigUpdater.Maps["prow/config.yaml"] = plugins.ConfigMapSpec{Name: ProwConfigMapName}
		pluginConfig.ConfigUpdater.Maps["prow/plugins.yaml"] = plugins.ConfigMapSpec{Name: ProwPluginsConfigMapName}

	}
	if len(pluginConfig.Plugins) == 0 {
		pluginConfig.Plugins = make(map[string][]string)
	}
	if len(pluginConfig.ExternalPlugins) == 0 {
		pluginConfig.ExternalPlugins = make(map[string][]plugins.ExternalPlugin)
	}
	if len(pluginConfig.Approve) == 0 {
		pluginConfig.Approve = []plugins.Approve{}
	}
	if len(pluginConfig.Welcome) == 0 {
		pluginConfig.Welcome = []plugins.Welcome{
			{
				MessageTemplate: "Welcome",
			},
		}
	}

	err = closure(pluginConfig, externalPlugins)
	if err != nil {
		return err
	}

	pluginYAML, err := yaml.Marshal(pluginConfig)
	if err != nil {
		return err
	}

	externalPluginsYAML, err := yaml.Marshal(externalPlugins)
	if err != nil {
		return err
	}

	data := make(map[string]string)
	data[ProwPluginsFilename] = string(pluginYAML)
	data[ProwExternalPluginsFilename] = string(externalPluginsYAML)
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

// AddProwPlugins adds plugins and external plugins to prow for any repos defined in o.Repos,
// or for all repos which have plugins if o.Repos is nil
func (o *Options) AddProwPlugins() error {
	pluginsList := []string{"config-updater", "approve", "assign", "blunderbuss", "help", "hold", "lgtm", "lifecycle", "size", "trigger", "wip", "heart", "cat", "override"}
	closure := func(pluginConfig *plugins.Configuration, externalPlugins *ExternalPlugins) error {
		if o.Repos == nil {
			// Then we need react for all repos defined in the plugins list
			o.Repos = make([]string, 0)
			for r, _ := range pluginConfig.Plugins {
				o.Repos = append(o.Repos, r)
			}
		}
		for _, r := range o.Repos {
			pluginConfig.Plugins[r] = pluginsList

			a := plugins.Approve{
				Repos: []string{r},
				//ReviewActsAsApprove: true,
				LgtmActsAsApprove: true,
			}
			pluginConfig.Approve = append(pluginConfig.Approve, a)

			parts := strings.Split(r, "/")
			t := plugins.Trigger{
				Repos:      []string{r},
				TrustedOrg: parts[0],
			}
			pluginConfig.Triggers = append(pluginConfig.Triggers, t)

			// External Plugins
			pluginConfig.ExternalPlugins[r] = externalPlugins.Items
		}
		return nil
	}
	return o.upsertPluginConfig(closure)
}

func (o *Options) AddExternalProwPlugins(adds []plugins.ExternalPlugin) error {
	closure := func(pluginConfig *plugins.Configuration, externalPlugins *ExternalPlugins) error {
		for _, add := range adds {
			foundIndex := -1
			for i, p := range externalPlugins.Items {
				if p.Name == add.Name {
					foundIndex = i
					break
				}
			}
			if foundIndex < 0 {
				externalPlugins.Items = append(externalPlugins.Items, add)
			} else {
				externalPlugins.Items[foundIndex] = add
			}
		}
		return nil
	}
	return o.upsertPluginConfig(closure)
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

func (o *Options) GetPostSubmitJob(org, repo, branch string) (config.Postsubmit, error) {

	p := config.Postsubmit{}
	cm, err := o.KubeClient.CoreV1().ConfigMaps(o.NS).Get(ProwConfigMapName, metav1.GetOptions{})
	if err != nil {
		return p, err
	}

	prowConfig := &config.Config{}
	err = yaml.Unmarshal([]byte(cm.Data[ProwConfigFilename]), &prowConfig)
	if err != nil {
		return p, err
	}

	key := fmt.Sprintf("%s/%s", org, repo)
	for _, p := range prowConfig.Postsubmits[key] {

		for _, a := range p.Branches {
			if a == branch {
				return p, nil
			}
		}
	}
	return p, fmt.Errorf("no prow config build spec found for %s/%s/%s", org, repo, branch)
}

func CreateProwJob(client kubernetes.Interface, ns string, j prowapi.ProwJob) (prowapi.ProwJob, error) {
	retJob := prowapi.ProwJob{}
	body, err := json.Marshal(j)
	if err != nil {
		return retJob, err
	}
	resp, err := client.CoreV1().RESTClient().Post().RequestURI(fmt.Sprintf("/apis/prow.k8s.io/v1/namespaces/%s/prowjobs", ns)).Body(body).DoRaw()
	if err != nil {
		return retJob, fmt.Errorf("failed to create prowjob %v: %s", err, string(resp))
	}
	return retJob, err
}
