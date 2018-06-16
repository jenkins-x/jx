package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"fmt"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
)

const (
	optionTitle = "title"
)

var (
	create_issue_long = templates.LongDesc(`
		Creates an issue in a the git project of the current directory
`)

	create_issue_example = templates.Examples(`
		# Create an issue in the current project
		jx create issue -t "something we should do"


		# Create an issue with a title and a body
		jx create issue -t "something we should do" --body "	
		some more
		text
		goes
		here
		""
"
	`)
)

// CreateIssueOptions the options for the create spring command
type CreateIssueOptions struct {
	CreateOptions

	Dir    string
	Title  string
	Body   string
	Labels []string
}

// NewCmdCreateIssue creates a command object for the "create" command
func NewCmdCreateIssue(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateIssueOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "issue",
		Short:   "Create an issue on the git project for the current directory",
		Aliases: []string{"env"},
		Long:    create_issue_long,
		Example: create_issue_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Dir, "dir", "", "", "The source directory used to detect the git repository. Defaults to the current directory")
	cmd.Flags().StringVarP(&options.Title, optionTitle, "t", "", "The title of the issue to create")
	cmd.Flags().StringVarP(&options.Body, "body", "", "", "The body of the issue")
	cmd.Flags().StringArrayVarP(&options.Labels, "label", "l", []string{}, "The labels to add to the issue")

	options.addCommonFlags(cmd)
	return cmd
}

// Run implements the command
func (o *CreateIssueOptions) Run() error {
	tracker, err := o.createIssueProvider(o.Dir)
	if err != nil {
		return err
	}
	issue := &gits.GitIssue{}

	err = o.PopulateIssue(issue)
	if err != nil {
		return err
	}

	createdIssue, err := tracker.CreateIssue(issue)
	if err != nil {
		return err
	}
	if createdIssue == nil {
		return fmt.Errorf("Failed to create issue: %s", issue.Title)
	}
	o.Printf("\nCreated issue %s at %s\n", util.ColorInfo(createdIssue.Name()), util.ColorInfo(createdIssue.URL))
	return nil
}

func (o *CreateIssueOptions) FindGitInfo(dir string) (*gits.GitRepositoryInfo, error) {
	_, gitConf, err := gits.FindGitConfigDir(dir)
	if err != nil {
		return nil, fmt.Errorf("Could not find a .git directory: %s\n", err)
	} else {
		if gitConf == "" {
			return nil, fmt.Errorf("No git conf dir found")
		}
		gitURL, err := gits.DiscoverUpstreamGitURL(gitConf)
		if err != nil {
			return nil, fmt.Errorf("Could not find the remote git source URL:  %s", err)
		}
		return gits.ParseGitURL(gitURL)
	}
}

func (o *CreateIssueOptions) PopulateIssue(issue *gits.GitIssue) error {
	title := o.Title
	body := o.Body
	var err error
	if title == "" {
		if o.BatchMode {
			return util.MissingOption(optionTitle)
		}
		title, err = util.PickValue("Issue title:", "", true)
		if err != nil {
			return err
		}
	}
	issue.Title = title
	issue.Body = body

	labels := []gits.GitLabel{}
	for _, label := range o.Labels {
		labels = append(labels, gits.GitLabel{
			Name: label,
		})
	}
	issue.Labels = labels

	if title == "" {
		return fmt.Errorf("No title specified!")
	}
	return nil
}
