package kube

import (
	"fmt"
	"io"
	"io/ioutil"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	survey "gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var useForkForEnvGitRepo = false

// CreateEnvironmentSurvey creates a Survey on the given environment using the default options
// from the CLI
func CreateEnvironmentSurvey(batchMode bool, authConfigSvc auth.ConfigService, devEnv *v1.Environment, data *v1.Environment,
	config *v1.Environment, forkEnvGitURL string, ns string, jxClient versioned.Interface, kubeClient kubernetes.Interface, envDir string,
	gitRepoOptions *gits.GitRepositoryOptions, helmValues config.HelmValuesConfig, prefix string, git gits.Gitter, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (gits.GitProvider, error) {
	surveyOpts := survey.WithStdio(in, out, errOut)
	name := data.Name
	createMode := name == ""
	if createMode {
		if config.Name != "" {
			err := ValidNameOption(OptionName, config.Name)
			if err != nil {
				return nil, err
			}
			err = ValidateEnvironmentDoesNotExist(jxClient, ns, config.Name)
			if err != nil {
				return nil, err
			}
			data.Name = config.Name
		} else {
			if batchMode {
				return nil, fmt.Errorf("environment name cannot be empty. Use --name option.")
			}

			validator := func(val interface{}) error {
				err := ValidateName(val)
				if err != nil {
					return err
				}
				str, ok := val.(string)
				if !ok {
					return fmt.Errorf("Expected string value")
				}
				v := ValidateEnvironmentDoesNotExist(jxClient, ns, str)
				return v
			}

			q := &survey.Input{
				Message: "Name:",
				Help:    "The Environment name must be unique, lower case and a valid DNS name",
			}
			err := survey.AskOne(q, &data.Name, validator, surveyOpts)
			if err != nil {
				return nil, err
			}
		}
	}
	if string(config.Spec.Kind) != "" {
		data.Spec.Kind = config.Spec.Kind
	} else {
		if string(data.Spec.Kind) == "" {
			data.Spec.Kind = v1.EnvironmentKindTypePermanent
		}
	}
	if config.Spec.Label != "" {
		data.Spec.Label = config.Spec.Label
	} else {
		defaultValue := data.Spec.Label
		if defaultValue == "" {
			defaultValue = strings.Title(data.Name)
		}
		q := &survey.Input{
			Message: "Label:",
			Default: defaultValue,
			Help:    "The Environment label is a person friendly descriptive text like 'Staging' or 'Production'",
		}
		err := survey.AskOne(q, &data.Spec.Label, survey.Required, surveyOpts)
		if err != nil {
			return nil, err
		}
	}
	if config.Spec.Namespace != "" {
		err := ValidNameOption(OptionNamespace, config.Spec.Namespace)
		if err != nil {
			return nil, err
		}
		data.Spec.Namespace = config.Spec.Namespace
	} else {
		defaultValue := data.Spec.Namespace
		if defaultValue == "" {
			// lets use the namespace as a team name
			defaultValue = data.Namespace
			if defaultValue == "" {
				defaultValue = ns
			}
			if data.Name != "" {
				if defaultValue == "" {
					defaultValue = data.Name
				} else {
					defaultValue += "-" + data.Name
				}
			}
		}
		if batchMode {
			data.Spec.Namespace = defaultValue
		} else {
			q := &survey.Input{
				Message: "Namespace:",
				Default: defaultValue,
				Help:    "The Kubernetes namespace name to use for this Environment",
			}
			err := survey.AskOne(q, &data.Spec.Namespace, ValidateName, surveyOpts)
			if err != nil {
				return nil, err
			}
		}
	}

	if helmValues.ExposeController.Config.Domain == "" {

		ic, err := GetIngressConfig(kubeClient, ns)
		if err != nil {
			return nil, err
		}

		if batchMode {
			log.Infof("Running in batch mode and no domain flag used so defaulting to team domain %s\n", ic.Domain)
			helmValues.ExposeController.Config.Domain = ic.Domain
		} else {
			q := &survey.Input{
				Message: "Domain:",
				Default: ic.Domain,
				Help:    "Domain to expose ingress endpoints.  Example: jenkinsx.io, leave blank if no appplications are to be exposed via ingress rules",
			}
			err := survey.AskOne(q, &helmValues.ExposeController.Config.Domain, nil, surveyOpts)
			if err != nil {
				return nil, err
			}
		}
	}

	if config.Spec.Cluster != "" {
		data.Spec.Cluster = config.Spec.Cluster
	} else {
		// lets not show the UI for this if users specify the namespace via arguments
		if !createMode || config.Spec.Namespace == "" {
			defaultValue := data.Spec.Cluster
			if batchMode {
				data.Spec.Cluster = defaultValue
			} else {
				q := &survey.Input{
					Message: "Cluster URL:",
					Default: defaultValue,
					Help:    "The Kubernetes cluster URL to use to host this Environment",
				}
				// TODO validate/transform to match valid kubnernetes cluster syntax
				err := survey.AskOne(q, &data.Spec.Cluster, nil, surveyOpts)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	if string(config.Spec.PromotionStrategy) != "" {
		data.Spec.PromotionStrategy = config.Spec.PromotionStrategy
	} else {
		promoteValues := []string{
			string(v1.PromotionStrategyTypeAutomatic),
			string(v1.PromotionStrategyTypeManual),
			string(v1.PromotionStrategyTypeNever),
		}
		defaultValue := string(data.Spec.PromotionStrategy)
		if defaultValue == "" {
			defaultValue = string(v1.PromotionStrategyTypeAutomatic)
		}
		q := &survey.Select{
			Message: "Promotion Strategy:",
			Options: promoteValues,
			Default: defaultValue,
			Help:    "Whether we promote to this Environment automatically, manually or never",
		}
		textValue := ""
		err := survey.AskOne(q, &textValue, survey.Required, surveyOpts)
		if err != nil {
			return nil, err
		}
		if textValue != "" {
			data.Spec.PromotionStrategy = v1.PromotionStrategyType(textValue)
		}
	}
	if string(data.Spec.PromotionStrategy) == "" {
		data.Spec.PromotionStrategy = v1.PromotionStrategyTypeAutomatic
	}
	if config.Spec.Order != 0 {
		data.Spec.Order = config.Spec.Order
	} else {
		order := data.Spec.Order
		if order == 0 {
			// TODO should we generate an order to default to last one?
			order = 100
		}
		defaultValue := util.Int32ToA(order)
		q := &survey.Input{
			Message: "Order:",
			Default: defaultValue,
			Help:    "This number is used to sort Environments in sequential order, lowest first",
		}
		textValue := ""
		err := survey.AskOne(q, &textValue, survey.Required, surveyOpts)
		if err != nil {
			return nil, err
		}
		if textValue != "" {
			i, err := util.AtoInt32(textValue)
			if err != nil {
				return nil, fmt.Errorf("Failed to convert input '%s' to number: %s", textValue, err)
			}
			data.Spec.Order = i
		}
	}
	if batchMode && gitRepoOptions.Owner == "" {
		devEnvGitOwner, err := GetDevEnvGitOwner(jxClient)
		if err != nil {
			return nil, fmt.Errorf("Failed to get default Git owner for repos: %s", err)
		}
		if devEnvGitOwner != "" {
			gitRepoOptions.Owner = devEnvGitOwner
		} else {
			gitRepoOptions.Owner = gitRepoOptions.Username
		}
		log.Infof("Using %s environment git owner in batch mode.\n", util.ColorInfo(gitRepoOptions.Owner))
	}
	_, gitProvider, err := CreateEnvGitRepository(batchMode, authConfigSvc, devEnv, data, config, forkEnvGitURL, envDir, gitRepoOptions, helmValues, prefix, git, in, out, errOut)
	return gitProvider, err
}

// CreateEnvGitRepository creates the git repository for the given Environment
func CreateEnvGitRepository(batchMode bool, authConfigSvc auth.ConfigService, devEnv *v1.Environment, data *v1.Environment, config *v1.Environment, forkEnvGitURL string, envDir string, gitRepoOptions *gits.GitRepositoryOptions, helmValues config.HelmValuesConfig, prefix string, git gits.Gitter, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (*gits.GitRepository, gits.GitProvider, error) {
	var gitProvider gits.GitProvider
	var repo *gits.GitRepository
	surveyOpts := survey.WithStdio(in, out, errOut)
	createRepo := false
	if config.Spec.Source.URL != "" {
		data.Spec.Source.URL = config.Spec.Source.URL
	} else {
		showURLEdit := devEnv.Spec.TeamSettings.UseGitOps
		if data.Spec.Source.URL == "" && !showURLEdit {
			if devEnv.Spec.TeamSettings.AskOnCreate {
				confirm := &survey.Confirm{
					Message: "Would you like to use GitOps to manage this environment? :",
					Default: false,
				}
				err := survey.AskOne(confirm, &showURLEdit, nil, surveyOpts)
				if err != nil {
					return repo, nil, err
				}
			} else {
				showURLEdit = true
			}
		}
		if showURLEdit {
			if data.Spec.Source.URL == "" {
				if batchMode {
					createRepo = true
				} else {
					confirm := &survey.Confirm{
						Message: fmt.Sprintf("We will now create a Git repository to store your %s environment, ok? :", data.Name),
						Default: true,
					}
					err := survey.AskOne(confirm, &createRepo, nil, surveyOpts)
					if err != nil {
						return repo, nil, err
					}
				}

				if createRepo {
					showURLEdit = false
					var err error
					repo, gitProvider, err = createEnvironmentGitRepo(batchMode, authConfigSvc, data, forkEnvGitURL, envDir, gitRepoOptions, helmValues, prefix, git, in, out, errOut)
					if err != nil {
						return repo, gitProvider, err
					}
					data.Spec.Source.URL = repo.CloneURL
				}
			} else {
				showURLEdit = true
			}
			if showURLEdit {
				q := &survey.Input{
					Message: "Git URL for the Environment source code:",
					Default: data.Spec.Source.URL,
					Help:    "The git clone URL for the Environment's Helm charts source code and custom configuration",
				}
				err := survey.AskOne(q, &data.Spec.Source.URL, survey.Required, surveyOpts)
				if err != nil {
					return repo, nil, err
				}
			}
		}
	}
	if config.Spec.Source.Ref != "" {
		data.Spec.Source.Ref = config.Spec.Source.Ref
	} else {
		if data.Spec.Source.URL != "" || data.Spec.Source.Ref != "" {
			if batchMode {
				createRepo = true
			} else {
				defaultBranch := data.Spec.Source.Ref
				if defaultBranch == "" {
					defaultBranch = "master"
				}
				q := &survey.Input{
					Message: "Git branch for the Environment source code:",
					Default: defaultBranch,
					Help:    "The Git release branch in the Environments Git repository used to store Helm charts source code and custom configuration",
				}
				err := survey.AskOne(q, &data.Spec.Source.Ref, nil, surveyOpts)
				if err != nil {
					return repo, nil, err
				}
			}
		}
	}
	return repo, gitProvider, nil
}

func createEnvironmentGitRepo(batchMode bool, authConfigSvc auth.ConfigService, env *v1.Environment, forkEnvGitURL string,
	environmentsDir string, gitRepoOptions *gits.GitRepositoryOptions, helmValues config.HelmValuesConfig, prefix string, git gits.Gitter, in terminal.FileReader, out terminal.FileWriter, outErr io.Writer) (*gits.GitRepository, gits.GitProvider, error) {
	defaultRepoName := fmt.Sprintf("environment-%s-%s", prefix, env.Name)
	details, err := gits.PickNewGitRepository(batchMode, authConfigSvc, defaultRepoName, gitRepoOptions, nil, nil, git, in, out, outErr)
	if err != nil {
		return nil, nil, err
	}
	org := details.Organisation

	repoName := details.RepoName
	owner := org
	if owner == "" {
		owner = details.User.Username
	}
	envDir := filepath.Join(environmentsDir, owner)
	provider := details.GitProvider

	repo, err := provider.GetRepository(owner, repoName)
	if err == nil {
		fmt.Fprintf(out, "Git repository %s/%s already exists\n", util.ColorInfo(owner), util.ColorInfo(repoName))
		// if the repo already exists then lets just modify it if required
		dir, err := util.CreateUniqueDirectory(envDir, details.RepoName, util.MaximumNewDirectoryAttempts)
		if err != nil {
			return nil, nil, err
		}
		pushGitURL, err := git.CreatePushURL(repo.CloneURL, details.User)
		if err != nil {
			return nil, nil, err
		}
		err = git.Clone(pushGitURL, dir)
		if err != nil {
			return nil, nil, err
		}
		err = ModifyNamespace(out, dir, env, git)
		if err != nil {
			return nil, nil, err
		}
		err = addValues(out, dir, helmValues, git)
		if err != nil {
			return nil, nil, err
		}
		err = git.PushMaster(dir)
		if err != nil {
			return nil, nil, err
		}
		fmt.Fprintf(out, "Pushed Git repository to %s\n\n", util.ColorInfo(repo.HTMLURL))
	} else {
		fmt.Fprintf(out, "Creating Git repository %s/%s\n", util.ColorInfo(owner), util.ColorInfo(repoName))

		if forkEnvGitURL != "" {
			gitInfo, err := gits.ParseGitURL(forkEnvGitURL)
			if err != nil {
				return nil, nil, err
			}
			originalOrg := gitInfo.Organisation
			originalRepo := gitInfo.Name
			if useForkForEnvGitRepo && gitInfo.IsGitHub() && provider.IsGitHub() && originalOrg != "" && originalRepo != "" {
				// lets try fork the repository and rename it
				repo, err := provider.ForkRepository(originalOrg, originalRepo, org)
				if err != nil {
					return nil, nil, fmt.Errorf("Failed to fork GitHub repo %s/%s to organisation %s due to %s", originalOrg, originalRepo, org, err)
				}
				if repoName != originalRepo {
					repo, err = provider.RenameRepository(owner, originalRepo, repoName)
					if err != nil {
						return nil, nil, fmt.Errorf("Failed to rename GitHub repo %s/%s to organisation %s due to %s", originalOrg, originalRepo, repoName, err)
					}
				}
				fmt.Fprintf(out, "Forked Git repository to %s\n\n", util.ColorInfo(repo.HTMLURL))

				dir, err := util.CreateUniqueDirectory(envDir, repoName, util.MaximumNewDirectoryAttempts)
				if err != nil {
					return nil, nil, err
				}
				err = git.Clone(repo.CloneURL, dir)
				if err != nil {
					return nil, nil, err
				}
				err = git.SetRemoteURL(dir, "upstream", forkEnvGitURL)
				if err != nil {
					return nil, nil, err
				}
				err = git.PullUpstream(dir)
				if err != nil {
					return nil, nil, err
				}
				err = ModifyNamespace(out, dir, env, git)
				if err != nil {
					return nil, nil, err
				}
				err = addValues(out, dir, helmValues, git)
				if err != nil {
					return nil, nil, err
				}
				err = git.Push(dir)
				if err != nil {
					return nil, nil, err
				}
				return repo, provider, nil
			}
		}

		// default to forking the URL if possible...
		repo, err = details.CreateRepository()
		if err != nil {
			return nil, nil, err
		}

		if forkEnvGitURL != "" {
			// now lets clone the fork and push it...
			dir, err := util.CreateUniqueDirectory(envDir, details.RepoName, util.MaximumNewDirectoryAttempts)
			if err != nil {
				return nil, nil, err
			}
			err = git.Clone(forkEnvGitURL, dir)
			if err != nil {
				return nil, nil, err
			}
			pushGitURL, err := git.CreatePushURL(repo.CloneURL, details.User)
			if err != nil {
				return nil, nil, err
			}
			err = git.AddRemote(dir, "upstream", forkEnvGitURL)
			if err != nil {
				return nil, nil, err
			}
			err = git.UpdateRemote(dir, pushGitURL)
			if err != nil {
				return nil, nil, err
			}
			err = ModifyNamespace(out, dir, env, git)
			if err != nil {
				return nil, nil, err
			}
			err = addValues(out, dir, helmValues, git)
			if err != nil {
				return nil, nil, err
			}
			err = git.PushMaster(dir)
			if err != nil {
				return nil, nil, err
			}
			fmt.Fprintf(out, "Pushed Git repository to %s\n\n", util.ColorInfo(repo.HTMLURL))
		}
	}
	return repo, provider, nil
}

// GetDevEnvGitOwner gets the default GitHub owner/organisation to use for Environment repos. This takes the setting
// from the 'jx' Dev Env to get the one that was selected at installation time.
func GetDevEnvGitOwner(jxClient versioned.Interface) (string, error) {
	adminDevEnv, err := GetDevEnvironment(jxClient, "jx")
	if err != nil {
		log.Errorf("Error loading team settings. %v\n", err)
		return "", err
	}
	if adminDevEnv != nil {
		return adminDevEnv.Spec.TeamSettings.EnvOrganisation, nil
	}
	return "", errors.New("Unable to find development environment in 'jx' to take git owner from")
}

// ModifyNamespace modifies the namespace
func ModifyNamespace(out io.Writer, dir string, env *v1.Environment, git gits.Gitter) error {
	ns := env.Spec.Namespace
	if ns == "" {
		return fmt.Errorf("No Namespace is defined for Environment %s", env.Name)
	}

	// makefile changes
	file := filepath.Join(dir, "Makefile")
	exists, err := util.FileExists(file)
	if err != nil {
		return err
	}
	if !exists {
		log.Warnf("WARNING: Could not find a Makefile in %s\n", dir)
		return nil
	}
	input, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	lines := strings.Split(string(input), "\n")
	err = ReplaceMakeVariable(lines, "NAMESPACE", "\""+ns+"\"")
	if err != nil {
		return err
	}
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(file, []byte(output), 0644)
	if err != nil {
		return err
	}

	// Jenkinsfile changes
	file = filepath.Join(dir, "Jenkinsfile")
	exists, err = util.FileExists(file)
	if err != nil {
		return err
	}
	if !exists {
		log.Warnf("WARNING: Could not find a Jenkinsfile in %s\n", dir)
	} else {
		input, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		lines := strings.Split(string(input), "\n")
		err = replaceEnvVar(lines, "DEPLOY_NAMESPACE", ns)
		if err != nil {
			return err
		}
		output := strings.Join(lines, "\n")
		err = ioutil.WriteFile(file, []byte(output), 0644)
		if err != nil {
			return err
		}
	}

	err = git.Add(dir, "*")
	if err != nil {
		return err
	}
	changes, err := git.HasChanges(dir)
	if err != nil {
		return err
	}
	if changes {
		return git.CommitDir(dir, "Use correct namespace for environment")
	}
	return nil
}

func addValues(out io.Writer, dir string, values config.HelmValuesConfig, git gits.Gitter) error {
	file := filepath.Join(dir, "env", "values.yaml")
	exists, err := util.FileExists(file)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("could not find a values.yaml in %s", dir)
	}

	oldText, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	text, err := values.String()
	if err != nil {
		return err
	}

	sourceMap := map[string]interface{}{}
	overrideMap := map[string]interface{}{}
	err = yaml.Unmarshal(oldText, &sourceMap)
	if err != nil {
		return errors.Wrapf(err, "failed to parse YAML for file %s", file)
	}
	err = yaml.Unmarshal([]byte(text), &overrideMap)
	if err != nil {
		return errors.Wrapf(err, "failed to parse YAML for file %s", file)
	}

	// now lets merge together the 2 blobs of YAML
	util.CombineMapTrees(sourceMap, overrideMap)

	output, err := yaml.Marshal(sourceMap)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal the combined values YAML files back to YAML")
	}
	err = ioutil.WriteFile(file, output, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "Failed to save YAML file %s", file)
	}

	err = git.Add(dir, "*")
	if err != nil {
		return err
	}
	changes, err := git.HasChanges(dir)
	if err != nil {
		return err
	}
	if changes {
		return git.CommitDir(dir, "Add environment configuration")
	}
	return nil
}

// ReplaceMakeVariable needs a description
func ReplaceMakeVariable(lines []string, name string, value string) error {
	re, err := regexp.Compile(name + "\\s*:?=\\s*(.*)")
	if err != nil {
		return err
	}
	replaceValue := name + " := " + value
	for i, line := range lines {
		lines[i] = re.ReplaceAllString(line, replaceValue)
	}
	return nil
}

func replaceEnvVar(lines []string, name string, value string) error {
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, name) {
			remain := strings.TrimSpace(strings.TrimPrefix(trimmed, name))
			if strings.HasPrefix(remain, "=") {
				// lets preserve whitespace
				idx := strings.Index(line, name)
				lines[i] = line[0:idx] + name + ` = "` + value + `"`
			}
		}
	}
	return nil
}

// GetEnvironmentNames returns the sorted list of environment names
func GetEnvironmentNames(jxClient versioned.Interface, ns string) ([]string, error) {
	envNames := []string{}
	envs, err := jxClient.JenkinsV1().Environments(ns).List(metav1.ListOptions{})
	if err != nil {
		return envNames, err
	}
	SortEnvironments(envs.Items)
	for _, env := range envs.Items {
		n := env.Name
		if n != "" {
			envNames = append(envNames, n)
		}
	}
	sort.Strings(envNames)
	return envNames, nil
}

func IsPreviewEnvironment(env *v1.Environment) bool {
	return env != nil && env.Spec.Kind == v1.EnvironmentKindTypePreview
}

// GetFilteredEnvironmentNames returns the sorted list of environment names
func GetFilteredEnvironmentNames(jxClient versioned.Interface, ns string, fn func(environment *v1.Environment) bool) ([]string, error) {
	envNames := []string{}
	envs, err := jxClient.JenkinsV1().Environments(ns).List(metav1.ListOptions{})
	if err != nil {
		return envNames, err
	}
	SortEnvironments(envs.Items)
	for _, env := range envs.Items {
		n := env.Name
		if n != "" && fn(&env) {
			envNames = append(envNames, n)
		}
	}
	sort.Strings(envNames)
	return envNames, nil
}

// GetOrderedEnvironments returns a map of the environments along with the correctly ordered  names
func GetOrderedEnvironments(jxClient versioned.Interface, ns string) (map[string]*v1.Environment, []string, error) {
	m := map[string]*v1.Environment{}

	envNames := []string{}
	envs, err := jxClient.JenkinsV1().Environments(ns).List(metav1.ListOptions{})
	if err != nil {
		return m, envNames, err
	}
	SortEnvironments(envs.Items)
	for _, env := range envs.Items {
		n := env.Name
		copy := env
		m[n] = &copy
		if n != "" {
			envNames = append(envNames, n)
		}
	}
	return m, envNames, nil
}

// GetEnvironments returns a map of the environments along with a sorted list of names
func GetEnvironments(jxClient versioned.Interface, ns string) (map[string]*v1.Environment, []string, error) {
	m := map[string]*v1.Environment{}

	envNames := []string{}
	envs, err := jxClient.JenkinsV1().Environments(ns).List(metav1.ListOptions{})
	if err != nil {
		return m, envNames, err
	}
	for _, env := range envs.Items {
		n := env.Name
		copy := env
		m[n] = &copy
		if n != "" {
			envNames = append(envNames, n)
		}
	}
	sort.Strings(envNames)
	return m, envNames, nil
}

// GetEnvironment find an environment by name
func GetEnvironment(jxClient versioned.Interface, ns string, name string) (*v1.Environment, error) {
	envs, err := jxClient.JenkinsV1().Environments(ns).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, env := range envs.Items {
		if env.GetName() == name {
			return &env, nil
		}
	}
	return nil, fmt.Errorf("no environment with name '%s' found", name)
}

// GetEnvironmentsByPrURL find an environment by a pull request URL
func GetEnvironmentsByPrURL(jxClient versioned.Interface, ns string, prURL string) (*v1.Environment, error) {
	envs, err := jxClient.JenkinsV1().Environments(ns).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, env := range envs.Items {
		if env.Spec.PullRequestURL == prURL {
			return &env, nil
		}
	}
	return nil, fmt.Errorf("no environment found for PR '%s'", prURL)
}

// GetEnvironments returns the namespace name for a given environment
func GetEnvironmentNamespace(jxClient versioned.Interface, ns, environment string) (string, error) {
	env, err := jxClient.JenkinsV1().Environments(ns).Get(environment, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	if env == nil {
		return "", fmt.Errorf("no environment found called %s, try running `jx get env`", environment)
	}
	return env.Spec.Namespace, nil
}

// GetEditEnvironmentNamespace returns the namespace of the current users edit environment
func GetEditEnvironmentNamespace(jxClient versioned.Interface, ns string) (string, error) {
	envs, err := jxClient.JenkinsV1().Environments(ns).List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	for _, env := range envs.Items {
		if env.Spec.Kind == v1.EnvironmentKindTypeEdit && env.Spec.PreviewGitSpec.User.Username == u.Username {
			return env.Spec.Namespace, nil
		}
	}
	return "", fmt.Errorf("The user %s does not have an Edit environment in home namespace %s", u.Username, ns)
}

// GetDevNamespace returns the developer environment namespace
// which is the namespace that contains the Environments and the developer tools like Jenkins
func GetDevNamespace(kubeClient kubernetes.Interface, ns string) (string, string, error) {
	env := ""
	namespace, err := kubeClient.CoreV1().Namespaces().Get(ns, metav1.GetOptions{})
	if err != nil {
		return ns, env, err
	}
	if namespace == nil {
		return ns, env, fmt.Errorf("No namespace found for %s", ns)
	}
	if namespace.Labels != nil {
		answer := namespace.Labels[LabelTeam]
		if answer != "" {
			ns = answer
		}
		env = namespace.Labels[LabelEnvironment]
	}
	return ns, env, nil
}

// GetTeams returns the Teams the user is a member of
func GetTeams(kubeClient kubernetes.Interface) ([]*corev1.Namespace, []string, error) {
	names := []string{}
	answer := []*corev1.Namespace{}
	namespaceList, err := kubeClient.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return answer, names, err
	}
	for idx, namespace := range namespaceList.Items {
		if namespace.Labels[LabelEnvironment] == LabelValueDevEnvironment {
			answer = append(answer, &namespaceList.Items[idx])
			names = append(names, namespace.Name)
		}
	}
	sort.Strings(names)
	return answer, names, nil
}

func PickEnvironment(envNames []string, defaultEnv string, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (string, error) {
	surveyOpts := survey.WithStdio(in, out, errOut)
	name := ""
	if len(envNames) == 0 {
		return "", nil
	} else if len(envNames) == 1 {
		name = envNames[0]
	} else {
		prompt := &survey.Select{
			Message: "Pick environment:",
			Options: envNames,
			Default: defaultEnv,
		}
		err := survey.AskOne(prompt, &name, nil, surveyOpts)
		if err != nil {
			return "", err
		}
	}
	return name, nil
}

type ByOrder []v1.Environment

func (a ByOrder) Len() int      { return len(a) }
func (a ByOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByOrder) Less(i, j int) bool {
	env1 := a[i]
	env2 := a[j]
	o1 := env1.Spec.Order
	o2 := env2.Spec.Order
	if o1 == o2 {
		return env1.Name < env2.Name
	}
	return o1 < o2
}

func SortEnvironments(environments []v1.Environment) {
	sort.Sort(ByOrder(environments))
}

// NewPermanentEnvironment creates a new permanent environment for testing
func NewPermanentEnvironment(name string) *v1.Environment {
	return &v1.Environment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "jx",
		},
		Spec: v1.EnvironmentSpec{
			Label:             strings.Title(name),
			Namespace:         "jx-" + name,
			PromotionStrategy: v1.PromotionStrategyTypeAutomatic,
			Kind:              v1.EnvironmentKindTypePermanent,
		},
	}
}

// NewPermanentEnvironment creates a new permanent environment for testing
func NewPermanentEnvironmentWithGit(name string, gitUrl string) *v1.Environment {
	env := NewPermanentEnvironment(name)
	env.Spec.Source.URL = gitUrl
	env.Spec.Source.Ref = "master"
	return env
}

// NewPreviewEnvironment creates a new preview environment for testing
func NewPreviewEnvironment(name string) *v1.Environment {
	return &v1.Environment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "jx",
		},
		Spec: v1.EnvironmentSpec{
			Label:             strings.Title(name),
			Namespace:         "jx-preview-" + name,
			PromotionStrategy: v1.PromotionStrategyTypeAutomatic,
			Kind:              v1.EnvironmentKindTypePreview,
		},
	}
}

// GetDevEnvironment returns the current development environment using the jxClient for the given ns.
// If the Dev Environment cannot be found, returns nil Environment (rather than an error). A non-nil error is only
// returned if there is an error fetching the Dev Environment.
func GetDevEnvironment(jxClient versioned.Interface, ns string) (*v1.Environment, error) {
	//Find the settings for the team
	environmentInterface := jxClient.JenkinsV1().Environments(ns)
	name := LabelValueDevEnvironment
	answer, err := environmentInterface.Get(name, metav1.GetOptions{})
	if err == nil {
		return answer, nil
	}
	selector := "env=dev"
	envList, err := environmentInterface.List(metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}
	if len(envList.Items) == 1 {
		return &envList.Items[0], nil
	}
	if len(envList.Items) == 0 {
		return nil, nil
	}
	return nil, fmt.Errorf("Error fetching dev environment resource definition in namespace %s, No Environment called: %s or with selector: %s found %d entries: %v",
		ns, name, selector, len(envList.Items), envList.Items)
}
