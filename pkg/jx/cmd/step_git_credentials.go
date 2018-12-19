package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

const (
	optionOutputFile = "output"
)

// StepGitCredentialsOptions contains the command line flags
type StepGitCredentialsOptions struct {
	StepOptions

	OutputFile string
}

var (
	StepGitCredentialsLong = templates.LongDesc(`
		This pipeline step generates a Git credentials file for the current Git provider pipeline Secrets

`)

	StepGitCredentialsExample = templates.Examples(`
		# generate the Git credentials file in the canonical location
		jx step git credentials

		# generate the Git credentials to a output file
		jx step git credentials -o /tmp/mycreds

`)
)

// NewCmdStepGitCredentials creates a new 'step git credentials' Command.
func NewCmdStepGitCredentials(commonOpts *CommonOptions) *cobra.Command {
	options := StepGitCredentialsOptions{
		StepOptions: StepOptions{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "credentials",
		Short:   "Creates the Git credentials file for the current pipeline Git credentials",
		Aliases: []string{"nexus_stage"},
		Long:    StepGitCredentialsLong,
		Example: StepGitCredentialsExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.OutputFile, optionOutputFile, "o", "", "The output file name")
	return cmd
}

func (o *StepGitCredentialsOptions) Run() error {
	outFile := o.OutputFile
	if outFile == "" {
		// lets figure out the default output file
		cfgHome := os.Getenv("XDG_CONFIG_HOME")
		if cfgHome == "" {
			cfgHome = util.HomeDir()
		}
		if cfgHome != "" {
			outFile = filepath.Join(cfgHome, "git", "credentials")
		}
	}
	if outFile == "" {
		return util.MissingOption(optionOutputFile)
	}
	dir, _ := filepath.Split(outFile)
	if dir != "" {
		err := os.MkdirAll(dir, DefaultWritePermissions)
		if err != nil {
			return err
		}
	}
	secrets, err := o.LoadPipelineSecrets(kube.ValueKindGit, "")
	if err != nil {
		return err
	}
	return o.createGitCredentialsFile(outFile, secrets)
}

func (o *StepGitCredentialsOptions) createGitCredentialsFile(fileName string, secrets *corev1.SecretList) error {
	data := o.CreateGitCredentialsFromSecrets(secrets)
	err := ioutil.WriteFile(fileName, data, DefaultWritePermissions)
	if err != nil {
		return fmt.Errorf("Failed to write to %s: %s", fileName, err)
	}
	log.Infof("Generated Git credentials file %s\n", util.ColorInfo(fileName))
	return nil
}

// CreateGitCredentialsFromSecrets Creates git credentials from secrets
func (o *StepGitCredentialsOptions) CreateGitCredentialsFromSecrets(secretList *corev1.SecretList) []byte {
	var buffer bytes.Buffer
	if secretList != nil {
		for _, secret := range secretList.Items {
			labels := secret.Labels
			annotations := secret.Annotations
			data := secret.Data
			if labels != nil && labels[kube.LabelKind] == kube.ValueKindGit && annotations != nil {
				u := annotations[kube.AnnotationURL]
				if u != "" && data != nil {
					username := data[kube.SecretDataUsername]
					pwd := data[kube.SecretDataPassword]
					if len(username) > 0 && len(pwd) > 0 {
						u2, err := url.Parse(u)
						if err != nil {
							log.Warnf("Ignoring invalid Git service URL %s for pipeline credential %s\n", u, secret.Name)
						} else {
							u2.User = url.UserPassword(string(username), string(pwd))
							buffer.WriteString(u2.String() + "\n")

							// lets write the other http protocol for completeness
							if u2.Scheme == "https" {
								u2.Scheme = "http"
							} else {
								u2.Scheme = "https"
							}
							buffer.WriteString(u2.String() + "\n")
						}
					}
				}
			}
		}
	}
	return buffer.Bytes()
}
