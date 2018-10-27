/*
Copyright 2018 The Kubernetes Authors & The Jenkins X Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	plugin_long = templates.LongDesc(`
		Provides utilities for interacting with plugins.

		Plugins provide extended functionality that is not part of the major command-line distribution.
		Please refer to the documentation and examples for more information about how write your own plugins.`)

	plugin_list_long = templates.LongDesc(`
		List all available plugin files on a user's PATH.

		Available plugin files are those that are:
		- executable
		- anywhere on the user's PATH
		- begin with "jx-"
`)
)

type PluginOptions struct {
	CommonOptions
}

func NewCmdPlugin(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &PluginOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:   "plugin [flags]",
		Short: "Provides utilities for interacting with plugins.",
		Long:  plugin_long,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()

			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdPluginList(f, in, out, errOut))
	return cmd
}

func (o *PluginOptions) Run() error {
	return o.Cmd.Help()
}

type PluginListOptions struct {
	PluginOptions
	Verifier PathVerifier
	NameOnly bool
}

// NewCmdPluginList provides a way to list all plugin executables visible to jx
func NewCmdPluginList(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &PluginListOptions{
		PluginOptions: PluginOptions{
			CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all visible plugin executables on a user's PATH",
		Long:  plugin_list_long,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Complete()
			CheckErr(err)
			err = options.Run()

			CheckErr(err)
		},
	}

	cmd.Flags().BoolVar(&options.NameOnly, "name-only", options.NameOnly, "If true, display only the binary name of each plugin, rather than its full path")
	return cmd
}

func (o *PluginListOptions) Complete() error {
	o.Verifier = &CommandOverrideVerifier{
		root:        o.Cmd.Root(),
		seenPlugins: make(map[string]string, 0),
	}
	return nil
}

func (o *PluginListOptions) Run() error {
	path := "PATH"
	if runtime.GOOS == "windows" {
		path = "path"
	}

	pluginsFound := false
	isFirstFile := true
	pluginWarnings := 0
	paths := sets.NewString(filepath.SplitList(os.Getenv(path))...)
	for _, dir := range paths.List() {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, f := range files {
			if f.IsDir() {
				continue
			}
			if !strings.HasPrefix(f.Name(), "jx-") {
				continue
			}

			if isFirstFile {
				log.Info("The following jx-compatible plugins are available:\n")
				pluginsFound = true
				isFirstFile = false
			}

			pluginPath := f.Name()
			if !o.NameOnly {
				pluginPath = filepath.Join(dir, pluginPath)
			}

			log.Infof("%s\n", util.ColorInfo(pluginPath))
			if errs := o.Verifier.Verify(filepath.Join(dir, f.Name())); len(errs) != 0 {
				for _, err := range errs {
					log.Errorf("  - %v\n", err)
					pluginWarnings++
				}
			}
		}
	}

	if !pluginsFound {
		return fmt.Errorf("error: unable to find any jx plugins in your PATH")
	} else {
		// Add a trailing line to make the output more readable
		log.Infoln("")
	}

	if pluginWarnings > 0 {
		if pluginWarnings == 1 {
			return fmt.Errorf("one plugin warning was found")
		}
		return fmt.Errorf("%v plugin warnings were found", pluginWarnings)
	}

	return nil
}

// pathVerifier receives a path and determines if it is valid or not
type PathVerifier interface {
	// Verify determines if a given path is valid
	Verify(path string) []error
}

type CommandOverrideVerifier struct {
	root        *cobra.Command
	seenPlugins map[string]string
}

// Verify implements PathVerifier and determines if a given path
// is valid depending on whether or not it overwrites an existing
// jx command path, or a previously seen plugin.
func (v *CommandOverrideVerifier) Verify(path string) []error {
	if v.root == nil {
		return []error{fmt.Errorf("unable to verify path with nil root")}
	}

	// extract the plugin binary name
	segs := strings.Split(path, "/")
	binName := segs[len(segs)-1]

	cmdPath := strings.Split(binName, "-")
	if len(cmdPath) > 1 {
		// the first argument is always "jx" for a plugin binary
		cmdPath = cmdPath[1:]
	}

	errors := []error{}

	if isExec, err := isExecutable(path); err == nil && !isExec {
		errors = append(errors, fmt.Errorf("warning: %s identified as a jx plugin, but it is not executable", path))
	} else if err != nil {
		errors = append(errors, fmt.Errorf("error: unable to identify %s as an executable file: %v", path, err))
	}

	if existingPath, ok := v.seenPlugins[binName]; ok {
		errors = append(errors, fmt.Errorf("warning: %s is overshadowed by a similarly named plugin: %s", path, existingPath))
	} else {
		v.seenPlugins[binName] = path
	}

	if cmd, _, err := v.root.Find(cmdPath); err == nil {
		errors = append(errors, fmt.Errorf("warning: %s overwrites existing command: %q", binName, cmd.CommandPath()))
	}

	return errors
}

func isExecutable(fullPath string) (bool, error) {
	info, err := os.Stat(fullPath)
	if err != nil {
		return false, err
	}

	if runtime.GOOS == "windows" {
		if strings.HasSuffix(info.Name(), ".exe") {
			return true, nil
		}
		return false, nil
	}

	if m := info.Mode(); !m.IsDir() && m&0111 != 0 {
		return true, nil
	}

	return false, nil
}
