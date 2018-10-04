package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	"github.com/stoewer/go-strcase"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// CreateExtensionsRepositoryOptions the flags for running create cluster
type UpdateExtensionsRepositoryOptions struct {
	UpdateExtensionsOptions
	Flags      InitFlags
	InputFile  string
	OutputFile string
}

type UpdateExtensionsRepositoryFlags struct {
}

type ExtensionsRepositoryLock struct {
	Version    int                       `json:"version"`
	Extensions []ExtensionRepositoryLock `json:"extensions"`
}

type ExtensionRepositoryLock struct {
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace"`
	UUID        string   `json:"uuid"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	When        []string `json:"when,omitempty"`
	Given       string   `json:"given,omitempty"`
	Type        string   `json:"type,omitempty"`
	Script      string   `json:"script"`
	Parameters  []struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		DefaultValue string `json:"defaultValue"`
	} `json:"parameters,omitempty"`
}

type ExtensionsRepositoryConstraints struct {
	Extensions []struct {
		Extension string `json:"extension"`
		Version   string `json:"version"`
	} `json:"extensions"`
}

type ExtensionDefinitions struct {
	Version    string `json:"version,omitempty"`
	Extensions []struct {
		Name        string   `json:"name"`
		Namespace   string   `json:"namespace"`
		UUID        string   `json:"uuid"`
		Description string   `json:"description"`
		When        []string `json:"when,omitempty"`
		Given       string   `json:"given,omitempty"`
		Type        string   `json:"type,omitempty"`
		ScriptFile  string   `json:"scriptFile,omitempty"`
		Parameters  []struct {
			Name         string `json:"name"`
			Description  string `json:"description"`
			DefaultValue string `json:"defaultValue"`
		} `json:"parameters"`
	} `json:"extensions"`
}

var (
	updateExtensionsRepositoryLong = templates.LongDesc(`
		This command updates the jenkins-x-extensions-repository.lock.yaml file from a jenkins-x-extensions-repository.yaml file

`)

	updateExtensionsRepositoryExample = templates.Examples(`
		
        # Updates a file called jenkins-x-extensions-repository.lock.yaml from a file called  jenkins-x-extensions-repository.yaml in the same directory
        jx update extensions repository
      
        # Allows the input and output file to specified
		jx update extensions repository -i my-repo.yaml -o my-repo.lock.yaml

`)
)

// NewCmdGet creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a Kubernetes cluster.
func NewCmdUpdateExtensionsRepository(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &UpdateExtensionsRepositoryOptions{
		UpdateExtensionsOptions: UpdateExtensionsOptions{
			UpdateOptions: UpdateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					In:      in,

					Out: out,
					Err: errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "repository",
		Short:   "Updates an extension repository",
		Long:    fmt.Sprintf(updateExtensionsRepositoryLong),
		Example: updateExtensionsRepositoryExample,
		Run: func(cmd2 *cobra.Command, args []string) {
			options.Cmd = cmd2
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.InputFile, "input-file", "i", "jenkins-x-extensions-repository.yaml", "The input file to read to generate the .lock file")
	cmd.Flags().StringVarP(&options.OutputFile, "output-file", "o", "jenkins-x-extensions-repository.lock.yaml", "The output .lock file")
	return cmd
}

func (o *UpdateExtensionsRepositoryOptions) Run() error {
	constraints := ExtensionsRepositoryConstraints{}
	err := constraints.LoadFromFile(o.InputFile)
	if err != nil {
		return err
	}
	oldLock := ExtensionsRepositoryLock{}
	err = oldLock.LoadFromFile(o.OutputFile)
	if err != nil {
		return err
	}
	oldVersion := oldLock.Version
	oldLockNameMap := make(map[string]ExtensionRepositoryLock, 0)
	newLock := ExtensionsRepositoryLock{
		Extensions: make([]ExtensionRepositoryLock, 0),
	}
	newLock.Version = oldVersion + 1
	for _, c := range constraints.Extensions {
		if strings.HasPrefix(c.Extension, "github.com") {
			s := strings.Split(c.Extension, "/")
			if len(s) != 3 {
				errors.New(fmt.Sprintf("Cannot parse extension path %s", util.ColorInfo(c.Extension)))
			}
			org := s[1]
			repo := s[2]
			version := c.Version
			if version == "latest" {
				version, err = util.GetLatestVersionStringFromGitHub(org, repo)
				if err != nil {
					return err
				}
			}
			definitionsUrl := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/v%s", org, repo, version)
			definitionsFileUrl := fmt.Sprintf("%s/jenkins-x-extension-definitions.yaml", definitionsUrl)
			extensionDefinitions := ExtensionDefinitions{}
			extensionDefinitions.LoadFromURL(definitionsFileUrl, c.Extension, c.Version)
			for _, ed := range extensionDefinitions.Extensions {
				// It's best practice to assign a UUID to an extension, but if it doesn't have one we
				// try to give it the one it had last
				UUID := ed.UUID
				if UUID == "" {
					// Lazy initialize the oldLockNameMap
					if len(oldLockNameMap) == 0 {
						for _, l := range oldLock.Extensions {
							oldLockNameMap[fmt.Sprintf("%s.%s", l.Namespace, l.Name)] = l
						}
					}
					UUID = oldLockNameMap[fmt.Sprintf("%s.%s", ed.Namespace, ed.Name)].UUID
				}
				// If the UUID is still empty, generate one
				if UUID == "" {
					UUID = uuid.New()
				}
				scriptFile := ed.ScriptFile
				if scriptFile == "" {
					scriptFile = fmt.Sprintf("%s.sh", strings.ToLower(strcase.SnakeCase(ed.Name)))
				}
				script, err := o.LoadAsStringFromURL(fmt.Sprintf("%s/%s", definitionsUrl, scriptFile))
				if err != nil {
					return err
				}
				eLock := ExtensionRepositoryLock{
					Name:        ed.Name,
					Namespace:   ed.Namespace,
					Version:     version,
					UUID:        UUID,
					Description: ed.Description,
					Parameters:  ed.Parameters,
					When:        ed.When,
					Given:       ed.Given,
					Type:        ed.Type,
					Script:      script,
				}
				newLock.Extensions = append(newLock.Extensions, eLock)
			}
		} else {
			return errors.New(fmt.Sprintf("Only github.com is supported, use a format like %s", util.ColorInfo("github.com/jenkins-x/ext-jacoco")))
		}
	}
	bytes, err := yaml.Marshal(newLock)
	if err != nil {
		return err
	}
	log.Printf("Updating extensions repository from %s to %s. Changes are %s\n", util.ColorInfo(oldLock.Version), util.ColorInfo(newLock.Version), util.ColorInfo("Unknown"))
	err = ioutil.WriteFile(o.OutputFile, bytes, 0755)
	if err != nil {
		return err
	}
	return nil
}

func (o *UpdateExtensionsRepositoryOptions) LoadAsStringFromURL(url string) (result string, err error) {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(fmt.Sprintf("%s?version=%d", url, time.Now().UnixNano()/int64(time.Millisecond)))

	defer resp.Body.Close()
	if err != nil {
		return "", err
	}

	bytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (constraints *ExtensionsRepositoryConstraints) LoadFromFile(inputFile string) (err error) {
	path, err := filepath.Abs(inputFile)
	if err != nil {
		return err
	}
	y, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(y, constraints)
	if err != nil {
		return err
	}
	return nil
}

func (lock *ExtensionsRepositoryLock) LoadFromFile(inputFile string) (err error) {
	path, err := filepath.Abs(inputFile)
	if err != nil {
		return err
	}
	y, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(y, lock)
	if err != nil {
		return err
	}
	return nil
}

func (lock *ExtensionDefinitions) LoadFromURL(definitionsUrl string, extension string, version string) (err error) {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(fmt.Sprintf("%s?version=%d", definitionsUrl, time.Now().UnixNano()/int64(time.Millisecond)))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Printf("Unable to find Extension Definitions at %s for %s with version %s", util.ColorWarning(definitionsUrl), util.ColorWarning(extension), util.ColorWarning(version))
		return nil
	}
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	err = yaml.Unmarshal(bytes, lock)
	if err != nil {
		return err
	}
	return nil
}
