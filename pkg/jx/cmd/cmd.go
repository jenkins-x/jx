package cmd

import (
	"io"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"

	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/version"
	"github.com/spf13/cobra"
)

const (
	//     * runs (aka 'run')

	valid_resources = `Valid resource types include:

    * environments (aka 'env')
    * pipelines (aka 'pipe')
    * urls (aka 'url')
    `
)

// NewJXCommand creates the `jx` command and its nested children.
func NewJXCommand(f cmdutil.Factory, in io.Reader, out, err io.Writer) *cobra.Command {
	cmds := &cobra.Command{
		Use:   "jx",
		Short: "jx is a command line tool for working with Jenkins X",
		Long: `
 `,
		Run: runHelp,
		/*
			BashCompletionFunction: bash_completion_func,
		*/
	}

	createCommands := NewCmdCreate(f, out, err)
	deleteCommands := NewCmdDelete(f, out, err)
	getCommands := NewCmdGet(f, out, err)
	editCommands := NewCmdEdit(f, out, err)
	updateCommands := NewCmdUpdate(f, out, err)

	installCommands := []*cobra.Command{
		NewCmdInstall(f, out, err),
		NewCmdUninstall(f, out, err),
		NewCmdUpgrade(f, out, err),
	}
	installCommands = append(installCommands, findCommands("cluster", createCommands, deleteCommands)...)
	installCommands = append(installCommands, findCommands("cluster", updateCommands)...)
	installCommands = append(installCommands, findCommands("jenkins token", createCommands, deleteCommands)...)
	installCommands = append(installCommands, NewCmdInit(f, out, err))

	addProjectCommands := []*cobra.Command{
		NewCmdImport(f, out, err),
	}
	addProjectCommands = append(addProjectCommands, findCommands("create archetype", createCommands, deleteCommands)...)
	addProjectCommands = append(addProjectCommands, findCommands("create spring", createCommands, deleteCommands)...)
	addProjectCommands = append(addProjectCommands, findCommands("create lile", createCommands, deleteCommands)...)
	addProjectCommands = append(addProjectCommands, findCommands("create micro", createCommands, deleteCommands)...)
	addProjectCommands = append(addProjectCommands, findCommands("create quickstart", createCommands, deleteCommands)...)

	gitCommands := []*cobra.Command{}
	gitCommands = append(gitCommands, findCommands("git server", createCommands, deleteCommands)...)
	gitCommands = append(gitCommands, findCommands("git token", createCommands, deleteCommands)...)
	gitCommands = append(gitCommands, NewCmdRepo(f, out, err))

	addonCommands := []*cobra.Command{}
	addonCommands = append(addonCommands, findCommands("addon", createCommands, deleteCommands)...)

	environmentsCommands := []*cobra.Command{
		NewCmdPreview(f, out, err),
		NewCmdPromote(f, out, err),
	}
	environmentsCommands = append(environmentsCommands, findCommands("environment", createCommands, deleteCommands, editCommands, getCommands)...)

	groups := templates.CommandGroups{
		{
			Message:  "Installing:",
			Commands: installCommands,
		},
		{
			Message:  "Adding Projects to Jenkins X:",
			Commands: addProjectCommands,
		},
		{
			Message:  "Addons:",
			Commands: addonCommands,
		},
		{
			Message:  "Git:",
			Commands: gitCommands,
		},
		{
			Message: "Working with Kubernetes:",
			Commands: []*cobra.Command{
				NewCompliance(f, out, err),
				NewCmdCompletion(f, out),
				NewCmdContext(f, out, err),
				NewCmdEnvironment(f, out, err),
				NewCmdTeam(f, out, err),
				NewCmdGC(f, out, err),
				NewCmdNamespace(f, out, err),
				NewCmdPrompt(f, out, err),
				NewCmdShell(f, out, err),
				NewCmdStatus(f, out, err),
			},
		},
		{
			Message: "Working with Applications:",
			Commands: []*cobra.Command{
				NewCmdConsole(f, out, err),
				NewCmdCloudBees(f, out, err),
				NewCmdLogs(f, out, err),
				NewCmdOpen(f, out, err),
				NewCmdRsh(f, out, err),
				NewCmdSync(f, out, err),
			},
		},
		{
			Message:  "Working with Environments:",
			Commands: environmentsCommands,
		},
		{
			Message: "Working with Jenkins X resources:",
			Commands: []*cobra.Command{
				getCommands,
				editCommands,
				createCommands,
				updateCommands,
				deleteCommands,
				NewCmdStart(f, out, err),
				NewCmdStop(f, out, err),
			},
		},
		{
			Message: "Jenkins X Pipeline Commands:",
			Commands: []*cobra.Command{
				NewCmdStep(f, out, err),
			},
		},
	}

	groups.Add(cmds)

	cmds.AddCommand(NewCmdVersion(f, out, err))
	cmds.Version = version.GetVersion()
	cmds.SetVersionTemplate("{{printf .Version}}\n")

	filters := []string{"options"}

	templates.ActsAsRootCommand(cmds, filters, groups...)

	return cmds
}

func findCommands(subCommand string, commands ...*cobra.Command) []*cobra.Command {
	answer := []*cobra.Command{}
	for _, parent := range commands {
		for _, c := range parent.Commands() {
			if commandHasParentName(c, subCommand) {
				answer = append(answer, c)
			} else {
				childCommands := findCommands(subCommand, c)
				if len(childCommands) > 0 {
					answer = append(answer, childCommands...)
				}
			}
		}
	}
	return answer
}

func commandHasParentName(command *cobra.Command, name string) bool {
	path := fullPath(command)
	return strings.Contains(path, name)
}

func fullPath(command *cobra.Command) string {
	name := command.Name()
	parent := command.Parent()
	if parent != nil {
		return fullPath(parent) + " " + name
	}
	return name
}

func runHelp(cmd *cobra.Command, args []string) {
	cmd.Help()
}
