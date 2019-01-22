package helm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/jenkins-x/jx/pkg/kube"
	"k8s.io/client-go/kubernetes"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

const (
	// ChartFileName file name for a chart
	ChartFileName = "Chart.yaml"
	// RequirementsFileName the file name for helm requirements
	RequirementsFileName = "requirements.yaml"
	// SecretsFileName the file name for secrets
	SecretsFileName = "secrets.yaml"
	// ValuesFileName the file name for values
	ValuesFileName = "values.yaml"
	// TemplatesDirName is the default name for the templates directory
	TemplatesDirName = "templates"

	// DefaultHelmRepositoryURL is the default cluster local helm repo
	DefaultHelmRepositoryURL = "http://jenkins-x-chartmuseum:8080"

	defaultEnvironmentChartDir = "env"
)

// copied from helm to minimise dependencies...

// Dependency describes a chart upon which another chart depends.
//
// Dependencies can be used to express developer intent, or to capture the state
// of a chart.
type Dependency struct {
	// Name is the name of the dependency.
	//
	// This must mach the name in the dependency's Chart.yaml.
	Name string `json:"name"`
	// Version is the version (range) of this chart.
	//
	// A lock file will always produce a single version, while a dependency
	// may contain a semantic version range.
	Version string `json:"version,omitempty"`
	// The URL to the repository.
	//
	// Appending `index.yaml` to this string should result in a URL that can be
	// used to fetch the repository index.
	Repository string `json:"repository"`
	// A yaml path that resolves to a boolean, used for enabling/disabling charts (e.g. subchart1.enabled )
	Condition string `json:"condition,omitempty"`
	// Tags can be used to group charts for enabling/disabling together
	Tags []string `json:"tags,omitempty"`
	// Enabled bool determines if chart should be loaded
	Enabled bool `json:"enabled,omitempty"`
	// ImportValues holds the mapping of source values to parent key to be imported. Each item can be a
	// string or pair of child/parent sublist items.
	ImportValues []interface{} `json:"import-values,omitempty"`
	// Alias usable alias to be used for the chart
	Alias string `json:"alias,omitempty"`
}

// ErrNoRequirementsFile to detect error condition
type ErrNoRequirementsFile error

// Requirements is a list of requirements for a chart.
//
// Requirements are charts upon which this chart depends. This expresses
// developer intent.
type Requirements struct {
	Dependencies []*Dependency `json:"dependencies"`
}

// DepSorter Used to avoid merge conflicts by sorting deps by name
type DepSorter []*Dependency

func (a DepSorter) Len() int           { return len(a) }
func (a DepSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DepSorter) Less(i, j int) bool { return a[i].Name < a[j].Name }

// SetAppVersion sets the version of the app to use
func (r *Requirements) SetAppVersion(app string, version string, repository string, alias string) {
	if r.Dependencies == nil {
		r.Dependencies = []*Dependency{}
	}
	for _, dep := range r.Dependencies {
		if dep != nil && dep.Name == app {
			dep.Version = version
			dep.Repository = repository
			dep.Alias = alias
			return
		}
	}
	r.Dependencies = append(r.Dependencies, &Dependency{
		Name:       app,
		Version:    version,
		Repository: repository,
		Alias:      alias,
	})
	sort.Sort(DepSorter(r.Dependencies))
}

// RemoveApplication removes the given app name. Returns true if a dependency was removed
func (r *Requirements) RemoveApplication(app string) bool {
	for i, dep := range r.Dependencies {
		if dep != nil && dep.Name == app {
			r.Dependencies = append(r.Dependencies[:i], r.Dependencies[i+1:]...)
			sort.Sort(DepSorter(r.Dependencies))
			return true
		}
	}
	return false
}

// FindRequirementsFileName returns the default requirements.yaml file name
func FindRequirementsFileName(dir string) (string, error) {
	return findFileName(dir, RequirementsFileName)
}

// FindChartFileName returns the default chart.yaml file name
func FindChartFileName(dir string) (string, error) {
	return findFileName(dir, ChartFileName)
}

// FindValuesFileName returns the default values.yaml file name
func FindValuesFileName(dir string) (string, error) {
	return findFileName(dir, ValuesFileName)
}

// FindTemplatesDirName returns the default templates/ dir name
func FindTemplatesDirName(dir string) (string, error) {
	return findFileName(dir, TemplatesDirName)
}

func findFileName(dir string, fileName string) (string, error) {
	names := []string{
		filepath.Join(dir, defaultEnvironmentChartDir, fileName),
		filepath.Join(dir, fileName),
	}
	for _, name := range names {
		exists, err := util.FileExists(name)
		if err != nil {
			return "", err
		}
		if exists {
			return name, nil
		}
	}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, f := range files {
		if f.IsDir() {
			name := filepath.Join(dir, f.Name(), fileName)
			exists, err := util.FileExists(name)
			if err != nil {
				return "", err
			}
			if exists {
				return name, nil
			}
		}
	}
	dirs := []string{
		filepath.Join(dir, defaultEnvironmentChartDir),
		dir,
	}
	for _, d := range dirs {
		name := filepath.Join(d, fileName)
		exists, err := util.FileExists(d)
		if err != nil {
			return "", err
		}
		if exists {
			return name, nil
		}
	}
	return "", fmt.Errorf("Could not deduce the default requirements.yaml file name")
}

// LoadRequirementsFile loads the requirements file or creates empty requirements if the file does not exist
func LoadRequirementsFile(fileName string) (*Requirements, error) {
	exists, err := util.FileExists(fileName)
	if err != nil {
		return nil, err
	}
	if exists {
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			return nil, err
		}
		return LoadRequirements(data)
	}
	r := &Requirements{}
	return r, nil
}

// LoadChartFile loads the chart file or creates empty chart if the file does not exist
func LoadChartFile(fileName string) (*chart.Metadata, error) {
	exists, err := util.FileExists(fileName)
	if err != nil {
		return nil, err
	}
	if exists {
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			return nil, err
		}
		return LoadChart(data)
	}
	return &chart.Metadata{}, nil
}

// LoadValuesFile loads the values file or creates empty map if the file does not exist
func LoadValuesFile(fileName string) (map[string]interface{}, error) {
	exists, err := util.FileExists(fileName)
	if err != nil {
		return nil, errors.Wrapf(err, "checking %s exists", fileName)
	}
	if exists {
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			return nil, errors.Wrapf(err, "reading %s", fileName)
		}
		v, err := LoadValues(data)
		if err != nil {
			return nil, errors.Wrapf(err, "unmarshaling %s", fileName)
		}
		return v, nil
	}
	return make(map[string]interface{}), nil
}

// LoadTemplatesDir loads the files in the templates dir or creates empty map if none exist
func LoadTemplatesDir(dirName string) (map[string]string, error) {
	exists, err := util.DirExists(dirName)
	if err != nil {
		return nil, err
	}
	answer := make(map[string]string)
	if exists {
		files, err := ioutil.ReadDir(dirName)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			filename, _ := filepath.Split(f.Name())
			answer[filename] = f.Name()
		}
	}
	return answer, nil
}

// LoadRequirements loads the requirements from some data
func LoadRequirements(data []byte) (*Requirements, error) {
	r := &Requirements{}
	return r, yaml.Unmarshal(data, r)
}

// LoadChart loads the requirements from some data
func LoadChart(data []byte) (*chart.Metadata, error) {
	r := &chart.Metadata{}
	return r, yaml.Unmarshal(data, r)
}

// LoadValues loads the values from some data
func LoadValues(data []byte) (map[string]interface{}, error) {
	r := make(map[string]interface{})

	return r, yaml.Unmarshal(data, &r)
}

// SaveFile saves contents (a pointer to a data structure) to a file
func SaveFile(fileName string, contents interface{}) error {
	data, err := yaml.Marshal(contents)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
}

func LoadChartName(chartFile string) (string, error) {
	chart, err := chartutil.LoadChartfile(chartFile)
	if err != nil {
		return "", err
	}
	return chart.Name, nil
}

func LoadChartNameAndVersion(chartFile string) (string, string, error) {
	chart, err := chartutil.LoadChartfile(chartFile)
	if err != nil {
		return "", "", err
	}
	return chart.Name, chart.Version, nil
}

// ModifyChart modifies the given chart using a callback
func ModifyChart(chartFile string, fn func(chart *chart.Metadata) error) error {
	chart, err := chartutil.LoadChartfile(chartFile)
	if err != nil {
		return errors.Wrapf(err, "Failed to load chart file %s", chartFile)
	}
	err = fn(chart)
	if err != nil {
		return errors.Wrapf(err, "Failed to modify chart for file %s", chartFile)
	}
	err = chartutil.SaveChartfile(chartFile, chart)
	if err != nil {
		return errors.Wrapf(err, "Failed to save modified chart file %s", chartFile)
	}
	return nil
}

// SetChartVersion modifies the given chart file to update the version
func SetChartVersion(chartFile string, version string) error {
	callback := func(chart *chart.Metadata) error {
		chart.Version = version
		return nil
	}
	return ModifyChart(chartFile, callback)
}

func AppendMyValues(valueFiles []string) ([]string, error) {
	// Overwrite the values with the content of myvalues.yaml files from the current folder if exists, otherwise
	// from ~/.jx folder also only if it's present
	curDir, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the current working directory")
	}
	myValuesFile := filepath.Join(curDir, "myvalues.yaml")
	exists, err := util.FileExists(myValuesFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if the myvaules.yaml file exists in the current directory")
	}
	if exists {
		valueFiles = append(valueFiles, myValuesFile)
		log.Infof("Using local value overrides file %s\n", util.ColorInfo(myValuesFile))
	} else {
		configDir, err := util.ConfigDir()
		if err != nil {
			return nil, errors.Wrap(err, "failed to read the config directory")
		}
		myValuesFile = filepath.Join(configDir, "myvalues.yaml")
		exists, err = util.FileExists(myValuesFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check if the myvaules.yaml file exists in the .jx directory")
		}
		if exists {
			valueFiles = append(valueFiles, myValuesFile)
			log.Infof("Using local value overrides file %s\n", util.ColorInfo(myValuesFile))
		}
	}
	return valueFiles, nil
}

// CombineValueFilesToFile iterates through the input files and combines them into a single Values object and then
// write it to the output file nested inside the chartName
func CombineValueFilesToFile(outFile string, inputFiles []string, chartName string, extraValues map[string]interface{}) error {
	answer := chartutil.Values{}
	for _, input := range inputFiles {
		values, err := chartutil.ReadValuesFile(input)
		if err != nil {
			return errors.Wrapf(err, "Failed to read helm values YAML file %s", input)
		}
		sourceMap := answer.AsMap()
		util.CombineMapTrees(sourceMap, values.AsMap())
		answer = chartutil.Values(sourceMap)
	}
	m := answer.AsMap()
	for k, v := range extraValues {
		m[k] = v
	}
	answerMap := map[string]interface{}{
		chartName: m,
	}
	answer = chartutil.Values(answerMap)
	text, err := answer.YAML()
	if err != nil {
		return errors.Wrap(err, "Failed to marshal the combined values YAML files back to YAML")
	}
	err = ioutil.WriteFile(outFile, []byte(text), util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "Failed to save combined helm values YAML file %s", outFile)
	}
	return nil
}

// GetLatestVersion get's the latest version of a chart in a repo using helmer
func GetLatestVersion(chart string, repo string, username string, password string, helmer Helmer) (string, error) {
	latest := ""
	version := ""
	err := InspectChart(chart, version, repo, username, password, helmer, func(dir string) error {
		var err error
		_, latest, err = LoadChartNameAndVersion(filepath.Join(dir, "Chart.yaml"))
		return err
	})
	return latest, err
}

// InspectChart fetches the specified chart in a repo using helmer, and then calls the closure on it, before cleaning up
func InspectChart(chart string, version string, repo string, username string, password string,
	helmer Helmer, closure func(dir string) error) error {
	dir, err := ioutil.TempDir("", fmt.Sprintf("jx-helm-fetch-%s-", chart))
	defer func() {
		err1 := os.RemoveAll(dir)
		if err1 != nil {
			log.Warnf("Error removing %s %v\n", dir, err1)
		}
	}()
	err = helmer.FetchChart(chart, version, true, dir, repo, username, password)
	if err != nil {
		return err
	}
	return closure(filepath.Join(dir, chart))
}

type InstallChartOptions struct {
	Dir         string
	ReleaseName string
	Chart       string
	Version     string
	Ns          string
	HelmUpdate  bool
	SetValues   []string
	ValueFiles  []string
	Repository  string
	Username    string
	Password    string
}

// InstallFromChartOptions uses the helmer and kubeClient interfaces to install the chart from the options,
// respeciting the installTimeout
func InstallFromChartOptions(options InstallChartOptions, helmer Helmer, kubeClient kubernetes.Interface,
	installTimeout string) error {
	if options.HelmUpdate {
		log.Infoln("Updating Helm repository...")
		err := helmer.UpdateRepo()
		if err != nil {
			return errors.Wrap(err, "failed to update repository")
		}
		log.Infoln("Helm repository update done.")
	}
	if options.Ns != "" {
		annotations := map[string]string{"jenkins-x.io/created-by": "Jenkins X"}
		kube.EnsureNamespaceCreated(kubeClient, options.Ns, nil, annotations)
	}
	timeout, err := strconv.Atoi(installTimeout)
	if err != nil {
		return errors.Wrap(err, "failed to convert the timeout to an int")
	}
	helmer.SetCWD(options.Dir)
	return helmer.UpgradeChart(options.Chart, options.ReleaseName, options.Ns, options.Version, true,
		timeout, true, false, options.SetValues, options.ValueFiles, options.Repository, options.Username,
		options.Password)
}
