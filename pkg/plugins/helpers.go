package plugins

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/blang/semver"
	"github.com/spf13/cobra"

	jenkinsv1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/extensions"
	"github.com/jenkins-x/jx-helpers/v3/pkg/homedir"
	"github.com/jenkins-x/jx-helpers/v3/pkg/httphelpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

const (
	jenkinsxOrganisation        = "jenkins-x"
	jenkinsxPluginsOrganisation = "jenkins-x-plugins"

	// OctantPluginName the default name of the octant plugin
	OctantPluginName = "octant"

	// OctantJXPluginName the name of the octant-jx plugin
	OctantJXPluginName = "octant-jx"

	// OctantJXOPluginName the name of the octant-jxo plugin
	OctantJXOPluginName = "octant-jxo"
)

// GetJXPlugin returns the path to the locally installed jx plugin
func GetJXPlugin(name, version string) (string, error) {
	pluginBinDir, err := homedir.DefaultPluginBinDir()
	if err != nil {
		return "", err
	}
	plugin := extensions.CreateJXPlugin(jenkinsxOrganisation, name, version)
	return extensions.EnsurePluginInstalled(plugin, pluginBinDir)
}

// GetOctantBinary returns the path to the locally installed octant plugin
func GetOctantBinary(version string) (string, error) {
	if version == "" {
		version = OctantVersion
	}
	pluginBinDir, err := homedir.DefaultPluginBinDir()
	if err != nil {
		return "", err
	}
	plugin := CreateOctantPlugin(version)
	return extensions.EnsurePluginInstalled(plugin, pluginBinDir)
}

// CreateOctantPlugin creates the helm 3 plugin
func CreateOctantPlugin(version string) jenkinsv1.Plugin {
	binaries := extensions.CreateBinaries(func(p extensions.Platform) string {
		kind := strings.ToLower(p.Goarch)
		if strings.HasSuffix(kind, "64") {
			kind = "64bit"
		}
		goos := p.Goos
		if goos == "Darwin" {
			goos = "macOS"
		}
		return fmt.Sprintf("https://github.com/vmware-tanzu/octant/releases/download/v%s/octant_%s_%s-%s.%s", version, version, goos, kind, p.Extension())
	})

	plugin := jenkinsv1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: OctantPluginName,
		},
		Spec: jenkinsv1.PluginSpec{
			SubCommand:  "octant",
			Binaries:    binaries,
			Description: "octant binary",
			Name:        OctantPluginName,
			Version:     version,
		},
	}
	return plugin
}

// GetOctantJXBinary returns the path to the locally installed octant-jx extension
func GetOctantJXBinary(version string) (string, error) {
	if version == "" {
		version = OctantJXVersion
	}
	pluginBinDir, err := homedir.DefaultPluginBinDir()
	if err != nil {
		return "", err
	}
	plugin := CreateOctantJXPlugin(version)
	return extensions.EnsurePluginInstalled(plugin, pluginBinDir)
}

// CreateOctantJXPlugin creates the helm 3 plugin
func CreateOctantJXPlugin(version string) jenkinsv1.Plugin {
	binaries := extensions.CreateBinaries(func(p extensions.Platform) string {
		return fmt.Sprintf("https://github.com/jenkins-x-plugins/octant-jx/releases/download/v%s/octant-jx-%s-%s.%s", version, strings.ToLower(p.Goos), strings.ToLower(p.Goarch), p.Extension())
	})

	plugin := jenkinsv1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: OctantJXPluginName,
		},
		Spec: jenkinsv1.PluginSpec{
			SubCommand:  "octant-jx",
			Binaries:    binaries,
			Description: "octant plugin for Jenkins X",
			Name:        OctantJXPluginName,
			Version:     version,
		},
	}
	return plugin
}

// GetOctantJXOBinary returns the path to the locally installed helmAnnotate extension
func GetOctantJXOBinary(version string) (string, error) {
	if version == "" {
		version = OctantJXVersion
	}
	pluginBinDir, err := homedir.DefaultPluginBinDir()
	if err != nil {
		return "", err
	}
	plugin := CreateOctantJXOPlugin(version)
	aliasFileName := "ha.tar.gz"
	if runtime.GOOS == "windows" {
		aliasFileName = "ha.zip"
	}
	return extensions.EnsurePluginInstalledForAliasFile(plugin, pluginBinDir, aliasFileName)
}

// CreateOctantJXOPlugin creates the octant-ojx plugin
func CreateOctantJXOPlugin(version string) jenkinsv1.Plugin {
	plugin := CreateOctantJXPlugin(version)
	plugin.Name = OctantJXOPluginName
	plugin.Spec.Name = OctantJXOPluginName
	plugin.Spec.SubCommand = OctantJXOPluginName
	return plugin
}

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// InstallStandardPlugin makes sure that latest version of plugin is installed and returns the path to the binary
func InstallStandardPlugin(dir, name string) (string, error) {
	u := "https://api.github.com/repos/jenkins-x-plugins/" + name + "/releases/latest"

	client := httphelpers.GetClient()
	req, err := http.NewRequest("GET", u, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("failed to create http request for %s: %w", u, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		if resp != nil {
			return "", fmt.Errorf("failed to GET endpoint %s with status %s: %w", u, resp.Status, err)
		}
		return "", fmt.Errorf("failed to GET endpoint %s: %w", u, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response from %s: %w", u, err)
	}

	release := &githubRelease{}
	err = json.Unmarshal(body, release)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal release from %s: %w", u, err)
	}
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	if latestVersion == "" {
		return "", fmt.Errorf("can't find latest version of plugin: %s", body)
	}

	plugin := extensions.CreateJXPlugin("jenkins-x-plugins", strings.TrimPrefix(name, "jx-"), latestVersion)
	if err != nil {
		return "", err
	}
	return extensions.EnsurePluginInstalled(plugin, dir)
}

func AllPlugins() (validArgs []string) {
	pluginBinDir, err := homedir.DefaultPluginBinDir()
	for k := range Plugins {
		validArgs = append(validArgs, Plugins[k].Name)
	}
	if err != nil {
		return
	}
	file, err := os.Open(pluginBinDir)
	if err != nil {
		return
	}
	defer file.Close()
	files, err := file.Readdirnames(0)
	if err != nil {
		return
	}
	pluginPattern := regexp.MustCompile("^jx-(.*)-[0-9.]+$")
	for _, plugin := range files {
		res := pluginPattern.FindStringSubmatch(plugin)
		if len(res) > 1 && PluginMap["jx-"+res[1]] == nil {
			validArgs = append(validArgs, res[1])
		}
	}
	return
}

// SetupPluginCompletion adds a Cobra command to the command tree for each
// plugin.  This is only done when performing shell completion that relate
// to plugins.
func SetupPluginCompletion(cmd *cobra.Command, args []string) {
	root := cmd.Root()
	if len(args) == 0 {
		return
	}
	if strings.HasPrefix(args[0], "-") {
		// Plugins are not supported if the first argument is a flag,
		// so no need to add them in that case.
		return
	}

	registerPluginCommands(root, true)
}

// registerPluginCommand allows adding Cobra command to the command tree or extracting them for usage in
// e.g. the help function or for registering the completion function
func registerPluginCommands(rootCmd *cobra.Command, list bool) (cmds []*cobra.Command) {
	var userDefinedCommands []*cobra.Command

	for _, plugin := range AllPlugins() {
		var args []string

		rawPluginArgs := strings.Split(plugin, "-")
		pluginArgs := rawPluginArgs[:1]
		if list {
			pluginArgs = rawPluginArgs
		}

		// Iterate through all segments, for kubectl-my_plugin-sub_cmd, we will end up with
		// two iterations: one for my_plugin and one for sub_cmd.
		for _, arg := range pluginArgs {
			// Underscores (_) in plugin's filename are replaced with dashes(-)
			// e.g. foo_bar -> foo-bar
			args = append(args, strings.ReplaceAll(arg, "_", "-"))
		}

		// In order to avoid that the same plugin command is added more than once,
		// find the lowest command given args from the root command
		parentCmd, remainingArgs, _ := rootCmd.Find(args)
		if parentCmd == nil {
			parentCmd = rootCmd
		}

		for _, remainingArg := range remainingArgs {
			cmd := &cobra.Command{
				Use: remainingArg,
				// Add a description that will be shown with completion choices.
				// Make each one different by including the plugin name to avoid
				// all plugins being grouped in a single line during completion for zsh.
				Short:              fmt.Sprintf("The command %s is a plugin", remainingArg),
				DisableFlagParsing: true,
				// Allow plugins to provide their own completion choices
				ValidArgsFunction: PluginCompletion,
				// A Run is required for it to be a valid command
				Run: func(_ *cobra.Command, _ []string) {},
			}
			// Add the plugin command to the list of user defined commands
			userDefinedCommands = append(userDefinedCommands, cmd)

			if list {
				parentCmd.AddCommand(cmd)
				parentCmd = cmd
			}
		}
	}

	return userDefinedCommands
}

// PluginCompletion deals with shell completion beyond the plugin name, it allows to complete
// plugin arguments and flags.
// It will call the plugin with __complete as the first argument.
//
// This completion command should print the completion choices to stdout, which is suported by default
// for commands developed with Cobra.
// The rest of the arguments will be the arguments for the plugin currently
// on the command-line.  For example, if a user types:
//
//	jx myplugin arg1 arg2 a<TAB>
//
// the plugin executable will be called with arguments: "__complete" "arg1" "arg2" "a".
// And if a user types:
//
//	jx myplugin arg1 arg2 <TAB>
//
// the completion executable will be called with arguments: "__complete" "arg1" "arg2" "".  Notice the empty
// last argument which indicates that a new word should be completed but that the user has not
// typed anything for it yet.
//
// JX's plugin completion logic supports Cobra's ShellCompDirective system.  This means a plugin
// can optionally print :<value of a shell completion directive> as its very last line to provide
// directives to the shell on how to perform completion.  If this directive is not present, the
// cobra.ShellCompDirectiveDefault will be used. Please see Cobra's documentation for more details:
// https://github.com/spf13/cobra/blob/master/shell_completions.md#dynamic-completion-of-nouns
func PluginCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Recreate the plugin name from the commandPath
	pluginName := strings.ReplaceAll(strings.ReplaceAll(cmd.CommandPath(), "-", "_"), " ", "-")

	pluginDir, err := homedir.DefaultPluginBinDir()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	path, err := Lookup(pluginName, pluginDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	newArgs := make([]string, len(args)+2) //nolint:mnd
	newArgs[0] = "__complete"
	for i, arg := range args {
		newArgs[i+1] = arg
	}
	newArgs[len(newArgs)-1] = toComplete
	cobra.CompDebugln(fmt.Sprintf("About to call: %s %s", path, strings.Join(args, " ")), true)
	return getPluginCompletions(path, newArgs, os.Environ())
}

// getPluginCompletions receives an executable's filepath, a slice
// of arguments, and a slice of environment variables
// to relay to the executable.
func getPluginCompletions(executablePath string, cmdArgs, environment []string) ([]string, cobra.ShellCompDirective) {
	buf := new(bytes.Buffer)

	prog := exec.Command(executablePath, cmdArgs...)
	prog.Stdin = os.Stdin
	prog.Stdout = buf
	prog.Stderr = os.Stderr
	prog.Env = environment

	var comps []string
	directive := cobra.ShellCompDirectiveNoFileComp
	if err := prog.Run(); err == nil {
		for _, comp := range strings.Split(buf.String(), "\n") {
			// Remove any empty lines
			if comp != "" {
				comps = append(comps, comp)
			}
		}

		// Check if the last line of output is of the form :<integer>, which
		// indicates a Cobra ShellCompDirective.  We do this for plugins
		// that use Cobra or the ones that wish to use this directive to
		// communicate a special behavior for the shell.
		if len(comps) > 0 {
			lastLine := comps[len(comps)-1]
			if len(lastLine) > 1 && lastLine[0] == ':' {
				if strInt, err := strconv.Atoi(lastLine[1:]); err == nil {
					directive = cobra.ShellCompDirective(strInt)
					comps = comps[:len(comps)-1]
				}
			}
		}
	}
	return comps, directive
}

func FindStandardPlugin(dir, name string) (string, error) {
	file, err := os.Open(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read plugin dir %s: %w", dir, err)
	}
	defer file.Close()
	files, err := file.Readdirnames(0)
	if err != nil {
		return "", fmt.Errorf("failed to read plugin dir %s: %w", dir, err)
	}
	pluginPattern, err := regexp.Compile("^" + name + "-([0-9.]+)$")
	if err != nil {
		return "", err
	}

	vers := make([]string, 0)
	for _, plugin := range files {
		res := pluginPattern.FindStringSubmatch(plugin)
		if len(res) > 1 {
			vers = append(vers, res[1])
		}
	}

	if len(vers) > 0 {
		vs := make([]semver.Version, 0)
		for _, r := range vers {
			v, err := semver.Parse(r)
			if err == nil {
				vs = append(vs, v)
			}
		}

		sort.Sort(sort.Reverse(semver.Versions(vs)))
		if len(vs) > 0 {
			return filepath.Join(dir, name+"-"+vs[0].String()), nil
		}
	}
	return InstallStandardPlugin(dir, name)
}

func Lookup(filename, pluginBinDir string) (string, error) {
	path, err := exec.LookPath(filename)
	if err != nil {
		path, err = FindStandardPlugin(pluginBinDir, filename)
		if err != nil {
			return "", fmt.Errorf("failed to load plugin %s: %w", filename, err)
		}
	}
	return path, nil
}

func Execute(executablePath string, cmdArgs, environment []string) error {
	// Windows does not support exec syscall.
	if runtime.GOOS == "windows" {
		cmd := exec.Command(executablePath, cmdArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Env = environment
		err := cmd.Run()
		if err == nil {
			os.Exit(0)
		}
		return err
	}

	// invoke cmd binary relaying the environment and args given
	// append executablePath to cmdArgs, as execve will make first argument the "binary name".
	// ToDo: Look at sanitizing the inputs passed to syscall exec, may be move away from syscall as it's deprecated.
	return syscall.Exec(executablePath, append([]string{executablePath}, cmdArgs...), environment) //nolint
}
