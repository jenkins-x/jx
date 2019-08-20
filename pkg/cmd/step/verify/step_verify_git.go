package verify

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// StepVerifyGitOptions contains the command line flags
type StepVerifyGitOptions struct {
	opts.StepOptions
}

// NewCmdStepVerifyGit creates the `jx step verify pod` command
func NewCmdStepVerifyGit(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepVerifyGitOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use: "git",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	return cmd
}

// Run implements this command
func (o *StepVerifyGitOptions) Run() error {
	log.Logger().Infof("verifying the git Secrets\n")

	secrets, err := o.LoadPipelineSecrets(kube.ValueKindGit, "")
	if err != nil {
		return err
	}
	info := util.ColorInfo
	for _, secret := range secrets.Items {
		log.Logger().Infof("verifying git Secret %s\n", info(secret.Name))
		annotations := secret.Annotations
		data := secret.Data
		if annotations == nil {
			return fmt.Errorf("no annotations on Secret %s", secret.Name)
		}
		if data == nil {
			return fmt.Errorf("no Data on Secret %s", secret.Name)
		}
		u := annotations[kube.AnnotationURL]
		if u == "" {
			return fmt.Errorf("secret %s does not have a Git URL annotation %s", secret.Name, kube.AnnotationURL)
		}
		username := data[kube.SecretDataUsername]
		pwd := data[kube.SecretDataPassword]
		if username == nil {
			return fmt.Errorf("secret %s does not have a Git username annotation %s", secret.Name, kube.SecretDataUsername)
		}
		if pwd == nil {
			return fmt.Errorf("secret %s does not have a Git password annotation %s", secret.Name, kube.SecretDataPassword)
		}
	}

	config := &auth.AuthConfig{}
	err = o.GetFactory().AuthMergePipelineSecrets(config, secrets, kube.ValueKindGit, true)
	if err != nil {
		return err
	}
	servers := config.Servers
	if len(servers) == 0 {
		return fmt.Errorf("failed to find any Git servers from the Git Secrets. There should be a Secret with label %s=%s", kube.LabelKind, kube.ValueKindGit)
	}
	pipeUserValid := false
	for _, server := range servers {
		for _, userAuth := range server.Users {

			log.Logger().Infof("verifying username %s at git server %s at %s\n", info(userAuth.Username), info(server.Name), info(server.URL))

			provider, err := gits.CreateProvider(server, userAuth, o.Git())
			if err != nil {
				return errors.Wrapf(err, "failed to create GitProvider for %s at git server %s", userAuth.Username, server.URL)
			}

			orgs, err := provider.ListOrganisations()
			if err != nil {
				return errors.Wrapf(err, "failed to list the organisations for %s at git server %s", userAuth.Username, server.URL)
			}
			orgNames := []string{}
			for _, org := range orgs {
				orgNames = append(orgNames, org.Login)
			}
			sort.Strings(orgNames)
			log.Logger().Infof("found %d organisations in git server %s: %s\n", len(orgs), info(server.URL), info(strings.Join(orgNames, ", ")))
			if config.PipeLineServer == server.URL && config.PipeLineUsername == userAuth.Username {
				pipeUserValid = true
			}
		}
	}

	if pipeUserValid {
		log.Logger().Infof("Validated pipeline user %s on git server %s", util.ColorInfo(config.PipeLineUsername), util.ColorInfo(config.PipeLineServer))
	} else {
		return errors.Errorf("pipeline user %s on git server %s not valid", util.ColorError(config.PipeLineUsername), util.ColorError(config.PipeLineServer))
	}

	log.Logger().Infof("git tokens seem to be setup correctly\n")
	return nil
}
