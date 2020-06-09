package create

import (
	"bytes"
	"fmt"
	"github.com/jenkins-x/jx/v2/pkg/log"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jenkins-x/jx/v2/pkg/cmd/create/options"
	"github.com/jenkins-x/jx/v2/pkg/cmd/deprecation"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/spf13/cobra/doc"

	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"

	"github.com/spf13/cobra"
)

const (
	gendocFrontmatterTemplate = `---
date: %s
title: "%s"
slug: %s
url: %s
description: %s
---
%s`
)

var (
	create_docs_long = templates.LongDesc(`
		Creates the documentation markdown files
`)

	create_docs_example = templates.Examples(`
		# Create the documentation files
		jx create docs
	`)
)

// CreateDocsOptions the options for the create spring command
type CreateDocsOptions struct {
	options.CreateOptions

	Dir string
}

// NewCmdCreateDocs creates a command object for the "create" command
func NewCmdCreateDocs(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &CreateDocsOptions{
		CreateOptions: options.CreateOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "docs",
		Short:   "Creates the documentation files",
		Aliases: []string{"doc"},
		Long:    create_docs_long,
		Example: create_docs_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run(cmd.Root())
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Dir, "dir", "d", ".", "the directory to generate the docs into")

	return cmd
}

// Run implements the command
func (o *CreateDocsOptions) Run(jxCommand *cobra.Command) error {
	dir := o.Dir

	exists, err := util.DirExists(dir)
	if err != nil {
		return errors.Wrapf(err, "checking whether the file %q exists", dir)
	}
	if !exists {
		err := os.Mkdir(dir, util.DefaultWritePermissions)
		if err != nil {
			return fmt.Errorf("failed to create %s: %s", dir, err)
		}
	}

	now := time.Now().Format(time.RFC3339)
	prepender := func(filename string) string {
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, path.Ext(name))
		url := "/commands/" + strings.ToLower(base) + "/"
		commandName := strings.Replace(base, "_", " ", -1)
		key := strings.Replace(commandName, "jx ", "", -1)
		deprecationNotice := ""
		log.Logger().Infof("Checking '%s' with key '%s'", commandName, key)
		dc, ok := deprecation.DeprecatedCommands[key]
		if ok {
			log.Logger().Infof("Command '%s' is deprecated", commandName)
			deprecationNotice = "### This command is deprecated\n\n"
			if dc.Replacement  != "" {
				deprecationNotice += fmt.Sprintf("It is recommended that you use '%s' instead\n\n", dc.Replacement)
			}
			if dc.Date != ""  {
				deprecationNotice += fmt.Sprintf("This command will be removed on '%s'\n\n", dc.Date)
			}
			if dc.Info != "" {
				deprecationNotice += fmt.Sprintf("%s\n\n", dc.Info)
			}
		}
		return fmt.Sprintf(gendocFrontmatterTemplate, now, commandName, base, url, "list of jx commands", deprecationNotice)
	}

	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, path.Ext(name))
		return "/commands/" + strings.ToLower(base) + "/"
	}

	if err := o.genMarkdownDeprecation(jxCommand, dir, now); err != nil {
		return errors.Wrapf(err, "generating the deprecation doc")
	}
	return doc.GenMarkdownTreeCustom(jxCommand, dir, prepender, linkHandler)
}

func (o *CreateDocsOptions) genMarkdownDeprecation(cmd *cobra.Command, dir string, date string) error {
	basename := "deprecation"
	filename := filepath.Join(dir, basename+".md")
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	header := fmt.Sprintf(gendocFrontmatterTemplate, date, "deprecated commands",
		basename, "/commands/"+strings.ToLower(basename)+"/", "list of jx commands which have been deprecated","")

	if _, err := io.WriteString(f, header); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	buf.WriteString("\n\n")
	buf.WriteString("## Deprecated Commands\n\n")
	buf.WriteString("\n\n")
	buf.WriteString("| Command        | Removal Date   | Replacement  |\n")
	buf.WriteString("|----------------|----------------|--------------|\n")
	o.genMarkdownTableRows(cmd, buf)

	_, err = buf.WriteTo(f)
	if err != nil {
		return err
	}

	return nil
}

//nolint:errcheck
func (o *CreateDocsOptions) genMarkdownTableRows(cmd *cobra.Command, buf io.StringWriter) {
	if cmd.Deprecated != "" {
		buf.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
			cmd.CommandPath(),
			deprecation.GetRemovalDate(cmd),
			deprecation.GetReplacement(cmd),
		))
		return
	}

	for _, c := range cmd.Commands() {
		o.genMarkdownTableRows(c, buf)
	}
}
