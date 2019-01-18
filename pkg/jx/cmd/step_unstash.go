package cmd

import (
	"github.com/jenkins-x/jx/pkg/cloud/buckets"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

// StepUnstashOptions contains the command line flags
type StepUnstashOptions struct {
	StepOptions

	URL    string
	OutDir string
	Timeout time.Duration
}

var (
	stepUnstashLong = templates.LongDesc(`
		This pipeline step unstashes the files in storage to a local file or the console
` + storageSupportDescription + SeeAlsoText("jx step stash", "jx edit storage"))

	stepUnstashExample = templates.Examples(`
		# unstash a file to the reports directory
		jx step unstash --url s3://mybucket/tests/myOrg/myRepo/mybranch/3/junit.xml -o reports

		# unstash the file to the from GCS to the console
		jx step unstash -u gs://mybucket/foo/bar/output.log
`)
)

// NewCmdStepUnstash creates the CLI command
func NewCmdStepUnstash(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := StepUnstashOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "unstash",
		Short:   "Unstashes files generated as part of a pipeline to a local file or directory or displays on the console",
		Aliases: []string{"collect"},
		Long:    stepUnstashLong,
		Example: stepUnstashExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.URL, "url", "u", "", "The fully qualified URL to the file to unstash including the storage host, path and file name")
	cmd.Flags().StringVarP(&options.OutDir, "output", "o", "", "The output file or directory")
	cmd.Flags().DurationVarP(&options.Timeout, "timeout", "t", time.Second * 30, "The timeout period before we should fail unstashing the entry")
	return cmd
}

// Run runs the command
func (o *StepUnstashOptions) Run() error {
	u := o.URL
	if u == "" {
		// TODO lets guess from the project etc...
		return util.MissingOption("url")
	}
	file := o.OutDir
	if file != "" {
		isDir, err := util.DirExists(file)
		if err != nil {
		  return errors.Wrapf(err, "failed to check if %s is a directory", file)
		}
		if isDir {
			u2, err := url.Parse(u)
			if err != nil {
			  return errors.Wrapf(err, "failed to parse URL %s", u)
			}
			name := u2.Path
			if name == "" || strings.HasSuffix(name, "/") {
				name += "output.txt"
			}
			file = filepath.Join(file, name)
		}
	}

	data, err := buckets.ReadURL(u, o.Timeout)
	if err != nil {
	  return err
	}
	if file == "" {
		log.Infof("%s\n", string(data))
		return nil
	}
	err = ioutil.WriteFile(file, data, util.DefaultWritePermissions)
	if err != nil {
	  return errors.Wrapf(err, "failed to write file %s", file)
	}
	log.Infof("wrote: %s\n", util.ColorInfo(file))
	return nil
}
