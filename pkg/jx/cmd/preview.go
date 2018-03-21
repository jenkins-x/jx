package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"strings"

	"path/filepath"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	preview_long = templates.LongDesc(`
		Creates or updates a Preview Environment for the given Pull Request or Branch.

		For more documentation on Preview Environments see: [http://jenkins-x.io/about/features/#preview-environments](http://jenkins-x.io/about/features/#preview-environments)

`)

	preview_example = templates.Examples(`
		# Create or updates the Preview Environment for the Pull Request
		jx preview
	`)
)

const (
	JENKINS_X_DOCKER_REGISTRY_SERVICE_HOST = "JENKINS_X_DOCKER_REGISTRY_SERVICE_HOST"
	JENKINS_X_DOCKER_REGISTRY_SERVICE_PORT = "JENKINS_X_DOCKER_REGISTRY_SERVICE_PORT"
	ORG                                    = "ORG"
	APP_NAME                               = "APP_NAME"
	PREVIEW_VERSION                        = "PREVIEW_VERSION"
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
	err = kube.RegisterEnvironmentCRD(apisClient)
	if err != nil {
		return err
	}
	err = kube.RegisterGitServiceCRD(apisClient)
	if err != nil {
		return err
	}

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

	if sourceURL == "" {
		return fmt.Errorf("No sourceURL could be defaulted for the Preview Environment. Use --dir flag to detect the git source URL")
	}

	if o.PullRequest == "" {
		o.PullRequest = os.Getenv("BRANCH_NAME")
	}
	prName := strings.TrimPrefix(o.PullRequest, "PR-")

	var gitInfo *gits.GitRepositoryInfo
	if sourceURL != "" {
		gitInfo, err = gits.ParseGitURL(sourceURL)
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
	envName = kube.ToValidName(envName)
	if envName == "" {
		return fmt.Errorf("No name could be defaulted for the Preview Environment. Please supply one!")
	}
	if ens == "" {
		ens = ns + "-" + envName
	}
	ens = kube.ToValidName(ens)

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

	if o.ReleaseName == "" {
		o.ReleaseName = ens
	}

	domain, err := kube.GetCurrentDomain(kubeClient, ns)
	if err != nil {
		return err
	}

	repository, err := getImageName()
	if err != nil {
		return err
	}

	tag, err := getImageTag()
	if err != nil {
		return err
	}

	values := config.PreviewValuesConfig{
		ExposeController: &config.ExposeController{
			Config: config.ExposeControllerConfig{
				Domain: domain,
			},
		},
		Preview: &config.Preview{
			Image: &config.Image{
				Repository: repository,
				Tag:        tag,
			},
		},
	}

	config, err := values.String()
	if err != nil {
		return err
	}

	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	configFileName := filepath.Join(dir, ExtraValuesFile)
	log.Infof("%s", config)
	err = ioutil.WriteFile(configFileName, []byte(config), 0644)
	if err != nil {
		return err
	}

	err = o.runCommand("helm", "upgrade", o.ReleaseName, ".", "--force", "--install", "--wait", "--namespace", ens, fmt.Sprintf("--values=%s", configFileName))
	if err != nil {
		return err
	}

	url := ""
	appNames := []string{app, o.ReleaseName, ens + "-preview", o.ReleaseName + "-" + app}
	for _, n := range appNames {
		url, err = kube.FindServiceURL(kubeClient, ens, n)
		if url != "" {
			break
		}
	}

	if url == "" {
		o.warnf("Could not find the service URL in namespace %s for names %s\n", ens, strings.Join(appNames, ", "))
	}

	comment := fmt.Sprintf(":star: PR built and available in a preview environment **%s**", envName)
	if url != "" {
		comment += fmt.Sprintf(" [here](%s) ", url)
	}

	stepPRCommentOptions := StepPRCommentOptions{
		Flags: StepPRCommentFlags{
			Owner:      gitInfo.Organisation,
			Repository: gitInfo.Name,
			Comment:    comment,
			PR:         prName,
		},
		StepPROptions: StepPROptions{
			StepOptions: StepOptions{
				CommonOptions: CommonOptions{
					BatchMode: true,
					Factory:   o.Factory,
				},
			},
		},
	}
	return stepPRCommentOptions.Run()

}

func getImageName() (string, error) {
	registryHost := os.Getenv(JENKINS_X_DOCKER_REGISTRY_SERVICE_HOST)
	if registryHost == "" {
		return "", fmt.Errorf("no %s environment variable found", JENKINS_X_DOCKER_REGISTRY_SERVICE_HOST)
	}
	registryPort := os.Getenv(JENKINS_X_DOCKER_REGISTRY_SERVICE_PORT)
	if registryHost == "" {
		return "", fmt.Errorf("no %s environment variable found", JENKINS_X_DOCKER_REGISTRY_SERVICE_PORT)
	}

	organisation := os.Getenv(ORG)
	if registryHost == "" {
		return "", fmt.Errorf("no %s environment variable found", ORG)
	}

	app := os.Getenv(APP_NAME)
	if registryHost == "" {
		return "", fmt.Errorf("no %s environment variable found", APP_NAME)
	}

	return fmt.Sprintf("%s:%s/%s/%s", registryHost, registryPort, organisation, app), nil
}

func getImageTag() (string, error) {

	tag := os.Getenv(PREVIEW_VERSION)
	if tag == "" {
		return "", fmt.Errorf("no %s environment variable found", PREVIEW_VERSION)
	}

	return tag, nil
}
