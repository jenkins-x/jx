package edit

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

const (
	// DeployKindKnative for knative serve based deployments
	DeployKindKnative = "knative"

	// DeployKindDefault for default kubernetes Deployment + Service deployment kinds
	DeployKindDefault = "default"
)

var (
	editDeployKindLong = templates.LongDesc(`
		Edits the deploy kind to use for your project or team
`)

	editDeployKindExample = templates.Examples(`
		# Edit the deploy kind for your current project and prompts you to pick one of the available kinds
		jx edit deploy

        # to switch to use Knative Serve deployments
		jx edit deploy knative

        # to switch to normal kubernetes deployments
		jx edit deploy default

		# Edit the default deploy kind for your team
		jx edit deploy --team

		# Set the default for your team to use knative
		jx edit deploy --team knative
	`)

	deployKinds = []string{DeployKindKnative, DeployKindDefault}

	knativeDeployKey = "knativeDeploy:"
)

// EditDeployKindOptions the options for the create spring command
type EditDeployKindOptions struct {
	EditOptions

	Kind string
	Dir  string
	Team bool
}

// NewCmdEditDeployKind creates a command object for the "create" command
func NewCmdEditDeployKind(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &EditDeployKindOptions{
		EditOptions: EditOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "deploy",
		Short:   "Edits the deploy kind to use for your project or team",
		Long:    editDeployKindLong,
		Example: editDeployKindExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.Team, "team", "t", false, "Edits the team default")
	cmd.Flags().StringVarP(&options.Kind, "kind", "k", "", fmt.Sprintf("The kind to use which should be one of: %s", strings.Join(deployKinds, ", ")))

	return cmd
}

// Run implements the command
func (o *EditDeployKindOptions) Run() error {
	settings, err := o.TeamSettings()
	if err != nil {
		return err
	}
	if o.Team {
		name, err := o.pickDeployKind(settings.DeployKind)
		if err != nil {
			return err
		}
		callback := func(env *v1.Environment) error {
			teamSettings := &env.Spec.TeamSettings
			teamSettings.DeployKind = name

			log.Logger().Infof("Setting the team deploy kind to: %s", util.ColorInfo(name))
			return nil
		}
		return o.ModifyDevEnvironment(callback)
	}

	fn := func(text string) (string, error) {
		defaultName := o.findDefaultDeployKindInValuesYaml(text)
		name := ""

		name, err := o.pickDeployKind(defaultName)
		if err != nil {
			return name, err
		}
		return o.setDeployKindInValuesYaml(text, name)
	}
	return o.ModifyHelmValuesFile(o.Dir, fn)
}

func (o *EditDeployKindOptions) findDefaultDeployKindInValuesYaml(yamlText string) string {
	// lets try find the current setting
	knativeFlag := ""
	lines := strings.Split(yamlText, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, knativeDeployKey) {
			knativeFlag = strings.TrimSpace(line[len(knativeDeployKey):])
			break
		}
	}
	if knativeFlag == "true" {
		return DeployKindKnative
	}
	return DeployKindDefault
}

// setDeployKindInValuesYaml sets the `knativeDeployKey` key to true or false based on the deployment kind
func (o *EditDeployKindOptions) setDeployKindInValuesYaml(yamlText string, deployKind string) (string, error) {
	var buffer strings.Builder

	lines := strings.Split(yamlText, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, knativeDeployKey) {
			buffer.WriteString(knativeDeployKey)
			buffer.WriteString(" ")
			if deployKind == DeployKindKnative {
				buffer.WriteString("true")
			} else {
				buffer.WriteString("false")
			}
		} else {
			buffer.WriteString(line)
		}
		buffer.WriteString("\n")
	}
	return buffer.String(), nil
}

func (o *EditDeployKindOptions) pickDeployKind(defaultName string) (string, error) {
	if o.Kind != "" {
		return o.Kind, nil
	}
	args := o.Args
	if len(args) > 0 {
		return args[0], nil
	}
	if o.BatchMode {
		return "", util.MissingArgument("kind")
	}
	if defaultName == "" || util.StringArrayIndex(deployKinds, defaultName) < 0 {
		defaultName = "default"
	}
	name, err := util.PickNameWithDefault(deployKinds, "Pick the deployment kind: ", defaultName, "lets you switch between knative serve based deployments and default kubernetes deployments", o.In, o.Out, o.Err)
	if err != nil {
		return name, err
	}
	if name == "" {
		return name, fmt.Errorf("no kind chosen")
	}
	return name, nil
}
