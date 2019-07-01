package opts

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
)

type GitRepositoryOptions struct {
	ServerURL  string
	ServerKind string
	Username   string
	ApiToken   string
	Owner      string
	RepoName   string
	Private    bool
}

// FindGitInfo parses the git information from the given directory
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

// CreateGitProvider creates a new git provider for the given URL
func (o *CommonOptions) CreateGitProvider(gitURL string) (gits.GitProvider, error) {
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	return o.factory.CreateGitProvider(gitURL, auth.AutoConfigKind, o.Git())
}

// CreateGitProviderFromDir creates a git providder from a given directory which contains a repostiory
func (o *CommonOptions) CreateGitProviderFromDir(dir string) (gits.GitProvider, error) {
	gitDir, gitConfDir, err := o.Git().FindGitConfigDir(dir)
	if err != nil {
		return nil, err
	}
	if gitDir == "" || gitConfDir == "" {
		return nil, fmt.Errorf("no git repository found in directory %q", dir)
	}
	gitURL, err := o.Git().DiscoverUpstreamGitURL(gitConfDir)
	if err != nil {
		return nil, err
	}
	if o.factory == nil {
		return nil, errors.New("command factory is not initialized")
	}
	gitProvider, err := o.factory.CreateGitProvider(gitURL, auth.AutoConfigKind, o.Git())
	if err != nil {
		return nil, err
	}
	return gitProvider, nil
}

// EnsureGitServiceCRD ensure that the GitService CRD is installed
func (o *CommonOptions) EnsureGitServiceCRD(server auth.Server) error {
	kind := server.Kind
	if kind == "github" && server.URL == gits.GitHubURL {
		return nil
	}
	if kind == "" {
		log.Logger().Warnf("Kind of git server %s with URL %s is empty", server.Name, server.URL)
		return nil
	}
	if server.Name == "" {
		server.Name = kind
	}
	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return errors.Wrap(err, "failed to create JX Client")
	}
	err = kube.EnsureGitServiceExistsForHost(jxClient, devNs, kind, server.Name, server.URL, o.Out)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure GitService exists for kind %s server %s in namespace %s", kind, server.URL, devNs)
	}
	log.Logger().Infof("Ensured we have a GitService called %s for URL %s in namespace %s", server.Name, server.URL, devNs)
	return nil
}

// DiscoverGitURL discovers the Git URL
func (o *CommonOptions) DiscoverGitURL(gitConf string) (string, error) {
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
			url, err = o.PickGitRemoteURL(cfg)
			if err != nil {
				return "", err
			}
		}
	}
	return url, nil
}

// AddGitRepoOptionsArguments adds common git flags to the given cobra command
func AddGitRepoOptionsArguments(cmd *cobra.Command, repositoryOptions *GitRepositoryOptions) {
	cmd.Flags().StringVarP(&repositoryOptions.ServerURL, "git-provider-url", "", "https://github.com", "The Git server URL to create new Git repositories inside")
	cmd.Flags().StringVarP(&repositoryOptions.ServerKind, "git-provider-kind", "", "",
		"Kind of Git server. If not specified, kind of server will be autodetected from Git provider URL. Possible values: bitbucketcloud, bitbucketserver, gitea, gitlab, github, fakegit")
	cmd.Flags().StringVarP(&repositoryOptions.Username, "git-username", "", "", "The Git username to use for creating new Git repositories")
	cmd.Flags().StringVarP(&repositoryOptions.ApiToken, "git-api-token", "", "", "The Git API token to use for creating new Git repositories")
	cmd.Flags().BoolVarP(&repositoryOptions.Private, "git-private", "", false, "Create new Git repositories as private")
}

// GitServerKind returns the kind of the git server
func (o *CommonOptions) GitServerKind(gitInfo *gits.GitRepository) (string, error) {
	return o.GitServerHostURLKind(gitInfo.HostURL())
}

// GitServerHostURLKind returns the kind of git server host URL
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

// InitGitConfigAndUser validates we have git setup
func (o *CommonOptions) InitGitConfigAndUser() error {
	// lets validate we have git configured
	_, _, err := gits.EnsureUserAndEmailSetup(o.Git())
	if err != nil {
		return err
	}

	err = o.RunCommandVerbose("git", "config", "--global", "credential.helper", "store")
	if err != nil {
		return err
	}
	if os.Getenv("XDG_CONFIG_HOME") == "" {
		log.Logger().Warnf("Note that the environment variable $XDG_CONFIG_HOME is not defined so we may not be able to push to git!")
	}
	return nil
}
