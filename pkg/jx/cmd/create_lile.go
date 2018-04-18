package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
)

const (
	optionOutputDir = "output-dir"
)

var (
	createLileLong = templates.LongDesc(`
		Creates a new lile application and then optionally setups CI/CD pipelines and GitOps promotion.

		Lile is an application generator for gRPC services in Go with a set of tools/libraries.

		This command is expected to be run within your '$GOHOME' directory. e.g. at '$GOHOME/src/github.com/myOrgOrUser/'

		For more documentation about lile see: [https://github.com/lileio/lile](https://github.com/lileio/lile)

	`)

	createLileExample = templates.Examples(`
		# Create a lile application and be prompted for the folder name
		jx create lile 

		# Create a lile application under test1
		jx create lile -o test1
	`)
)

// CreateLileOptions the options for the create spring command
type CreateLileOptions struct {
	CreateProjectOptions
	OutDir string
}

// NewCmdCreateLile creates a command object for the "create" command
func NewCmdCreateLile(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateLileOptions{
		CreateProjectOptions: CreateProjectOptions{
			ImportOptions: ImportOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "lile",
		Short:   "Create a new lile based application and import the generated code into git and Jenkins for CI/CD",
		Long:    createLileLong,
		Example: createLileExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.OutDir, optionOutputDir, "o", "", "Relative directory to output the project to. Defaults to current directory")

	return cmd
}

// checkLileInstalled lazily install lile if its not installed already
func (o CreateLileOptions) checkLileInstalled() error {
	_, err := o.getCommandOutput("", "lile", "help")
	if err != nil {
		o.Printf("Installing lile's dependencies...\n")
		// lets install lile
		err = o.installBrewIfRequired()
		if err != nil {
			return err
		}
		if runtime.GOOS == "darwin" && !o.NoBrew {
			err = o.runCommand("brew", "install", "protobuf")
			if err != nil {
				return err
			}
		}

		o.Printf("Downloading and building lile - this can take a while...\n")
		err = o.runCommand("go", "get", "-u", "github.com/lileio/lile/...")
		if err == nil {
			o.Printf("Installed lile and its dependencies!\n")
		}
	}
	return err
}

// GenerateLile creates a fresh lile project by running lile on local shell
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
		dir, err = util.PickValue("Pick a name for the new project:", "myapp", true)
		if err != nil {
			return err
		}
		if dir == "" || dir == "." {
			return fmt.Errorf("Invalid project name: %s", dir)
		}
	}

	// generate lile project
	err = o.GenerateLile(dir)
	if err != nil {
		return err
	}

	o.Printf("Created lile project at %s\n\n", util.ColorInfo(dir))

	return o.ImportCreatedProject(dir)
}
