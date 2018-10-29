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
	"strings"

	"github.com/jenkins-x/jx/pkg/extensions"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/spf13/cobra"
)

var (
	get_plugins_long = templates.LongDesc(`
		List all available plugin files on a user's PATH.

		Plugins provide extended functionality that is not part of the major command-line distribution.

		Available plugin files are those that are:
		- executable
		- anywhere on the user's PATH
		- begin with "jx-"

		Plugins defined by extensions are automatically installed when the plugin is called.

		Please refer to the documentation and examples for more information about how write your own plugins.

`)
)

type GetPluginsOptions struct {
	CommonOptions
	Verifier extensions.PathVerifier
}

// NewCmdGetPlugins provides a way to list all plugin executables visible to jx
func NewCmdGetPlugins(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetPluginsOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:   "plugins",
		Short: "List all visible plugin executables on a user's PATH",
		Long:  get_plugins_long,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Complete()
			CheckErr(err)
			err = options.Run()

			CheckErr(err)
		},
	}

	return cmd
}

func (o *GetPluginsOptions) Complete() error {
	o.Verifier = &extensions.CommandOverrideVerifier{
		Root:        o.Cmd.Root(),
		SeenPlugins: make(map[string]string, 0),
	}
	return nil
}

func (o *GetPluginsOptions) Run() error {
	return o.PrintExtensionPlugins()
}

func (o *GetPluginsOptions) PrintExtensionPlugins() error {
	pcgs, err := o.getPluginCommandGroups(o.Verifier)
	if err != nil {
		return err
	}
	maxLength := 0
	for _, pcg := range pcgs {
		for _, pc := range pcg.Commands {

			if len(pc.SubCommand) > maxLength {
				maxLength = len(pc.SubCommand)
			}
		}
	}

	for _, pcg := range pcgs {
		log.Infof("%s\n", pcg.Message)
		for _, pc := range pcg.Commands {
			var description string
			url, err := extensions.FindPluginUrl(pc.PluginSpec)
			if err != nil {
				// No-op as this might just be a local plugin
			}
			if pc.Name != "" && pc.Version != "" && url != "" {
				description = fmt.Sprintf("%s (app %s version %s installed from %s)", pc.Description, pc.Name,
					pc.Version, url)
			} else {
				description = pc.Description
			}
			log.Infof("  %s %s%s\n", util.ColorInfo(pc.SubCommand), strings.Repeat(" ", maxLength-len(pc.SubCommand)), description)
		}
		log.Infoln("")
	}

	if len(pcgs) > 0 {
		// Add a trailing line to make the output more readable
		log.Infoln("")
	}
	return nil
}
