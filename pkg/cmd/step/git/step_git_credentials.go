package git

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/cmd/opts/step"

	"github.com/jenkins-x/jx/pkg/cmd/helper"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

const (
	optionOutputFile = "output"
)

type credentials struct {
	user       string
	password   string
	serviceURL string
}

// StepGitCredentialsOptions contains the command line flags
type StepGitCredentialsOptions struct {
	step.StepOptions

	OutputFile string
	AskPass    bool
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

		# respond to a GIT_ASKPASS request
		jx step git credentials --ask-pass
`)
)

func NewCmdStepGitCredentials(commonOpts *opts.CommonOptions) *cobra.Command {
	options := StepGitCredentialsOptions{
		StepOptions: step.StepOptions{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "credentials",
		Short:   "Creates the Git credentials file for the current pipeline",
		Long:    StepGitCredentialsLong,
		Example: StepGitCredentialsExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.OutputFile, optionOutputFile, "o", "", "The output file name")
	cmd.Flags().BoolVar(&options.AskPass, "ask-pass", false, "respond to a GIT_ASKPASS request")
	return cmd
}

func (o *StepGitCredentialsOptions) Run() error {
	secrets, err := o.LoadPipelineSecrets(kube.ValueKindGit, "")
	if err != nil {
		return err
	}

	if o.AskPass {
		// TODO issue-5772 handle input from GIT_ASKPASS poperly by parsing the git provider
		credentialList := o.readCredentials(secrets)
		fmt.Println(credentialList[0].password)
		return nil
	} else {
		outFile, err := o.determineOutputFile()
		if err != nil {
			return err
		}

		return o.createGitCredentialsFile(outFile, secrets)
	}
}

func (o *StepGitCredentialsOptions) determineOutputFile() (string, error) {
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
		return "", util.MissingOption(optionOutputFile)
	}
	dir, _ := filepath.Split(outFile)
	if dir != "" {
		err := os.MkdirAll(dir, util.DefaultWritePermissions)
		if err != nil {
			return "", err
		}
	}
	return outFile, nil
}

func (o *StepGitCredentialsOptions) createGitCredentialsFile(fileName string, secrets *corev1.SecretList) error {
	data := o.CreateGitCredentialsFromSecrets(secrets)
	err := ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err != nil {
		return fmt.Errorf("failed to write to %s: %s", fileName, err)
	}
	log.Logger().Infof("Generated Git credentials file %s", util.ColorInfo(fileName))
	return nil
}

// CreateGitCredentialsFromSecrets Creates git credentials from secrets
func (o *StepGitCredentialsOptions) CreateGitCredentialsFromSecrets(secretList *corev1.SecretList) []byte {
	credentialList := o.readCredentials(secretList)

	var buffer bytes.Buffer
	for _, credential := range credentialList {
		u2, err := url.Parse(credential.serviceURL)
		if err != nil {
			log.Logger().Warnf("Ignoring pipeline credential for due to invalid Git service URL %s for user %s", credential.user, credential.serviceURL)
		} else {
			u2.User = url.UserPassword(credential.user, credential.password)
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

	return buffer.Bytes()
}

func (o *StepGitCredentialsOptions) readCredentials(secretList *corev1.SecretList) []credentials {
	var credentialList []credentials
	if secretList != nil {
		for _, secret := range secretList.Items {
			labels := secret.Labels
			annotations := secret.Annotations
			data := secret.Data
			if labels != nil && labels[kube.LabelKind] == kube.ValueKindGit && annotations != nil {
				url := annotations[kube.AnnotationURL]
				if url != "" && data != nil {
					username := data[kube.SecretDataUsername]
					pwd := data[kube.SecretDataPassword]
					if len(username) > 0 && len(pwd) > 0 {
						creds := credentials{
							user:       string(username),
							password:   string(pwd),
							serviceURL: url,
						}
						credentialList = append(credentialList, creds)
					}
				}
			}
		}
	}
	return credentialList
}
