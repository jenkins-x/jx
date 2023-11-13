// Copyright © 2019 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const descriptionSourcePath = "docs/reference/cmd/"

func generateCliYaml(opts *options) error {
	root := cmd.Main(nil)
	disableFlagsInUseLine(root)
	source := filepath.Join(opts.source, descriptionSourcePath)
	if err := loadLongDescription(root, source); err != nil {
		return err
	}

	switch opts.kind {
	case "markdown":
		return GenMarkdownTree(root, opts.target)
	case "man":
		header := &GenManHeader{
			Section: "1",
		}
		return GenManTree(root, header, opts.target)
	default:
		return fmt.Errorf("invalid docs kind : %s", opts.kind)
	}
}

func disableFlagsInUseLine(cmd *cobra.Command) {
	visitAll(cmd, func(ccmd *cobra.Command) {
		// do not add a `[flags]` to the end of the usage line.
		ccmd.DisableFlagsInUseLine = true
	})
}

// visitAll will traverse all commands from the root.
// This is different from the VisitAll of cobra.Command where only parents
// are checked.
func visitAll(root *cobra.Command, fn func(*cobra.Command)) {
	for _, c := range root.Commands() {
		visitAll(c, fn)
	}
	fn(root)
}

func loadLongDescription(cmd *cobra.Command, path ...string) error {
	for _, c := range cmd.Commands() {
		if c.Name() == "" {
			continue
		}
		fullpath := filepath.Join(path[0], strings.Join(append(path[1:], c.Name()), "_")+".md")
		if c.HasSubCommands() {
			if err := loadLongDescription(c, path[0], c.Name()); err != nil {
				return err
			}
		}

		if _, err := os.Stat(fullpath); err != nil {
			log.Printf("WARN: %s does not exist, skipping\n", fullpath)
			continue
		}

		content, err := os.ReadFile(fullpath)
		if err != nil {
			return err
		}
		description, examples := parseMDContent(string(content))
		c.Long = description
		c.Example = examples
	}
	return nil
}

type options struct {
	source string
	target string
	kind   string
}

func parseArgs() (*options, error) {
	opts := &options{}
	cwd, _ := os.Getwd()
	flags := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	flags.StringVar(&opts.source, "root", cwd, "Path to project root")
	flags.StringVar(&opts.target, "target", "/tmp", "Target path for generated yaml files")
	flags.StringVar(&opts.kind, "kind", "markdown", "Kind of docs to generate (supported: man, markdown)")
	err := flags.Parse(os.Args[1:])
	return opts, err
}

func parseMDContent(mdString string) (description, examples string) {
	parsedContent := strings.Split(mdString, "\n## ")
	for _, s := range parsedContent {
		if strings.Index(s, "Description") == 0 {
			description = strings.TrimSpace(strings.TrimPrefix(s, "Description"))
		}
		if strings.Index(s, "Examples") == 0 {
			examples = strings.TrimSpace(strings.TrimPrefix(s, "Examples"))
		}
	}
	return description, examples
}

func main() {
	opts, err := parseArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
	if opts != nil {
		fmt.Printf("Project root: %s\n", opts.source)
		fmt.Printf("Generating yaml files into %s\n", opts.target)

		if err := generateCliYaml(opts); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate yaml files: %s\n", err.Error())
		}
	}
}
