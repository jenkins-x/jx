package update

import (
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdateWebhooksOptions the flags for updating webhooks
type UpdateWebhooksOptions struct {
	*opts.CommonOptions
	Org             string
	User            string
	Repo            string
	ExactHookMatch  bool
	PreviousHookUrl string
	HMAC            string
	Endpoint        string
	DryRun          bool
}

var (
	updateWebhooksLong = templates.LongDesc(`
		
		Updates the webhook for one repository, or all repositories in an organization.

`)

	updateWebhooksExample = templates.Examples(`

		jx update webhooks --org=mycorp

`)
)

func NewCmdUpdateWebhooks(commonOpts *opts.CommonOptions) *cobra.Command {
	options := createUpdateWebhooksOptions(commonOpts)

	cmd := &cobra.Command{
		Use:     "webhooks",
		Short:   "Updates all webhooks for an existing org",
		Long:    updateWebhooksLong,
		Example: updateWebhooksExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Org, "org", "o", "jenkins-x", "The name of the git organisation to query")
	cmd.Flags().StringVarP(&options.Repo, "repo", "r", "", "The name of the repository to query")
	cmd.Flags().BoolVarP(&options.ExactHookMatch, "exact-hook-url-match", "", true, "Whether to exactly match the hook based on the URL")
	cmd.Flags().StringVarP(&options.PreviousHookUrl, "previous-hook-url", "", "", "Whether to match based on an another URL")
	cmd.Flags().StringVarP(&options.HMAC, "hmac", "", "", "Don't use the HMAC token from the cluster, use the provided token")
	cmd.Flags().StringVarP(&options.Endpoint, "endpoint", "", "", "Don't use the endpoint from the cluster, use the provided endpoint")

	return cmd
}

func createUpdateWebhooksOptions(commonOpts *opts.CommonOptions) UpdateWebhooksOptions {
	options := UpdateWebhooksOptions{
		CommonOptions: commonOpts,
	}
	return options
}

func (options *UpdateWebhooksOptions) Run() error {
	client, currentNamespace, err := options.KubeClientAndNamespace()
	if err != nil {
		return errors.Wrap(err, "failed to get kube client")
	}

	ns, _, err := kube.GetDevNamespace(client, currentNamespace)
	if err != nil {
		return err
	}

	webhookURL := ""
	if options.Endpoint != "" {
		webhookURL = options.Endpoint
	} else {
		webhookURL, err = options.GetWebHookEndpoint()
		if err != nil {
			return err
		}
	}

	isProwEnabled, err := options.IsProw()
	if err != nil {
		return err
	}

	hmacToken := ""
	if isProwEnabled {
		if options.HMAC != "" {
			hmacToken = options.HMAC
		} else {
			hmacTokenSecret, err := client.CoreV1().Secrets(ns).Get("hmac-token", metav1.GetOptions{})
			if err != nil {
				return err
			}
			hmacToken = string(hmacTokenSecret.Data["hmac"])
		}
	}

	git, err := options.GitProviderForGitServerURL(gits.GitHubURL, "github")
	if err != nil {
		return errors.Wrap(err, "unable to determine git provider")
	}
	owner := GetOrgOrUserFromOptions(options)

	if options.Verbose {
		log.Logger().Infof("Updating webhooks for Owner %v and Repo %v", owner, options.Repo)
	}

	if options.Repo != "" {
		options.updateRepoHook(git, options.Repo, webhookURL, isProwEnabled, hmacToken)
	} else {
		if owner == "" {
			return errors.Wrap(err, "unable to list repositories - no repo owner")
		}

		repositories, err := git.ListRepositories(owner)
		if err != nil {
			return errors.Wrap(err, "unable to list repositories")
		}

		log.Logger().Infof("Found %v repos", util.ColorInfo(len(repositories)))

		for _, repo := range repositories {
			options.updateRepoHook(git, repo.Name, webhookURL, isProwEnabled, hmacToken)
		}
	}

	return nil
}

// GetOrgOrUserFromOptions returns the Org if set,
// if not set, returns the user if that is set
// or "" if neither is set
func GetOrgOrUserFromOptions(options *UpdateWebhooksOptions) string {
	owner := options.Org
	if owner == "" && options.Username != "" {
		owner = options.Username
	}
	return owner
}

func (options *UpdateWebhooksOptions) updateRepoHook(git gits.GitProvider, repoName string, webhookURL string, isProwEnabled bool, hmacToken string) error {
	log.Logger().Infof("Checking hooks for repository %s with user %s", util.ColorInfo(repoName), util.ColorInfo(git.UserAuth().Username))

	webhooks, err := git.ListWebHooks(options.Org, repoName)
	if err != nil {
		return errors.Wrap(err, "unable to list webhooks")
	}

	webHookArgs := &gits.GitWebHookArguments{
		Owner: options.Org,
		Repo: &gits.GitRepository{
			Name: repoName,
		},
		URL: webhookURL,
	}
	if isProwEnabled {
		webHookArgs.Secret = hmacToken
	}
	if len(webhooks) > 0 {
		// find matching hook
		for _, webHook := range webhooks {
			if options.matches(webhookURL, webHook) {
				log.Logger().Infof("Found matching hook for url %s", util.ColorInfo(webHook.URL))
				webHookArgs.ID = webHook.ID
				webHookArgs.ExistingURL = options.PreviousHookUrl
				if !options.DryRun {
					if err := git.UpdateWebHook(webHookArgs); err != nil {
						return errors.Wrapf(err, "updating the webhook %q on repository '%s/%s'",
							webhookURL, options.Org, repoName)
					}
				}
			}
		}
	} else {
		if !options.DryRun {
			if err := git.CreateWebHook(webHookArgs); err != nil {
				return errors.Wrapf(err, "creating the webhook %q on repository '%s/%s'",
					webhookURL, options.Org, repoName)
			}
		}
	}
	return nil
}

func (options *UpdateWebhooksOptions) matches(webhookURL string, webHookArgs *gits.GitWebHookArguments) bool {
	if "" != options.PreviousHookUrl {
		return options.PreviousHookUrl == webHookArgs.URL
	}

	if options.ExactHookMatch {
		return webhookURL == webHookArgs.URL
	} else {
		return strings.Contains(webHookArgs.URL, "hook.jx")
	}
}
