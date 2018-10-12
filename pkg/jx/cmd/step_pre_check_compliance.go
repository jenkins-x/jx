package cmd

import (
	"fmt"
	"io"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/kube"

	"github.com/jenkins-x/jx/pkg/log"

	jenkinsv1client "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// StepPreBuildOptions contains the command line flags
type StepPreCheckComplianceOptions struct {
	StepOptions
}

var (
	StepPreCheckComplianceLong = templates.LongDesc(`
		This pipeline step adds a pending state on compliance checks
`)

	StepPreCheckComplianceExample = templates.Examples(`
		jx step pre check compliance
`)
)

func NewCmdStepPreCheckCompliance(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := StepPreCheckComplianceOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "compliance",
		Short:   "Adds a pending state",
		Long:    StepPreCheckComplianceLong,
		Example: StepPreCheckComplianceExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	return cmd
}

func (o *StepPreCheckComplianceOptions) Run() error {

	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	apisClient, err := o.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterComplianceCheckCRD(apisClient)
	if err != nil {
		return err
	}
	err = kube.RegisterPipelineActivityCRD(apisClient)
	if err != nil {
		return err
	}

	err = o.Init(jxClient, ns)
	return nil
}

func (o *StepPreCheckComplianceOptions) Init(jxClient jenkinsv1client.Interface, ns string) error {

	org := os.Getenv("REPO_OWNER")
	repo := os.Getenv("REPO_NAME")
	build := os.Getenv("BUILD_ID")
	pullRequest := os.Getenv("PULL_REQUEST")
	sha := os.Getenv("PULL_PULL_SHA")
	// TODO Fix this once PROW supports other providers
	url := fmt.Sprintf("https://github.com/%s/%s.git", org, repo)
	pipeline := fmt.Sprintf("%s-%s", org, repo)
	if pipeline != "" && build != "" {
		name := kube.ToValidName(pipeline + "-" + build)
		log.Infof("Creating compliance check for %s\n", name)
		_, err := jxClient.JenkinsV1().ComplianceChecks(ns).Create(&jenkinsv1.ComplianceCheck{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: jenkinsv1.ComplianceCheckSpec{
				Checked: false,
				Commit: jenkinsv1.ComplianceCheckCommitReference{
					GitURL:      url,
					PullRequest: pullRequest,
					SHA:         sha,
				},
			},
		})
		if err != nil {
			return err
		}
	} else {
		log.Errorf("Cannot determine pipeline (found %s) and build number (found %s) for compliance check\n", pipeline, build)
	}
	return nil
}
