package cmd

import (
	"io"

	"errors"

	"fmt"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

const (
	VERSION = "version"
)

// CreateClusterOptions the flags for running create cluster
type StepTagOptions struct {
	StepOptions

	Flags StepTagFlags
}

type StepTagFlags struct {
	Version string
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

func NewCmdStepTag(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {

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
			cmdutil.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Flags.Version, VERSION, "v", "", "version number for the tag [required]")

	return cmd
}

func (o *StepTagOptions) Run() error {
	if o.Flags.Version == "" {
		return errors.New("No version flag")
	}

	tag := "v" + o.Flags.Version

	err := gits.GitCmd("", "commit", "-a", "-m", fmt.Sprintf("release %s", o.Flags.Version), "--allow-empty")
	if err != nil {
		return err
	}

	err = gits.GitCmd("", "tag", "-fa", tag, "-m", fmt.Sprintf("release %s", o.Flags.Version))
	if err != nil {
		return err
	}

	err = gits.GitCmd("", "push", "origin", tag)
	if err != nil {
		return err
	}

	log.Successf("Tag %s created and pushed to remote origin", tag)
	return nil
}
