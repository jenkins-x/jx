package kube

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/AlecAivazis/survey.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var useForkForEnvGitRepo = false

// CreateEnvironmentSurvey creates a Survey on the given environment using the default options
// from the CLI
func CreateEnvironmentSurvey(out io.Writer, batchMode bool, authConfigSvc auth.AuthConfigService, devEnv *v1.Environment, data *v1.Environment, config *v1.Environment, forkEnvGitURL string, ns string, jxClient *versioned.Clientset, envDir string, gitRepoOptions gits.GitRepositoryOptions) (gits.GitProvider, error) {
	var gitProvider gits.GitProvider
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
				return nil, util.InvalidOptionError(OptionName, config.Name, err)
			}
			data.Name = config.Name
		} else {
			validator := func(val interface{}) error {
				err := ValidateName(val)
				if err != nil {
					return err
				}
				str, ok := val.(string)
				if !ok {
					return fmt.Errorf("Expected string value!")
				}
				return ValidateEnvironmentDoesNotExist(jxClient, ns, str)
			}

			q := &survey.Input{
				Message: "Name:",
				Help:    "The Environment name must be unique, lower case and a valid DNS name",
			}
			err := survey.AskOne(q, &data.Name, validator)
			if err != nil {
				return nil, err
			}
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
		err := survey.AskOne(q, &data.Spec.Label, survey.Required)
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
		q := &survey.Input{
			Message: "Namespace:",
			Default: defaultValue,
			Help:    "The kubernetes namespace name to use for this Environment",
		}
		err := survey.AskOne(q, &data.Spec.Namespace, ValidateName)
		if err != nil {
			return nil, err
		}
	}
	if config.Spec.Cluster != "" {
		data.Spec.Cluster = config.Spec.Cluster
	} else {
		// lets not show the UI for this if users specify the namespace via arguments
		if !createMode || config.Spec.Namespace == "" {
			defaultValue := data.Spec.Cluster
			q := &survey.Input{
				Message: "Cluster URL:",
				Default: defaultValue,
				Help:    "The kubernetes cluster URL to use to host this Environment",
			}
			// TODO validate/transform to match valid kubnernetes cluster syntax
			err := survey.AskOne(q, &data.Spec.Cluster, nil)
			if err != nil {
				return nil, err
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
		err := survey.AskOne(q, &textValue, survey.Required)
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
		err := survey.AskOne(q, &textValue, survey.Required)
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
	createRepo := false
	if config.Spec.Source.URL != "" {
		data.Spec.Source.URL = config.Spec.Source.URL
	} else {
		showUrlEdit := devEnv.Spec.TeamSettings.UseGitOPs
		if data.Spec.Source.URL == "" {
			if devEnv.Spec.TeamSettings.AskOnCreate {
				confirm := &survey.Confirm{
					Message: "Would you like to use GitOps to manage this environment? :",
					Default: false,
				}
				err := survey.AskOne(confirm, &showUrlEdit, nil)
				if err != nil {
					return nil, err
				}
			} else {
				showUrlEdit = true
			}
		}
		if showUrlEdit {
			if data.Spec.Source.URL == "" {
				confirm := &survey.Confirm{
					Message: "Would you like to create a new Git repository to store this Environments source code? :",
					Default: true,
				}
				err := survey.AskOne(confirm, &createRepo, nil)
				if err != nil {
					return nil, err
				}
				if createRepo {
					showUrlEdit = false
					url, p, err := createEnvironmentGitRepo(out, batchMode, authConfigSvc, data, forkEnvGitURL, envDir, gitRepoOptions)
					if err != nil {
						return nil, err
					}
					gitProvider = p
					data.Spec.Source.URL = url
				}
			} else {
				showUrlEdit = true
			}
			if showUrlEdit {
				q := &survey.Input{
					Message: "Git URL for the Environment source code:",
					Default: data.Spec.Source.URL,
					Help:    "The git clone URL for the Environment's Helm charts source code and custom configuration",
				}
				err := survey.AskOne(q, &data.Spec.Source.URL, survey.Required)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	if config.Spec.Source.Ref != "" {
		data.Spec.Source.Ref = config.Spec.Source.Ref
	} else {
		if data.Spec.Source.URL != "" || data.Spec.Source.Ref != "" {
			q := &survey.Input{
				Message: "Git Ref for the Environment source code:",
				Default: data.Spec.Source.Ref,
				Help:    "The git clone Ref for the Environment's Helm charts source code and custom configuration",
			}
			err := survey.AskOne(q, &data.Spec.Source.Ref, nil)
			if err != nil {
				return nil, err
			}
		}
	}
	return gitProvider, nil
}

func createEnvironmentGitRepo(out io.Writer, batchMode bool, authConfigSvc auth.AuthConfigService, env *v1.Environment, forkEnvGitURL string, environmentsDir string, gitRepoOptions gits.GitRepositoryOptions) (string, gits.GitProvider, error) {
	defaultRepoName := "environment-" + env.Name
	details, err := gits.PickNewGitRepository(out, batchMode, authConfigSvc, defaultRepoName, gitRepoOptions)
	if err != nil {
		return "", nil, err
	}
	org := details.Organisation
	repoName := details.RepoName
	owner := org
	if owner == "" {
		owner = details.User.Username
	}
	envDir := filepath.Join(environmentsDir, owner)
	provider := details.GitProvider
	if forkEnvGitURL != "" {
		gitInfo, err := gits.ParseGitURL(forkEnvGitURL)
		if err != nil {
			return "", nil, err
		}
		originalOrg := gitInfo.Organisation
		originalRepo := gitInfo.Name
		if useForkForEnvGitRepo && gitInfo.IsGitHub() && provider.IsGitHub() && originalOrg != "" && originalRepo != "" {
			// lets try fork the repository and rename it
			repo, err := provider.ForkRepository(originalOrg, originalRepo, org)
			if err != nil {
				return "", nil, fmt.Errorf("Failed to fork github repo %s/%s to organisation %s due to %s", originalOrg, originalRepo, org, err)
			}
			if repoName != originalRepo {
				repo, err = provider.RenameRepository(owner, originalRepo, repoName)
				if err != nil {
					return "", nil, fmt.Errorf("Failed to rename github repo %s/%s to organisation %s due to %s", originalOrg, originalRepo, repoName, err)
				}
			}
			fmt.Fprintf(out, "Forked git repository to %s\n\n", util.ColorInfo(repo.HTMLURL))

			dir, err := util.CreateUniqueDirectory(envDir, repoName, util.MaximumNewDirectoryAttempts)
			if err != nil {
				return "", nil, err
			}
			err = gits.GitClone(repo.CloneURL, dir)
			if err != nil {
				return "", nil, err
			}
			err = gits.SetRemoteURL(dir, "upstream", forkEnvGitURL)
			if err != nil {
				return "", nil, err
			}
			err = gits.GitCmd(dir, "pull", "-r", "upstream", "master")
			if err != nil {
				return "", nil, err
			}
			err = modifyNamespace(out, dir, env)
			if err != nil {
				return "", nil, err
			}
			err = gits.GitPush(dir)
			if err != nil {
				return "", nil, err
			}
			return repo.CloneURL, provider, nil
		}
	}
	// default to forking the URL if possible...
	repo, err := details.CreateRepository()
	if err != nil {
		return "", nil, err
	}

	if forkEnvGitURL != "" {
		// now lets clone the fork and push it...
		dir, err := util.CreateUniqueDirectory(envDir, details.RepoName, util.MaximumNewDirectoryAttempts)
		if err != nil {
			return "", nil, err
		}
		err = gits.GitClone(forkEnvGitURL, dir)
		if err != nil {
			return "", nil, err
		}
		pushGitURL, err := gits.GitCreatePushURL(repo.CloneURL, details.User)
		if err != nil {
			return "", nil, err
		}
		err = gits.GitCmd(dir, "remote", "add", "upstream", forkEnvGitURL)
		if err != nil {
			return "", nil, err
		}
		err = gits.GitCmd(dir, "remote", "set-url", "origin", pushGitURL)
		if err != nil {
			return "", nil, err
		}
		err = modifyNamespace(out, dir, env)
		if err != nil {
			return "", nil, err
		}
		err = gits.GitCmd(dir, "push", "-u", "origin", "master")
		if err != nil {
			return "", nil, err
		}
		fmt.Fprintf(out, "Pushed git repository to %s\n\n", util.ColorInfo(repo.HTMLURL))
	}
	return repo.CloneURL, provider, nil
}

func modifyNamespace(out io.Writer, dir string, env *v1.Environment) error {
	ns := env.Spec.Namespace
	if ns == "" {
		return fmt.Errorf("No Namespace is defined for Environment %s", env.Name)
	}
	file := filepath.Join(dir, "Makefile")
	exists, err := util.FileExists(file)
	if err != nil {
		return err
	}
	if !exists {
		fmt.Printf(util.ColorWarning("WARNING: Could not find a Makefile in %s\n"), dir)
		return nil
	}
	input, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")

	err = replaceMakeVariable(lines, "NAMESPACE", "\""+ns+"\"")
	if err != nil {
		return err
	}
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(file, []byte(output), 0644)
	if err != nil {
		return err
	}
	err = gits.GitAdd(dir, "*")
	if err != nil {
		return err
	}
	return gits.GitCommit(dir, "Use correct namespace for environment")
}

func replaceMakeVariable(lines []string, name string, value string) error {
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

// GetEnvironmentNames returns the sorted list of environment names
func GetEnvironmentNames(jxClient *versioned.Clientset, ns string) ([]string, error) {
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

// GetEnvironments returns a map of the enviroments along with a sorted list of names
func GetEnvironments(jxClient *versioned.Clientset, ns string) (map[string]*v1.Environment, []string, error) {
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

// GetDevNamespace returns the developer environment namespace
// which is the namespace that contains the Environments and the developer tools like Jenkins
func GetDevNamespace(kubeClient *kubernetes.Clientset, ns string) (string, string, error) {
	env := ""
	namespace, err := kubeClient.CoreV1().Namespaces().Get(ns, metav1.GetOptions{})
	if err != err {
		return ns, env, err
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

func PickEnvironment(envNames []string, defaultEnv string) (string, error) {
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
		err := survey.AskOne(prompt, &name, nil)
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
