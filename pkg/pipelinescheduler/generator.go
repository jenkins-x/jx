package pipelinescheduler

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/environments"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	uuid "github.com/satori/go.uuid"
	v1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/helm/pkg/proto/hapi/chart"

	"k8s.io/client-go/kubernetes"

	"github.com/ghodss/yaml"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/plugins"
)

// GenerateProw will generate the prow config for the namespace
func GenerateProw(gitOps bool, jxClient versioned.Interface, namespace string, teamSchedulerName string, devEnv *jenkinsv1.Environment) (*config.Config,
	*plugins.Configuration, error) {
	var defaultScheduler *jenkinsv1.Scheduler
	var err error
	if teamSchedulerName != "" {
		defaultScheduler, err = jxClient.JenkinsV1().Schedulers(namespace).Get(teamSchedulerName, metav1.GetOptions{})
		if err != nil {
			return nil, nil, errors.Wrapf(err, "Error finding default scheduler in team settings")
		}
	}
	schedulers, err := jxClient.JenkinsV1().Schedulers(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	if len(schedulers.Items) == 0 {
		return nil, nil, errors.New("No pipeline schedulers are configured")
	}
	leaves := make([]*SchedulerLeaf, 0)
	lookup := make(map[string]*jenkinsv1.Scheduler)
	for _, item := range schedulers.Items {
		lookup[item.Name] = item.DeepCopy()
	}
	// Process Schedulers linked to SourceRepositoryGroups
	sourceRepoGroups, err := jxClient.JenkinsV1().SourceRepositoryGroups(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Error finding source repository groups")
	}
	// Process Schedulers linked to SourceRepositoryGroups
	sourceRepos, err := jxClient.JenkinsV1().SourceRepositories(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Error finding source repositories")
	}
	for _, sourceRepo := range sourceRepos.Items {
		applicableSchedulers := []*jenkinsv1.SchedulerSpec{}
		// Apply config-updater to devEnv
		applicableSchedulers = addConfigUpdaterToDevEnv(gitOps, applicableSchedulers, devEnv, &sourceRepo.Spec)
		// Apply repo scheduler
		applicableSchedulers = addRepositoryScheduler(sourceRepo, lookup, applicableSchedulers)
		// Apply project schedulers
		applicableSchedulers = addProjectSchedulers(sourceRepoGroups, sourceRepo, lookup, applicableSchedulers)
		// Apply team scheduler
		applicableSchedulers = addTeamScheduler(teamSchedulerName, defaultScheduler, applicableSchedulers)
		if len(applicableSchedulers) < 1 {
			continue
		}
		merged, err := Build(applicableSchedulers)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "building scheduler")
		}
		leaves = append(leaves, &SchedulerLeaf{
			Repo:          sourceRepo.Spec.Repo,
			Org:           sourceRepo.Spec.Org,
			SchedulerSpec: merged,
		})
		if err != nil {
			return nil, nil, errors.Wrapf(err, "building prow config")
		}
	}
	cfg, plugs, err := BuildProwConfig(leaves)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "building prow config")
	}
	if cfg != nil {
		cfg.PodNamespace = namespace
		cfg.ProwJobNamespace = namespace
	}
	return cfg, plugs, nil
}

func addTeamScheduler(defaultSchedulerName string, defaultScheduler *jenkinsv1.Scheduler, applicableSchedulers []*jenkinsv1.SchedulerSpec) []*jenkinsv1.SchedulerSpec {
	if defaultSchedulerName != "" {
		if defaultScheduler != nil {
			applicableSchedulers = append([]*jenkinsv1.SchedulerSpec{&defaultScheduler.Spec}, applicableSchedulers...)
		} else {
			log.Logger().Warnf("A team pipeline scheduler named %s was configured but could not be found", defaultSchedulerName)
		}
	}
	return applicableSchedulers
}

func addRepositoryScheduler(sourceRepo jenkinsv1.SourceRepository, lookup map[string]*jenkinsv1.Scheduler, applicableSchedulers []*jenkinsv1.SchedulerSpec) []*jenkinsv1.SchedulerSpec {
	if sourceRepo.Spec.Scheduler.Name != "" {
		scheduler := lookup[sourceRepo.Spec.Scheduler.Name]
		if scheduler != nil {
			applicableSchedulers = append([]*jenkinsv1.SchedulerSpec{&scheduler.Spec}, applicableSchedulers...)
		} else {
			log.Logger().Warnf("A scheduler named %s is referenced by repository(%s) but could not be found", sourceRepo.Spec.Scheduler.Name, sourceRepo.Name)
		}
	}
	return applicableSchedulers
}

func addProjectSchedulers(sourceRepoGroups *jenkinsv1.SourceRepositoryGroupList, sourceRepo jenkinsv1.SourceRepository, lookup map[string]*jenkinsv1.Scheduler, applicableSchedulers []*jenkinsv1.SchedulerSpec) []*jenkinsv1.SchedulerSpec {
	for _, sourceGroup := range sourceRepoGroups.Items {
		for _, groupRepo := range sourceGroup.Spec.SourceRepositorySpec {
			if groupRepo.Name == sourceRepo.Name {
				if sourceGroup.Spec.Scheduler.Name != "" {
					scheduler := lookup[sourceGroup.Spec.Scheduler.Name]
					if scheduler != nil {
						applicableSchedulers = append([]*jenkinsv1.SchedulerSpec{&scheduler.Spec}, applicableSchedulers...)
					} else {
						log.Logger().Warnf("A scheduler named %s is referenced by repository group(%s) but could not be found", sourceGroup.Spec.Scheduler.Name, sourceGroup.Name)
					}
				}
			}
		}
	}
	return applicableSchedulers
}

func addConfigUpdaterToDevEnv(gitOps bool, applicableSchedulers []*jenkinsv1.SchedulerSpec, devEnv *jenkinsv1.Environment, sourceRepo *jenkinsv1.SourceRepositorySpec) []*jenkinsv1.SchedulerSpec {
	if gitOps && strings.Contains(devEnv.Spec.Source.URL, sourceRepo.Org+"/"+sourceRepo.Repo) {
		maps := make(map[string]jenkinsv1.ConfigMapSpec)
		maps["env/prow/config.yaml"] = jenkinsv1.ConfigMapSpec{
			Name: "config",
		}
		maps["env/prow/plugins.yaml"] = jenkinsv1.ConfigMapSpec{
			Name: "plugins",
		}
		environmentUpdaterSpec := &jenkinsv1.SchedulerSpec{
			ConfigUpdater: &jenkinsv1.ConfigUpdater{
				Map: maps,
			},
			Plugins: &jenkinsv1.ReplaceableSliceOfStrings{
				Items: []string{"config-updater"},
			},
		}
		applicableSchedulers = append([]*jenkinsv1.SchedulerSpec{environmentUpdaterSpec}, applicableSchedulers...)
	}
	return applicableSchedulers
}

//ApplyDirectly directly applies the prow config to the cluster
func ApplyDirectly(kubeClient kubernetes.Interface, namespace string, cfg *config.Config,
	plugs *plugins.Configuration) error {
	cfgYaml, err := yaml.Marshal(cfg)
	if err != nil {
		return errors.Wrapf(err, "marshalling config to yaml")
	}
	cfgConfigMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "config",
			Namespace: namespace,
		},
		Data: map[string]string{
			"config.yaml": string(cfgYaml),
		},
	}
	plugsYaml, err := yaml.Marshal(plugs)
	if err != nil {
		return errors.Wrapf(err, "marshalling plugins to yaml")
	}
	plugsConfigMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "plugins",
			Namespace: namespace,
		},
		Data: map[string]string{
			"plugins.yaml": string(plugsYaml),
		},
	}
	_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(cfgConfigMap)
	if kubeerrors.IsNotFound(err) {
		_, err := kubeClient.CoreV1().ConfigMaps(namespace).Create(cfgConfigMap)
		if err != nil {
			return errors.Wrapf(err, "creating ConfigMap config")
		}
	} else if err != nil {
		return errors.Wrapf(err, "updating ConfigMap config")
	}
	_, err = kubeClient.CoreV1().ConfigMaps(namespace).Update(plugsConfigMap)
	if kubeerrors.IsNotFound(err) {
		_, err := kubeClient.CoreV1().ConfigMaps(namespace).Create(plugsConfigMap)
		if err != nil {
			return errors.Wrapf(err, "creating ConfigMap plugins")
		}
	} else if err != nil {
		return errors.Wrapf(err, "updating ConfigMap plugins")
	}
	return nil
}

//GitOpsOptions are options for running AddToEnvironmentRepo
type GitOpsOptions struct {
	ConfigureGitFn  gits.ConfigureGitFn
	Gitter          gits.Gitter
	Verbose         bool
	Helmer          helm.Helmer
	GitProvider     gits.GitProvider
	DevEnv          *jenkinsv1.Environment
	EnvironmentsDir string
}

// AddToEnvironmentRepo adds the prow config to the gitops environment repo
func (o *GitOpsOptions) AddToEnvironmentRepo(cfg *config.Config, plugs *plugins.Configuration) error {
	branchNameUUID, err := uuid.NewV4()
	if err != nil {
		return errors.Wrapf(err, "creating creating branch name")
	}
	validBranchName := "add-prow-config" + branchNameUUID.String()
	details := gits.PullRequestDetails{
		BranchName: validBranchName,
		Title:      fmt.Sprintf("Add Prow config"),
		Message:    fmt.Sprintf("Add Prow config generated on %s", time.Now()),
	}

	modifyChartFn := func(requirements *helm.Requirements, metadata *chart.Metadata,
		existingValues map[string]interface{},
		templates map[string]string, dir string, pullRequestDetails *gits.PullRequestDetails) error {
		prowDir := filepath.Join(dir, "prow")
		err := os.MkdirAll(prowDir, 0700)
		if err != nil {
			return errors.Wrapf(err, "creating prow dir in gitops repo %s", prowDir)
		}
		cfgBytes, err := yaml.Marshal(cfg)
		if err != nil {
			return errors.Wrapf(err, "marshaling prow config to yaml")
		}
		cfgPath := filepath.Join(prowDir, "config.yaml")
		err = ioutil.WriteFile(cfgPath, cfgBytes, 0600)
		if err != nil {
			return errors.Wrapf(err, "writing %s", cfgPath)
		}

		plugsBytes, err := yaml.Marshal(plugs)
		if err != nil {
			return errors.Wrapf(err, "marshaling prow plugins config to yaml")
		}
		plugsPath := filepath.Join(prowDir, "plugins.yaml")
		err = ioutil.WriteFile(plugsPath, plugsBytes, 0600)
		if err != nil {
			return errors.Wrapf(err, "writing %s", plugsPath)
		}
		return nil
	}

	options := environments.EnvironmentPullRequestOptions{
		ConfigGitFn:   o.ConfigureGitFn,
		Gitter:        o.Gitter,
		ModifyChartFn: modifyChartFn,
		GitProvider:   o.GitProvider,
	}

	info, err := options.Create(o.DevEnv, o.EnvironmentsDir, &details, nil, "", false)

	if err != nil {
		return errors.Wrapf(err, "creating pr for prow config")
	}
	if info != nil {
		log.Logger().Infof("Added prow config via Pull Request %s\n", info.PullRequest.URL)
	}
	return nil
}
