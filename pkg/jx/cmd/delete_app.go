package cmd

import (
	"fmt"
	"io"

	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/log"

	"github.com/pkg/errors"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/util"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

var (
	deleteAppLong = templates.LongDesc(`
		Deletes one or more Apps

`)

	deleteAppExample = templates.Examples(`
		# prompt for the available apps to delete
		jx delete apps 

		# delete a specific app 
		jx delete app jx-app-cheese
	`)
)

const (
	optionPurge = "purge"
)

// DeleteAppOptions are the flags for this delete commands
type DeleteAppOptions struct {
	commoncmd.CommonOptions

	GitOps bool
	DevEnv *jenkinsv1.Environment

	// for testing
	FakePullRequests commoncmd.CreateEnvPullRequestFn

	ReleaseName string
	Namespace   string
	Purge       bool
	Alias       string

	// allow git to be configured externally before a PR is created
	ConfigureGitCallback commoncmd.ConfigureGitFolderFn
}

// NewCmdDeleteApp creates a command object for this command
func NewCmdDeleteApp(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	o := &DeleteAppOptions{
		CommonOptions: commoncmd.CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "application",
		Short:   "Deletes one or more applications from Jenkins",
		Long:    deleteAppLong,
		Example: deleteAppExample,
		Aliases: []string{"applications"}, // FIXME - naming conflict with 'app'
		Run: func(cmd *cobra.Command, args []string) {
			o.Cmd = cmd
			o.Args = args
			err := o.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&o.ReleaseName, optionRelease, "r", "",
		"The chart release name (available when NOT using GitOps for your dev environment)")
	cmd.Flags().BoolVarP(&o.Purge, optionPurge, "", true,
		"Should we run helm update first to ensure we use the latest version (available when NOT using GitOps for your dev environment)")
	cmd.Flags().StringVarP(&o.Namespace, optionNamespace, "n", defaultNamespace, "The Namespace to install into (available when NOT using GitOps for your dev environment)")
	cmd.Flags().StringVarP(&o.Alias, optionAlias, "", "",
		"An alias to use for the app (available when using GitOps for your dev environment)")

	return cmd
}

// Run implements this command
func (o *DeleteAppOptions) Run() error {
	o.GitOps, o.DevEnv = o.GetDevEnv()

	if o.GitOps {
		msg := "Unable to specify --%s when using GitOps for your dev environment"
		if o.ReleaseName != "" {
			return util.InvalidOptionf(optionRelease, o.ReleaseName, msg, optionRelease)
		}
		if o.Namespace != "" {
			return util.InvalidOptionf(optionNamespace, o.Namespace, msg, optionNamespace)
		}
	}
	if !o.GitOps {
		if o.Alias != "" {
			return util.InvalidOptionf(optionAlias, o.Alias,
				"Unable to specify --%s when NOT using GitOps for your dev environment", optionAlias)
		}
	}

	args := o.Args
	if len(args) == 0 {
		return o.Cmd.Help()
	}
	if len(args) > 1 {
		return o.Cmd.Help()
	}

	for _, app := range args {
		if o.GitOps {
			err := o.createPR(app)
			if err != nil {
				return err
			}
		} else {
			err := o.deleteApp(app)
			if err != nil {
				return err
			}
		}
		return nil
	}

	return nil
}

func (o *DeleteAppOptions) createPR(app string) error {

	modifyChartFn := func(requirements *helm.Requirements, metadata *chart.Metadata, values map[string]interface{},
		templates map[string]map[string]interface{}) error {
		// See if the app already exists in requirements
		found := false
		for i, d := range requirements.Dependencies {
			if d.Name == app && d.Alias == o.Alias {
				found = true
				requirements.Dependencies[i] = nil
			}
		}
		// If app not found, add it
		if !found {
			return fmt.Errorf("unable to delete %s as not installed", app)
		}
		return nil
	}
	branchNameText := "delete-app-" + app
	title := fmt.Sprintf("Delete %s", app)
	message := fmt.Sprintf("Delete app %s", app)
	var pullRequestInfo *gits.PullRequestInfo
	if o.FakePullRequests != nil {
		var err error
		pullRequestInfo, err = o.FakePullRequests(o.DevEnv, modifyChartFn, branchNameText, title, message,
			nil)
		if err != nil {
			return err
		}
	} else {
		var err error
		pullRequestInfo, err = o.CreateEnvironmentPullRequest(o.DevEnv, modifyChartFn, &branchNameText, &title,
			&message,
			nil, o.ConfigureGitCallback)
		if err != nil {
			return err
		}
	}
	log.Infof("Delete app via Pull Request %s\n", pullRequestInfo.PullRequest.URL)
	return nil
}

func (o *DeleteAppOptions) deleteApp(name string) error {
	err := o.EnsureHelm()
	if err != nil {
		return errors.Wrap(err, "failed to ensure that helm is present")
	}
	releaseName := name
	if o.ReleaseName != "" {
		releaseName = o.ReleaseName
	}
	err = o.DeleteChart(releaseName, o.Purge)
	if err != nil {
	}
	return err
}
