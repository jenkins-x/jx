package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/gits"

	"github.com/jenkins-x/jx/pkg/vault"

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

const (
	appsGeneratedSecretKey = "appsGeneratedSecrets"
)

const secretTemplate = `
{{- range .Values.generatedSecrets }}
apiVersion: v1
data:
  {{ .key }}: {{ .value }}
kind: Secret
metadata:
  name: {{ .name }} 
type: Opaque
{{- end }}
`

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
	if o.Repo == "" {
		o.Repo = DEFAULT_CHARTMUSEUM_URL
	}

	var vaultBasepath string
	var vaultClient vault.Client
	useVault := o.UseVault()
	if useVault {
		var err error
		if o.GitOps {
			gitInfo, err := gits.ParseGitURL(o.DevEnv.Spec.Source.URL)
			if err != nil {
				return err
			}
			vaultBasepath = strings.Join([]string{"gitOps", gitInfo.Organisation, gitInfo.Name}, "/")
		} else {

			teamName, _, err := o.TeamAndEnvironmentNames()
			if err != nil {
				return err
			}
			vaultBasepath = strings.Join([]string{"teams", teamName}, "/")
		}
		vaultClient, err = o.CreateSystemVaultClient("")
		if err != nil {
			return err
		}
	}

	if o.GitOps {
		msg := "unable to specify --%s when using GitOps for your dev environment"
		if o.ReleaseName != "" {
			return util.InvalidOptionf(optionRelease, o.ReleaseName, msg, optionRelease)
		}
		if !o.HelmUpdate {
			return util.InvalidOptionf(optionHelmUpdate, o.HelmUpdate, msg, optionHelmUpdate)
		}
		if o.Namespace != "" && o.Namespace != kube.DefaultNamespace {
			return util.InvalidOptionf(optionNamespace, o.Namespace, msg, optionNamespace)
		}
		if len(o.SetValues) > 0 {
			return util.InvalidOptionf(optionSet, o.SetValues, msg, optionSet)
		}
		if len(o.ValueFiles) > 1 {
			return util.InvalidOptionf(optionValues, o.SetValues,
				"no more than one --%s can be specified when using GitOps for your dev environment", optionValues)
		}
		if !useVault {
			return fmt.Errorf("cannot install apps without a vault when using GitOps for your dev environment")
		}
	}
	if !o.GitOps {
		if o.Alias != "" {
			return util.InvalidOptionf(optionAlias, o.Alias,
				"unable to specify --%s when NOT using GitOps for your dev environment", optionAlias)
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
		var version string
		if o.Version != "" {
			version = o.Version
		}
		var schema []byte
		err := helm.InspectChart(app, version, o.Repo, o.Username, o.Password, o.Helm(), func(dir string) error {
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
				secrets := make([]*surveyutils.GeneratedSecret, 0)
				schemaOptions := surveyutils.JSONSchemaOptions{
					CreateSecret: func(name string, key string, value string) (*jenkinsv1.ResourceReference, error) {
						secret := &surveyutils.GeneratedSecret{
							Name:  name,
							Key:   key,
							Value: value,
						}
						secrets = append(secrets, secret)
						return &jenkinsv1.ResourceReference{
							Name: name,
							Kind: "Secret",
						}, nil

					},
				}
				values, err := schemaOptions.GenerateValues(schema, []string{}, o.In, o.Out, o.Err)
				if err != nil {
					return errors.Wrapf(err, "error generating values for schema %s", schemaFile)
				}
				valuesYaml, err := yaml.JSONToYAML(values)
				if err != nil {
					return errors.Wrapf(err, "error converting values from json to yaml\n\n%v", values)
				}
				if o.Verbose {
					log.Infof("Generated values.yaml:\n\n%v\n", util.ColorInfo(string(valuesYaml)))
				}

				var secretsYaml []byte
				// We write a secret template into the chart, append the values for the generated secrets to values.yaml
				if len(secrets) > 0 {
					if useVault {
						for _, secret := range secrets {
							path := strings.Join([]string{vaultBasepath, secret.Name}, "/")
							err := vault.WriteMap(vaultClient, path, map[string]interface{}{
								secret.Key: secret.Value,
							})
							if err != nil {
								return err
							}
						}
					} else {
						// For each secret, we write a file into the chart
						templatesDir := filepath.Join(dir, "templates")
						err = os.MkdirAll(templatesDir, 0700)
						if err != nil {
							return err
						}
						fileName := filepath.Join(templatesDir, "app-generated-secret-template.yaml")
						err := ioutil.WriteFile(fileName, []byte(secretTemplate), 0755)
						if err != nil {
							return err
						}
						allSecrets := map[string][]*surveyutils.GeneratedSecret{
							appsGeneratedSecretKey: secrets,
						}
						secretsYaml, err = yaml.Marshal(allSecrets)
						if err != nil {
							return err
						}
					}
				}

				if err != nil {
					return err
				}
				if len(o.ValueFiles) > 0 && schema != nil {
					log.Warnf("values.yaml specified by --valuesFiles will be used despite presence of schema in app")
				} else if schema != nil {
					valuesFile, err := ioutil.TempFile("", fmt.Sprintf("%s-values.yaml", app))
					defer func() {
						err = valuesFile.Close()
						if err != nil {
							log.Warnf("Error closing %s because %v\n", valuesFile.Name(), err)
						}
						err = util.DeleteFile(valuesFile.Name())
						if err != nil {
							log.Warnf("Error deleting %s because %v\n", valuesFile.Name(), err)
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
					if !o.GitOps {
						if len(secretsYaml) > 0 {
							secretsFile, err := ioutil.TempFile("", fmt.Sprintf("%s-secrets.yaml", app))
							defer func() {
								err = secretsFile.Close()
								if err != nil {
									log.Warnf("Error closing %s because %v\n", secretsFile.Name(), err)
								}
								err = util.DeleteFile(secretsFile.Name())
								if err != nil {
									log.Warnf("Error deleting %s because %v\n", secretsFile.Name(), err)
								}
							}()
							if err != nil {
								return err
							}
							_, err = secretsFile.Write(secretsYaml)
							if err != nil {
								return err
							}
							o.ValueFiles = append(o.ValueFiles, secretsFile.Name())
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

	branchNameText := "add-app-" + app + "-" + version
	title := fmt.Sprintf("Add %s %s", app, version)
	message := fmt.Sprintf("Add app %s %s", app, version)

	pullRequestInfo, err := o.createEnvironmentPullRequest(o.DevEnv, o.CreateAddRequirementFn(app, o.Alias, version,
		o.Repo, o.ValueFiles, dir),
		&branchNameText, &title,
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

	err = o.OnAppInstall(name, version)
	if err != nil {
		return errors.Wrapf(err, "running postinstall hooks for %s, name")
	}
	log.Infof("Successfully installed %s %s\n", util.ColorInfo(name), util.ColorInfo(version))
	return nil
}
