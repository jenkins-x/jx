package cmd

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

var (
	editDeployKindLong = templates.LongDesc(`
		Edits the deploy kind to use for your team
`)

	editDeployKindExample = templates.Examples(`
		# Edit the deploy kind and prompts you to pick one of the available kinds
		jx edit deploy

        # to switch to use Knative Serve deployments
		jx edit deploy knative

        # to switch to normal kubernetes deployments
		jx edit deploy default
	`)

	deployKinds = []string{"knative", "default"}
)

// EditDeployKindOptions the options for the create spring command
type EditDeployKindOptions struct {
	EditOptions

	Kind string
}

// NewCmdEditDeployKind creates a command object for the "create" command
func NewCmdEditDeployKind(commonOpts *CommonOptions) *cobra.Command {
	options := &EditDeployKindOptions{
		EditOptions: EditOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "deploy",
		Short:   "Edits the deploy kind to use for your team",
		Long:    editDeployKindLong,
		Example: editDeployKindExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	return cmd
}

// Run implements the command
func (o *EditDeployKindOptions) Run() error {
	args := o.Args
	name := ""
	if len(args) == 0 {
		if o.BatchMode {
			return util.MissingArgument("kind")
		}
		settings, err := o.TeamSettings()
		if err != nil {
			return err
		}
		defaultName := settings.DeployKind
		if defaultName == "" || util.StringArrayIndex(deployKinds, defaultName) < 0 {
			defaultName = "default"
		}
		name, err = util.PickNameWithDefault(deployKinds, "Pick the deployment kind: ", defaultName, "lets you switch between knative serve based deployments and default kubernetes deployments", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
		if name == "" {
			return fmt.Errorf("no kind chosen")
		}
	} else {
		name = args[0]
	}

	callback := func(env *v1.Environment) error {
		teamSettings := &env.Spec.TeamSettings
		teamSettings.DeployKind = name

		log.Infof("Setting the team deploy kind to: %s\n", util.ColorInfo(name))
		return nil
	}
	return o.ModifyDevEnvironment(callback)
}
