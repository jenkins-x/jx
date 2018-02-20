package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

var (
	preview_long = templates.LongDesc(`
		Creates or updates a Preview environment for the given Pull Request or Branch
`)

	preview_example = templates.Examples(`
		# Create or updates the Preview Environment for the Pull Request
		jx preview
	`)
)

// PreviewOptions the options for the create spring command
type PreviewOptions struct {
	PromoteOptions

	Name           string
	Label          string
	Namespace      string
	Cluster        string
	PullRequestURL string
	PullRequest    string
	SourceURL      string
	SourceRef      string
	Dir            string
	GitConfDir     string
	GitProvider    gits.GitProvider

	HelmValuesConfig config.HelmValuesConfig
}

// NewCmdPreview creates a command object for the "create" command
func NewCmdPreview(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &PreviewOptions{
		HelmValuesConfig: config.HelmValuesConfig{
			ExposeController: &config.ExposeController{},
		},
		PromoteOptions: PromoteOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "preview",
		Short:   "Creates or updates a Preview Environment for the current version of an application",
		Long:    preview_long,
		Example: preview_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	//addCreateAppFlags(cmd, &options.CreateOptions)

	cmd.Flags().StringVarP(&options.Name, kube.OptionName, "n", "", "The Environment resource name. Must follow the kubernetes name conventions like Services, Namespaces")
	cmd.Flags().StringVarP(&options.Label, "label", "l", "", "The Environment label which is a descriptive string like 'Production' or 'Staging'")
	cmd.Flags().StringVarP(&options.Namespace, kube.OptionNamespace, "", "", "The Kubernetes namespace for the Environment")
	cmd.Flags().StringVarP(&options.Cluster, "cluster", "c", "", "The Kubernetes cluster for the Environment. If blank and a namespace is specified assumes the current cluster")
	cmd.Flags().StringVarP(&options.Dir, "dir", "", "", "The source directory used to detect the git source URL and reference")
	cmd.Flags().StringVarP(&options.PullRequest, "pr", "", "", "The Pull Request Name (e.g. 'PR-23' or just '23'")
	cmd.Flags().StringVarP(&options.PullRequestURL, "pr-url", "", "", "The Pull Request URL")
	cmd.Flags().StringVarP(&options.SourceURL, "source-url", "s", "", "The source code git URL")
	cmd.Flags().StringVarP(&options.SourceRef, "source-ref", "", "", "The source code git ref (branch/sha)")

	options.HelmValuesConfig.AddExposeControllerValues(cmd, false)
	options.PromoteOptions.addPromoteOptions(cmd)

	return cmd
}

// Run implements the command
func (o *PreviewOptions) Run() error {
	/*
	args := o.Args
	if len(args) > 0 && o.Name == "" {
		o.Name = args[0]
	}
	*/
	f := o.Factory
	jxClient, currentNs, err := f.CreateJXClient()
	if err != nil {
		return err
	}
	kubeClient, _, err := f.CreateClient()
	if err != nil {
		return err
	}
	apisClient, err := f.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	kube.RegisterEnvironmentCRD(apisClient)

	ns, _, err := kube.GetDevNamespace(kubeClient, currentNs)
	if err != nil {
		return err
	}

	app := o.Application
	if app == "" {
		app, err = o.DiscoverAppName()
		if err != nil {
			return err
		}
	}
	o.Application = app

	// TODO fill in default values!
	envName := o.Name
	ens := o.Namespace
	label := o.Label
	prURL := o.PullRequestURL
	sourceRef := o.SourceRef
	sourceURL := o.SourceURL

	if sourceURL == "" {
		// lets discover the git dir
		if o.Dir == "" {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			o.Dir = dir
		}
		root, gitConf, err := gits.FindGitConfigDir(o.Dir)
		if err != nil {
			o.warnf("Could not find a .git directory: %s\n", err)
		} else {
			if root != "" {
				o.Dir = root
				sourceURL, err = o.discoverGitURL(gitConf)
				if err != nil {
					o.warnf("Could not find the remote git source URL:  %s\n", err)
				} else {
					if sourceRef == "" {
						sourceRef, err = gits.GitGetBranch(root)
						if err != nil {
							o.warnf("Could not find the remote git source ref:  %s\n", err)
						}

					}
				}
			}
		}

	}
	if o.PullRequest == "" {
		o.PullRequest = os.Getenv("BRANCH_NAME")
	}
	prName := strings.TrimPrefix(o.PullRequest, "PR-")

	if sourceURL != "" {
		gitInfo, err := gits.ParseGitURL(sourceURL)
		if err != nil {
			o.warnf("Could not parse the git URL %s due to %s\n", sourceURL, err)
		} else {
			sourceURL = gitInfo.HttpCloneURL()
			if prURL == "" {
				if o.PullRequest == "" {
					o.warnf("No Pull Request name or URL specified nor could one be found via $BRANCH_NAME\n")
				} else {
					prURL = gitInfo.PullRequestURL(prName)
				}
			}

			if envName == "" && prName != "" {
				envName = gitInfo.Organisation + "-" + gitInfo.Name + "-pr-" + prName
			}

			if label == "" {
				label = gitInfo.Organisation + "/" + gitInfo.Name + " PR-" + prName
			}
		}
	}

	if envName == "" {
		return fmt.Errorf("No name could be defaulted for the Preview Environment. Please supply one!")
	}
	if ens == "" {
		ens = ns + "-" + envName
	}
	if label == "" {
		label = envName
	}

	env, err := jxClient.JenkinsV1().Environments(ns).Get(envName, metav1.GetOptions{})
	if err == nil {
		// lets check for updates...
		update := false

		spec := &env.Spec
		source := &spec.Source
		if spec.Label != label {
			spec.Label = label
			update = true
		}
		if spec.Namespace != ens {
			spec.Namespace = ens
			update = true
		}
		if spec.Namespace != ens {
			spec.Namespace = ens
			update = true
		}
		if spec.Kind != v1.EnvironmentKindTypePreview {
			spec.Kind = v1.EnvironmentKindTypePreview
			update = true
		}
		if source.Kind != v1.EnvironmentRepositoryTypeGit {
			source.Kind = v1.EnvironmentRepositoryTypeGit
			update = true
		}
		if source.URL != sourceURL {
			source.URL = sourceURL
			update = true
		}
		if source.Ref != sourceRef {
			source.Ref = sourceRef
			update = true
		}

		if update {
			_, err = jxClient.JenkinsV1().Environments(ns).Update(env)
			if err != nil {
				return fmt.Errorf("Failed to update Environment %s due to %s", envName, err)
			}
		}
	}
	if err != nil {
		// lets create a new preview environment
		env = &v1.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name: envName,
			},
			Spec: v1.EnvironmentSpec{
				Namespace:         ens,
				Label:             label,
				Kind:              v1.EnvironmentKindTypePreview,
				PromotionStrategy: v1.PromotionStrategyTypeAutomatic,
				PullRequestURL:    prURL,
				Order:             999,
				Source: v1.EnvironmentRepository{
					Kind: v1.EnvironmentRepositoryTypeGit,
					URL:  sourceURL,
					Ref:  sourceRef,
				},
			},
		}
		_, err = jxClient.JenkinsV1().Environments(ns).Create(env)
		if err != nil {
			return err
		}
		o.Printf("Created environment %s\n", util.ColorInfo(env.Name))
	}

	err = kube.EnsureEnvironmentNamespaceSetup(kubeClient, jxClient, env, ns)
	if err != nil {
		return err
	}
	return nil
}

func (o *PreviewOptions) discoverGitURL(gitConf string) (string, error) {
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
	url := gits.GetRemoteUrl(cfg, "origin")
	if url == "" {
		url = gits.GetRemoteUrl(cfg, "upstream")
		if url == "" {
			url, err = o.pickRemoteURL(cfg)
			if err != nil {
				return "", err
			}
		}
	}
	return url, nil
}
