package app

import (
	"fmt"
	"go/build"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/cmd/codegen/util"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
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
	
	outputPath := filepath.Join(o.OutputBase, o.OutputPackage)

	// Work out the InputPackage relative to GOROOT
	o.GoPathInputPackage = o.InputPackage

	// Work out the OutputPackage relative to GOROOT
	o.GoPathOutputPackage = strings.TrimPrefix(outputPath,
		fmt.Sprintf("%s/", filepath.Join(build.Default.GOPATH, "src")))
	return nil
}
