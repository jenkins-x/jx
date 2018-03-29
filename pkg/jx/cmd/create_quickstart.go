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

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
)

const (
	JenkinsXQuickstartsOrganisation = "jenkins-x-quickstarts"
)

var (
	createQuickstartLong = templates.LongDesc(`
		Create a new project from a sample/starter (found in https://github.com/jenkins-x-quickstarts)

		This will create a new project for you from the selected template.

		For more documentation see: [http://jenkins-x.io/developing/create-quickstart/](http://jenkins-x.io/developing/create-quickstart/)

`)

	createQuickstartExample = templates.Examples(`
		Create a new project from a sample/starter (found in https://github.com/jenkins-x-quickstarts)

		This will create a new project for you from the selected template.

		jx create quickstart

		jx create quickstart -f http
	`)
)

// CreateQuickstartOptions the options for the create spring command
type CreateQuickstartOptions struct {
	CreateProjectOptions

	GitHubOrganisations []string
	Filter              quickstarts.QuickstartFilter
	GitProvider         gits.GitProvider
	GitHost             string
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
		Long:    createQuickstartLong,
		Example: createQuickstartExample,
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
	cmd.Flags().StringVarP(&options.GitHost, "git-host", "", "", "The Git server host if not using GitHub when pushing created project")
	cmd.Flags().StringVarP(&options.Filter.Text, "filter", "f", "", "The text filter")
	return cmd
}

// Run implements the generic Create command
func (o *CreateQuickstartOptions) Run() error {
	installOpts := InstallOptions{
		CommonOptions: CommonOptions{
			Factory: o.Factory,
			Out:     o.Out,
		},
	}
	userAuth, err := installOpts.getGitUser("git username to create the quickstart")
	if err != nil {
		return err
	}

	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	var server *auth.AuthServer
	config := authConfigSvc.Config()
	if o.GitHub {
		server = config.GetOrCreateServer(gits.GitHubURL)
	} else {
		if o.GitHost != "" {
			server = config.GetOrCreateServer(o.GitHost)
		} else {
			server, err = config.PickServer("Pick the git server to search for repositories")
			if err != nil {
				return err
			}
		}
	}
	if server == nil {
		return fmt.Errorf("no git server provided")
	}

	o.GitProvider, err = gits.CreateProvider(server, userAuth)

	if err != nil {
		return err
	}

	model, err := o.LoadQuickstarts()
	if err != nil {
		return fmt.Errorf("failed to load quickstarts: %s", err)
	}
	q, err := model.CreateSurvey(&o.Filter)
	if err != nil {
		return err
	}
	if q == nil {
		return fmt.Errorf("no quickstart chosen")
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

	o.CreateProjectOptions.ImportOptions.GitProvider = o.GitProvider
	o.Organisation = userAuth.Username
	return o.ImportCreatedProject(genDir)
}

func (o *CreateQuickstartOptions) createQuickstart(f *quickstarts.QuickstartForm, dir string) (string, error) {
	q := f.Quickstart
	answer := filepath.Join(dir, f.Name)
	u := q.DownloadZipURL
	if u == "" {
		return answer, fmt.Errorf("quickstart %s does not have a download zip URL", q.ID)
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
		return answer, fmt.Errorf("failed to download file %s due to %s", zipFile, err)
	}
	tmpDir, err := ioutil.TempDir("", "jx-source-")
	if err != nil {
		return answer, fmt.Errorf("failed to create temporary directory: %s", err)
	}
	err = util.Unzip(zipFile, tmpDir)
	if err != nil {
		return answer, fmt.Errorf("failed to unzip new project file %s due to %s", zipFile, err)
	}
	err = os.Remove(zipFile)
	if err != nil {
		return answer, err
	}
	tmpDir, err = findFirstDirectory(tmpDir)
	if err != nil {
		return answer, fmt.Errorf("failed to find a directory inside the source download: %s", err)
	}
	err = util.RenameDir(tmpDir, answer, false)
	if err != nil {
		return answer, fmt.Errorf("failed to rename temp dir %s to %s: %s", tmpDir, answer, err)
	}
	o.Printf("Generated quickstart at %s\n", answer)
	return answer, nil
}

func findFirstDirectory(dir string) (string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return dir, err
	}
	for _, f := range files {
		if f.IsDir() {
			return filepath.Join(dir, f.Name()), nil
		}
	}
	return "", fmt.Errorf("no child directory found in %s", dir)
}

func (o *CreateQuickstartOptions) LoadQuickstarts() (*quickstarts.QuickstartModel, error) {
	model := quickstarts.NewQuickstartModel()

	groups := o.GitHubOrganisations
	if util.StringArrayIndex(groups, JenkinsXQuickstartsOrganisation) < 0 {
		groups = append(groups, JenkinsXQuickstartsOrganisation)
	}

	model.LoadGithubQuickstarts(o.GitProvider, groups)
	return model, nil
}
