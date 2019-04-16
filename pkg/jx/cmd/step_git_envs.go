package cmd

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

// StepGitEnvsOptions contains the command line flags
type StepGitEnvsOptions struct {
	StepOptions

	ServiceKind string
}

var (
	// StepGitEnvsLong command long description
	StepGitEnvsLong = templates.LongDesc(`
		This pipeline step generates a Git environment variables from the current Git provider pipeline Secrets

`)
	// StepGitEnvsExample command example
	StepGitEnvsExample = templates.Examples(`
		# Sets the Git environment variables for the current GitHub provider
		jx step git envs

		# Sets the Gie environment variables for the current Gtilab provider
		jx step git envs --service-kind=gitlab
`)
)

// NewCmdStepGitEnvs create the 'step git envs' command
func NewCmdStepGitEnvs(commonOpts *opts.CommonOptions) *cobra.Command {
	options := StepGitEnvsOptions{
		StepOptions: StepOptions{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "envs",
		Short:   "Creates the Git environment variables for the current pipeline Git credentials",
		Long:    StepGitEnvsLong,
		Example: StepGitEnvsExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.ServiceKind, "service-kind", "", "github", "The kind of git service")
	return cmd
}

// Run implements the command
func (o *StepGitEnvsOptions) Run() error {
	secrets, err := o.LoadPipelineSecrets(kube.ValueKindGit, "")
	if err != nil {
		return errors.Wrap(err, "loading the pipeline secrets")
	}

	username, token, err := o.getGitCredentials(secrets, o.ServiceKind)
	if err != nil {
		return errors.Wrap(err, "retrieving the environment variables values")
	}

	_, _ = fmt.Fprintf(o.Out, "export GIT_USERNAME=%s\nexport GIT_API_TOKEN=%s\n", username, token)

	return nil
}

func (o *StepGitEnvsOptions) getGitCredentials(secrets *corev1.SecretList, serviceKind string) (string, string, error) {
	if secrets == nil {
		return "", "", errors.New("no git credentials found")
	}
	for _, secret := range secrets.Items {
		labels := secret.Labels
		data := secret.Data
		if data == nil {
			continue
		}
		if labels != nil && labels[kube.LabelKind] == kube.ValueKindGit {
			foundServiceKind, ok := labels[kube.LabelServiceKind]
			if !ok {
				continue
			}
			if strings.EqualFold(serviceKind, foundServiceKind) {
				username := string(data[kube.SecretDataUsername])
				pwd := string(data[kube.SecretDataPassword])
				return username, pwd, nil
			}
		}
	}

	return "", "", errors.New("no git credentials found")
}
