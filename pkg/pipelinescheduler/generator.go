package pipelinescheduler

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	v1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/client-go/kubernetes"

	yaml "gopkg.in/yaml.v2"

	"k8s.io/helm/pkg/proto/hapi/chart"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"

	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/environments"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/pipelinescheduler/prow"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/plugins"
)

const (
	orgLabel         = "jenkins.io/git-org"
	repoLabel        = "jenkins.io/git-repo"
	parentAnnotation = "jenkins.io/parent-scheduler"
)

// GenerateProw will generate the prow config for the namespace
func GenerateProw(jxClient versioned.Interface, namespace string) (*config.Config,
	*plugins.Configuration, error) {
	schedulers, err := jxClient.JenkinsV1().Schedulers(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	if len(schedulers.Items) == 0 {
		log.Infof("No schedulers found\n")
	}
	leaves := make([]*SchedulerLeaf, 0)
	lookup := make(map[string]*jenkinsv1.Scheduler)
	for _, item := range schedulers.Items {
		lookup[item.Name] = &item
	}
	for _, s := range lookup {
		org, orgOk := s.Labels[orgLabel]
		repo, repoOk := s.Labels[repoLabel]
		if orgOk && repoOk {
			schedulers := []*jenkinsv1.SchedulerSpec{
				&s.Spec,
			}
			// locate the parents
			for {
				if parentName, ok := s.Annotations[parentAnnotation]; ok {
					if parent, ok := lookup[parentName]; ok {
						// prepend the parent
						schedulers = append([]*jenkinsv1.SchedulerSpec{&parent.Spec}, schedulers...)
						continue
					} else {
						log.Warnf("no parent %s available", parentName)
					}
				}
				break
			}
			merged, err := Build(schedulers)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "building scheduler")
			}
			leaves = append(leaves, &SchedulerLeaf{
				Repo:          repo,
				Org:           org,
				SchedulerSpec: merged,
			})
		}
		// Otherwise ignore
	}
	cfg, plugs, err := prow.Build(leaves)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "building prow config")
	}
	return cfg, plugs, nil
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
			Name: "config",
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
			Name: "plugins",
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
	ConfigureGitFn  environments.ConfigureGitFn
	Gitter          gits.Gitter
	Verbose         bool
	Helmer          helm.Helmer
	GitProvider     gits.GitProvider
	DevEnv          *jenkinsv1.Environment
	EnvironmentsDir string
}

// AddToEnvironmentRepo adds the prow config to the gitops environment repo
func (o *GitOpsOptions) AddToEnvironmentRepo(cfg *config.Config, plugs *plugins.Configuration) error {
	details := environments.PullRequestDetails{
		BranchName: "add-prow-config",
		Title:      fmt.Sprintf("Add Prow config"),
		Message:    fmt.Sprintf("Add Prow config generated on %s", time.Now),
	}

	modifyChartFn := func(requirements *helm.Requirements, metadata *chart.Metadata,
		existingValues map[string]interface{},
		templates map[string]string, dir string, pullRequestDetails *environments.PullRequestDetails) error {
		prowDir := filepath.Join(dir, "prow")
		err := os.MkdirAll(prowDir, 0700)
		if err != nil {
			return errors.Wrapf(err, "creating prow dir in gitops repo %s")
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

	info, err := options.Create(o.DevEnv, o.EnvironmentsDir, &details, nil)

	if err != nil {
		return errors.Wrapf(err, "creating pr for prow config")
	}
	log.Infof("Added prow config via Pull Request %s\n", info.PullRequest.URL)
	return nil
}
