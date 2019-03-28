package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

var (
	createMicroLong = templates.LongDesc(`
		Creates a new micro application and then optionally setups CI/CD pipelines and GitOps promotion.

		Micro is an application generator for gRPC services in Go with a set of tools/libraries.

		This command is expected to be run within your '$GOHOME' directory. e.g. at '$GOHOME/src/github.com/myOrgOrUser/'

		For more documentation about micro see: [https://github.com/microio/micro](https://github.com/microio/micro)

	`)

	createMicroExample = templates.Examples(`
		# Create a micro application and be prompted for the folder name
		jx create micro 

		# Create a micro application under test1
		jx create micro -o test1
	`)
)

// CreateMicroOptions the options for the create spring command
type CreateMicroOptions struct {
	CreateProjectOptions
}

// NewCmdCreateMicro creates a command object for the "create" command
func NewCmdCreateMicro(commonOpts *CommonOptions) *cobra.Command {
	options := &CreateMicroOptions{
		CreateProjectOptions: CreateProjectOptions{
			ImportOptions: ImportOptions{
				CommonOptions: commonOpts,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "micro [github.com/myuser/myapp]",
		Short:   "Create a new micro based application and import the generated code into Git and Jenkins for CI/CD",
		Long:    createMicroLong,
		Example: createMicroExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	return cmd
}

// checkMicroInstalled lazily install micro if its not installed already
func (o CreateMicroOptions) checkMicroInstalled() error {
	_, err := o.getCommandOutput("", "micro", "help")
	if err != nil {
		log.Infoln("Installing micro's dependencies...")
		// lets install micro
		err = o.installBrewIfRequired()
		if err != nil {
			return err
		}
		if runtime.GOOS == "darwin" && !o.NoBrew {
			err = o.RunCommand("brew", "install", "protobuf")
			if err != nil {
				return err
			}
		}
		log.Infoln("Downloading and building micro dependencies...")
		packages := []string{"github.com/golang/protobuf/proto", "github.com/golang/protobuf/protoc-gen-go", "github.com/micro/protoc-gen-micro"}
		for _, p := range packages {
			log.Infof("Installing %s\n", p)
			err = o.RunCommand("go", "get", "-u", p)
			if err != nil {
				return fmt.Errorf("Failed to install %s: %s", p, err)
			}
		}
		log.Infoln("Installed micro dependencies")

		log.Infoln("Downloading and building micro - this can take a minute or so...")
		err = o.RunCommand("go", "get", "-u", "github.com/micro/micro")
		if err == nil {
			log.Infoln("Installed micro and its dependencies!")
		}
	}
	return err
}

// GenerateMicro creates a fresh micro project by running micro on local shell
func (o CreateMicroOptions) GenerateMicro(dir string) error {
	return o.RunCommand("micro", "new", dir)
}

// Run implements the command
func (o *CreateMicroOptions) Run() error {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		log.Warnf(`No $GOPATH found. 

You need to have installed go on your machine to be able to create micro services. 

For instructions please see: %s 

`, util.ColorInfo("https://golang.org/doc/install#install"))
		return nil
	}

	err := o.checkMicroInstalled()
	if err != nil {
		return err
	}

	dir := ""
	args := o.Args
	if len(args) > 0 {
		dir = args[0]
	}
	if dir == "" {
		if o.BatchMode {
			return util.MissingOption(optionOutputDir)
		}
		dir, err = util.PickValue("Pick a fully qualified name for the new project:", "github.com/myuser/myapp", true, "", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
		if dir == "" || dir == "." {
			return fmt.Errorf("Invalid project name: %s", dir)
		}
	}
	log.Blank()

	// generate micro project
	err = o.GenerateMicro(dir)
	if err != nil {
		return err
	}

	path := filepath.Join(gopath, "src", dir)
	log.Infof("Created micro project at %s\n\n", util.ColorInfo(path))

	return o.ImportCreatedProject(path)
}
