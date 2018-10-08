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
type UpgradeExtensionsRepositoryOptions struct {
	UpgradeExtensionsOptions
	Flags      InitFlags
	InputFile  string
	OutputFile string
}

type UpgradeExtensionsRepositoryFlags struct {
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
	Description string   `json:"description,omitempty"`
	When        []string `json:"when,omitempty"`
	Given       string   `json:"given,omitempty"`
	Type        string   `json:"type,omitempty"`
	Script      string   `json:"script,omitempty"`
	Children    []string `json:"children,omitempty"`
	Parameters  []struct {
		Name         string `json:"name"`
		Description  string `json:"description,omitempty"`
		DefaultValue string `json:"defaultValue,omitempty"`
	} `json:"parameters,omitempty"`
}

type ExtensionsRepositoryConstraints struct {
	Remotes []struct {
		Remote string `json:"remote"`
		Tag    string `json:"tag"`
	} `json:"remotes"`
}

type ExtensionDefinitions struct {
	Version    string                `json:"version,omitempty"`
	Extensions []ExtensionDefinition `json:"extensions"`
}

type ExtensionDefinition struct {
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace"`
	UUID        string   `json:"uuid"`
	Description string   `json:"description,omitempty"`
	When        []string `json:"when,omitempty"`
	Given       string   `json:"given,omitempty"`
	Type        string   `json:"type,omitempty"`
	Children    map[string]struct {
		UUID   string `json:"uuid,omitempty"`
		Remote string `json:"remote,omitempty"`
	} `json:"children,omitempty"`
	ScriptFile string `json:"scriptFile,omitempty"`
	Parameters []struct {
		Name         string `json:"name"`
		Description  string `json:"description,omitempty"`
		DefaultValue string `json:"defaultValue,omitempty"`
	} `json:"parameters,omitempty"`
}

type httpError struct {
	StatusCode int
	Status     string
}

func (e *httpError) Error() string {
	return e.Status
}

var (
	upgradeExtensionsRepositoryLong = templates.LongDesc(`
		This command upgrades the jenkins-x-extensions-repository.lock.yaml file from a jenkins-x-extensions-repository.yaml file

`)

	upgradeExtensionsRepositoryExample = templates.Examples(`
		
        # Updates a file called jenkins-x-extensions-repository.lock.yaml from a file called  jenkins-x-extensions-repository.yaml in the same directory
        jx upgrade extensions repository
      
        # Allows the input and output file to specified
		jx upgrade extensions repository -i my-repo.yaml -o my-repo.lock.yaml

`)
)

// NewCmdGet creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a Kubernetes cluster.
func NewCmdUpgradeExtensionsRepository(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &UpgradeExtensionsRepositoryOptions{
		UpgradeExtensionsOptions: UpgradeExtensionsOptions{
			CreateOptions: CreateOptions{
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
		Long:    fmt.Sprintf(upgradeExtensionsRepositoryLong),
		Example: upgradeExtensionsRepositoryExample,
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

func (o *UpgradeExtensionsRepositoryOptions) Run() error {
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

	lookupByName := make(map[string]ExtensionRepositoryLock, 0)
	lookupByUUID := make(map[string]ExtensionRepositoryLock, 0)
	for _, c := range constraints.Remotes {
		if strings.HasPrefix(c.Remote, "github.com") {
			s := strings.Split(c.Remote, "/")
			if len(s) != 3 {
				errors.New(fmt.Sprintf("Cannot parse extension path %s", util.ColorInfo(c.Remote)))
			}
			org := s[1]
			repo := s[2]
			tag := c.Tag
			if tag == "latest" {
				tag, err = util.GetLatestVersionStringFromGitHub(org, repo)
				if err != nil {
					return err
				}
				tag = fmt.Sprintf("v%s", tag)
			}
			definitionsUrl := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", org, repo, tag)
			definitionsFileUrl := fmt.Sprintf("%s/jenkins-x-extension-definitions.yaml", definitionsUrl)
			extensionDefinitions := ExtensionDefinitions{}
			extensionDefinitions.LoadFromURL(definitionsFileUrl, c.Remote, c.Tag)
			for _, ed := range extensionDefinitions.Extensions {
				// It's best practice to assign a UUID to an extension, but if it doesn't have one we
				// try to give it the one it had last
				UUID := ed.UUID
				if UUID == "" {
					// Lazy initialize the oldLockNameMap
					if len(oldLockNameMap) == 0 {
						for _, l := range oldLock.Extensions {
							oldLockNameMap[l.FullyQualifiedName()] = l
						}
					}
					UUID = oldLockNameMap[ed.FullyQualifiedName()].UUID
				}
				// If the UUID is still empty, generate one
				if UUID == "" {
					UUID = uuid.New()
					log.Printf("No UUID found for %s. Generated UUID %s, please update your extension definition "+
						"accordingly.", ed.FullyQualifiedName(), UUID)
				}
				var script string
				children := make([]string, 0)
				// If the children is present, there is no script
				if len(ed.Children) == 0 {
					scriptFile := ed.ScriptFile
					if scriptFile == "" {
						scriptFile = fmt.Sprintf("%s.sh", strings.ToLower(strcase.SnakeCase(ed.Name)))
					}
					script, err = o.LoadAsStringFromURL(fmt.Sprintf("%s/%s", definitionsUrl, scriptFile))
					if err != nil {
						return err
					}
				} else {
					for fqn, c := range ed.Children {
						if c.UUID != "" {
							children = append(children, c.UUID)
						} else {
							children = append(children, fqn)
						}
					}
				}
				eLock := ExtensionRepositoryLock{
					Name:        ed.Name,
					Namespace:   ed.Namespace,
					Version:     tag,
					UUID:        UUID,
					Description: ed.Description,
					Parameters:  ed.Parameters,
					When:        ed.When,
					Given:       ed.Given,
					Type:        ed.Type,
					Script:      script,
					Children:    children,
				}
				lookupByName[eLock.FullyQualifiedName()] = eLock
				lookupByUUID[eLock.UUID] = eLock
				newLock.Extensions = append(newLock.Extensions, eLock)
			}
		} else {
			return errors.New(fmt.Sprintf("Only github.com is supported, use a format like %s", util.ColorInfo("github.com/jenkins-x/ext-jacoco")))
		}
	}
	uuidResolveErrors := make([]string, 0)
	// Second pass over extensions to allow us to do things like resolve fqns into UUIDs
	for i, lock := range newLock.Extensions {
		newLock.Extensions[i].Children = o.recursivelyFixChildren(lock, lookupByName, lookupByUUID, &uuidResolveErrors)
	}
	if len(uuidResolveErrors) > 0 {
		bytes, err := yaml.Marshal(newLock)
		if err != nil {
			return err
		}
		errFile, err := ioutil.TempFile("", o.OutputFile)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(errFile.Name(), bytes, 0755)
		if err != nil {
			return err
		}
		return errors.New(fmt.Sprintf("Cannot resolve children %s in repository. Partial .lock file written to %s.", uuidResolveErrors, errFile.Name()))
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

func (o *UpgradeExtensionsRepositoryOptions) recursivelyFixChildren(lock ExtensionRepositoryLock, lookupByName map[string]ExtensionRepositoryLock, lookupByUUID map[string]ExtensionRepositoryLock, resolveErrors *[]string) (children []string) {
	children = make([]string, 0)
	for _, u := range lock.Children {
		if uuid.Parse(u) == nil {
			if c, ok := lookupByName[u]; ok {
				log.Printf("We recommend you explicitly specify the UUID for child %s on extension %s as this will stop the "+
					"extension breaking if names are changed.\n"+
					"If you are the maintainer of the extension definition add \n"+
					"\n"+
					"      UUID: %s\n"+
					"\n"+
					"to the child definition. \n"+
					"If you aren't the maintainer of the extension definition we recommend you contact them and ask"+
					"them to add the UUID.\n"+
					"\n"+
					"This does not stop you using the extension, as we have been able to discover and attach the "+
					"UUID to the child based on the fully qualified name.\n", util.ColorWarning(u), util.ColorWarning(lock.FullyQualifiedName()), util.ColorWarning(c.UUID))
				u = c.UUID
			} else {
				// Record the error in the loop, but don't error until end. This allows us to report all the errors
				// up front
				*resolveErrors = append(*resolveErrors, u)
			}
		}
		if c, ok := lookupByUUID[u]; ok {
			if len(c.Children) == 0 {
				// Leaf, so add
				children = append(children, u)
			} else {
				// Now we need to recursively resolve any children
				children = append(children, o.recursivelyFixChildren(c, lookupByName, lookupByUUID, resolveErrors)...)
			}
		}
	}
	return children
}

func (o *UpgradeExtensionsRepositoryOptions) LoadAsStringFromURL(url string) (result string, err error) {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(fmt.Sprintf("%s?version=%d", url, time.Now().UnixNano()/int64(time.Millisecond)))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &httpError{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
		}
	}

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
		log.Printf("Unable to find Extension Definitions at %s for %s with version %s\n", util.ColorWarning(definitionsUrl), util.ColorWarning(extension), util.ColorWarning(version))
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

func (e *ExtensionRepositoryLock) FullyQualifiedName() (fqn string) {
	return fmt.Sprintf("%s.%s", e.Namespace, e.Name)
}

func (e *ExtensionDefinition) FullyQualifiedName() (fqn string) {
	return fmt.Sprintf("%s.%s", e.Namespace, e.Name)
}
