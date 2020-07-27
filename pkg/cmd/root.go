package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/jenkins-x/jx-api/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-cli/pkg/cmd/namespace"
	"github.com/jenkins-x/jx-cli/pkg/cmd/upgrade"
	"github.com/jenkins-x/jx-cli/pkg/cmd/version"
	"github.com/jenkins-x/jx-cli/pkg/plugins"
	"github.com/jenkins-x/jx-helpers/pkg/cobras"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/extensions"
	"github.com/jenkins-x/jx-helpers/pkg/homedir"
	"github.com/jenkins-x/jx-helpers/pkg/kube/jxclient"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Main creates the new command
func Main(args []string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jx",
		Short: "Jenkins X 3.x alpha command line",
		Run:   runHelp,
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
		return pluginCommandGroups, po.ManagedPluginsEnabled
	}
	doCmd := func(cmd *cobra.Command, args []string) {
		handleCommand(po, cmd, args, getPluginCommandGroups)
	}

	generalCommands := []*cobra.Command{
		cobras.SplitCommand(namespace.NewCmdNamespace()),
		cobras.SplitCommand(upgrade.NewCmdUpgrade()),
		cobras.SplitCommand(version.NewCmdVersion()),
	}

	// aliases to classic jx commands...
	getCmd := &cobra.Command{
		Use:   "get TYPE [flags]",
		Short: "Display one or more resources",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			helper.CheckErr(err)
		},
		SuggestFor: []string{"list", "ps"},
	}
	getBuildCmd := &cobra.Command{
		Use:     "build TYPE [flags]",
		Short:   "Display one or more resources relating to a pipeline build",
		Aliases: []string{"builds"},
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			helper.CheckErr(err)
		},
	}
	createCmd := &cobra.Command{
		Use:   "create TYPE [flags]",
		Short: "Create one or more resources",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			helper.CheckErr(err)
		},
		SuggestFor: []string{"new", "make"},
	}
	startCmd := &cobra.Command{
		Use:   "start TYPE [flags]",
		Short: "Starts a resource",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			helper.CheckErr(err)
		},
	}
	stopCmd := &cobra.Command{
		Use:   "stop TYPE [flags]",
		Short: "Stops a resource",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			helper.CheckErr(err)
		},
	}
	getCmd.AddCommand(
		getBuildCmd,
		aliasCommand(cmd, doCmd, "activities", []string{"pipeline", "activities"}, "act", "activity"),
		aliasCommand(cmd, doCmd, "application", []string{"application"}, "app", "apps", "applications"),
		aliasCommand(cmd, doCmd, "pipelines", []string{"pipeline", "get"}, "pipeline"),
		aliasCommand(cmd, doCmd, "previews", []string{"pipeline", "previews"}, "preview"),
	)
	getBuildCmd.AddCommand(
		aliasCommand(cmd, doCmd, "logs", []string{"pipeline", "logs"}, "log"),
		aliasCommand(cmd, doCmd, "pods", []string{"pipeline", "pods"}, "pod"),
	)
	createCmd.AddCommand(
		aliasCommand(cmd, doCmd, "quickstart", []string{"project", "quickstart"}, "qs"),
		aliasCommand(cmd, doCmd, "project", []string{"project"}),
		aliasCommand(cmd, doCmd, "pullrequest", []string{"project", "pullrequest"}, "pr"),
	)
	startCmd.AddCommand(
		aliasCommand(cmd, doCmd, "pipeline", []string{"pipeline", "start"}, "pipelines"),
	)
	stopCmd.AddCommand(
		aliasCommand(cmd, doCmd, "pipeline", []string{"pipeline", "stop"}, "pipelines"),
	)
	generalCommands = append(generalCommands, getCmd, createCmd, startCmd, stopCmd,
		aliasCommand(cmd, doCmd, "import", []string{"project", "import"}, "log"),
	)

	cmd.AddCommand(generalCommands...)
	groups := templates.CommandGroups{
		{
			Message:  "General:",
			Commands: generalCommands,
		},
	}
	groups.Add(cmd)
	filters := []string{"options"}

	templates.ActsAsRootCommand(cmd, filters, getPluginCommandGroups, groups...)
	handleCommand(po, cmd, args, getPluginCommandGroups)
	return cmd
}

func handleCommand(po *templates.Options, cmd *cobra.Command, args []string, getPluginCommandGroups func() (templates.PluginCommandGroups, bool)) {
	managedPlugins := &managedPluginHandler{
		JXClient:  po.JXClient,
		Namespace: po.Namespace,
	}
	localPlugins := &localPluginHandler{}

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
			if _, managedPluginsEnabled := getPluginCommandGroups(); managedPluginsEnabled {
				if err := handleEndpointExtensions(managedPlugins, cmdPathPieces, pluginDir); err != nil {
					log.Logger().Errorf("%v", err)
					os.Exit(1)
				}
			} else {
				if err := handleEndpointExtensions(localPlugins, cmdPathPieces, pluginDir); err != nil {
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
		Short:   "alias for: jx " + name,
		Aliases: aliases,
		Run: func(cmd *cobra.Command, args []string) {
			realArgs = append(realArgs, args...)
			log.Logger().Debugf("about to invoke alias: %s", strings.Join(realArgs, " "))
			fn(rootCmd, realArgs)
		},
		SuggestFor:         []string{"jx " + name},
		DisableFlagParsing: true,
	}
	return cmd
}

func runHelp(cmd *cobra.Command, args []string) {
	cmd.Help() //nolint:errcheck
}

// PluginHandler is capable of parsing command line arguments
// and performing executable filename lookups to search
// for valid plugin files, and execute found plugins.
type PluginHandler interface {
	// Lookup receives a potential filename and returns
	// a full or relative path to an executable, if one
	// exists at the given filename, or an error.
	Lookup(filename string, pluginBinDir string) (string, error)
	// Execute receives an executable's filepath, a slice
	// of arguments, and a slice of environment variables
	// to relay to the executable.
	Execute(executablePath string, cmdArgs, environment []string) error
}

type managedPluginHandler struct {
	JXClient  versioned.Interface
	Namespace string
	localPluginHandler
}

// Lookup implements PluginHandler
func (h *managedPluginHandler) Lookup(filename, pluginBinDir string) (string, error) {
	jxClient, ns, err := jxclient.LazyCreateJXClientAndNamespace(h.JXClient, h.Namespace)
	if err != nil {
		return "", err
	}

	possibles, err := jxClient.JenkinsV1().Plugins(ns).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", extensions.PluginCommandLabel, filename),
	})
	if err != nil {
		return "", err
	}
	if len(possibles.Items) > 0 {
		found := possibles.Items[0]
		if len(possibles.Items) > 1 {
			// There is a warning about this when you install extensions as well
			log.Logger().Warnf("More than one plugin installed for %s by apps. Selecting the one installed by %s at random.",
				filename, found.Name)

		}
		return extensions.EnsurePluginInstalled(found, pluginBinDir)
	}
	return h.localPluginHandler.Lookup(filename, pluginBinDir)
}

// Execute implements PluginHandler
func (h *managedPluginHandler) Execute(executablePath string, cmdArgs, environment []string) error {
	return h.localPluginHandler.Execute(executablePath, cmdArgs, environment)
}

type localPluginHandler struct{}

// Lookup implements PluginHandler
func (h *localPluginHandler) Lookup(filename, pluginBinDir string) (string, error) {
	// if on Windows, append the "exe" extension
	// to the filename that we are looking up.
	if runtime.GOOS == "windows" {
		filename += ".exe"
	}

	return exec.LookPath(filename)
}

// Execute implements PluginHandler
func (h *localPluginHandler) Execute(executablePath string, cmdArgs, environment []string) error {
	return syscall.Exec(executablePath, cmdArgs, environment)
}

func handleEndpointExtensions(pluginHandler PluginHandler, cmdArgs []string, pluginBinDir string) error {
	remainingArgs := []string{} // all "non-flag" arguments

	for idx := range cmdArgs {
		if strings.HasPrefix(cmdArgs[idx], "-") {
			break
		}
		remainingArgs = append(remainingArgs, strings.Replace(cmdArgs[idx], "-", "_", -1))
	}

	foundBinaryPath := ""

	// attempt to find binary, starting at longest possible name with given cmdArgs
	for len(remainingArgs) > 0 {
		commandName := fmt.Sprintf("jx-%s", strings.Join(remainingArgs, "-"))
		path, err := pluginHandler.Lookup(commandName, pluginBinDir)
		if err != nil || path == "" {
			// lets see if we have previously downloaded this binary plugin
			path = FindPluginBinary(pluginBinDir, commandName)
			if path != "" {
				foundBinaryPath = path
				break
			}

			/* Usually "executable file not found in $PATH", spams output of jx help subcommand:
			if err != nil {
				log.Logger().Errorf("Error installing plugin for command %s. %v\n", remainingArgs, err)
			}
			*/
			remainingArgs = remainingArgs[:len(remainingArgs)-1]
			continue
		}

		foundBinaryPath = path
		break
	}

	if foundBinaryPath == "" {
		return nil
	}

	// invoke cmd binary relaying the current environment and args given
	// remainingArgs will always have at least one element.
	// execve will make remainingArgs[0] the "binary name".
	if err := pluginHandler.Execute(foundBinaryPath, append([]string{foundBinaryPath}, cmdArgs[len(remainingArgs):]...), os.Environ()); err != nil {
		return err
	}

	return nil
}

// FindPluginBinary tries to find the jx-foo binary plugin in the plugins dir `~/.jx/plugins/jx/bin` dir `
func FindPluginBinary(pluginDir, commandName string) string {
	if pluginDir != "" {
		files, err := ioutil.ReadDir(pluginDir)
		if err != nil {
			log.Logger().Debugf("failed to read plugin dir %s", err.Error())
		} else {
			prefix := commandName + "-"
			for _, f := range files {
				name := f.Name()
				if strings.HasPrefix(name, prefix) {
					path := filepath.Join(pluginDir, name)
					log.Logger().Debugf("found plugin %s at %s", commandName, path)
					return path
				}
			}
		}
	}
	return ""
}
