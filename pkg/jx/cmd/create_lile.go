package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

const (
	optionOutputDir = "output-dir"
)

var (
	createLileLong = templates.LongDesc(`
		Creates a new Lile application and then optionally setups CI/CD pipelines and GitOps promotion.

		Lile is an application generator for gRPC services in Go with a set of tools/libraries.

		This command is expected to be run within your '$GOHOME' directory. e.g. at '$GOHOME/src/github.com/myOrgOrUser/'

		For more documentation about Lile see: [https://github.com/lileio/lile](https://github.com/lileio/lile)

	`)

	createLileExample = templates.Examples(`
		# Create a Lile application and be prompted for the folder name
		jx create lile 

		# Create a Lile application under test1
		jx create lile -o test1
	`)
)

// CreateLileOptions the options for the create spring command
type CreateLileOptions struct {
	CreateProjectOptions
	OutDir string
}

// NewCmdCreateLile creates a command object for the "create" command
func NewCmdCreateLile(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &CreateLileOptions{
		CreateProjectOptions: CreateProjectOptions{
			ImportOptions: ImportOptions{
				CommonOptions: commonOpts,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "lile",
		Short:   "Create a new Lile based application and import the generated code into Git and Jenkins for CI/CD",
		Long:    createLileLong,
		Example: createLileExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.OutDir, optionOutputDir, "o", "", "Relative directory to output the project to. Defaults to current directory")

	return cmd
}

// checkLileInstalled lazily install lile if its not installed already
func (o CreateLileOptions) checkLileInstalled() error {
	_, err := o.GetCommandOutput("", "lile", "help")
	if err != nil {
		log.Logger().Info("Installing Lile's dependencies...")
		// lets install lile
		err = o.InstallBrewIfRequired()
		if err != nil {
			return err
		}
		if runtime.GOOS == "darwin" && !o.NoBrew {
			err = o.RunCommand("brew", "install", "protobuf")
			if err != nil {
				return err
			}
		}

		log.Logger().Info("Downloading and building Lile - this can take a while...")
		err = o.RunCommand("go", "get", "-u", "github.com/lileio/lile/...")
		if err == nil {
			log.Logger().Info("Installed Lile and its dependencies!")
		}
	}
	return err
}

// GenerateLile creates a fresh Lile project by running lile on local shell
func (o CreateLileOptions) GenerateLile(dir string) error {
	var cmdOut bytes.Buffer
	e := exec.Command("lile", "new", dir)
	e.Env = os.Environ()
	e.Env = append(e.Env, "CI=do_not_prompt")
	e.Stdout = &cmdOut
	e.Stderr = o.Err
	err := e.Run()
	return err
}

// Run implements the command
func (o *CreateLileOptions) Run() error {
	err := o.checkLileInstalled()
	if err != nil {
		return err
	}

	dir := o.OutDir
	if dir == "" {
		if o.BatchMode {
			return util.MissingOption(optionOutputDir)
		}
		dir, err = util.PickValue("Pick a name for the new project:", "myapp", true, "", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
		if dir == "" || dir == "." {
			return fmt.Errorf("Invalid project name: %s", dir)
		}
	}

	// generate Lile project
	err = o.GenerateLile(dir)
	if err != nil {
		return err
	}

	log.Logger().Infof("Created Lile project at %s\n\n", util.ColorInfo(dir))

	return o.ImportCreatedProject(dir)
}
