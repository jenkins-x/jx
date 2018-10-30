package cmd

import (
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdateWebhooks the flags for running create cluster
type UpdateWebhooksOptions struct {
	CommonOptions
	Org             string
	ExactHookMatch  bool
	PreviousHookUrl string
}

var (
	updateWebhooksLong = templates.LongDesc(`
		
		Not currently implemented.

`)

	updateWebhooksExample = templates.Examples(`

		jx update webhooks

`)
)

func NewCmdUpdateWebhooks(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := createUpdateWebhooksOptions(f, in, out, errOut)

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
	cmd.Flags().BoolVarP(&options.ExactHookMatch, "exact-hook-url-match", "", true, "Whether to exactly match the hook based on the URL")
	cmd.Flags().StringVarP(&options.PreviousHookUrl, "previous-hook-url", "", "", "Whether to match based on an another URL")

	return cmd
}

func createUpdateWebhooksOptions(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) UpdateWebhooksOptions {
	commonOptions := CommonOptions{
		Factory: f,
		In:      in,
		Out:     out,
		Err:     errOut,
	}
	options := UpdateWebhooksOptions{
		CommonOptions: commonOptions,
	}
	return options
}

func (options *UpdateWebhooksOptions) Run() error {
	authConfigService, err := options.CreateGitAuthConfigService()
	if err != nil {
		return errors.Wrap(err, "failed to create git auth service")
	}

	_, _, err = options.KubeClient()
	if err != nil {
		return errors.Wrap(err, "failed to get kube client")
	}

	ns, _, err := kube.GetDevNamespace(options.KubeClientCached, options.currentNamespace)
	if err != nil {
		return err
	}

	baseURL, err := kube.GetServiceURLFromName(options.KubeClientCached, "hook", ns)
	if err != nil {
		return err
	}

	webhookUrl := util.UrlJoin(baseURL, "hook")
	hmacToken, err := options.KubeClientCached.CoreV1().Secrets(ns).Get("hmac-token", metav1.GetOptions{})
	if err != nil {
		return err
	}

	gitServer := authConfigService.Config().CurrentServer

	git, err := options.gitProviderForGitServerURL(gitServer, "github")
	if err != nil {
		return errors.Wrap(err, "unable to determine git provider")
	}

	repositories, err := git.ListRepositories(options.Org)
	if err != nil {
		return errors.Wrap(err, "unable to list repositories")
	}

	log.Infof("Found %v repos\n", util.ColorInfo(len(repositories)))

	for _, repo := range repositories {
		repoName := repo.Name
		webhooks, err := git.ListWebHooks(options.Org, repoName)
		if err != nil {
			return errors.Wrap(err, "unable to list webhooks")
		}

		log.Infof("Checking hooks for repository %s\n", util.ColorInfo(repo.Name))

		if len(webhooks) > 0 {
			// find matching hook
			for _, webHook := range webhooks {
				if options.matches(webhookUrl, webHook) {
					log.Infof("Found matching hook for url %s\n", util.ColorInfo(webHook.URL))

					// update
					webHookArgs := &gits.GitWebHookArguments{
						Owner: options.Org,
						Repo: &gits.GitRepositoryInfo{
							Name: repo.Name,
						},
						URL:    webhookUrl,
						Secret: string(hmacToken.Data["hmac"]),
					}

					log.Infof("Updating WebHook with new args\n")

					git.UpdateWebHook(webHookArgs)
				}
			}
		}
	}

	return nil
}

func (options *UpdateWebhooksOptions) matches(webhookUrl string, webHookArgs *gits.GitWebHookArguments) bool {
	if "" != options.PreviousHookUrl {
		return options.PreviousHookUrl == webHookArgs.URL
	}

	if options.ExactHookMatch {
		return webhookUrl == webHookArgs.URL
	} else {
		return strings.Contains(webHookArgs.URL, "hook.jx")
	}
}
