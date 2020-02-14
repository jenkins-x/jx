package helmfile

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/envctx"
	helmfile2 "github.com/jenkins-x/jx/pkg/helmfile"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/versionstream"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
		Creates a new helmfile.yaml from a jx-apps.yaml
`)

	createHelmfileExample = templates.Examples(`
		# Create a new helmfile.yaml from a jx-apps.yaml
		jx create helmfile
	`)
)

// GeneratedValues is a struct that gets marshalled into helm values for creating namespaces via helm
type GeneratedValues struct {
	Namespaces []string `json:"namespaces"`
}

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
	cmd.Flags().StringArrayVarP(&o.valueFiles, "values", "", nil, "specify values in a YAML file or a URL(can specify multiple)")

	return cmd
}

// Run implements the command
func (o *CreateHelmfileOptions) Run() error {
	apps, _, err := config.LoadAppConfig(o.dir)
	if err != nil {
		return errors.Wrap(err, "failed to load applications")
	}

	ec, err := o.EnvironmentContext(o.dir, true)
	if err != nil {
		return err
	}

	o.valueFiles = append(o.valueFiles, "../jx-requirements.values.yaml.gotmpl")

	secretsYaml := os.Getenv("JX_SECRETS_YAML")
	if secretsYaml != "" {
		o.valueFiles = append(o.valueFiles, secretsYaml)
	}

	err = o.ensureJxRequirementsYamlExists(ec.Requirements)
	if err != nil {
		return err
	}

	helmPrefixes, err := ec.VersionResolver.GetRepositoryPrefixes()
	if err != nil {
		return err
	}

	helm := o.Helm()
	localHelmRepos, err := helm.ListRepos()
	if err != nil {
		return errors.Wrap(err, "failed listing helm repos")
	}

	// iterate over all apps and split them into phases to generate separate helmfiles for each
	var applications []config.App
	var systemApplications []config.App
	charts := make(map[string]*envctx.ChartDetails)

	for i := range apps.Apps {
		app := &apps.Apps[i]
		details, err := ec.ChartDetails(app.Name, app.Repository)
		if err != nil {
			return errors.Wrapf(err, "failed to resolve chart details for %s repository %s", app.Name, app.Repository)
		}
		charts[app.Name] = details

		defaults, valuesFiles, err := ec.ResolveApplicationDefaults(details.Name)
		if err != nil {
			return err
		}
		app.Values = append(app.Values, valuesFiles...)
		if app.Namespace == "" {
			app.Namespace = defaults.Namespace
		}
		if app.Phase == "" {
			app.Phase = config.Phase(defaults.Phase)
		}

		// default phase is apps so set it in if empty
		if app.Phase == "" || app.Phase == config.PhaseApps {
			applications = append(applications, *app)
		}
		if app.Phase == config.PhaseSystem {
			systemApplications = append(systemApplications, *app)
		}
	}

	err = o.generateHelmFile(ec, helmPrefixes, applications, charts, err, localHelmRepos, apps, string(config.PhaseApps))
	if err != nil {
		return errors.Wrap(err, "failed to generate apps helmfile")
	}
	err = o.generateHelmFile(ec, helmPrefixes, systemApplications, charts, err, localHelmRepos, apps, string(config.PhaseSystem))
	if err != nil {
		return errors.Wrap(err, "failed to generate system helmfile")
	}

	return nil
}

func (o *CreateHelmfileOptions) generateHelmFile(ec *envctx.EnvironmentContext, helmPrefixes *versionstream.RepositoryPrefixes, applications []config.App, charts map[string]*envctx.ChartDetails, err error, localHelmRepos map[string]string, apps *config.AppConfig, phase string) error {
	// use a map to dedupe repositories
	repos := make(map[string]string)
	for _, app := range applications {
		details := charts[app.Name]
		if details == nil {
			continue
		}

		_, err = url.ParseRequestURI(details.Repository)
		if err != nil {
			// if the repository isn't a valid URL lets just use whatever was supplied in the application repository field, probably it is a directory path
			repos[details.Repository] = details.Repository
		} else {
			matched := false
			// check if URL matches a repo in helms local list
			for key, value := range localHelmRepos {
				if details.Repository == value {
					repos[details.Repository] = key
					matched = true
				}
			}
			if !matched {
				prefix := helmPrefixes.PrefixForURL(details.Repository)
				if prefix == "" {
					prefix = details.Prefix
				}
				if prefix == "" {
					prefix = uuid.New().String()
				}
				repos[details.Repository] = prefix
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
			found := false
			for _, r := range repositories {
				if r.URL == repoURL {
					found = true
					break
				}
			}
			if !found {
				repositories = append(repositories, repository)
			}
		}
	}

	for _, ar := range apps.Repositories {
		found := false
		for _, r := range repositories {
			if r.URL == ar.URL {
				found = true
				break
			}
		}
		if !found {
			repositories = append(repositories, ar)
		}
	}

	defaultNamespace := apps.DefaultNamespace
	if defaultNamespace == "" && ec.Requirements != nil {
		defaultNamespace = ec.Requirements.Cluster.Namespace
	}
	if defaultNamespace == "" && ec.DevEnv != nil {
		defaultNamespace = ec.DevEnv.Namespace
	}
	var releases []helmfile2.ReleaseSpec
	for i := range applications {
		app := &applications[i]
		details := charts[app.Name]
		if details == nil {
			continue
		}
		chartName := details.Name
		version := app.Version
		if ec.VersionResolver != nil {
			if version == "" {
				sv, err := ec.VersionResolver.StableVersion(versionstream.KindChart, details.Name)
				if err != nil {
					return errors.Wrapf(err, "failed to resolve version of chart %s", details.Name)
				}
				if sv != nil {
					version = sv.Version
				}
			}
		}
		if app.Namespace == "" {
			app.Namespace = defaultNamespace
		}

		// check if a local directory and values file exists for the app
		extraValuesFiles := append(app.Values, o.valueFiles...)
		extraValuesFiles = o.addExtraAppValues(*app, extraValuesFiles, "values.yaml", phase)
		extraValuesFiles = o.addExtraAppValues(*app, extraValuesFiles, "values.yaml.gotmpl", phase)

		release := helmfile2.ReleaseSpec{
			Name:      details.LocalName,
			Namespace: applications[i].Namespace,
			Version:   version,
			Chart:     chartName,
			Values:    extraValuesFiles,
		}
		releases = append(releases, release)
	}

	// if we have no releases then lets add a dummy entry to avoid `helmfile sync` failing
	if len(releases) == 0 {
		release := helmfile2.ReleaseSpec{
			Name:      "empty",
			Chart:     "jenkins-x/empty",
			Namespace: defaultNamespace,
		}
		releases = append(releases, release)

		found := false
		for _, repo := range repositories {
			if repo.Name == "jenkins-x" {
				found = true
				break
			}
		}
		if !found {
			repository := helmfile2.RepositorySpec{
				Name: "jenkins-x",
				URL:  kube.DefaultChartMuseumURL,
			}
			repositories = append(repositories, repository)
		}
	}

	// ensure any namespaces referenced are created first, do this via an extra chart that creates namespaces
	// so that helm manages the k8s resources, useful when cleaning up, this is a workaround for a helm 3 limitation
	// which is expected to be fixed
	repositories, releases, err = o.ensureNamespaceExist(repositories, releases, phase)
	if err != nil {
		return errors.Wrapf(err, "failed to check namespaces exists")
	}

	// lets sort the repositories in name order to minimise PR differences.
	// the releases are ordered via the `jx-apps.yml` file
	sort.Slice(repositories, func(i, j int) bool {
		r1 := repositories[i]
		r2 := repositories[j]
		return strings.Compare(r1.Name, r2.Name) < 0
	})

	h := helmfile2.HelmState{
		Bases: []string{"../environments.yaml"},
		HelmDefaults: helmfile2.HelmSpec{
			Atomic:  true,
			Verify:  false,
			Wait:    true,
			Timeout: 520,
			// need Force to be false https://github.com/helm/helm/issues/6378
			Force: false,
		},
		Repositories: repositories,
		Releases:     releases,
	}

	data, err := yaml.Marshal(h)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal helmfile data")
	}

	err = o.writeHelmfile(err, phase, data)
	if err != nil {
		return errors.Wrapf(err, "failed to write helmfile")
	}
	return nil
}

func (o *CreateHelmfileOptions) writeHelmfile(err error, phase string, data []byte) error {
	exists, err := util.DirExists(path.Join(o.outputDir, phase))
	if err != nil || !exists {
		err = os.MkdirAll(path.Join(o.outputDir, phase), os.ModePerm)
		if err != nil {
			return errors.Wrapf(err, "cannot create phase directory %s ", path.Join(o.outputDir, phase))
		}
	}
	err = ioutil.WriteFile(path.Join(o.outputDir, phase, helmfile), data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", helmfile)
	}

	return nil
}

func (o *CreateHelmfileOptions) addExtraAppValues(app config.App, newValuesFiles []string, valuesFilename, phase string) []string {
	fileName := path.Join(o.dir, phase, app.Name, valuesFilename)
	exists, _ := util.FileExists(fileName)
	if exists {
		newValuesFiles = append(newValuesFiles, path.Join(app.Name, valuesFilename))
	}
	parts := strings.Split(app.Name, "/")
	if len(parts) == 2 {
		localName := parts[1]
		fileName := path.Join(o.dir, phase, localName, valuesFilename)
		exists, _ := util.FileExists(fileName)
		if exists {
			newValuesFiles = append(newValuesFiles, path.Join(localName, valuesFilename))
		}
	}
	return newValuesFiles
}

// this is a temporary function that wont be needed once helm 3 supports creating namespaces
func (o *CreateHelmfileOptions) ensureNamespaceExist(helmfileRepos []helmfile2.RepositorySpec, helmfileReleases []helmfile2.ReleaseSpec, phase string) ([]helmfile2.RepositorySpec, []helmfile2.ReleaseSpec, error) {

	// start by deleting the existing generated directory
	err := os.RemoveAll(path.Join(o.outputDir, phase, "generated"))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot delete generated values directory %s ", path.Join(phase, "generated"))
	}

	client, currentNamespace, err := o.KubeClientAndNamespace()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to create kube client")
	}

	namespaces, err := client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to list namespaces")
	}

	namespaceMatched := false
	// loop over each application and check if the namespace it references exists, if not add the namespace creator chart to the helmfile
	for k, release := range helmfileReleases {
		for _, ns := range namespaces.Items {
			if ns.Name == release.Namespace {
				namespaceMatched = true
			}
		}
		if !namespaceMatched {
			existingCreateNamespaceChartFound := false
			for _, release := range helmfileReleases {
				if release.Name == "namespace-"+release.Namespace {
					existingCreateNamespaceChartFound = true
				}
			}
			if !existingCreateNamespaceChartFound {

				err := o.writeGeneratedNamespaceValues(release.Namespace, phase)
				if err != nil {
					errors.Wrapf(err, "failed to write generated namespace values file")
				}

				repository := helmfile2.RepositorySpec{
					Name: "zloeber",
					URL:  "git+https://github.com/zloeber/helm-namespace@chart",
				}
				helmfileRepos = append(helmfileRepos, repository)

				createNamespaceChart := helmfile2.ReleaseSpec{
					Name:      "namespace-" + release.Namespace,
					Namespace: currentNamespace,
					Chart:     "zloeber/namespace",

					Values: []string{path.Join("generated", release.Namespace, "values.yaml")},
				}

				// add a dependency so that the create namespace chart is installed before the app chart
				helmfileReleases[k].Needs = []string{fmt.Sprintf("%s/namespace-%s", currentNamespace, release.Namespace)}

				helmfileReleases = append(helmfileReleases, createNamespaceChart)
			}
		}
	}

	return helmfileRepos, helmfileReleases, nil
}

func (o *CreateHelmfileOptions) writeGeneratedNamespaceValues(namespace, phase string) error {
	// workaround with using []interface{} for values, this causes problems with (un)marshalling so lets write a file and
	// add the file path to the []string values
	err := os.MkdirAll(path.Join(o.outputDir, phase, "generated", namespace), os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "cannot create generated values directory %s ", path.Join(phase, "generated", namespace))
	}
	value := GeneratedValues{
		Namespaces: []string{namespace},
	}
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path.Join(o.outputDir, phase, "generated", namespace, "values.yaml"), data, util.DefaultWritePermissions)
	return nil
}

func (o *CreateHelmfileOptions) ensureJxRequirementsYamlExists(requirements *config.RequirementsConfig) error {
	fileName := filepath.Join(o.dir, config.RequirementsValuesFileName)
	exists, err := util.FileExists(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to check if file exists %s", fileName)

	}
	if exists {
		return nil
	}
	err = config.SaveRequirementsValuesFile(requirements, o.dir)
	if err != nil {
		return errors.Wrap(err, "failed to save requirements yaml file")
	}
	return nil
}
