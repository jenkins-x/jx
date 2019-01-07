package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateAddonOptions the options for the create spring command
type CreateAddonOptions struct {
	CreateOptions

	Namespace   string
	Version     string
	ReleaseName string
	SetValues   string
	ValueFiles  []string
	HelmUpdate  bool
}

// NewCmdCreateAddon creates a command object for the "create" command
func NewCmdCreateAddon(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateAddonOptions{
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
		Use:     "addon",
		Short:   "Creates an addon",
		Aliases: []string{"scm"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateAddonAmbassador(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonAnchore(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonCloudBees(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonGitea(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonIstio(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonKnativeBuild(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonKubeless(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonOwasp(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonPipelineEvents(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonPrometheus(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonProw(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonSSO(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonVault(f, in, out, errOut))

	options.addFlags(cmd, kube.DefaultNamespace, "", "")
	return cmd
}

func (options *CreateAddonOptions) addFlags(cmd *cobra.Command, defaultNamespace string, defaultOptionRelease string, defaultVersion string) {
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", defaultNamespace, "The Namespace to install into")
	cmd.Flags().StringVarP(&options.ReleaseName, optionRelease, "r", defaultOptionRelease, "The chart release name")
	cmd.Flags().StringVarP(&options.SetValues, "set", "s", "", "The chart set values (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	cmd.Flags().StringArrayVarP(&options.ValueFiles, "values", "f", []string{}, "List of locations for values files, can be local files or URLs")
	cmd.Flags().BoolVarP(&options.HelmUpdate, "helm-update", "", true, "Should we run helm update first to ensure we use the latest version")
	cmd.Flags().StringVarP(&options.Version, "version", "v", defaultVersion, "The chart version to install)")
}

// Run implements this command
func (o *CreateAddonOptions) Run() error {
	args := o.Args
	if len(args) == 0 {
		return o.Cmd.Help()
	}

	for _, arg := range args {
		err := o.CreateAddon(arg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *CreateAddonOptions) CreateAddon(addon string) error {
	err := o.ensureHelm()
	if err != nil {
		return errors.Wrap(err, "failed to ensure that helm is present")
	}
	charts := kube.AddonCharts
	chart := charts[addon]
	if chart == "" {
		return util.InvalidArg(addon, util.SortedMapKeys(charts))
	}
	setValues := strings.Split(o.SetValues, ",")

	err = o.installChart(addon, chart, o.Version, o.Namespace, o.HelmUpdate, setValues, o.ValueFiles, "")
	if err != nil {
		return fmt.Errorf("Failed to install chart %s: %s", chart, err)
	}
	return o.ExposeAddon(addon)
}

func (o *CreateAddonOptions) ExposeAddon(addon string) error {
	service, ok := kube.AddonServices[addon]
	if !ok {
		return nil
	}

	client, err := o.KubeClient()
	if err != nil {
		return err
	}

	svc, err := client.CoreV1().Services(o.Namespace).Get(service, meta_v1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "getting the addon service: %s", service)
	}

	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}
	if svc.Annotations[kube.AnnotationExpose] == "" {
		svc.Annotations[kube.AnnotationExpose] = "true"
		svc, err = client.CoreV1().Services(o.Namespace).Update(svc)
		if err != nil {
			return errors.Wrap(err, "updating the service annotations")
		}
	}
	devNamespace, _, err := kube.GetDevNamespace(client, o.currentNamespace)
	if err != nil {
		return errors.Wrap(err, "retrieving the dev namespace")
	}
	return o.expose(devNamespace, o.Namespace, "")
}
