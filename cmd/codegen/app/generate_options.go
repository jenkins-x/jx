package app

import (
	"fmt"
	"go/build"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/cmd/codegen/util"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/pkg/errors"
)

const (
	optionGroupWithVersion = "group-with-version"
	optionInputPackage     = "input-package"
	optionOutputPackage    = "output-package"

	optionInputBase       = "input-base"
	optionOutputBase      = "output-base"
	optionBoilerplateFile = "boilerplate-file"
	optionModuleName      = "module-name"
)

// GenerateOptions contain common code generation options
type GenerateOptions struct {
	*opts.CommonOptions
	OutputBase          string
	BoilerplateFile     string
	GroupsWithVersions  []string
	InputPackage        string
	GoPathInputPackage  string
	GoPathOutputPackage string
	GoPathOutputBase    string
	OutputPackage       string
	ClientGenVersion    string
	InputBase           string
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
