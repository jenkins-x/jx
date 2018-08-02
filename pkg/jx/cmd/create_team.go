package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	createTeamLong = templates.LongDesc(`
		Creates an issue in a the git project of the current directory
`)

	createTeamExample = templates.Examples(`
		# Create an issue in the current project
		jx create issue -t "something we should do"


		# Create an issue with a title and a body
		jx create issue -t "something we should do" --body "	
		some more
		text
		goes
		here
		""
"
	`)
)

// CreateTeamOptions the options for the create spring command
type CreateTeamOptions struct {
	CreateOptions

	Name    string
	Members []string
}

// NewCmdCreateTeam creates a command object for the "create" command
func NewCmdCreateTeam(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateTeamOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "team",
		Short:   "Create a new Team which is then provisioned by the team controller",
		Aliases: []string{"env"},
		Long:    createTeamLong,
		Example: createTeamExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Name, optionName, "n", "", "The name of the new Team. Should be all lower case and no special characters other than '-'")
	cmd.Flags().StringArrayVarP(&options.Members, "member", "m", []string{}, "The usernames of the members to add to the Team")

	options.addCommonFlags(cmd)
	return cmd
}

// Run implements the command
func (o *CreateTeamOptions) Run() error {
	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return err
	}
	apisClient, err := o.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterTeamCRD(apisClient)
	if err != nil {
		return err
	}

	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	ns, err := kube.GetAdminNamespace(kubeClient, devNs)
	if err != nil {
		return err
	}

	_, names, err := kube.GetPendingTeams(jxClient, ns)
	if err != nil {
		return err
	}

	name := o.Name
	if name == "" {
		args := o.Args
		if len(args) > 0 {
			name = args[0]
		}
	}
	if name == "" {
		return util.MissingOption(optionName)
	}

	if util.StringArrayIndex(names, name) >= 0 {
		return fmt.Errorf("The Team %s already exists!", name)
	}

	// TODO configure?
	kind := v1.TeamKindTypeCD

	team := &v1.Team{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: v1.TeamSpec{
			Label:   strings.Title(name),
			Members: o.Members,
			Kind:    kind,
		},
	}
	_, err = jxClient.JenkinsV1().Teams(ns).Create(team)
	if err != nil {
		return fmt.Errorf("Failed to create Team %s: %s", name, err)
	}
	log.Infof("Created Team: %s\n", util.ColorInfo(name))
	return nil
}
