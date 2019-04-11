package cmd

import (
	"strings"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/sirupsen/logrus"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdateWebhooksOptions the flags for updating webhooks
type UpdateWebhooksOptions struct {
	*opts.CommonOptions
	Org             string
	Repo            string
	ExactHookMatch  bool
	PreviousHookUrl string
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
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Org, "org", "o", "jenkins-x", "The name of the git organisation to query")
	cmd.Flags().StringVarP(&options.Repo, "repo", "r", "", "The name of the repository to query")
	cmd.Flags().BoolVarP(&options.ExactHookMatch, "exact-hook-url-match", "", true, "Whether to exactly match the hook based on the URL")
	cmd.Flags().StringVarP(&options.PreviousHookUrl, "previous-hook-url", "", "", "Whether to match based on an another URL")

	return cmd
}

func createUpdateWebhooksOptions(commonOpts *opts.CommonOptions) UpdateWebhooksOptions {
	options := UpdateWebhooksOptions{
		CommonOptions: commonOpts,
	}
	return options
}

func (options *UpdateWebhooksOptions) Run() error {
	authConfigService, err := options.CreateGitAuthConfigService()
	if err != nil {
		return errors.Wrap(err, "failed to create git auth service")
	}

	client, currentNamespace, err := options.KubeClientAndNamespace()
	if err != nil {
		return errors.Wrap(err, "failed to get kube client")
	}

	ns, _, err := kube.GetDevNamespace(client, currentNamespace)
	if err != nil {
		return err
	}

	webhookUrl, err := options.GetWebHookEndpoint()
	if err != nil {
		return err
	}

	isProwEnabled, err := options.IsProw()
	if err != nil {
		return err
	}

	hmacToken, err := client.CoreV1().Secrets(ns).Get("hmac-token", metav1.GetOptions{})
	if err != nil {
		return err
	}

	gitServer := authConfigService.Config().CurrentServer

	git, err := options.GitProviderForGitServerURL(gitServer, "github")
	if err != nil {
		return errors.Wrap(err, "unable to determine git provider")
	}

	if options.Repo != "" {
		options.updateRepoHook(git, options.Repo, webhookUrl, isProwEnabled, hmacToken)
	} else {
		repositories, err := git.ListRepositories(options.Org)
		if err != nil {
			return errors.Wrap(err, "unable to list repositories")
		}

		logrus.Infof("Found %v repos\n", util.ColorInfo(len(repositories)))

		for _, repo := range repositories {
			options.updateRepoHook(git, repo.Name, webhookUrl, isProwEnabled, hmacToken)
		}
	}

	return nil
}

func (options *UpdateWebhooksOptions) updateRepoHook(git gits.GitProvider, repoName string, webhookURL string, isProwEnabled bool, hmacToken *corev1.Secret) error {
	webhooks, err := git.ListWebHooks(options.Org, repoName)
	if err != nil {
		return errors.Wrap(err, "unable to list webhooks")
	}

	logrus.Infof("Checking hooks for repository %s\n", util.ColorInfo(repoName))

	if len(webhooks) > 0 {
		// find matching hook
		for _, webHook := range webhooks {
			if options.matches(webhookURL, webHook) {
				logrus.Infof("Found matching hook for url %s\n", util.ColorInfo(webHook.URL))

				// update
				webHookArgs := &gits.GitWebHookArguments{
					ID:    webHook.ID,
					Owner: options.Org,
					Repo: &gits.GitRepository{
						Name: repoName,
					},
					URL:         webhookURL,
					ExistingURL: options.PreviousHookUrl,
				}

				if isProwEnabled {
					webHookArgs.Secret = string(hmacToken.Data["hmac"])
				}

				if !options.DryRun {
					git.UpdateWebHook(webHookArgs)
				}
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
