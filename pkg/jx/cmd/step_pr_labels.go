package cmd

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
)

// DefaultPrefix for all PR labels environment keys
const DefaultPrefix = "JX_PR_LABELS"

// StepPRLabelsOptions holds the options for the cmd
type StepPRLabelsOptions struct {
	CommonOptions

	Dir         string
	Prefix      string
	PullRequest string
}

var (
	labelLong = templates.LongDesc(`
		Creates environment variables from the labels in a pull request.

		Environment variables are prefixed per default with ` + DefaultPrefix + `.
        You can use the '--prefix' argument to set a different prefix.
    `)

	labelExample = templates.Examples(`
		# List all labels of a given pull-request
		jx step pr labels

		# List all labels of a given pull-request using a custom prefix
		jx step pr --prefix PRL

		# List all labels of a given pull-request using a custom pull-request number
		jx step pr --pr PR-34
		jx step pr --pr 34

    `)
)

// NewCmdStepPRLabels creates the new cmd
func NewCmdStepPRLabels(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepPRLabelsOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,

			Out: out,
			Err: errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "labels",
		Short:   "List all labels of a given pull-request",
		Long:    labelLong,
		Example: labelExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)
	cmd.Flags().StringVarP(&options.PullRequest, "pr", "", "", "Git Pull Request number")
	cmd.Flags().StringVarP(&options.Prefix, "prefix", "p", "", "Environment variable prefix")
	return cmd
}

// Run implements the execution
func (o *StepPRLabelsOptions) Run() error {
	gitInfo, provider, _, err := o.createGitProvider(o.Dir)
	if err != nil {
		return err
	}
	if provider == nil {
		return fmt.Errorf("No Git provider could be found. Are you in a directory containing a `.git/config` file?")
	}

	if o.PullRequest == "" {
		o.PullRequest = os.Getenv("BRANCH_NAME")
	}

	if o.Prefix == "" {
		o.Prefix = DefaultPrefix
	}

	prNum, err := strconv.Atoi(o.PullRequest)
	if err != nil {
		log.Warn("Unable to convert PR " + o.PullRequest + " to a number" + "\n")
	}

	pr, err := provider.GetPullRequest(gitInfo.Organisation, gitInfo, prNum)
	if err != nil {
		return errors.Wrapf(err, "failed to find PullRequest %d", prNum)
	}

	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		return errors.Wrapf(err, "failed to create regex %v", reg)
	}

	for _, v := range pr.Labels {
		envKey := reg.ReplaceAllString(*v.Name, "_")
		log.Infof("%v_%v=%v\n", o.Prefix, strings.ToUpper(envKey), *v.Name)
	}
	return nil
}
