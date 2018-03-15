package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/quickstarts"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
)

const (
	JenkinsXQuickstartsOrganisation = "jenkins-x-quickstarts"
)

var (
	create_quickstart_long = templates.LongDesc(`
		Creates a new Maven project using an Archetype

		You then get the option to import the generated source code into a git repository and Jenkins for CI / CD

`)

	create_quickstart_example = templates.Examples(`
		# Create a new application from a Maven Archetype using the UI to choose which archetype to use
		jx create archetype

		# Creates a Camel Archetype, filtering on the archetypes containing the text 'spring'
		jx create archetype -g  org.apache.camel.archetypes -a spring
	`)
)

// CreateQuickstartOptions the options for the create spring command
type CreateQuickstartOptions struct {
	CreateProjectOptions

	GitHubOrganisations []string
	Filter              quickstarts.QuickstartFilter
}

// NewCmdCreateQuickstart creates a command object for the "create" command
func NewCmdCreateQuickstart(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateQuickstartOptions{
		CreateProjectOptions: CreateProjectOptions{
			ImportOptions: ImportOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "quickstart",
		Short:   "Create a new app from a Quickstart and import the generated code into git and Jenkins for CI / CD",
		Long:    create_quickstart_long,
		Example: create_quickstart_example,
		Aliases: []string{"arch"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	options.addCreateAppFlags(cmd)

	cmd.Flags().StringArrayVarP(&options.GitHubOrganisations, "organisations", "g", []string{}, "The github organisations to query for quickstarts")
	cmd.Flags().StringArrayVarP(&options.Filter.Tags, "tag", "t", []string{}, "The tags on the quickstarts to filter")
	cmd.Flags().StringVarP(&options.Filter.Owner, "owner", "", "", "The owner to filter on")
	cmd.Flags().StringVarP(&options.Filter.Language, "language", "l", "", "The language to filter on")
	cmd.Flags().StringVarP(&options.Filter.Framework, "framework", "", "", "The framework to filter on")

	cmd.Flags().StringVarP(&options.Filter.Text, "filter", "f", "", "The text filter")
	return cmd
}

// Run implements the generic Create command
func (o *CreateQuickstartOptions) Run() error {
	model, err := o.LoadQuickstarts()
	if err != nil {
		return fmt.Errorf("Failed to load quickstarts: %s", err)
	}
	q, err := model.CreateSurvey(&o.Filter)
	if err != nil {
		return err
	}
	if q == nil {
		return fmt.Errorf("No quickstart chosen")
	}

	dir := o.OutDir
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	genDir, err := o.createQuickstart(q, dir)
	if err != nil {
		return err
	}

	o.Printf("Created project at %s\n\n", util.ColorInfo(genDir))

	return o.ImportCreatedProject(genDir)
}

func (o *CreateQuickstartOptions) createQuickstart(q *quickstarts.Quickstart, dir string) (string, error) {
	answer := filepath.Join(dir, q.Name)
	err := os.Mkdir(answer, DefaultWritePermissions)
	if err != nil {
		return answer, fmt.Errorf("Failed to create %s: %s", answer, err)
	}
	u := q.DownloadZipURL
	if u == "" {
		return answer, fmt.Errorf("Quickstart %s does not have a download zip URL", q.ID)
	}
	client := http.Client{}

	//fmt.Printf("generating spring project from: %s\n", u)
	req, err := http.NewRequest(http.MethodGet, u, strings.NewReader(""))
	if err != nil {
		return answer, err
	}
	res, err := client.Do(req)
	if err != nil {
		return answer, err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return answer, err
	}

	zipFile := filepath.Join(dir, "source.zip")
	err = ioutil.WriteFile(zipFile, body, util.DefaultWritePermissions)
	if err != nil {
		return answer, fmt.Errorf("Failed to download file %s due to %s", zipFile, err)
	}
	err = util.Unzip(zipFile, dir)
	if err != nil {
		return answer, fmt.Errorf("Failed to unzip new project file %s due to %s", zipFile, err)
	}
	err = os.Remove(zipFile)
	if err != nil {
		return answer, err
	}
	o.Printf("Generated quickstart at %s\n", answer)
	return answer, nil
}

func (o *CreateQuickstartOptions) LoadQuickstarts() (*quickstarts.QuickstartModel, error) {
	model := quickstarts.NewQuickstartModel()

	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return model, err
	}
	config := authConfigSvc.Config()
	server := config.GetOrCreateServer(gits.GitHubHost)

	userAuth, err := config.PickServerUserAuth(server, "git user name", o.BatchMode)
	if err != nil {
		return model, err
	}
	provider, err := gits.CreateProvider(server, userAuth)
	if err != nil {
		return model, err
	}
	groups := o.GitHubOrganisations
	if util.StringArrayIndex(groups, JenkinsXQuickstartsOrganisation) < 0 {
		groups = append(groups, JenkinsXQuickstartsOrganisation)
	}

	model.LoadGithubQuickstarts(provider, groups)
	return model, nil
}
