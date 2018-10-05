package cmd

import (
	"fmt"
	"github.com/stoewer/go-strcase"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/blang/semver"
	"github.com/jenkins-x/jx/pkg/kube"

	"github.com/jenkins-x/jx/pkg/util"

	"github.com/jenkins-x/jx/pkg/log"

	typev1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/ghodss/yaml"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

const upstreamExtensionsRepositoryUrl = "https://raw.githubusercontent.com/jenkins-x/jenkins-x-extensions/master/jenkins-x-extensions-repository.yaml"

var (
	upgradeExtensionsLong = templates.LongDesc(`
		Upgrades the Jenkins X extensions available to this Jenkins X install if there are new versions available
`)

	upgradeExtensionsExample = templates.Examples(`
		
		# upgrade extensions
		jx upgrade extensions 

        # upgrade a particular extension
		jx upgrade extensions -f hello-world
	`)
)

type UpgradeExtensionsOptions struct {
	CreateOptions
	Filter               string
	ExtensionsRepository string
}

type ExtensionsRepository struct {
	Version    string                       `json:"version,omitempty"`
	Extensions []jenkinsv1.ExtensionDetails `json:"extensions,omitempty"`
}

func NewCmdUpgradeExtensions(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &UpgradeExtensionsOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "extensions",
		Short:   "Upgrades the Jenkins X extensions available to this Jenkins X install if there are new versions available",
		Long:    upgradeExtensionsLong,
		Example: upgradeBInariesExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.Verbose, "verbose", "", false, "Enable verbose logging")
	cmd.Flags().StringVarP(&options.Filter, "filter", "f", "", "Filter the extensions to upgrade")
	cmd.Flags().StringVarP(&options.ExtensionsRepository, "extensions-repository", "", upstreamExtensionsRepositoryUrl, "Specify the extensions repository yaml file to read from")
	return cmd
}

func (o *UpgradeExtensionsOptions) Run() error {

	apisClient, err := o.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	kube.RegisterExtensionCRD(apisClient)
	extensionsRepository := ExtensionsRepository{}
	var bytes []byte

	if err != nil {
		return err
	}

	if strings.HasPrefix(o.ExtensionsRepository, "http://") || strings.HasPrefix(o.ExtensionsRepository, "https://") {
		httpClient := &http.Client{Timeout: 10 * time.Second}
		resp, err := httpClient.Get(fmt.Sprintf("%s?version=%d", o.ExtensionsRepository, time.Now().UnixNano()/int64(time.Millisecond)))

		defer resp.Body.Close()

		bytes, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
	} else {
		path := o.ExtensionsRepository
		// if it starts with a ~ it's the users homedir
		if strings.HasPrefix(path, "~") {
			usr, err := user.Current()
			if err == nil {
				path = filepath.Join(usr.HomeDir, strings.TrimPrefix(path, "~"))
			}
		}
		// Perhaps it's an absolute file path
		bytes, err = ioutil.ReadFile(path)
		if err != nil {
			// Perhaps it's a relative path
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			bytes, err = ioutil.ReadFile(filepath.Join(cwd, path))
			if err != nil {
				return errors.New(fmt.Sprintf("Unable to open Extensions Repository at %s", path))
			}
		}
	}

	err = yaml.Unmarshal(bytes, &extensionsRepository)
	if err != nil {
		return err
	}
	log.Infof("Current Extension Repository version %s\n", util.ColorInfo(extensionsRepository.Version))
	client, ns, err := o.Factory.CreateJXClient()
	if err != nil {
		return err
	}
	extensionsClient := client.JenkinsV1().Extensions(ns)
	kubeClient, curNs, err := o.KubeClient()
	if err != nil {
		return err
	}
	extensionsConfig, err := (&kube.ExtensionsConfig{}).LoadFromConfigMap(kubeClient, curNs)
	if err != nil {
		return err
	}
	installedExtensions, err := o.GetInstalledExtensions(extensionsClient)
	if err != nil {
		return err
	}
	for _, e := range extensionsRepository.Extensions {
		_, _, err = o.UpsertExtension(e, extensionsClient, installedExtensions, extensionsConfig.Extensions[e.Name])
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *UpgradeExtensionsOptions) UpsertExtension(extension jenkinsv1.ExtensionDetails, extensions typev1.ExtensionInterface, installedExtensions map[string]jenkinsv1.Extension, extensionConfig kube.ExtensionConfig) (*jenkinsv1.Extension, bool, error) {
	// TODO Validate extension
	newVersion, err := semver.Parse(extension.Version)
	if err != nil {
		return nil, false, err
	}
	existing, ok := installedExtensions[extension.UUID]
	if !ok {
		// Doesn't exist
		res, err := extensions.Create(&jenkinsv1.Extension{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s.%s", strcase.KebabCase(extension.Namespace), strcase.KebabCase(extension.Name)),
			},
			Spec: extension,
		})
		log.Infof("Adding Extension %s version %s\n", util.ColorInfo(fmt.Sprintf("%s.%s", extension.Namespace, extension.Name)), util.ColorInfo(newVersion))
		if err != nil {
			return res, true, err
		}
		if o.Contains(extension.When, jenkinsv1.ExtensionWhenInstall) {
			o.UpstallExtension(extension, extensionConfig)
		}
		return res, true, err
	}
	// TODO Handle uninstalling existing extension if name has changed but UUID hasn't
	existingVersion, err := semver.Parse(existing.Spec.Version)
	if existingVersion.LT(newVersion) {
		existing.Spec = extension
		res, err := extensions.Update(&existing)
		if o.Contains(extension.When, jenkinsv1.ExtensionWhenUpgrade) {
			o.UpstallExtension(extension, extensionConfig)
		}
		log.Infof("Upgrading Extension %s from %s to %s\n", util.ColorInfo(fmt.Sprintf("%s.%s", extension.Namespace, extension.Name)), util.ColorInfo(existingVersion), util.ColorInfo(newVersion))
		return res, false, err
	} else {
		return &existing, false, nil
	}
}

func (o *UpgradeExtensionsOptions) UpstallExtension(e jenkinsv1.ExtensionDetails, extensionConfig kube.ExtensionConfig) (err error) {
	ext, envVarsFormatted, err := e.ToExecutable(extensionConfig.Parameters)
	if err != nil {
		return err
	}
	log.Infof("Installing Extension %s version %s to pipeline with environment variables [ %s ]\n", util.ColorInfo(fmt.Sprintf("%s:%s", e.Namespace, e.Name)), util.ColorInfo(e.Version), util.ColorInfo(envVarsFormatted))
	return ext.Execute(o.Verbose)
}

func (o *UpgradeExtensionsOptions) Contains(whens []jenkinsv1.ExtensionWhen, when jenkinsv1.ExtensionWhen) bool {
	for _, w := range whens {
		if when == w {
			return true
		}
	}
	return false
}

func (o *UpgradeExtensionsOptions) GetInstalledExtensions(extensions typev1.ExtensionInterface) (installedExtensions map[string]jenkinsv1.Extension, err error) {
	exts, err := extensions.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	installedExtensions = make(map[string]jenkinsv1.Extension)
	for _, ext := range exts.Items {
		if ext.Spec.UUID == "" {
			return nil, errors.New(fmt.Sprintf("Extension %s does not have a UUID", util.ColorInfo(fmt.Sprintf("%s:%s", ext.Namespace, ext.Name))))
		}
		installedExtensions[ext.Spec.UUID] = ext
	}
	return installedExtensions, nil
}
