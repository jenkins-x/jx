package opts

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/secreturl"
	"github.com/jenkins-x/jx/pkg/secreturl/localvault"
	"github.com/pborman/uuid"

	"github.com/jenkins-x/jx/pkg/environments"
	"gopkg.in/src-d/go-git.v4/plumbing"

	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/kube/services"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	version2 "github.com/jenkins-x/jx/pkg/version"
	"github.com/pkg/errors"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/src-d/go-git.v4"
	gitconfig "gopkg.in/src-d/go-git.v4/config"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DefaultChartRepo default URL for charts repository
	DefaultChartRepo = "http://jenkins-x-chartmuseum:8080"
	// DefaultTillerNamesapce default namespace for helm tiller server
	DefaultTillerNamesapce = "kube-system"
	// DefaultTillerRole default cluster role for service account of helm tiller server
	DefaultTillerRole = "cluster-admin"
	// DefaultOnlyHelmClient indicates if only the client is initialized
	DefaultOnlyHelmClient = false
	// DefaultHelm3 indicates if helm 3 is used
	DefaultHelm3 = false
	// DefaultSkipTiller skips the tiller server initialization
	DefaultSkipTiller = false
	// DefaultGlobalTiller indicates if a global tiller server is used
	DefaultGlobalTiller = true
	// DefaultRemoteTiller indicates that a remote tiller server is used
	DefaultRemoteTiller = true
	// DefaultSkipClusterRole skips the cluster role creation
	DefaultSkipClusterRole = false
)

// InitHelmConfig configuration for helm initialization
type InitHelmConfig struct {
	Namespace       string
	OnlyHelmClient  bool
	Helm3           bool
	SkipTiller      bool
	GlobalTiller    bool
	TillerNamespace string
	TillerRole      string
}

// defaultInitHelmConfig builds the default configuration for init helm
func (o *CommonOptions) defaultInitHelmConfig() InitHelmConfig {
	return InitHelmConfig{
		Namespace:       kube.DefaultNamespace,
		OnlyHelmClient:  DefaultOnlyHelmClient,
		Helm3:           DefaultHelm3,
		SkipTiller:      DefaultSkipTiller,
		GlobalTiller:    DefaultGlobalTiller,
		TillerNamespace: DefaultTillerNamesapce,
		TillerRole:      DefaultTillerRole,
	}
}

// InitHelm initializes helm client and server (tiller)
func (o *CommonOptions) InitHelm(config InitHelmConfig) error {
	var err error

	skipTiller := config.SkipTiller
	if config.Helm3 {
		log.Logger().Debugf("Using %s", util.ColorInfo("helm3"))
		skipTiller = true
	} else {
		log.Logger().Debugf("Using %s", util.ColorInfo("helm2"))
	}
	if !skipTiller {
		log.Logger().Infof("Configuring %s", util.ColorInfo("tiller"))
		client, curNs, err := o.KubeClientAndNamespace()
		if err != nil {
			return err
		}

		tillerNamespace := config.TillerNamespace
		serviceAccountName := "tiller"
		if config.GlobalTiller {
			if tillerNamespace == "" {
				return errors.New("tiller namespace is empty: glboal tiller requires a namesapce")
			}
		} else {
			if config.Namespace == "" {
				config.Namespace = curNs
			}
			if config.Namespace == "" {
				return errors.New("empty namespace")
			}
			tillerNamespace = config.Namespace
		}

		err = o.EnsureServiceAccount(tillerNamespace, serviceAccountName)
		if err != nil {
			return err
		}

		if config.GlobalTiller {
			clusterRoleBindingName := serviceAccountName
			err = o.EnsureClusterRoleBinding(clusterRoleBindingName, config.TillerRole, tillerNamespace, serviceAccountName)
			if err != nil {
				return err
			}
		} else {
			// lets create a tiller service account
			roleName := "tiller-manager"
			roleBindingName := "tiller-binding"

			_, err = client.RbacV1().Roles(tillerNamespace).Get(roleName, metav1.GetOptions{})
			if err != nil {
				// lets create a Role for tiller
				role := &rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      roleName,
						Namespace: tillerNamespace,
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{"", "extensions", "apps"},
							Resources: []string{"*"},
							Verbs:     []string{"*"},
						},
					},
				}
				_, err = client.RbacV1().Roles(tillerNamespace).Create(role)
				if err != nil {
					return fmt.Errorf("Failed to create Role %s in namespace %s: %s", roleName, tillerNamespace, err)
				}
				log.Logger().Infof("Created Role %s in namespace %s", util.ColorInfo(roleName), util.ColorInfo(tillerNamespace))
			}
			_, err = client.RbacV1().RoleBindings(tillerNamespace).Get(roleBindingName, metav1.GetOptions{})
			if err != nil {
				// lets create a RoleBinding for tiller
				roleBinding := &rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      roleBindingName,
						Namespace: tillerNamespace,
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      serviceAccountName,
							Namespace: tillerNamespace,
						},
					},
					RoleRef: rbacv1.RoleRef{
						Kind:     "Role",
						Name:     roleName,
						APIGroup: "rbac.authorization.k8s.io",
					},
				}
				_, err = client.RbacV1().RoleBindings(tillerNamespace).Create(roleBinding)
				if err != nil {
					return fmt.Errorf("Failed to create RoleBinding %s in namespace %s: %s", roleName, tillerNamespace, err)
				}
				log.Logger().Infof("Created RoleBinding %s in namespace %s", util.ColorInfo(roleName), util.ColorInfo(tillerNamespace))
			}
		}

		running, err := kube.IsDeploymentRunning(client, "tiller-deploy", tillerNamespace)
		if running {
			log.Logger().Infof("Tiller Deployment is running in namespace %s", util.ColorInfo(tillerNamespace))
			return nil
		}
		if err == nil && !running {
			return fmt.Errorf("existing tiller deployment found but not running, please check the %s namespace and resolve any issues", tillerNamespace)
		}

		if !running {
			log.Logger().Infof("Initialising helm using ServiceAccount %s in namespace %s", util.ColorInfo(serviceAccountName), util.ColorInfo(tillerNamespace))

			err = o.Helm().Init(false, serviceAccountName, tillerNamespace, false)
			if err != nil {
				return err
			}
			err = kube.WaitForDeploymentToBeReady(client, "tiller-deploy", tillerNamespace, 10*time.Minute)
			if err != nil {
				return err
			}

			err = o.Helm().Init(false, serviceAccountName, tillerNamespace, true)
			if err != nil {
				return err
			}
		}

		log.Logger().Infof("Waiting for tiller-deploy to be ready in tiller namespace %s", tillerNamespace)
		err = kube.WaitForDeploymentToBeReady(client, "tiller-deploy", tillerNamespace, 10*time.Minute)
		if err != nil {
			return err
		}
	} else {
		log.Logger().Debugf("Skipping %s", util.ColorInfo("tiller"))
	}

	if config.Helm3 {
		err = o.Helm().Init(false, "", "", false)
		if err != nil {
			return err
		}
	} else if config.OnlyHelmClient || config.SkipTiller {
		err = o.Helm().Init(true, "", "", false)
		if err != nil {
			return err
		}
	}

	err = o.Helm().AddRepo("jenkins-x", kube.DefaultChartMuseumURL, "", "")
	if err != nil {
		return err
	}
	log.Logger().Info("helm installed and configured")

	return nil
}

func (o *CommonOptions) RegisterLocalHelmRepo(repoName, ns string) error {
	if repoName == "" {
		repoName = kube.LocalHelmRepoName
	}
	// TODO we should use the auth package to keep a list of server login/pwds
	// TODO we have a chartmuseumAuth.yaml now but sure yet if that's the best thing to do
	username := "admin"
	password := "admin"

	// lets check if we have a local helm repository
	client, err := o.KubeClient()
	if err != nil {
		return errors.Wrap(err, "failed to create the kube client")
	}
	u, err := services.FindServiceURL(client, ns, kube.ServiceChartMuseum)
	if err != nil {
		return errors.Wrapf(err, "failed to find the service URL of the ChartMuseum")
	}
	u2, err := url.Parse(u)
	if err != nil {
		return errors.Wrap(err, "failed to parse the ChartMuseum URL")
	}
	if u2.User == nil {
		u2.User = url.UserPassword(username, password)
	}
	helmUrl := u2.String()
	// lets check if we already have the helm repo installed or if we need to add it or remove + add it
	remove := false
	repos, err := o.Helm().ListRepos()
	if err != nil {
		return errors.Wrap(err, "failed to list the repositories")
	}
	for repo, repoURL := range repos {
		if repo == repoName {
			if repoURL == helmUrl {
				return nil
			} else {
				remove = true
			}
		}
	}
	if remove {
		err = o.Helm().RemoveRepo(repoName)
		if err != nil {
			return errors.Wrapf(err, "failed to remove the repository '%s'", repoName)
		}
	}
	return o.Helm().AddRepo(repoName, helmUrl, "", "")
}

//AddHelmBinaryRepoIfMissing adds the helm repo at url if it's missing. If a repoName is specified it will be used (if
// the repo is added) otherwise one will be generated. The username and password will be used, and stored in vault, if
// possible. The name of the repo (regardless of whether it was added or already there) is returned - this may well be
// different from the requested name (if it's already there).
func (o *CommonOptions) AddHelmBinaryRepoIfMissing(url, repoName, username, password string) (string, error) {
	vaultClient, err := o.SystemVaultClient("")
	if err != nil {
		vaultClient = nil
	}
	name, err := helm.AddHelmRepoIfMissing(url, repoName, username, password, o.Helm(), vaultClient, o.In, o.Out, o.Err)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return name, nil
}

// InstallChartOrGitOps if using gitOps lets write files otherwise lets use helm
func (o *CommonOptions) InstallChartOrGitOps(isGitOps bool, gitOpsDir string, gitOpsEnvDir string, releaseName string, chart string, alias string, version string, ns string, helmUpdate bool,
	setValues []string, setSecrets []string, valueFiles []string, repo string) error {

	if !isGitOps {
		return o.InstallChartWithOptions(helm.InstallChartOptions{ReleaseName: releaseName, Chart: chart, Version: version,
			Ns: ns, HelmUpdate: helmUpdate, SetValues: append(setValues, setSecrets...), ValueFiles: valueFiles, Repository: repo})
	}

	if gitOpsEnvDir == "" {
		return fmt.Errorf("currently GitOps mode is only supported using the local file system for install time use only")
	}

	if version == "" {
		var err error
		version, err = o.GetVersionNumber(version2.KindChart, chart, "", "")
		if err != nil {
			return err
		}
	}
	if repo == "" {
		repo = kube.DefaultChartMuseumURL
	}

	// lets strip the repo name from the helm chart name
	paths := strings.SplitN(chart, "/", 2)
	if len(paths) > 1 {
		chart = paths[1]
	}

	valuesFiles := &environments.ValuesFiles{
		Items: valueFiles,
	}
	if len(setValues) > 0 {
		extraValues := helm.SetValuesToMap(setValues)
		fileName, err := ioutil.TempFile("", "values.yaml")
		defer func() {
			err := util.DeleteFile(fileName.Name())
			if err != nil {
				log.Logger().Errorf("deleting file %s: %v", fileName.Name(), err)
			}
		}()
		if err != nil {
			return errors.Wrapf(err, "creating temp file to write extra values to")
		}
		err = helm.SaveFile(fileName.Name(), extraValues)
		if err != nil {
			return errors.Wrapf(err, "writing extra values to %s\n%s\n", fileName.Name(), extraValues)
		}
		valuesFiles.Items = append(valuesFiles.Items, fileName.Name())
	}

	//Needed for generated Apps - Otherwise the main repo's Chart.yml is used and the Apps metadata is left empty
	chartUntarDir, _ := ioutil.TempDir("", chart+uuid.NewUUID().String())
	err := o.Helm().FetchChart(chart, version, true, chartUntarDir, repo, "", "")
	if err != nil {
		log.Logger().Errorf("unable to fetch chart - %s", err)
	}

	// Maybe modify the gitOpsEnvDir to after a fetch
	modifyFn := environments.CreateAddRequirementFn(chart, alias, version, repo, valuesFiles, filepath.Join(chartUntarDir, chart), o.Verbose, o.Helm())

	if len(setSecrets) > 0 {
		secretsFile := filepath.Join(gitOpsEnvDir, helm.SecretsFileName)
		secretValues, err := helm.LoadValuesFile(secretsFile)
		if err != nil {
			return err
		}
		secretValues[alias] = helm.SetValuesToMap(setSecrets)
		err = helm.SaveFile(secretsFile, secretValues)
		if err != nil {
			return err
		}
	}

	// if we are part of an initial installation we won't have done a git push yet so lets just write to the gitOpsEnvDir where the dev env chart is
	return environments.ModifyChartFiles(gitOpsEnvDir, nil, modifyFn, "")
}

// InstallChartAt installs the given chart
func (o *CommonOptions) InstallChartAt(dir string, releaseName string, chart string, version string, ns string,
	helmUpdate bool, setValues []string, valueFiles []string, repo string) error {
	return o.InstallChartWithOptions(helm.InstallChartOptions{Dir: dir, ReleaseName: releaseName, Chart: chart,
		Version: version, Ns: ns, HelmUpdate: helmUpdate, SetValues: setValues, ValueFiles: valueFiles, Repository: repo})
}

// InstallChartWithOptions uses the options to run helm install or helm upgrade
func (o *CommonOptions) InstallChartWithOptions(options helm.InstallChartOptions) error {
	return o.InstallChartWithOptionsAndTimeout(options, DefaultInstallTimeout)
}

// InstallChartWithOptionsAndTimeout uses the options and the timeout to run helm install or helm upgrade
func (o *CommonOptions) InstallChartWithOptionsAndTimeout(options helm.InstallChartOptions, timeout string) error {
	client, err := o.KubeClient()
	if err != nil {
		return err
	}
	if options.VersionsDir == "" {
		options.VersionsDir, err = o.CloneJXVersionsRepo(options.VersionsGitURL, options.VersionsGitRef)
		if err != nil {
			return err
		}
	}
	secretURLClient, err := o.GetSecretURLClient()
	if err != nil {
		return errors.Wrap(err, "failed to create a Secret RL client")
	}
	return helm.InstallFromChartOptions(options, o.Helm(), client, timeout, secretURLClient)
}

// GetSecretURLClient create a new secret URL client
func (o *CommonOptions) GetSecretURLClient() (secreturl.Client, error) {
	if o.secretURLClient == nil {
		var err error
		o.secretURLClient, err = o.SystemVaultClient(o.devNamespace)
		if err != nil {
			log.Logger().Warnf("failed to create system vault in namespace %s due to %s\n", o.devNamespace, err.Error())
			o.secretURLClient = nil
		}
	}
	if o.secretURLClient == nil {
		dir, err := util.LocalFileSystemSecretsDir()
		if err != nil {
			return o.secretURLClient, err
		}
		o.secretURLClient = localvault.NewFileSystemClient(dir)
	}
	return o.secretURLClient, nil
}

// SetSecretURLClient sets the Secret URL Client
func (o *CommonOptions) SetSecretURLClient(client secreturl.Client) {
	o.secretURLClient = client
}

// CloneJXVersionsRepo clones the jenkins-x versions repo to a local working dir
func (o *CommonOptions) CloneJXVersionsRepo(versionRepository string, versionRef string) (string, error) {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	configDir, err := util.ConfigDir()
	if err != nil {
		return "", fmt.Errorf("error determining config dir %v", err)
	}
	wrkDir := filepath.Join(configDir, "jenkins-x-versions")

	if versionRepository == "" || versionRef == "" {
		settings, err := o.TeamSettings()
		if err != nil {
			return "", errors.Wrapf(err, "failed to load TeamSettings")
		}
		if versionRepository == "" {
			versionRepository = settings.VersionStreamURL
		}
		if versionRef == "" {
			versionRef = settings.VersionStreamRef
		}
	}
	if versionRepository == "" {
		versionRepository = DefaultVersionsURL
	}

	log.Logger().Debugf("Current configuration dir: %s", configDir)
	log.Logger().Debugf("versionRepository: %s git ref: %s", versionRepository, versionRef)

	// If the repo already exists let's try to fetch the latest version
	if exists, err := util.DirExists(wrkDir); err == nil && exists {
		repo, err := git.PlainOpen(wrkDir)
		if err != nil {
			log.Logger().Errorf("Error opening %s", wrkDir)
			return o.deleteAndReClone(wrkDir, versionRepository, versionRef, o.Out)
		}
		remote, err := repo.Remote("origin")
		if err != nil {
			log.Logger().Errorf("Error getting remote origin")
			return o.deleteAndReClone(wrkDir, versionRepository, versionRef, o.Out)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		remoteRefs := "+refs/heads/master:refs/remotes/origin/master"
		if versionRef != "" {
			remoteRefs = "+refs/heads/" + versionRef + ":refs/remotes/origin/" + versionRef
		}
		err = remote.FetchContext(ctx, &git.FetchOptions{
			RefSpecs: []gitconfig.RefSpec{
				gitconfig.RefSpec(remoteRefs),
			},
		})

		// The repository is up to date
		if err == git.NoErrAlreadyUpToDate {
			if versionRef != "" {
				err = o.Git().Checkout(wrkDir, versionRef)
				if err != nil {
					return o.deleteAndReClone(wrkDir, versionRepository, versionRef, o.Out)
				}
			}
			return wrkDir, nil
		} else if err != nil {
			return o.deleteAndReClone(wrkDir, versionRepository, versionRef, o.Out)
		}

		pullLatest := false
		if o.BatchMode {
			pullLatest = true
		} else if o.AdvancedMode {
			confirm := &survey.Confirm{
				Message: "A local Jenkins X versions repository already exists, pull the latest?",
				Default: true,
			}
			err = survey.AskOne(confirm, &pullLatest, nil, surveyOpts)
			if err != nil {
				log.Logger().Errorf("Error confirming if we should pull latest, skipping %s", wrkDir)
			}
		} else {
			pullLatest = true
			log.Logger().Infof(util.QuestionAnswer("A local Jenkins X versions repository already exists, pulling the latest", util.YesNo(pullLatest)))
		}
		if pullLatest {
			w, err := repo.Worktree()
			if err == nil {
				err := w.Pull(&git.PullOptions{RemoteName: "origin"})
				if err != nil {
					return "", errors.Wrap(err, "pulling the latest")
				}
			}
		}
		if versionRef != "" {
			err = o.Git().Checkout(wrkDir, versionRef)
			if err != nil {
				return o.deleteAndReClone(wrkDir, versionRepository, versionRef, o.Out)
			}
		}
		return wrkDir, err
	} else {
		return o.deleteAndReClone(wrkDir, versionRepository, versionRef, o.Out)
	}
}

func (o *CommonOptions) deleteAndReClone(wrkDir string, versionRepository string, referenceName string, fw terminal.FileWriter) (string, error) {
	log.Logger().Info("Deleting and cloning the Jenkins X versions repo")
	err := os.RemoveAll(wrkDir)
	if err != nil {
		return "", errors.Wrapf(err, "failed to delete dir %s: %s\n", wrkDir, err.Error())
	}
	err = os.MkdirAll(wrkDir, util.DefaultWritePermissions)
	if err != nil {
		return "", errors.Wrapf(err, "failed to ensure directory is created %s", wrkDir)
	}
	err = o.clone(wrkDir, versionRepository, referenceName, fw)
	if err != nil {
		return "", err
	}
	return wrkDir, err
}

func (o *CommonOptions) clone(wrkDir string, versionRepository string, referenceName string, fw terminal.FileWriter) error {
	if referenceName == "" || referenceName == "master" {
		referenceName = "refs/heads/master"
	} else if !strings.Contains(referenceName, "/") {
		if strings.HasPrefix(referenceName, "PR-") {
			prNumber := strings.TrimPrefix(referenceName, "PR-")

			log.Logger().Infof("Cloning the Jenkins X versions repo %s with PR: %s to %s", util.ColorInfo(versionRepository), util.ColorInfo(referenceName), util.ColorInfo(wrkDir))
			return o.shallowCloneGitRepositoryToDir(wrkDir, versionRepository, prNumber, "")
		}
		log.Logger().Infof("Cloning the Jenkins X versions repo %s with revision %s to %s", util.ColorInfo(versionRepository), util.ColorInfo(referenceName), util.ColorInfo(wrkDir))

		err := o.Git().Clone(versionRepository, wrkDir)
		if err != nil {
			return errors.Wrapf(err, "failed to clone repository: %s to dir %s", versionRepository, wrkDir)
		}
		err = o.RunCommandFromDir(wrkDir, "git", "fetch", "origin", referenceName)
		if err != nil {
			return errors.Wrapf(err, "failed to git fetch origin %s for repo: %s in dir %s", referenceName, versionRepository, wrkDir)
		}
		err = o.Git().Checkout(wrkDir, "FETCH_HEAD")
		if err != nil {
			return errors.Wrapf(err, "failed to checkout FETCH_HEAD of repo: %s in dir %s", versionRepository, wrkDir)
		}
		return nil

		// TODO doesn't seem to work at all for a git ref....
		//return o.shallowCloneGitRepositoryToDir(wrkDir, versionRepository, "", referenceName)
	}
	log.Logger().Infof("Cloning the Jenkins X versions repo %s with ref %s to %s", util.ColorInfo(versionRepository), util.ColorInfo(referenceName), util.ColorInfo(wrkDir))
	_, err := git.PlainClone(wrkDir, false, &git.CloneOptions{
		URL:           versionRepository,
		ReferenceName: plumbing.ReferenceName(referenceName),
		SingleBranch:  true,
		Progress:      fw,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to clone reference: %s", referenceName)
	}
	return err
}

func (o *CommonOptions) shallowCloneGitRepositoryToDir(dir string, gitURL string, pullRequestNumber string, revision string) error {
	if pullRequestNumber != "" {
		log.Logger().Infof("shallow cloning pull request %s of repository %s to temp dir %s", gitURL,
			pullRequestNumber, dir)
		err := o.Git().ShallowClone(dir, gitURL, "", pullRequestNumber)
		if err != nil {
			return errors.Wrapf(err, "shallow cloning pull request %s of repository %s to temp dir %s\n", gitURL,
				pullRequestNumber, dir)
		}
	} else if revision != "" {
		log.Logger().Infof("shallow cloning revision %s of repository %s to temp dir %s", gitURL,
			revision, dir)
		err := o.Git().ShallowClone(dir, gitURL, revision, "")
		if err != nil {
			return errors.Wrapf(err, "shallow cloning revision %s of repository %s to temp dir %s\n", gitURL,
				revision, dir)
		}
	} else {
		log.Logger().Infof("shallow cloning master of repository %s to temp dir %s", gitURL, dir)
		err := o.Git().ShallowClone(dir, gitURL, "", "")
		if err != nil {
			return errors.Wrapf(err, "shallow cloning master of repository %s to temp dir %s\n", gitURL, dir)
		}
	}

	return nil
}

// DeleteChart deletes the given chart
func (o *CommonOptions) DeleteChart(releaseName string, purge bool) error {
	_, ns, err := o.KubeClientAndNamespace()
	if err != nil {
		return err
	}

	return o.Helm().DeleteRelease(ns, releaseName, purge)
}

// FindHelmChart finds the helm chart in the current dir
func (o *CommonOptions) FindHelmChart() (string, error) {
	return o.FindHelmChartInDir("")
}

// FindHelmChartInDir finds the helm chart in the given dir. If no dir is specified then the current dir is used
func (o *CommonOptions) FindHelmChartInDir(dir string) (string, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", errors.Wrap(err, "failed to get the current working directory")
		}
	}
	helmer := o.Helm()
	helmer.SetCWD(dir)
	return helmer.FindChart()
}

// FindChartValuesYaml finds the helm chart value.yaml in the given dir. If no dir is specified then the current dir is used
func (o *CommonOptions) FindChartValuesYaml(dir string) (string, error) {
	chartFile, err := o.FindHelmChartInDir(dir)
	if err != nil {
		return "", errors.Wrapf(err, "failed to find helm chart")
	}
	chartDir, _ := filepath.Split(chartFile)
	valuesFileName, err := helm.FindValuesFileName(chartDir)
	if valuesFileName == "" {
		return "", fmt.Errorf("could not find a helm chart values.yaml in the charts folder")
	}

	exists, err := util.FileExists(valuesFileName)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("could not find a helm chart values.yaml in the charts folder %s", chartDir)
	}
	return valuesFileName, nil
}

// DiscoverAppNam discovers an app name from a helm chart installation
func (o *CommonOptions) DiscoverAppName() (string, error) {
	answer := ""
	chartFile, err := o.FindHelmChart()
	if err != nil {
		return answer, err
	}
	if chartFile != "" {
		return helm.LoadChartName(chartFile)
	}

	gitInfo, err := o.Git().Info("")
	if err != nil {
		return answer, err
	}

	if gitInfo == nil {
		return answer, fmt.Errorf("no git info found to discover app name from")
	}
	answer = gitInfo.Name

	if answer == "" {
	}
	return answer, nil
}

// AddChartRepos add chart repositories
func (o *CommonOptions) AddChartRepos(dir string, helmBinary string, chartRepos []string) error {
	installedChartRepos, err := o.GetInstalledChartRepos(helmBinary)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve the install charts")
	}
	if chartRepos != nil {
		for _, url := range chartRepos {
			if !util.StringMapHasValue(installedChartRepos, url) {
				_, err = o.AddHelmBinaryRepoIfMissing(url, "", "", "")
				if err != nil {
					return errors.Wrapf(err, "failed to add the Helm repository with URL '%s'", url)
				}
			}
		}
	}

	reqfile := filepath.Join(dir, helm.RequirementsFileName)
	exists, err := util.FileExists(reqfile)
	if err != nil {
		return errors.Wrapf(err, "requirements.yaml file not found in the chart directory '%s'", dir)
	}
	if exists {
		requirements, err := helm.LoadRequirementsFile(reqfile)
		if err != nil {
			return errors.Wrap(err, "failed to load the Helm requirements file")
		}
		if requirements != nil {
			changed := false
			// lets replace the release chart museum URL if required
			chartRepoURL := o.ReleaseChartMuseumUrl()
			if chartRepoURL != "" && chartRepoURL != DefaultChartRepo {
				for i := range requirements.Dependencies {
					if requirements.Dependencies[i].Repository == DefaultChartRepo {
						requirements.Dependencies[i].Repository = chartRepoURL
						changed = true
					}
				}
			}
			for _, dep := range requirements.Dependencies {
				repo := dep.Repository
				if repo != "" && !util.StringMapHasValue(installedChartRepos, repo) && repo != DefaultChartRepo && !strings.HasPrefix(repo, "file:") && !strings.HasPrefix(repo, "alias:") && !strings.HasPrefix(repo, "@") {
					name, err := o.AddHelmBinaryRepoIfMissing(repo, "", "", "")
					if err != nil {
						return errors.Wrapf(err, "failed to add Helm repository '%s'", repo)
					}
					dep.Repository = fmt.Sprintf("@%s", name)
					changed = true
				}
			}
			if changed {
				err := helm.SaveFile(reqfile, requirements)
				if err != nil {
					return errors.Wrap(err, "failed to save the Helm requirements file")
				}
			}
		}
	}
	return nil
}

// GetInstalledChartRepos retruns the installed chart repositories
func (o *CommonOptions) GetInstalledChartRepos(helmBinary string) (map[string]string, error) {
	return o.Helm().ListRepos()
}

// HelmInit initialises helm
func (o *CommonOptions) HelmInit(dir string) error {
	o.Helm().SetCWD(dir)
	if o.Helm().HelmBinary() == "helm" {
		// need to check the tiller settings at this point
		_, noTiller, helmTemplate, err := o.TeamHelmBin()
		if err != nil {
			return errors.Wrap(err, "failed to access team settings")
		}

		if noTiller || helmTemplate {
			return o.Helm().Init(true, "", "", false)
		} else {
			return o.Helm().Init(false, "", "", true)
		}
	} else {
		return o.Helm().Init(false, "", "", false)
	}
}

// HelmInitDependency initialises helm dependencies
func (o *CommonOptions) HelmInitDependency(dir string, chartRepos []string) (string, error) {
	o.Helm().SetCWD(dir)
	err := o.Helm().RemoveRequirementsLock()
	if err != nil {
		return o.Helm().HelmBinary(),
			errors.Wrapf(err, "failed to remove requirements.lock file from chart '%s'", dir)
	}

	if o.Helm().HelmBinary() == "helm" {
		// need to check the tiller settings at this point
		_, noTiller, helmTemplate, err := o.TeamHelmBin()
		if err != nil {
			return o.Helm().HelmBinary(),
				errors.Wrap(err, "failed to access team settings")
		}

		if noTiller || helmTemplate {
			err = o.Helm().Init(true, "", "", false)
		} else {
			err = o.Helm().Init(false, "", "", true)
		}
	} else {
		err = o.Helm().Init(false, "", "", false)
	}

	if err != nil {
		return o.Helm().HelmBinary(),
			errors.Wrap(err, "failed to initialize Helm")
	}
	err = o.AddChartRepos(dir, o.Helm().HelmBinary(), chartRepos)
	if err != nil {
		return o.Helm().HelmBinary(),
			errors.Wrap(err, "failed to add chart repositories")
	}

	return o.Helm().HelmBinary(), nil
}

// HelmInitDependencyBuild initialises the dependencies an run the build
func (o *CommonOptions) HelmInitDependencyBuild(dir string, chartRepos []string, valuesFiles []string) (string, error) {
	helmBin, err := o.HelmInitDependency(dir, chartRepos)
	if err != nil {
		return helmBin, err
	}
	// TODO due to this issue: https://github.com/kubernetes/helm/issues/4230
	// lets stick with helm2 for this step
	//
	helmBinary := o.Helm().HelmBinary()
	o.Helm().SetHelmBinary("helm")
	o.Helm().SetCWD(dir)
	err = o.Helm().BuildDependency()
	if err != nil {
		return helmBinary, errors.Wrapf(err, "failed to build the dependencies of chart '%s'", dir)
	}

	o.Helm().SetHelmBinary(helmBinary)
	_, err = o.Helm().Lint(valuesFiles)
	if err != nil {
		return helmBinary, errors.Wrapf(err, "failed to lint the chart '%s'", dir)
	}
	return helmBinary, nil
}

// HelmInitRecursiveDependencyBuild helm initialises the dependencies recursively
func (o *CommonOptions) HelmInitRecursiveDependencyBuild(dir string, chartRepos []string, valuesFiles []string) error {
	_, err := o.HelmInitDependency(dir, chartRepos)
	if err != nil {
		return errors.Wrap(err, "initializing Helm")
	}

	helmBinary := o.Helm().HelmBinary()
	o.Helm().SetHelmBinary("helm")
	o.Helm().SetCWD(dir)
	err = o.Helm().BuildDependency()
	if err != nil {
		return errors.Wrapf(err, "failed to build the dependencies of chart '%s'", dir)
	}

	reqFilePath := filepath.Join(dir, helm.RequirementsFileName)
	reqs, err := helm.LoadRequirementsFile(reqFilePath)
	if err != nil {
		return errors.Wrap(err, "loading the requirements file")
	}

	type chartDep struct {
		path string
		deps []*helm.Dependency
	}

	baseChartPath := filepath.Join(dir, "charts")
	depQueue := []chartDep{{
		path: baseChartPath,
		deps: reqs.Dependencies,
	}}

	for {
		if len(depQueue) == 0 {
			break
		}
		currChartDep := depQueue[0]
		depQueue = depQueue[1:]
		for _, dep := range currChartDep.deps {
			chartArchive := filepath.Join(currChartDep.path, fmt.Sprintf("%s-%s.tgz", dep.Name, dep.Version))
			chartPath := filepath.Join(currChartDep.path, dep.Name)
			err := os.MkdirAll(chartPath, os.ModePerm)
			if err != nil {
				return errors.Wrap(err, "creating directory")
			}
			err = util.UnTargz(chartArchive, chartPath, []string{})
			if err != nil {
				return errors.Wrap(err, "extracting chart")
			}
			o.Helm().SetCWD(chartPath)
			err = o.Helm().BuildDependency()
			if err != nil {
				return errors.Wrap(err, "building Helm dependency")
			}
			chartReqFile := filepath.Join(chartPath, helm.RequirementsFileName)
			reqs, err := helm.LoadRequirementsFile(chartReqFile)
			if err != nil {
				return errors.Wrap(err, "loading the requirements file")
			}
			if len(reqs.Dependencies) > 0 {
				depQueue = append(depQueue, chartDep{
					path: filepath.Join(chartPath, "charts"),
					deps: reqs.Dependencies,
				})
			}
		}
	}

	o.Helm().SetHelmBinary(helmBinary)
	_, err = o.Helm().Lint(valuesFiles)
	if err != nil {
		return errors.Wrapf(err, "linting the chart '%s'", dir)
	}

	return nil
}

// DefaultReleaseCharts returns the default release charts
func (o *CommonOptions) DefaultReleaseCharts() []string {
	releasesURL := o.ReleaseChartMuseumUrl()
	answer := []string{
		kube.DefaultChartMuseumURL,
	}
	if releasesURL != "" {
		answer = append(answer, releasesURL)
	}
	return answer
}

func (o *CommonOptions) ReleaseChartMuseumUrl() string {
	if o.RemoteCluster {
		return ""
	}
	chartRepo := os.Getenv("CHART_REPOSITORY")
	if chartRepo == "" {
		if o.factory.IsInCDPipeline() {
			chartRepo = DefaultChartRepo
			log.Logger().Warnf("No $CHART_REPOSITORY defined so using the default value of: %s", DefaultChartRepo)
		} else {
			return ""
		}
	}
	return chartRepo
}

// EnsureHelm ensures helm is installed
func (o *CommonOptions) EnsureHelm() error {
	_, err := o.Helm().Version(false)
	if err == nil {
		return nil
	}
	err = o.InstallHelm()
	if err != nil {
		return errors.Wrap(err, "failed to install Helm")
	}
	cfg := o.defaultInitHelmConfig()
	err = o.InitHelm(cfg)
	if err != nil {
		return errors.Wrapf(err, "initializing helm with config: %v", cfg)
	}
	return nil
}

// ModifyHelmValuesFile modifies the helm values.yaml file using some kind of callback
func (o *CommonOptions) ModifyHelmValuesFile(dir string, fn func(string) (string, error)) error {
	// lets find the project dir...
	valuesFileName, err := o.FindChartValuesYaml(dir)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(valuesFileName)
	if err != nil {
		return errors.Wrapf(err, "failed to read file %s", valuesFileName)
	}

	text := string(data)
	text, err = fn(text)
	if err != nil {
		return errors.Wrapf(err, "failed to process file %s", valuesFileName)
	}

	err = ioutil.WriteFile(valuesFileName, []byte(text), util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to write file %s", valuesFileName)
	}

	log.Logger().Infof("modified the helm file: %s", util.ColorInfo(valuesFileName))
	return nil
}
