package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/quickstarts"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
)

const (
	// JenkinsXQuickstartsOrganisation is the default organisation for quickstarts
	JenkinsXQuickstartsOrganisation = "jenkins-x-quickstarts"
)

var (
	createQuickstartLong = templates.LongDesc(`
		Create a new project from a sample/starter (found in https://github.com/jenkins-x-quickstarts)

		This will create a new project for you from the selected template.
		It will exclude any work-in-progress repos (containing the "WIP-" pattern)

		For more documentation see: [https://jenkins-x.io/developing/create-quickstart/](https://jenkins-x.io/developing/create-quickstart/)

` + SeeAlsoText("jx create project"))

	createQuickstartExample = templates.Examples(`
		Create a new project from a sample/starter (found in https://github.com/jenkins-x-quickstarts)

		This will create a new project for you from the selected template.
		It will exclude any work-in-progress repos (containing the "WIP-" pattern)

		jx create quickstart

		jx create quickstart -f http
	`)
)

// CreateQuickstartOptions the options for the create quickstart command
type CreateQuickstartOptions struct {
	CreateProjectOptions

	GitHubOrganisations []string
	Filter              quickstarts.QuickstartFilter
	GitProvider         gits.GitProvider
	GitHost             string
	IgnoreTeam          bool
}

// NewCmdCreateQuickstart creates a command object for the "create" command
func NewCmdCreateQuickstart(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateQuickstartOptions{
		CreateProjectOptions: CreateProjectOptions{
			ImportOptions: ImportOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					In:      in,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "quickstart",
		Short:   "Create a new app from a Quickstart and import the generated code into Git and Jenkins for CI/CD",
		Long:    createQuickstartLong,
		Example: createQuickstartExample,
		Aliases: []string{"arch"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addCreateAppFlags(cmd)

	cmd.Flags().StringArrayVarP(&options.GitHubOrganisations, "organisations", "g", []string{}, "The GitHub organisations to query for quickstarts")
	cmd.Flags().StringArrayVarP(&options.Filter.Tags, "tag", "t", []string{}, "The tags on the quickstarts to filter")
	cmd.Flags().StringVarP(&options.Filter.Owner, "owner", "", "", "The owner to filter on")
	cmd.Flags().StringVarP(&options.Filter.Language, "language", "l", "", "The language to filter on")
	cmd.Flags().StringVarP(&options.Filter.Framework, "framework", "", "", "The framework to filter on")
	cmd.Flags().StringVarP(&options.GitHost, "git-host", "", "", "The Git server host if not using GitHub when pushing created project")
	cmd.Flags().StringVarP(&options.Filter.Text, "filter", "f", "", "The text filter")
	cmd.Flags().StringVarP(&options.Filter.ProjectName, "project-name", "p", "", "The project name (for use with -b batch mode)")
	return cmd
}

// Run implements the generic Create command
func (o *CreateQuickstartOptions) Run() error {
	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()

	var locations []v1.QuickStartLocation
	if !o.IgnoreTeam {
		jxClient, ns, err := o.JXClientAndDevNamespace()
		if err != nil {
			return err
		}

		locations, err = kube.GetQuickstartLocations(jxClient, ns)
		if err != nil {
			return err
		}
	}

	// lets add any extra github organisations if they are not already configured
	for _, org := range o.GitHubOrganisations {
		found := false
		for _, loc := range locations {
			if loc.GitURL == gits.GitHubURL && loc.Owner == org {
				found = true
				break
			}
		}
		if !found {
			locations = append(locations, v1.QuickStartLocation{
				GitURL:   gits.GitHubURL,
				GitKind:  gits.KindGitHub,
				Owner:    org,
				Includes: []string{"*"},
				Excludes: []string{"WIP-*"},
			})
		}
	}

	gitMap := map[string]map[string]v1.QuickStartLocation{}
	for _, loc := range locations {
		m := gitMap[loc.GitURL]
		if m == nil {
			m = map[string]v1.QuickStartLocation{}
			gitMap[loc.GitURL] = m
		}
		m[loc.Owner] = loc
	}

	var details *gits.CreateRepoData

	if !o.BatchMode {
		details, err = o.GetGitRepositoryDetails()
		if err != nil {
			return err
		}

		o.Filter.ProjectName = details.RepoName
	}


	model, err := o.LoadQuickstartsFromMap(config, gitMap)
	if err != nil {
		return fmt.Errorf("failed to load quickstarts: %s", err)
	}
	q, err := model.CreateSurvey(&o.Filter, o.BatchMode, o.In, o.Out, o.Err)
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

	// if there is a charts folder named after the app name, lets rename it to the generated app name
	folder := ""
	if q.Quickstart != nil {
		folder = q.Quickstart.Name
	}
	idx := strings.LastIndex(folder, "/")
	if idx > 0 {
		folder = folder[idx+1:]
	}
	if folder != "" {
		chartsDir := filepath.Join(genDir, "charts", folder)
		exists, err := util.FileExists(chartsDir)
		if err != nil {
			return err
		}
		if exists {
			o.PostDraftPackCallback = func() error {
				_, appName := filepath.Split(genDir)
				appChartDir := filepath.Join(genDir, "charts", appName)

				log.Infof("### PostDraftPack callback copying from %s to %s!!!s\n", chartsDir, appChartDir)
				err := util.CopyDirOverwrite(chartsDir, appChartDir)
				if err != nil {
					return err
				}
				err = os.RemoveAll(chartsDir)
				if err != nil {
					return err
				}
				return o.Git().Remove(genDir, filepath.Join("charts", folder))
			}
		} else {
			log.Infof("### NO charts folder %s\n", chartsDir)
		}
	}
	log.Infof("Created project at %s\n\n", util.ColorInfo(genDir))

	o.CreateProjectOptions.ImportOptions.GitProvider = o.GitProvider

	if details != nil {
		o.ConfigureImportOptions(details)
	}

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

	req, err := http.NewRequest(http.MethodGet, u, strings.NewReader(""))
	if err != nil {
		return answer, err
	}
	userAuth := q.GitProvider.UserAuth()
	token := userAuth.ApiToken
	username := userAuth.Username
	if token != "" && username != "" {
		o.Debugf("Downloading Quickstart source zip from %s with basic auth for user: %s\n", u, username)
		req.SetBasicAuth(username, token)
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
	log.Infof("Generated quickstart at %s\n", answer)
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

// LoadQuickstartsFromMap Load all quickstarts
func (o *CreateQuickstartOptions) LoadQuickstartsFromMap(config *auth.AuthConfig, gitMap map[string]map[string]v1.QuickStartLocation) (*quickstarts.QuickstartModel, error) {
	model := quickstarts.NewQuickstartModel()

	for gitURL, m := range gitMap {
		for _, location := range m {
			kind := location.GitKind
			if kind == "" {
				kind = gits.KindGitHub
			}
			gitProvider, err := o.gitProviderForGitServerURL(gitURL, kind)
			if err != nil {
				return model, err
			}
			o.Debugf("Searching for repositories in Git server %s owner %s includes %s excludes %s as user %s \n", gitProvider.ServerURL(), location.Owner, strings.Join(location.Includes, ", "), strings.Join(location.Excludes, ", "), gitProvider.CurrentUsername())
			err = model.LoadGithubQuickstarts(gitProvider, location.Owner, location.Includes, location.Excludes)
			if err != nil {
				o.Debugf("Quickstart load error: %s\n", err.Error())
			}
		}
	}
	return model, nil
}
