package cmd

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/issues"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (o *CommonOptions) FindGitInfo(dir string) (*gits.GitRepository, error) {
	_, gitConf, err := o.Git().FindGitConfigDir(dir)
	if err != nil {
		return nil, fmt.Errorf("Could not find a .git directory: %s\n", err)
	} else {
		if gitConf == "" {
			return nil, fmt.Errorf("No git conf dir found")
		}
		gitURL, err := o.Git().DiscoverUpstreamGitURL(gitConf)
		if err != nil {
			return nil, fmt.Errorf("Could not find the remote git source URL:  %s", err)
		}
		return gits.ParseGitURL(gitURL)
	}
}

// createGitProvider creates a git from the given directory
func (o *CommonOptions) createGitProvider(dir string) (*gits.GitRepository, gits.GitProvider, issues.IssueProvider, error) {
	gitDir, gitConfDir, err := o.Git().FindGitConfigDir(dir)
	if err != nil {
		return nil, nil, nil, err
	}
	if gitDir == "" || gitConfDir == "" {
		log.Warnf("No git directory could be found from dir %s\n", dir)
		return nil, nil, nil, nil
	}

	gitUrl, err := o.Git().DiscoverUpstreamGitURL(gitConfDir)
	if err != nil {
		return nil, nil, nil, err
	}
	gitInfo, err := gits.ParseGitURL(gitUrl)
	if err != nil {
		return nil, nil, nil, err
	}
	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return gitInfo, nil, nil, err
	}
	gitKind, err := o.GitServerKind(gitInfo)
	gitProvider, err := gitInfo.CreateProvider(o.IsInCluster(), authConfigSvc, gitKind, o.Git(), o.BatchMode, o.In, o.Out, o.Err)
	if err != nil {
		return gitInfo, gitProvider, nil, err
	}
	tracker, err := o.createIssueProvider(dir)
	if err != nil {
		return gitInfo, gitProvider, tracker, err
	}
	return gitInfo, gitProvider, tracker, nil
}

func (o *CommonOptions) updatePipelineGitCredentialsSecret(server *auth.AuthServer, userAuth *auth.UserAuth) (string, error) {
	client, curNs, err := o.KubeClientAndNamespace()
	if err != nil {
		return "", err
	}
	ns, _, err := kube.GetDevNamespace(client, curNs)
	if err != nil {
		return "", err
	}
	options := metav1.GetOptions{}
	serverName := server.Name
	name := kube.ToValidName(kube.SecretJenkinsPipelineGitCredentials + server.Kind + "-" + serverName)
	secrets := client.CoreV1().Secrets(ns)
	secret, err := secrets.Get(name, options)
	create := false
	operation := "update"
	labels := map[string]string{
		kube.LabelCredentialsType: kube.ValueCredentialTypeUsernamePassword,
		kube.LabelCreatedBy:       kube.ValueCreatedByJX,
		kube.LabelKind:            kube.ValueKindGit,
		kube.LabelServiceKind:     server.Kind,
	}
	annotations := map[string]string{
		kube.AnnotationCredentialsDescription: fmt.Sprintf("API Token for acccessing %s Git service inside pipelines", server.URL),
		kube.AnnotationURL:                    server.URL,
		kube.AnnotationName:                   serverName,
	}
	if err != nil {
		// lets create a new secret
		create = true
		operation = "create"
		secret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Annotations: annotations,
				Labels:      labels,
			},
			Data: map[string][]byte{},
		}
	} else {
		secret.Annotations = util.MergeMaps(secret.Annotations, annotations)
		secret.Labels = util.MergeMaps(secret.Labels, labels)
	}
	if userAuth.Username != "" {
		secret.Data["username"] = []byte(userAuth.Username)
	}
	if userAuth.ApiToken != "" {
		secret.Data["password"] = []byte(userAuth.ApiToken)
	}
	if create {
		_, err = secrets.Create(secret)
	}
	if err != nil {
		return name, fmt.Errorf("Failed to %s secret %s due to %s", operation, secret.Name, err)
	}

	prow, err := o.isProw()
	if err != nil {
		return name, err
	}
	if prow {
		return name, nil
	}

	// update the Jenkins config
	cm, err := client.CoreV1().ConfigMaps(ns).Get(kube.ConfigMapJenkinsX, metav1.GetOptions{})
	if err != nil {
		return name, fmt.Errorf("Could not load Jenkins ConfigMap: %s", err)
	}

	updated, err := kube.UpdateJenkinsGitServers(cm, server, userAuth, name)
	if err != nil {
		return name, err
	}
	if updated {
		_, err = client.CoreV1().ConfigMaps(ns).Update(cm)
		if err != nil {
			return name, fmt.Errorf("Failed to update Jenkins ConfigMap: %s", err)
		}
		log.Infof("Updated the Jenkins ConfigMap %s\n", kube.ConfigMapJenkinsX)

		// wait a little bit to give k8s chance to sync the ConfigMap to the file system
		time.Sleep(time.Second * 2)

		// lets ensure that the git server + credential is in the Jenkins server configuration
		jenk, err := o.JenkinsClient()
		if err != nil {
			return name, err
		}
		// TODO reload does not seem to reload the plugin content
		//err = jenk.Reload()
		err = jenk.SafeRestart()
		if err != nil {
			log.Warnf("Failed to safe restart Jenkins after configuration change %s\n", err)
		} else {
			log.Infoln("Safe Restarted Jenkins server")

			// Let's wait 5 minutes for Jenkins to come back up.
			// This is kinda gross, but it's just polling Jenkins every second for 5 minutes.
			timeout := time.Duration(5) * time.Minute
			start := int64(time.Now().Nanosecond())
			for int64(time.Now().Nanosecond())-start < timeout.Nanoseconds() {
				_, err := jenk.GetJobs()
				if err == nil {
					break
				}
				log.Infoln("Jenkins returned an error. Waiting for it to recover...")
				time.Sleep(1 * time.Second)
			}
		}
	}

	return name, nil
}

func (o *CommonOptions) ensureGitServiceCRD(server *auth.AuthServer) error {
	kind := server.Kind
	if kind == "github" && server.URL == gits.GitHubURL {
		return nil
	}
	if kind == "" {
		log.Warnf("Kind of git server %s with URL %s is empty\n", server.Name, server.URL)
		return nil
	}
	// lets lazily populate the name if its empty
	if server.Name == "" {
		server.Name = kind
	}
	apisClient, err := o.ApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterGitServiceCRD(apisClient)
	if err != nil {
		return err
	}

	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	err = kube.EnsureGitServiceExistsForHost(jxClient, devNs, kind, server.Name, server.URL, o.Out)
	if err != nil {
		return err
	}
	log.Infof("Ensured we have a GitService called %s for URL %s in namespace %s\n", server.Name, server.URL, devNs)
	return nil
}

func (o *CommonOptions) discoverGitURL(gitConf string) (string, error) {
	if gitConf == "" {
		return "", fmt.Errorf("No GitConfDir defined!")
	}
	cfg := gitcfg.NewConfig()
	data, err := ioutil.ReadFile(gitConf)
	if err != nil {
		return "", fmt.Errorf("Failed to load %s due to %s", gitConf, err)
	}

	err = cfg.Unmarshal(data)
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshal %s due to %s", gitConf, err)
	}
	remotes := cfg.Remotes
	if len(remotes) == 0 {
		return "", nil
	}
	url := o.Git().GetRemoteUrl(cfg, "origin")
	if url == "" {
		url = o.Git().GetRemoteUrl(cfg, "upstream")
		if url == "" {
			url, err = o.pickRemoteURL(cfg)
			if err != nil {
				return "", err
			}
		}
	}
	return url, nil
}

func addGitRepoOptionsArguments(cmd *cobra.Command, repositoryOptions *gits.GitRepositoryOptions) {
	cmd.Flags().StringVarP(&repositoryOptions.ServerURL, "git-provider-url", "", "https://github.com", "The Git server URL to create new Git repositories inside")
	cmd.Flags().StringVarP(&repositoryOptions.ServerKind, "git-provider-kind", "", "",
		"Kind of Git server. If not specified, kind of server will be autodetected from Git provider URL. Possible values: bitbucketcloud, bitbucketserver, gitea, gitlab, github, fakegit")
	cmd.Flags().StringVarP(&repositoryOptions.Username, "git-username", "", "", "The Git username to use for creating new Git repositories")
	cmd.Flags().StringVarP(&repositoryOptions.ApiToken, "git-api-token", "", "", "The Git API token to use for creating new Git repositories")
	cmd.Flags().BoolVarP(&repositoryOptions.Private, "git-private", "", false, "Create new Git repositories as private")
}

func (o *CommonOptions) GitServerKind(gitInfo *gits.GitRepository) (string, error) {
	return o.GitServerHostURLKind(gitInfo.HostURL())
}

func (o *CommonOptions) GitServerHostURLKind(hostURL string) (string, error) {
	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return "", err
	}

	kubeClient, err := o.KubeClient()
	if err != nil {
		return "", err
	}

	kind, err := kube.GetGitServiceKind(jxClient, kubeClient, devNs, hostURL)
	if err != nil {
		return kind, err
	}
	if kind == "" {
		if o.BatchMode {
			return "", fmt.Errorf("No Git server kind could be found for URL %s\nPlease try specify it via: jx create git server someKind %s", hostURL, hostURL)
		}
		kind, err = util.PickName(gits.KindGits, fmt.Sprintf("Pick what kind of Git server is: %s", hostURL), "", o.In, o.Out, o.Err)
		if err != nil {
			return "", err
		}
		if kind == "" {
			return "", fmt.Errorf("No Git kind chosen!")
		}
	}
	return kind, nil
}

// gitProviderForURL returns a GitProvider for the given git URL
func (o *CommonOptions) gitProviderForURL(gitURL string, message string) (gits.GitProvider, error) {
	if o.fakeGitProvider != nil {
		return o.fakeGitProvider, nil
	}
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return nil, err
	}
	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return nil, err
	}
	gitKind, err := o.GitServerKind(gitInfo)
	if err != nil {
		return nil, err
	}
	return gitInfo.PickOrCreateProvider(authConfigSvc, message, o.BatchMode, gitKind, o.Git(), o.In, o.Out, o.Err)
}

// gitProviderForURL returns a GitProvider for the given Git server URL
func (o *CommonOptions) gitProviderForGitServerURL(gitServiceUrl string, gitKind string) (gits.GitProvider, error) {
	if o.fakeGitProvider != nil {
		return o.fakeGitProvider, nil
	}
	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return nil, err
	}
	return gits.CreateProviderForURL(o.IsInCluster(), authConfigSvc, gitKind, gitServiceUrl, o.Git(), o.BatchMode, o.In, o.Out, o.Err)
}

func (o *CommonOptions) createGitProviderForURLWithoutKind(gitURL string) (gits.GitProvider, *gits.GitRepository, error) {
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return nil, gitInfo, err
	}
	gitKind, err := o.GitServerKind(gitInfo)
	if err != nil {
		return nil, gitInfo, err
	}
	provider, err := o.gitProviderForGitServerURL(gitURL, gitKind)
	return provider, gitInfo, err
}
