package helmfile

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"path"

	"github.com/jenkins-x/jx/pkg/config"
	helmfile2 "github.com/jenkins-x/jx/pkg/helmfile"

	"github.com/google/uuid"
	"github.com/jenkins-x/jx/pkg/util"

	"github.com/ghodss/yaml"

	"github.com/jenkins-x/jx/pkg/cmd/create/options"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	helmfile = "helmfile.yaml"
)

var (
	createHelmfileLong = templates.LongDesc(`
		** EXPERIMENTAL COMMAND **

		Creates a new helmfile.yaml from a jx-apps.yaml
`)

	createHelmfileExample = templates.Examples(`
		** EXPERIMENTAL COMMAND **

		# Create a new helmfile.yaml from a jx-apps.yaml
		jx create helmfile
	`)
)

// CreateHelmfileOptions the options for the create helmfile command
type CreateHelmfileOptions struct {
	options.CreateOptions
	outputDir  string
	dir        string
	valueFiles []string
}

// NewCmdCreateHelmfile  creates a command object for the "create" command
func NewCmdCreateHelmfile(commonOpts *opts.CommonOptions) *cobra.Command {
	o := &CreateHelmfileOptions{
		CreateOptions: options.CreateOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "helmfile",
		Short:   "Create a new helmfile",
		Long:    createHelmfileLong,
		Example: createHelmfileExample,
		Run: func(cmd *cobra.Command, args []string) {
			o.Cmd = cmd
			o.Args = args
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.dir, "dir", "", ".", "the directory to look for a 'jx-apps.yml' file")
	cmd.Flags().StringVarP(&o.outputDir, "outputDir", "", "", "The directory to write the helmfile.yaml file")
	cmd.Flags().StringArrayVarP(&o.valueFiles, "values", "", []string{""}, "specify values in a YAML file or a URL(can specify multiple)")

	return cmd
}

// Run implements the command
func (o *CreateHelmfileOptions) Run() error {

	apps, err := config.LoadApplicationsConfig(o.dir)
	if err != nil {
		return errors.Wrap(err, "failed to load applications")
	}

	helm := o.Helm()
	localHelmRepos, err := helm.ListRepos()
	if err != nil {
		return errors.Wrap(err, "failed listing helm repos")
	}

	// contains the repo url and name to reference it by in the release spec
	// use a map to dedupe repositories
	repos := make(map[string]string)
	for _, app := range apps.Applications {
		_, err = url.ParseRequestURI(app.Repository)
		if err != nil {
			// if the repository isn't a valid URL lets just use whatever was supplied in the application repository field, probably it is a directory path
			repos[app.Repository] = app.Repository
		} else {
			matched := false
			// check if URL matches a repo in helms local list
			for key, value := range localHelmRepos {
				if app.Repository == value {
					repos[app.Repository] = key
					matched = true
				}
			}
			if !matched {
				repos[app.Repository] = uuid.New().String()
			}
		}
	}

	var repositories []helmfile2.RepositorySpec
	for repoURL, name := range repos {
		_, err = url.ParseRequestURI(repoURL)
		// skip non URLs as they're probably local directories which don't need to be in the helmfile.repository section
		if err == nil {
			repository := helmfile2.RepositorySpec{
				Name: name,
				URL:  repoURL,
			}
			repositories = append(repositories, repository)
		}

	}

	var releases []helmfile2.ReleaseSpec
	for _, app := range apps.Applications {
		if app.Namespace == "" {
			app.Namespace = apps.DefaultNamespace
		}

		// check if a local directory and values file exists for the app
		extraValuesFiles := o.valueFiles
		extraValuesFiles = o.addExtraAppValues(app, extraValuesFiles, "values.yaml")
		extraValuesFiles = o.addExtraAppValues(app, extraValuesFiles, "values.yaml.gotmpl")

		chartName := fmt.Sprintf("%s/%s", repos[app.Repository], app.Name)
		release := helmfile2.ReleaseSpec{
			Name:      app.Name,
			Namespace: app.Namespace,
			Chart:     chartName,
			Values:    extraValuesFiles,
		}
		releases = append(releases, release)
	}

	h := helmfile2.HelmState{
		Bases: []string{"../environments.yaml"},
		HelmDefaults: helmfile2.HelmSpec{
			Atomic:  true,
			Verify:  false,
			Wait:    true,
			Timeout: 180,
			// need Force to be false https://github.com/helm/helm/issues/6378
			Force: false,
		},
		Repositories: repositories,
		Releases:     releases,
	}

	data, err := yaml.Marshal(h)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(o.outputDir, helmfile), data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", helmfile)
	}

	return nil
}

func (o *CreateHelmfileOptions) addExtraAppValues(app config.Application, newValuesFiles []string, valuesFilename string) []string {
	fileName := path.Join(o.dir, "apps", app.Name, valuesFilename)
	exists, _ := util.FileExists(fileName)
	if exists {
		newValuesFiles = append(newValuesFiles, path.Join(app.Name, valuesFilename))
	}
	return newValuesFiles
}
