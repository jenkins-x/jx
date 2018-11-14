package cmd

import (
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/version"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
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
func NewJXCommand(f Factory, in terminal.FileReader, out terminal.FileWriter, err io.Writer) *cobra.Command {
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

	createCommands := NewCmdCreate(f, in, out, err)
	deleteCommands := NewCmdDelete(f, in, out, err)
	getCommands := NewCmdGet(f, in, out, err)
	editCommands := NewCmdEdit(f, in, out, err)
	updateCommands := NewCmdUpdate(f, in, out, err)

	installCommands := []*cobra.Command{
		NewCmdInstall(f, in, out, err),
		NewCmdUninstall(f, in, out, err),
		NewCmdUpgrade(f, in, out, err),
	}
	installCommands = append(installCommands, findCommands("cluster", createCommands, deleteCommands)...)
	installCommands = append(installCommands, findCommands("cluster", updateCommands)...)
	installCommands = append(installCommands, findCommands("jenkins token", createCommands, deleteCommands)...)
	installCommands = append(installCommands, NewCmdInit(f, in, out, err))

	addProjectCommands := []*cobra.Command{
		NewCmdImport(f, in, out, err),
	}
	addProjectCommands = append(addProjectCommands, findCommands("create archetype", createCommands, deleteCommands)...)
	addProjectCommands = append(addProjectCommands, findCommands("create spring", createCommands, deleteCommands)...)
	addProjectCommands = append(addProjectCommands, findCommands("create lile", createCommands, deleteCommands)...)
	addProjectCommands = append(addProjectCommands, findCommands("create micro", createCommands, deleteCommands)...)
	addProjectCommands = append(addProjectCommands, findCommands("create quickstart", createCommands, deleteCommands)...)

	gitCommands := []*cobra.Command{}
	gitCommands = append(gitCommands, findCommands("git server", createCommands, deleteCommands)...)
	gitCommands = append(gitCommands, findCommands("git token", createCommands, deleteCommands)...)
	gitCommands = append(gitCommands, NewCmdRepo(f, in, out, err))

	addonCommands := []*cobra.Command{}
	addonCommands = append(addonCommands, findCommands("addon", createCommands, deleteCommands)...)

	environmentsCommands := []*cobra.Command{
		NewCmdPreview(f, in, out, err),
		NewCmdPromote(f, in, out, err),
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
				NewCompliance(f, in, out, err),
				NewCmdCompletion(f, in, out, err),
				NewCmdContext(f, in, out, err),
				NewCmdEnvironment(f, in, out, err),
				NewCmdTeam(f, in, out, err),
				NewCmdNamespace(f, in, out, err),
				NewCmdPrompt(f, in, out, err),
				NewCmdScan(f, in, out, err),
				NewCmdShell(f, in, out, err),
				NewCmdStatus(f, in, out, err),
			},
		},
		{
			Message: "Working with Applications:",
			Commands: []*cobra.Command{
				NewCmdConsole(f, in, out, err),
				NewCmdLogs(f, in, out, err),
				NewCmdOpen(f, in, out, err),
				NewCmdRsh(f, in, out, err),
				NewCmdSync(f, in, out, err),
			},
		},
		{
			Message: "Working with CloudBees application:",
			Commands: []*cobra.Command{
				NewCmdCloudBees(f, in, out, err),
				NewCmdLogin(f, in, out, err),
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
				NewCmdStart(f, in, out, err),
				NewCmdStop(f, in, out, err),
			},
		},
		{
			Message: "Jenkins X Pipeline Commands:",
			Commands: []*cobra.Command{
				NewCmdStep(f, in, out, err),
			},
		},
		{
			Message: "Jenkins X services:",
			Commands: []*cobra.Command{
				NewCmdController(f, in, out, err),
				NewCmdGC(f, in, out, err),
				NewCmdServeBuildNumbers(f, in, out, err),
			},
		},
	}

	groups.Add(cmds)

	filters := []string{"options"}
	templates.ActsAsRootCommand(cmds, filters, groups...)
	cmds.AddCommand(NewCmdDocs(f, in, out, err))
	cmds.AddCommand(NewCmdVersion(f, in, out, err))
	cmds.Version = version.GetVersion()
	cmds.SetVersionTemplate("{{printf .Version}}\n")
	cmds.AddCommand(NewCmdOptions(out))
	cmds.AddCommand(NewCmdDiagnose(f, in, out, err))

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
