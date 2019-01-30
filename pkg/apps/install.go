package apps

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/AlecAivazis/survey.v1/terminal"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/environments"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/vault"
	"k8s.io/client-go/kubernetes"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/surveyutils"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

// InstallOptions are shared options for installing, removing or upgrading apps for either GitOps or HelmOps
type InstallOptions struct {
	Helmer          helm.Helmer
	KubeClient      kubernetes.Interface
	InstallTimeout  string
	JxClient        versioned.Interface
	Namespace       string
	EnvironmentsDir string
	GitProvider     gits.GitProvider
	ConfigureGitFn  environments.ConfigureGitFn
	Gitter          gits.Gitter
	Verbose         bool
	DevEnv          *jenkinsv1.Environment
	BatchMode       bool
	In              terminal.FileReader
	Out             terminal.FileWriter
	Err             io.Writer
	GitOps          bool
	TeamName        string
	VaultClient     *vault.Client

	valuesFiles []string // internal variable used to track, most be passed in
}

// AddApp adds the app at a particular version (
// or latest if not specified) from the repository with username and password. A releaseName can be specified.
// Values can be passed with in files or as a slice of name=value pairs. An alias can be specified.
// GitOps or HelmOps will be automatically chosen based on the o.GitOps flag
func (o *InstallOptions) AddApp(app string, version string, repository string, username string, password string,
	releaseName string, valuesFiles []string, setValues []string, alias string, helmUpdate bool) error {
	inspectChartFunc := o.createInspectChartFn(version, app, repository, username, password, releaseName, setValues,
		alias, helmUpdate)
	o.valuesFiles = valuesFiles
	err := helm.InspectChart(app, version, repository, username, password, o.Helmer, inspectChartFunc)
	if err != nil {
		return err
	}
	return nil
}

//DeleteApp deletes the app. An alias and releaseName can be specified. GitOps or HelmOps will be automatically chosen based on the o.GitOps flag
func (o *InstallOptions) DeleteApp(app string, alias string, releaseName string, purge bool) error {
	if o.GitOps {
		opts := GitOpsOptions{
			InstallOptions: o,
		}
		err := opts.DeleteApp(app, alias)
		if err != nil {
			return err
		}
	} else {
		opts := HelmOpsOptions{
			InstallOptions: o,
		}
		err := opts.DeleteApp(app, releaseName, true)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpgradeApp upgrades the app (or all apps if empty) to a particular version (
// or the latest if not specified) from the repository with username and password. An alias can be specified.
// GitOps or HelmOps will be automatically chosen based on the o.GitOps flag
func (o *InstallOptions) UpgradeApp(app string, version string, repository string, username string, password string,
	alias string, update bool) error {
	if o.GitOps {
		opts := GitOpsOptions{
			InstallOptions: o,
		}
		err := opts.UpgradeApp(app, version, repository, username, password, alias)
		if err != nil {
			return err
		}
	} else {
		opts := HelmOpsOptions{
			InstallOptions: o,
		}
		err := opts.UpgradeApp(app, version, repository, username, password, alias, update)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *InstallOptions) createInspectChartFn(version string, app string, repository string, username string,
	password string, releaseName string, setValues []string, alias string, helmUpdate bool) func(dir string) error {
	var schema []byte
	var values []byte
	inspectChartFunc := func(dir string) error {
		if version == "" {
			var err error
			_, version, err = helm.LoadChartNameAndVersion(filepath.Join(dir, "Chart.yaml"))
			if err != nil {
				return errors.Wrapf(err, "error loading chart from %s", dir)
			}
			if o.Verbose {
				log.Infof("No version specified so using latest version which is %s\n", util.ColorInfo(version))
			}
		}
		schemaFile := filepath.Join(dir, "values.schema.json")
		if _, err := os.Stat(schemaFile); !os.IsNotExist(err) {
			schema, err = ioutil.ReadFile(schemaFile)
			if err != nil {
				return errors.Wrapf(err, "error reading schema file %s", schemaFile)
			}
		}

		if schema != nil {
			if len(o.valuesFiles) > 0 {
				log.Warnf("values.yaml specified by --valuesFiles will be used despite presence of schema in app")
			}
			var secrets []*surveyutils.GeneratedSecret
			var err error
			values, secrets, err = GenerateQuestions(schema, o.BatchMode, o.In, o.Out, o.Err)
			if err != nil {
				return errors.Wrapf(err, "asking questions for schema %s", schemaFile)
			}
			cleanup, err := o.handleValues(dir, app, values)
			defer cleanup()
			if err != nil {
				return err
			}
			cleanup, err = o.handleSecrets(dir, app, secrets)
			defer cleanup()
			if err != nil {
				return err
			}
		}

		if o.GitOps {
			opts := GitOpsOptions{
				InstallOptions: o,
			}
			err := opts.AddApp(app, dir, version, repository, alias)
			if err != nil {
				return err
			}
		} else {

			opts := HelmOpsOptions{
				InstallOptions: o,
			}
			err := opts.AddApp(app, dir, version, values, repository, username, password, releaseName, setValues,
				helmUpdate)
			if err != nil {
				return err
			}
		}
		return nil

	}
	return inspectChartFunc
}

func (o *InstallOptions) handleValues(dir string, app string, values []byte) (func(), error) {
	valuesFile, cleanup, err := AddValuesToChart(dir, app, values, o.Verbose)
	if err != nil {
		return cleanup, err
	}

	if o.valuesFiles == nil {
		o.valuesFiles = make([]string, 0)
	}
	o.valuesFiles = append(o.valuesFiles, valuesFile)
	return cleanup, nil
}

func (o *InstallOptions) handleSecrets(dir string, app string, generatedSecrets []*surveyutils.GeneratedSecret) (func(),
	error) {
	if o.VaultClient != nil {
		var vaultBasepath string
		if o.GitOps {
			gitInfo, err := gits.ParseGitURL(o.DevEnv.Spec.Source.URL)
			if err != nil {
				return nil, err
			}
			vaultBasepath = strings.Join([]string{"gitOps", gitInfo.Organisation, gitInfo.Name}, "/")
		} else {
			vaultBasepath = strings.Join([]string{"teams", o.TeamName}, "/")
		}
		f, err := AddSecretsToVault(generatedSecrets, *o.VaultClient, vaultBasepath)
		if err != nil {
			return func() {}, errors.Wrapf(err, "adding secrets to vault with basepath %s for %s", vaultBasepath,
				app)
		}
		return f, nil
	}
	secretsFile, f, err := AddSecretsToTemplate(dir, app, generatedSecrets)
	if err != nil {
		return func() {}, errors.Wrapf(err, "adding secrets to template for %s", app)
	}
	if o.valuesFiles == nil {
		o.valuesFiles = make([]string, 0)
	}
	o.valuesFiles = append(o.valuesFiles, secretsFile)
	return f, nil
}
