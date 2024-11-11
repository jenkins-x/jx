package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/extensions"
	"github.com/jenkins-x/jx-helpers/v3/pkg/homedir"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/jenkins-x/jx/pkg/cmd/dashboard"
	"github.com/jenkins-x/jx/pkg/cmd/namespace"
	"github.com/jenkins-x/jx/pkg/cmd/ui"
	"github.com/jenkins-x/jx/pkg/cmd/upgrade"
	"github.com/jenkins-x/jx/pkg/cmd/version"
	"github.com/jenkins-x/jx/pkg/plugins"
	"github.com/spf13/cobra"
)

// Main creates the new command
func Main(args []string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jx",
		Short: "Jenkins X 3.x command line",
		Run:   runHelp,
		// Hook before and after Run initialize and write profiles to disk,
		// respectively.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

			if cmd.Name() == cobra.ShellCompRequestCmd || cmd.Name() == cobra.ShellCompNoDescRequestCmd {
				// This is the __complete or __completeNoDesc command which
				// indicates shell completion has been requested.
				plugins.SetupPluginCompletion(cmd, args)
			}
			return nil
		},
	}

	po := &templates.Options{}
	getPluginCommandGroups := func() (templates.PluginCommandGroups, bool) {
		verifier := &extensions.CommandOverrideVerifier{
			Root:        cmd,
			SeenPlugins: make(map[string]string),
		}
		pluginCommandGroups, err := po.GetPluginCommandGroups(verifier, plugins.Plugins)
		if err != nil {
			log.Logger().Errorf("%v", err)
		}
		return pluginCommandGroups, false
	}
	doCmd := func(cmd *cobra.Command, args []string) {
		handleCommand(cmd, args)
	}

	generalCommands := []*cobra.Command{
		cobras.SplitCommand(dashboard.NewCmdDashboard()),
		cobras.SplitCommand(namespace.NewCmdNamespace()),
		cobras.SplitCommand(ui.NewCmdUI()),
		cobras.SplitCommand(upgrade.NewCmdUpgrade()),
		cobras.SplitCommand(version.NewCmdVersion()),
	}

	// aliases to classic jx commands...
	getCmd := &cobra.Command{
		Use:   "get TYPE [flags]",
		Short: "Display one or more resources",
		Run: func(cmd *cobra.Command, _ []string) {
			err := cmd.Help()
			helper.CheckErr(err)
		},
		SuggestFor: []string{"list", "ps"},
	}
	addCmd := &cobra.Command{
		Use:   "add TYPE [flags]",
		Short: "Adds one or more resources",
		Run: func(cmd *cobra.Command, _ []string) {
			err := cmd.Help()
			helper.CheckErr(err)
		},
	}
	getBuildCmd := &cobra.Command{
		Use:     "build TYPE [flags]",
		Short:   "Display one or more resources relating to a pipeline build",
		Aliases: []string{"builds"},
		Run: func(cmd *cobra.Command, _ []string) {
			err := cmd.Help()
			helper.CheckErr(err)
		},
	}
	createCmd := &cobra.Command{
		Use:   "create TYPE [flags]",
		Short: "Create one or more resources",
		Run: func(cmd *cobra.Command, _ []string) {
			err := cmd.Help()
			helper.CheckErr(err)
		},
		SuggestFor: []string{"new", "make"},
	}
	startCmd := &cobra.Command{
		Use:   "start TYPE [flags]",
		Short: "Starts a resource",
		Run: func(cmd *cobra.Command, _ []string) {
			err := cmd.Help()
			helper.CheckErr(err)
		},
	}
	stopCmd := &cobra.Command{
		Use:   "stop TYPE [flags]",
		Short: "Stops a resource",
		Run: func(cmd *cobra.Command, _ []string) {
			err := cmd.Help()
			helper.CheckErr(err)
		},
	}
	addCmd.AddCommand(
		aliasCommand(cmd, doCmd, "app", []string{"gitops", "helmfile", "add"}, "chart"),
	)
	getCmd.AddCommand(
		getBuildCmd,
		aliasCommand(cmd, doCmd, "activities", []string{"pipeline", "activities"}, "act", "activity"),
		aliasCommand(cmd, doCmd, "application", []string{"application", "get"}, "app", "apps", "applications"),
		aliasCommand(cmd, doCmd, "pipelines", []string{"pipeline", "get"}, "pipeline"),
		aliasCommand(cmd, doCmd, "previews", []string{"preview", "get"}, "preview"),
	)
	getBuildCmd.AddCommand(
		aliasCommand(cmd, doCmd, "logs", []string{"pipeline", "logs"}, "log"),
		aliasCommand(cmd, doCmd, "pods", []string{"pipeline", "pods"}, "pod"),
	)
	createCmd.AddCommand(
		aliasCommand(cmd, doCmd, "quickstart", []string{"project", "quickstart"}, "qs"),
		aliasCommand(cmd, doCmd, "spring", []string{"project", "spring"}, "sb"),
		aliasCommand(cmd, doCmd, "project", []string{"project"}),
		aliasCommand(cmd, doCmd, "pullrequest", []string{"project", "pullrequest"}, "pr"),
	)
	startCmd.AddCommand(
		aliasCommand(cmd, doCmd, "pipeline", []string{"pipeline", "start"}, "pipelines"),
	)
	stopCmd.AddCommand(
		aliasCommand(cmd, doCmd, "pipeline", []string{"pipeline", "stop"}, "pipelines"),
	)
	generalCommands = append(generalCommands, addCmd, getCmd, createCmd, startCmd, stopCmd,
		aliasCommand(cmd, doCmd, "import", []string{"project", "import"}),
		aliasCommand(cmd, doCmd, "ctx", []string{"context"}),
	)

	cmd.AddCommand(generalCommands...)

	var groups templates.CommandGroups
	command := templates.CommandGroup{

		Message:  "General:",
		Commands: generalCommands,
	}
	groups = append(groups, command)

	groups.Add(cmd)
	filters := []string{"options"}

	templates.ActsAsRootCommand(cmd, filters, getPluginCommandGroups, groups...)
	handleCommand(cmd, args)
	return cmd
}

func handleCommand(cmd *cobra.Command, args []string) {

	if len(args) == 0 {
		args = os.Args
	}
	if len(args) > 1 {
		cmdPathPieces := args[1:]
		pluginDir, err := homedir.DefaultPluginBinDir()
		if err != nil {
			log.Logger().Errorf("%v", err)
			os.Exit(1)
		}

		// only look for suitable executables if
		// the specified command does not already exist
		if _, _, err := cmd.Find(cmdPathPieces); err != nil {
			var cmdName string // first "non-flag" arguments
			for _, arg := range cmdPathPieces {
				if !strings.HasPrefix(arg, "-") {
					cmdName = arg
					break
				}
			}
			switch cmdName {
			case "help", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd, "completion":
				// Don't search for a plugin
			default:
				if err := handleEndpointExtensions(cmdPathPieces, pluginDir); err != nil {
					log.Logger().Errorf("%v", err)
					os.Exit(1)
				}
			}
		}
	}
}

func aliasCommand(rootCmd *cobra.Command, fn func(cmd *cobra.Command, args []string), name string, args []string, aliases ...string) *cobra.Command {
	realArgs := append([]string{"jx"}, args...)
	cmd := &cobra.Command{
		Use:     name,
		Short:   "alias for: " + strings.Join(realArgs, " "),
		Aliases: aliases,
		ValidArgsFunction: func(_ *cobra.Command, completeArgs []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			cmd, pluginArgs, err := rootCmd.Find(args)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return plugins.PluginCompletion(cmd, append(pluginArgs, completeArgs...), toComplete)
		},
		Run: func(_ *cobra.Command, args []string) {
			realArgs = append(realArgs, args...)
			log.Logger().Debugf("about to invoke alias: %s", strings.Join(realArgs, " "))
			fn(rootCmd, realArgs)
		},
		DisableFlagParsing: true,
	}
	return cmd
}

func runHelp(cmd *cobra.Command, _ []string) {
	cmd.Help() //nolint:errcheck
}

func handleEndpointExtensions(cmdArgs []string, pluginBinDir string) error {
	var remainingArgs []string // all "non-flag" arguments

	for idx := range cmdArgs {
		if strings.HasPrefix(cmdArgs[idx], "-") {
			break
		}
		remainingArgs = append(remainingArgs, strings.ReplaceAll(cmdArgs[idx], "-", "_"))
	}

	foundBinaryPath := ""

	// attempt to find binary, starting at longest possible name with given cmdArgs
	var err error
	for len(remainingArgs) > 0 {
		commandName := fmt.Sprintf("jx-%s", strings.Join(remainingArgs, "-"))

		// lets try the correct plugin versions first
		path := ""
		if plugins.PluginMap[commandName] != nil {
			p := *plugins.PluginMap[commandName]
			path, err = extensions.EnsurePluginInstalled(p, pluginBinDir)
			if err != nil {
				return fmt.Errorf("failed to install binary plugin %s version %s to %s: %w", commandName, p.Spec.Version, pluginBinDir, err)
			}
		}

		// lets see if there's a local build of the plugin on the PATH for developers...
		if path == "" {
			path, err = plugins.Lookup(commandName, pluginBinDir)
		}
		if path != "" {
			foundBinaryPath = path
			break
		}
		remainingArgs = remainingArgs[:len(remainingArgs)-1]
	}

	if foundBinaryPath == "" {
		return err
	}

	nextArgs := cmdArgs[len(remainingArgs):]
	log.Logger().Debugf("using the plugin command: %s", termcolor.ColorInfo(foundBinaryPath+" "+strings.Join(nextArgs, " ")))

	// Giving plugin information about how it was invoked, so it can give correct help
	pluginCommandName := os.Args[0] + " " + strings.Join(remainingArgs, " ")
	environ := append(os.Environ(),
		fmt.Sprintf("BINARY_NAME=%s", pluginCommandName),
		fmt.Sprintf("TOP_LEVEL_COMMAND=%s", pluginCommandName))
	// invoke cmd binary relaying the current environment and args given
	// remainingArgs will always have at least one element.
	// execute will make remainingArgs[0] the "binary name".
	return plugins.Execute(foundBinaryPath, nextArgs, environ)
}
