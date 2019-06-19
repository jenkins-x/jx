package apps

import (
	"encoding/base64"
	"fmt"

	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/vaulturl"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/environments"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"

	"github.com/jenkins-x/jx/pkg/surveyutils"
	"github.com/jenkins-x/jx/pkg/vault"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	//ValuesAnnotation is the name of the annotation used to stash values
	ValuesAnnotation       = "jenkins.io/values.yaml"
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

// StashValues takes the values used to configure an app and annotates the APP CRD with them allowing them to be used
// at a later date e.g. when the app is upgraded
func StashValues(values []byte, name string, jxClient versioned.Interface, ns string, chartDir string, repository string) (bool, *jenkinsv1.App, error) {
	// locate the app CRD
	create := false
	app, err := jxClient.JenkinsV1().Apps(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		create = true
		app = &jenkinsv1.App{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: jenkinsv1.AppSpec{},
		}
	}
	// base64 encode the values.yaml
	encoded := base64.StdEncoding.EncodeToString(values)
	if app.Annotations == nil {
		app.Annotations = make(map[string]string)
	}
	app.Annotations[ValuesAnnotation] = encoded

	err = environments.AddAppMetaData(chartDir, app, repository)
	if err != nil {
		return false, nil, errors.Wrapf(err, "adding metadata to app %s", app.Name)
	}

	return create, app, nil
}

// AddSecretsToVault adds the generatedSecrets into the vault using client at basepath
func AddSecretsToVault(generatedSecrets []*surveyutils.GeneratedSecret, client vault.Client) (func(), error) {
	if len(generatedSecrets) > 0 {
		for _, secret := range generatedSecrets {
			err := vault.WriteMap(client, secret.Path, map[string]interface{}{
				secret.Key: secret.Value,
			})
			if err != nil {
				return func() {}, err
			}
		}
	}
	return func() {}, nil
}

// AddSecretsToTemplate adds the generatedSecrets into the template (rooted at dir) as Kubernetes Secrets,
// using app as the base of the name
func AddSecretsToTemplate(dir string, chartName string, generatedSecrets []*surveyutils.GeneratedSecret) (string, func(),
	error) {
	// We write a secret template into the chart, append the values for the generated generatedSecrets to values.yaml
	if len(generatedSecrets) > 0 {
		// For each secret, we write a file into the chart
		templatesDir := filepath.Join(dir, "templates")
		err := os.MkdirAll(templatesDir, 0700)
		if err != nil {
			return "", func() {}, err
		}
		fileName := filepath.Join(templatesDir, "chartName-generated-secret-template.yaml")
		err = ioutil.WriteFile(fileName, []byte(secretTemplate), 0755)
		if err != nil {
			return "", func() {}, err
		}
		allSecrets := map[string][]*surveyutils.GeneratedSecret{
			appsGeneratedSecretKey: generatedSecrets,
		}
		secretsYaml, err := yaml.Marshal(allSecrets)
		if err != nil {
			return "", func() {}, err
		}
		secretsFile, err := ioutil.TempFile("", fmt.Sprintf("%s-generatedSecrets.yaml", ToValidFileSystemName(chartName)))
		cleanup := func() {
			err = secretsFile.Close()
			if err != nil {
				log.Logger().Warnf("Error closing %s because %v", secretsFile.Name(), err)
			}
			err = util.DeleteFile(secretsFile.Name())
			if err != nil {
				log.Logger().Warnf("Error deleting %s because %v", secretsFile.Name(), err)
			}
		}
		if err != nil {
			return "", func() {}, err
		}
		_, err = secretsFile.Write(secretsYaml)
		if err != nil {
			return "", func() {}, err
		}
		return secretsFile.Name(), cleanup, nil

	}
	return "", func() {}, nil
}

// AddValuesToChart adds a values file to the chart rooted at dir
func AddValuesToChart(chartName string, values []byte, verbose bool) (string, func(), error) {
	valuesYaml, err := yaml.JSONToYAML(values)
	if err != nil {
		return "", func() {}, errors.Wrapf(err, "error converting values from json to yaml\n\n%v", values)
	}
	if verbose {
		log.Logger().Infof("Generated values.yaml:\n\n%v", util.ColorInfo(string(valuesYaml)))
	}

	valuesFile, err := ioutil.TempFile("", fmt.Sprintf("%s-values.yaml", ToValidFileSystemName(chartName)))
	cleanup := func() {
		err = valuesFile.Close()
		if err != nil {
			log.Logger().Warnf("Error closing %s because %v", valuesFile.Name(), err)
		}
		err = util.DeleteFile(valuesFile.Name())
		if err != nil {
			log.Logger().Warnf("Error deleting %s because %v", valuesFile.Name(), err)
		}
	}
	if err != nil {
		return "", func() {}, errors.Wrapf(err, "creating tempfile to write values for %s", chartName)
	}
	_, err = valuesFile.Write(valuesYaml)
	if err != nil {
		return "", func() {}, errors.Wrapf(err, "writing values to %s for %s", valuesFile.Name(), chartName)
	}
	return valuesFile.Name(), cleanup, nil
}

//GenerateQuestions asks questions based on the schema
func GenerateQuestions(schema []byte, batchMode bool, askExisting bool, basePath string, useVault bool,
	existing map[string]interface{}, in terminal.FileReader, out terminal.FileWriter, outErr io.Writer) ([]byte,
	[]*surveyutils.GeneratedSecret, error) {
	secrets := make([]*surveyutils.GeneratedSecret, 0)
	schemaOptions := surveyutils.JSONSchemaOptions{
		CreateSecret: func(name string, key string, value string, passthrough bool) (interface{},
			error) {
			secret := &surveyutils.GeneratedSecret{
				Name:  name,
				Key:   key,
				Value: value,
				Path:  strings.Join([]string{basePath, name}, "/"),
			}
			secrets = append(secrets, secret)
			if passthrough {
				if useVault {
					return vaulturl.ToURI(secret.Path, secret.Key), nil
				}
				return value, nil
			}
			return map[string]interface{}{
				"Name": name,
				"Kind": "Secret",
			}, nil
		},
		Out:                 out,
		In:                  in,
		OutErr:              outErr,
		IgnoreMissingValues: false,
		NoAsk:               batchMode,
		AutoAcceptDefaults:  batchMode,
		AskExisting:         askExisting,
	}
	// For adding an app there are by definition no existing values,
	// and whether we auto-accept defaults is determined by batch mode
	values, err := schemaOptions.GenerateValues(schema, existing)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	return values, secrets, nil
}

func addApp(create bool, jxClient versioned.Interface, app *jenkinsv1.App) error {
	if create {
		_, err := jxClient.JenkinsV1().Apps(app.Namespace).Create(app)
		if err != nil {
			return errors.Wrapf(err, "creating App %s to annotate with values.yaml", app.Name)
		}
	} else {
		_, err := jxClient.JenkinsV1().Apps(app.Namespace).PatchUpdate(app)
		if err != nil {
			return errors.Wrapf(err, "updating App %s to annotate with values.yaml", app.Name)
		}
	}
	return nil
}
