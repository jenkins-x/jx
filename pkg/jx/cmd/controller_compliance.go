package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/governance"

	"github.com/jenkins-x/jx/pkg/log"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jenkins-x/jx/pkg/kube"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// ControllerComplianceOptions the options for the create spring command
type ControllerComplianceOptions struct {
	ControllerOptions
	IsPre bool
}

// NewCmdControllerCompliance creates a command object for the "create" command
func NewCmdControllerCompliance(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &ControllerComplianceOptions{
		ControllerOptions: ControllerOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "compliance",
		Short: "Updates a pull request with compliance status",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&options.IsPre, "pre", "", false, "Run the pre-submit phase which creates a pending status")

	return cmd
}

// Run implements this command
func (o *ControllerComplianceOptions) Run() error {
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
	watch, err := jxClient.JenkinsV1().ComplianceChecks(ns).Watch(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for event := range watch.ResultChan() {
		check, ok := event.Object.(*jenkinsv1.ComplianceCheck)
		if !ok {
			log.Fatalf("unexpected type %s\n", event)
		}
		err = o.Check(check.Spec)
		if err != nil {
			gitProvider, gitRepoInfo, err1 := o.createGitProviderForURLWithoutKind(check.Spec.Commit.GitURL)
			if err1 != nil {
				return err1
			}
			_, err1 = governance.NotifyComplianceState(check.Spec.Commit, "error", "", "Internal Error performing compliance checks", "", gitProvider, gitRepoInfo)
			if err1 != nil {
				log.Fatalf("Error updating git commit state on error %s\n", err1)
			}
			log.Fatalf("Error updating git commit state %s\n", err1)
		}
	}

	if err != nil {
		return err
	}
	return nil
}

func (o *ControllerComplianceOptions) Check(check jenkinsv1.ComplianceCheckSpec) error {
	gitProvider, gitRepoInfo, err := o.createGitProviderForURLWithoutKind(check.Commit.GitURL)
	if err != nil {
		return err
	}
	pass := false
	if check.Checked {
		var commentBuilder strings.Builder
		pass = true
		for _, c := range check.Checks {
			if !c.Pass {
				pass = false
				fmt.Fprintf(&commentBuilder, "%s | %s | %s | TODO | `/test this`\n", c.Name, c.Description, check.Commit.SHA)
			}
		}
		if pass {
			_, err := governance.NotifyComplianceState(check.Commit, "success", "", "Compliance checks completed successfully", "", gitProvider, gitRepoInfo)
			if err != nil {
				return err
			}
		} else {

			comment := fmt.Sprintf(
				"The following compliance checks **failed**, say `/retest` to rerun them all:\n"+
					"\n"+
					"Name | Description | Commit | Details | Rerun command\n"+
					"--- | --- | --- | --- | --- \n"+
					"%s\n"+
					"<details>\n"+
					"\n"+
					"Instructions for interacting with me using PR comments are available [here](https://git.k8s.io/community/contributors/guide/pull-requests.md).  If you have questions or suggestions related to my behavior, please file an issue against the [kubernetes/test-infra](https://github.com/kubernetes/test-infra/issues/new?title=Prow%%20issue:) repository. I understand the commands that are listed [here](https://go.k8s.io/bot-commands).\n"+
					"</details>", commentBuilder.String())
			_, err := governance.NotifyComplianceState(check.Commit, "failure", "", "Some compliance checks failed", comment, gitProvider, gitRepoInfo)
			if err != nil {
				return err
			}
		}
	} else {
		_, err := governance.NotifyComplianceState(check.Commit, "pending", "", "Waiting for compliance checks to complete", "", gitProvider, gitRepoInfo)
		if err != nil {
			return err
		}
	}
	return nil
}
