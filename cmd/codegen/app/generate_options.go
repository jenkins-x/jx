package app

import (
	"fmt"
	"go/build"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/cmd/codegen/util"
)

const (
	optionGroupWithVersion = "group-with-version"
	optionInputPackage     = "input-package"
	optionOutputPackage    = "output-package"

	optionInputBase       = "input-base"
	optionOutputBase      = "output-base"
	optionBoilerplateFile = "boilerplate-file"
	optionModuleName      = "module-name"
	global                = "global"
)

// CommonOptions contains the common options
type CommonOptions struct {
	Args             []string
	Cmd              *cobra.Command
	LogLevel         string
	GeneratorVersion string
}

// GenerateOptions contain common code generation options
type GenerateOptions struct {
	*CommonOptions
	OutputBase          string
	BoilerplateFile     string
	GroupsWithVersions  []string
	InputPackage        string
	GoPathInputPackage  string
	GoPathOutputPackage string
	GoPathOutputBase    string
	OutputPackage       string
	InputBase           string
	Global              bool
}

func (o *GenerateOptions) configure() error {
	err := util.EnsureGoPath()
	if err != nil {
		return err
	}

	inputPath := filepath.Join(o.InputBase, o.InputPackage)
	outputPath := filepath.Join(o.OutputBase, o.OutputPackage)

	if !strings.HasPrefix(inputPath, build.Default.GOPATH) {
		return errors.Errorf("input %s is not in GOPATH (%s)", inputPath, build.Default.GOPATH)
	}

	if !strings.HasPrefix(outputPath, build.Default.GOPATH) {
		return errors.Errorf("output %s is not in GOPATH (%s)", outputPath, build.Default.GOPATH)
	}

	// Work out the InputPackage relative to GOROOT
	o.GoPathInputPackage = strings.TrimPrefix(inputPath,
		fmt.Sprintf("%s/", filepath.Join(build.Default.GOPATH, "src")))

	// Work out the OutputPackage relative to GOROOT
	o.GoPathOutputPackage = strings.TrimPrefix(outputPath,
		fmt.Sprintf("%s/", filepath.Join(build.Default.GOPATH, "src")))
	return nil
}
