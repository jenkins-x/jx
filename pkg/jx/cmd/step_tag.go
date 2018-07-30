package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

const (
	VERSION = "version"

	defaultVersionFile = "VERSION"
)

// CreateClusterOptions the flags for running create cluster
type StepTagOptions struct {
	StepOptions

	Flags StepTagFlags
}

type StepTagFlags struct {
	Version     string
	VersionFile string
}

var (
	stepTagLong = templates.LongDesc(`
		This pipeline step command creates a git tag using a version number prefixed with 'v' and pushes it to a
		remote origin repo.

		This commands effectively runs:

		git commit -a -m "release $(VERSION)" --allow-empty
		git tag -fa v$(VERSION) -m "Release version $(VERSION)"
		git push origin v$(VERSION)

`)

	stepTagExample = templates.Examples(`

		jx step tag --version 1.0.0

`)
)

func NewCmdStepTag(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {

	options := StepTagOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "tag",
		Short:   "Creates a git tag and pushes to remote repo",
		Long:    stepTagLong,
		Example: stepTagExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Flags.Version, VERSION, "v", "", "version number for the tag [required]")
	cmd.Flags().StringVarP(&options.Flags.VersionFile, "version-file", "", defaultVersionFile, "The file name used to load the version number from if no '--version' option is specified")

	return cmd
}

func (o *StepTagOptions) Run() error {
	if o.Flags.Version == "" {
		// lets see if its defined in the VERSION file
		path := o.Flags.VersionFile
		if path == "" {
			path = "VERSION"
		}
		exists, err := util.FileExists(path)
		if exists && err == nil {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			o.Flags.Version = string(data)
		}
	}
	if o.Flags.Version == "" {
		return errors.New("No version flag")
	}

	tag := "v" + o.Flags.Version

	err := o.Git().AddCommmit("", fmt.Sprintf("release %s", o.Flags.Version))
	if err != nil {
		return err
	}

	err = o.Git().CreateTag("", tag, fmt.Sprintf("release %s", o.Flags.Version))
	if err != nil {
		return err
	}

	err = o.Git().PushTag("", tag)
	if err != nil {
		return err
	}

	log.Successf("Tag %s created and pushed to remote origin", tag)
	return nil
}
