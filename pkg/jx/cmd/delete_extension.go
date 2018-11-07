package cmd

import (
	"fmt"
	"io"
	"strings"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/extensions"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

var (
	deleteExtension = templates.LongDesc(`
		Deletes one or more Extensions from Jenkins X

		Some extensions may have defined scripts to run when being uninstalled.

`)

	deleteExtensionExample = templates.Examples(`
		# prompt for the available extensions to delete
		jx delete extension

		# delete a specific extension
		jx delete extension jx.spotbugs-analyzer

		# delete a specific extensions
		jx delete extension jx.spotbugs-analyzer jx.jacoco-analyzer


		# delete all extension
		jx delete extension all
	`)
)

// DeleteExtensionOptions are the flags for delete commands
type DeleteExtensionOptions struct {
	CommonOptions
	All bool
}

// NewCmdDeleteExtension creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdDeleteExtension(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteExtensionOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "extension",
		Short:   "Deletes one or more extensions",
		Long:    deleteExtension,
		Example: deleteExtensionExample,
		Aliases: []string{"extensions", "ext"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
		SuggestFor: []string{"remove", "rm"},
	}
	cmd.Flags().BoolVarP(&options.All, "all", "", false, "Remove all extensions")
	cmd.Flags().BoolVarP(&options.BatchMode, optionBatchMode, "b", false, "Run in batch mode")
	return cmd
}

// Run implements this command
func (o *DeleteExtensionOptions) Run() error {
	args := o.Args
	if len(args) == 0 && !o.All {
		return o.Cmd.Help()
	}

	apisClient, err := o.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterExtensionCRD(apisClient)
	if err != nil {
		return err
	}

	// Let's register the release CRD as charts built using Jenkins X use it, and it's very likely that people installing
	// apps are using Helm
	err = kube.RegisterReleaseCRD(apisClient)
	if err != nil {
		return err
	}

	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return err
	}
	jxClient, ns, err := o.Factory.CreateJXClient()
	if err != nil {
		return err
	}
	extensionsClient := jxClient.JenkinsV1().Extensions(ns)
	exts, err := extensionsClient.List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	extensionsList, err := (&jenkinsv1.ExtensionConfigList{}).LoadFromConfigMap(extensions.ExtensionsConfigDefaultConfigMap, kubeClient, ns)
	if err != nil {
		return err
	}

	if len(exts.Items) == 0 {
		return fmt.Errorf("There are no Extensions installed for team %s. You install them using: %s\n", util.ColorInfo(ns), util.ColorInfo("jx upgrade extensions"))
	}

	names := make([]string, 0)
	lookup := make(map[string]jenkinsv1.ExtensionSpec)
	for _, e := range exts.Items {
		lookup[e.Spec.FullyQualifiedKebabName()] = e.Spec
		names = append(names, e.Spec.FullyQualifiedKebabName())
	}
	if o.All {
		args = names
	}
	if len(args) == 0 && !o.BatchMode {
		args, err = util.PickNames(names, "Pick Extension(s):", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
	}
	if len(args) == 0 {
		return fmt.Errorf("Specify the extensions to delete using %s or delete them all using %s.", util.ColorInfo("jx delete extensions <extension name>"), util.ColorInfo("jx delete extension --all"))
	}
	deleteExtensions := strings.Join(args, ", ")

	if !o.BatchMode && !util.Confirm(fmt.Sprintf("You are about to delete the Extensions: %s", deleteExtensions), false, "The list of Extensions to be deleted", o.In, o.Out, o.Err) {
		return nil
	}
	for _, name := range args {
		if util.StringArrayIndex(names, name) < 0 {
			return util.InvalidOption(optionLabel, name, names)
		}
		ext, _ := lookup[name]
		if ext.IsOnUninstall() {
			// Find the config
			var config jenkinsv1.ExtensionConfig
			for _, c := range extensionsList.Extensions {
				if c.FullyQualifiedName() == ext.FullyQualifiedName() {
					config = c
					break
				}
			}
			e, _, err := extensions.ToExecutable(&ext, config.Parameters, o.devNamespace, extensionsClient)
			if err != nil {
				return err
			}
			err = e.Execute(o.Verbose)
			if err != nil {
				return err
			}
		}
		err := extensionsClient.Delete(name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	log.Infof("Deleted Extensions %s\n", util.ColorInfo(deleteExtensions))

	return nil
}
