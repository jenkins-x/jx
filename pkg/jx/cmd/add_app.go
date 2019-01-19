package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	helmchart "k8s.io/helm/pkg/proto/hapi/chart"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"

	"github.com/jenkins-x/jx/pkg/surveyutils"

	"github.com/ghodss/yaml"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/util"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/jenkins-x/jx/pkg/log"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// AddAppOptions the options for the create spring command
type AddAppOptions struct {
	AddOptions

	GitOps bool
	DevEnv *jenkinsv1.Environment

	Repo     string
	Username string
	Password string
	Alias    string

	// allow git to be configured externally before a PR is created
	ConfigureGitCallback ConfigureGitFolderFn

	Namespace   string
	Version     string
	ReleaseName string
	SetValues   []string
	ValueFiles  []string
	HelmUpdate  bool
}

const (
	optionHelmUpdate = "helm-update"
	optionValues     = "values"
	optionSet        = "set"
	optionAlias      = "alias"
)

// NewCmdAddApp creates a command object for the "create" command
func NewCmdAddApp(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &AddAppOptions{
		AddOptions: AddOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "app",
		Short: "Adds an app",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addFlags(cmd, kube.DefaultNamespace, "", "")
	return cmd
}

func (o *AddAppOptions) addFlags(cmd *cobra.Command, defaultNamespace string, defaultOptionRelease string, defaultVersion string) {

	// Common flags

	cmd.Flags().StringVarP(&o.Version, "version", "v", defaultVersion,
		"The chart version to install")
	cmd.Flags().StringVarP(&o.Repo, "repository", "", "",
		"The repository from which the app should be installed (default specified in your dev environment)")
	cmd.Flags().StringVarP(&o.Username, "username", "", "",
		"The username for the repository")
	cmd.Flags().StringVarP(&o.Password, "password", "", "",
		"The password for the repository")
	cmd.Flags().BoolVarP(&o.BatchMode, optionBatchMode, "b", false, "In batch mode the command never prompts for user input")
	cmd.Flags().BoolVarP(&o.Verbose, optionVerbose, "", false, "Enable verbose logging")
	cmd.Flags().StringVarP(&o.Alias, optionAlias, "", "",
		"An alias to use for the app (available when using GitOps for your dev environment)")
	cmd.Flags().StringVarP(&o.ReleaseName, optionRelease, "r", defaultOptionRelease,
		"The chart release name (available when NOT using GitOps for your dev environment)")
	cmd.Flags().BoolVarP(&o.HelmUpdate, optionHelmUpdate, "", true,
		"Should we run helm update first to ensure we use the latest version (available when NOT using GitOps for your dev environment)")
	cmd.Flags().StringVarP(&o.Namespace, optionNamespace, "n", defaultNamespace, "The Namespace to install into (available when NOT using GitOps for your dev environment)")
	cmd.Flags().StringArrayVarP(&o.ValueFiles, optionValues, "f", []string{}, "List of locations for values files, "+
		"can be local files or URLs (available when NOT using GitOps for your dev environment)")
	cmd.Flags().StringArrayVarP(&o.SetValues, optionSet, "s", []string{},
		"The chart set values (can specify multiple or separate values with commas: key1=val1,key2=val2) (available when NOT using GitOps for your dev environment)")

}

// Run implements this command
func (o *AddAppOptions) Run() error {
	o.GitOps, o.DevEnv = o.GetDevEnv()
	if o.Repo == "" {
		o.Repo = o.DevEnv.Spec.TeamSettings.AppsRepository
	}

	if o.GitOps {
		msg := "Unable to specify --%s when using GitOps for your dev environment"
		if o.ReleaseName != "" {
			return util.InvalidOptionf(optionRelease, o.ReleaseName, msg, optionRelease)
		}
		if !o.HelmUpdate {
			return util.InvalidOptionf(optionHelmUpdate, o.HelmUpdate, msg, optionHelmUpdate)
		}
		if o.Namespace != "" {
			return util.InvalidOptionf(optionNamespace, o.Namespace, msg, optionNamespace)
		}
		if len(o.SetValues) > 0 {
			return util.InvalidOptionf(optionSet, o.SetValues, msg, optionSet)
		}
		if len(o.ValueFiles) > 1 {
			return util.InvalidOptionf(optionValues, o.SetValues,
				"No more than one --%s can be specified when using GitOps for your dev environment", optionValues)
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

	if o.Repo == "" {
		return fmt.Errorf("must specify a repository")
	}

	for _, app := range args {
		version := o.Version
		var schema []byte
		err := helm.InspectChart(app, version, o.Repo, o.Username, o.Password, o.Helm(), func(dir string) error {
			if version == "" {
				_, v, err := helm.LoadChartNameAndVersion(filepath.Join(dir, "Chart.yaml"))
				if err != nil {
					return err
				}
				version = v
				if o.Verbose {
					log.Infof("No version specified so using latest version which is %s\n", util.ColorInfo(version))
				}
			}
			schemaFile := filepath.Join(dir, "values.schema.json")
			if _, err := os.Stat(schemaFile); !os.IsNotExist(err) {
				schema, err = ioutil.ReadFile(schemaFile)
				if err != nil {
					return err
				}
			}

			if schema != nil {
				secrets := make([]*corev1.Secret, 0)
				schemaOptions := surveyutils.JSONSchemaOptions{
					CreateSecret: func(name string, key string, value string) (*jenkinsv1.ResourceReference, error) {
						secret := &corev1.Secret{
							TypeMeta: metav1.TypeMeta{
								Kind:       "Secret",
								APIVersion: corev1.SchemeGroupVersion.Version,
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: name,
							},
							Data: map[string][]byte{
								key: []byte(value),
							},
						}
						secrets = append(secrets, secret)
						return &jenkinsv1.ResourceReference{
							Name: name,
							Kind: "Secret",
						}, nil

					},
				}
				values, err := schemaOptions.GenerateValues(schema, []string{app}, o.In, o.Out, o.Err)
				if err != nil {
					return err
				}
				valuesYaml, err := yaml.JSONToYAML(values)
				if err != nil {
					return err
				}
				if o.Verbose {
					log.Infof("Generated values.yaml:\n\n%v\n", util.ColorInfo(string(valuesYaml)))
				}

				// For each secret, we write a file into the chart
				templatesDir := filepath.Join(dir, "templates")
				err = os.MkdirAll(templatesDir, 0700)
				if err != nil {
					return err
				}
				for _, secret := range secrets {
					file, err := ioutil.TempFile(templatesDir, fmt.Sprintf("%s-*.yaml", secret.Name))
					defer func() {
						err = file.Close()
						if err != nil {
							log.Warnf("Error closing %s because %v\n", file.Name(), err)
						}
					}()
					if err != nil {
						return err
					}
					bs, err := json.Marshal(secret)
					if err != nil {
						return err
					}
					ybs, err := yaml.JSONToYAML(bs)
					if err != nil {
						return err
					}
					_, err = file.Write(ybs)
					if err != nil {
						return err
					}
					if o.Verbose {
						log.Infof("Added secret %s\n\n%v\n", secret.Name, util.ColorInfo(string(ybs)))
					}
					if err != nil {
						return err
					}
				}
				if o.BatchMode {
					if schema != nil && o.Verbose {
						log.Warnf("%s prevents questions from schema being asked", optionBatchMode)
					}
				} else {
					if schema != nil && len(o.ValueFiles) > 0 {
						return fmt.Errorf("if you want to use %s you must use %s as %s has configuration questions",
							optionValues, optionBatchMode, app)
					} else if schema != nil {
						valuesFile, err := ioutil.TempFile("", fmt.Sprintf("%s-values.yaml", app))
						defer func() {
							err = valuesFile.Close()
							if err != nil {
								log.Warnf("Error closing %s because %v\n", valuesFile.Name(), err)
							}
						}()
						if err != nil {
							return err
						}
						_, err = valuesFile.Write(valuesYaml)
						if err != nil {
							return err
						}
						o.ValueFiles = []string{
							valuesFile.Name(),
						}
					}
				}

				if err != nil {
					return err
				}
			}

			if o.GitOps {
				err := o.createPR(app, dir, version)
				if err != nil {
					return err
				}
			} else {
				err := o.installApp(app, dir, version)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *AddAppOptions) createPR(app string, dir string, version string) error {

	modifyChartFn := func(requirements *helm.Requirements, chart *helmchart.Metadata, values map[string]interface{},
		templates map[string]string, envDir string) error {
		// See if the app already exists in requirements
		found := false
		for _, d := range requirements.Dependencies {
			if d.Name == app && d.Alias == o.Alias {
				// App found
				log.Infof("App %s already installed.\n", util.ColorWarning(app))
				if version != d.Version {
					log.Infof("To upgrade the app use %s or %s\n",
						util.ColorInfo("jx upgrade app <app>"),
						util.ColorInfo("jx upgrade apps --all"))
				}
				found = true
				break
			}
		}
		// If app not found, add it
		if !found {
			requirements.Dependencies = append(requirements.Dependencies, &helm.Dependency{
				Alias:      o.Alias,
				Repository: o.Repo,
				Name:       app,
				Version:    version,
			})
			appDir := filepath.Join(envDir, app)
			rootValuesFileName := filepath.Join(appDir, helm.ValuesFileName)
			err := os.MkdirAll(appDir, 0700)
			if err != nil {
				return errors.Wrapf(err, "cannot create app directory %s", appDir)
			}
			if o.Verbose {
				log.Infof("Using %s for app files\n", appDir)
			}
			if len(o.ValueFiles) == 1 {
				// We need to write the values file into the right spot for the app
				err = util.CopyFile(o.ValueFiles[0], rootValuesFileName)
				if err != nil {
					return errors.Wrapf(err, "cannot copy values."+
						"yaml to app directory %s", appDir)
				}
				if o.Verbose {
					log.Infof("Writing file provided by --%s to %s\n", optionValues, rootValuesFileName)
				}
			}
			// Write the release.yaml
			var gitRepo, releaseNotesURL, appReadme, description string
			templatesDir := filepath.Join(dir, "templates")
			if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
				if o.Verbose {
					log.Infof("No templates directory exists in %s", dir)
				}
			} else if err != nil {
				return errors.Wrapf(err, "stat directory %s", appDir)
			} else {
				releaseYamlPath := filepath.Join(templatesDir, "release.yaml")
				if _, err := os.Stat(releaseYamlPath); err == nil {
					bytes, err := ioutil.ReadFile(releaseYamlPath)
					if err != nil {
						return errors.Wrapf(err, "release.yaml from %s", templatesDir)
					}
					release := jenkinsv1.Release{}
					err = yaml.Unmarshal(bytes, &release)
					if err != nil {
						return errors.Wrapf(err, "unmarshal %s", releaseYamlPath)
					}
					gitRepo = release.Spec.GitHTTPURL
					releaseNotesURL = release.Spec.ReleaseNotesURL
					releaseYamlOutPath := filepath.Join(appDir, "release.yaml")
					err = ioutil.WriteFile(releaseYamlOutPath, bytes, 0755)
					if err != nil {
						return errors.Wrapf(err, "write file %s", releaseYamlOutPath)
					}
					if o.Verbose {
						log.Infof("Read release notes URL %s and git repo url %s from release.yaml\nWriting release."+
							"yaml from app to %s\n", releaseNotesURL, gitRepo, releaseYamlOutPath)
					}
				} else if os.IsNotExist(err) {
					if o.Verbose {

						log.Infof("Not adding release.yaml as not present in chart. Only files in %s are:\n",
							templatesDir)
						err := util.ListDirectory(templatesDir, true)
						if err != nil {
							return err
						}
					}
				} else {
					return errors.Wrapf(err, "reading release.yaml from %s", templatesDir)
				}
			}
			chartYamlPath := filepath.Join(dir, helm.ChartFileName)
			if _, err := os.Stat(chartYamlPath); err == nil {
				bytes, err := ioutil.ReadFile(chartYamlPath)
				if err != nil {
					return errors.Wrapf(err, "read %s from %s", helm.ChartFileName, dir)
				}
				chart := helmchart.Metadata{}
				err = yaml.Unmarshal(bytes, &chart)
				if err != nil {
					return errors.Wrapf(err, "unmarshal %s", chartYamlPath)
				}
				description = chart.Description

			} else if os.IsNotExist(err) {
				if o.Verbose {
					log.Infof("Not adding %s as not present in chart. Only files in %s are:\n", helm.ChartFileName, dir)
					err := util.ListDirectory(dir, true)
					if err != nil {
						return err
					}
				}
			} else {
				return errors.Wrapf(err, "stat Chart.yaml from %s", dir)
			}
			// Need to copy over any referenced files, and their schemas
			rootValues, err := helm.LoadValuesFile(rootValuesFileName)
			if err != nil {
				return err
			}
			schemas := make(map[string][]string)
			possibles := make(map[string]string)
			if _, err := os.Stat(dir); err == nil {
				files, err := ioutil.ReadDir(dir)
				if err != nil {
					return errors.Wrapf(err, "unable to list files in %s", dir)
				}
				possibleReadmes := make([]string, 0)
				for _, file := range files {
					fileName := strings.ToUpper(file.Name())
					if fileName == "README.MD" || fileName == "README" {
						possibleReadmes = append(possibleReadmes, filepath.Join(dir, file.Name()))
					}
				}
				if len(possibleReadmes) > 1 {
					if o.Verbose {
						log.Warnf("Unable to add README to PR for %s as more than one exists and not sure which to"+
							" use %s\n", app, possibleReadmes)
					}
				} else if len(possibleReadmes) == 1 {
					bytes, err := ioutil.ReadFile(possibleReadmes[0])
					if err != nil {
						return errors.Wrapf(err, "unable to read file %s", possibleReadmes[0])
					}
					appReadme = string(bytes)
				}

				for _, f := range files {
					ignore, err := util.IgnoreFile(f.Name(), helm.DefaultValuesTreeIgnores)
					if err != nil {
						return err
					}
					if !f.IsDir() && !ignore {
						key := f.Name()
						// Handle .schema. files specially
						if parts := strings.Split(key, ".schema."); len(parts) > 1 {
							// this is a file named *.schema.*, the part before .schema is the key
							if _, ok := schemas[parts[0]]; !ok {
								schemas[parts[0]] = make([]string, 0)
							}
							schemas[parts[0]] = append(schemas[parts[0]], filepath.Join(dir, f.Name()))
						}
						possibles[key] = filepath.Join(dir, f.Name())

					}
				}
			} else if !os.IsNotExist(err) {
				return errors.Wrap(err, fmt.Sprintf("error reading %s", dir))
			}
			if o.Verbose && appReadme == "" {
				log.Infof("Not adding App Readme as no README, README.md, readme or readme.md found in %s\n", dir)
			}
			readme := o.generateReadme(app, version, description, o.Repo, gitRepo, releaseNotesURL, appReadme)
			readmeOutPath := filepath.Join(appDir, "README.MD")
			err = ioutil.WriteFile(readmeOutPath, []byte(readme), 0755)
			if err != nil {
				return errors.Wrapf(err, "write README.md to %s", appDir)
			}
			if o.Verbose {
				log.Infof("Writing README.md to %s\n", readmeOutPath)
			}
			externalFileHandler := func(path string, element map[string]interface{}, key string) error {
				fileName, _ := filepath.Split(path)
				err := util.CopyFile(path, filepath.Join(appDir, fileName))
				if err != nil {
					return errors.Wrapf(err, "copy %s to %s", path, appDir)
				}
				// key for schema is the filename without the extension
				schemaKey := strings.TrimSuffix(fileName, filepath.Ext(fileName))
				if schemaPaths, ok := schemas[schemaKey]; ok {
					for _, schemaPath := range schemaPaths {
						fileName, _ := filepath.Split(schemaPath)
						schemaOutPath := filepath.Join(appDir, fileName)
						err := util.CopyFile(schemaPath, schemaOutPath)
						if err != nil {
							return errors.Wrapf(err, "copy %s to %s", schemaPath, appDir)
						}
						if o.Verbose {
							log.Infof("Writing %s to %s\n", fileName, schemaOutPath)
						}
					}
				}
				return nil
			}
			err = helm.HandleExternalFileRefs(rootValues, possibles, "", externalFileHandler)
			if err != nil {
				return err
			}
		}

		return nil
	}
	branchNameText := "add-app-" + app + "-" + version
	title := fmt.Sprintf("Add %s %s", app, version)
	message := fmt.Sprintf("Add app %s %s", app, version)

	pullRequestInfo, err := o.createEnvironmentPullRequest(o.DevEnv, modifyChartFn, &branchNameText, &title,
		&message,
		nil, o.ConfigureGitCallback)
	if err != nil {
		return errors.Wrapf(err, "creating pr for %s", app)
	}
	log.Infof("Added app via Pull Request %s\n", pullRequestInfo.PullRequest.URL)
	return nil
}

func (o *AddAppOptions) installApp(name string, chart string, version string) error {
	err := o.ensureHelm()
	if err != nil {
		return errors.Wrap(err, "failed to ensure that helm is present")
	}
	setValues := make([]string, 0)
	for _, vs := range o.SetValues {
		setValues = append(setValues, strings.Split(vs, ",")...)
	}

	err = o.installChartOptions(helm.InstallChartOptions{
		ReleaseName: name,
		Chart:       chart,
		Version:     version,
		Ns:          o.Namespace,
		HelmUpdate:  o.HelmUpdate,
		SetValues:   setValues,
		ValueFiles:  o.ValueFiles,
		Repository:  o.Repo,
		Username:    o.Username,
		Password:    o.Password,
	})
	if err != nil {
		return fmt.Errorf("failed to install name %s: %v", name, err)
	}
	// Attach the secrets to the name CRD

	return o.OnAppInstall(name, version)
}

func (o *AddAppOptions) generateReadme(app string, version string, description string, chartRepo string,
	gitRepo string, releaseNotesURL string, appReadme string) string {
	var readme strings.Builder
	readme.WriteString(fmt.Sprintf("# %s\n\n|App Metadata|---|\n", unknownZeroValue(app)))
	if version != "" {
		readme.WriteString(fmt.Sprintf("| **Version** | %s |\n", version))
	}
	if description != "" {
		readme.WriteString(fmt.Sprintf("| **Description** | %s |\n", description))
	}
	if chartRepo != "" {
		readme.WriteString(fmt.Sprintf("| **Chart Repository** | %s |\n", chartRepo))
	}
	if gitRepo != "" {
		readme.WriteString(fmt.Sprintf("| **Git Repository** | %s |\n", gitRepo))
	}
	if releaseNotesURL != "" {
		readme.WriteString(fmt.Sprintf("| **Release Notes** | %s |\n", releaseNotesURL))
	}

	if appReadme != "" {
		readme.WriteString(fmt.Sprintf("\n## App README.MD\n\n%s\n", appReadme))
	}
	return readme.String()
}

func unknownZeroValue(value string) string {
	if value == "" {
		return "unknown"
	}
	return value

}
