package jenkinsfile

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
)

const (
	// ModuleFileName the name of the module imports file name
	ModuleFileName = "imports.yaml"
)

// ImportFile represents an import of a file from a module (usually a version of a git repo)
type ImportFile struct {
	Import string
	File   string
}

// ImportFileResolver resolves a build pack file resolver strategy
type ImportFileResolver func(importFile *ImportFile) (string, error)

// Modules defines the dependent modules for a build pack
type Modules struct {
	Modules []*Module `yaml:"modules,omitempty"`
}

// Module defines a dependent module for a build pack
type Module struct {
	Name   string `yaml:"name,omitempty"`
	GitURL string `yaml:"gitUrl,omitempty"`
	GitRef string `yaml:"gitRef,omitempty"`
}


// ModulesResolver resolves a number of modules into a structure we can use to resolve imports
type ModulesResolver struct {
	Modules map[string]*ModuleResolver
}

// ModuleResolver a resolver for a single module
type ModuleResolver struct {
	Module   *Module
	PacksDir string
}


// LoadModules loads the modules in the given build pack directory if there are any
func LoadModules(dir string) (*Modules, error) {
	fileName := filepath.Join(dir, ModuleFileName)
	config := Modules{}
	exists, err := util.FileExists(fileName)
	if err != nil || !exists {
		return &config, err
	}
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return &config, fmt.Errorf("Failed to load file %s due to %s", fileName, err)
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return &config, fmt.Errorf("Failed to unmarshal YAML file %s due to %s", fileName, err)
	}
	return &config, err
}

// Resolve resolves this module to a directory
func (m *Module) Resolve(gitter gits.Gitter) (*ModuleResolver, error) {
	err := m.Validate()
	if err != nil {
	  return nil, err
	}

	dir, err := InitBuildPack(gitter, m.GitURL, m.GitRef)
	if err != nil {
		return nil, err
	}

	answer := &ModuleResolver{
		Module: m,
		PacksDir: dir,
	}
	return answer, nil
}

// Validate returns an error if any data is missing
func (m *Module) Validate() error {
	if m.GitURL == "" {
		return util.MissingOption("GitURL")
	}
	return nil
}


// Resolve Resolve the modules into a module resolver
func (m *Modules) Resolve(gitter gits.Gitter) (*ModulesResolver, error) {
	answer := &ModulesResolver{
		Modules: map[string]*ModuleResolver{},
	}
	for _, mod := range m.Modules {
		resolver, err := mod.Resolve(gitter)
		if err != nil {
			return answer, err
		}
		answer.Modules[mod.Name] = resolver
	}
	return answer, nil
}

// AsImportResolver returns an ImportFileResolver for these modules
func (m *ModulesResolver) AsImportResolver() ImportFileResolver {
	return m.ResolveImport
}

// ResolveImport resolves an import relative to the local git clone of the import
func (m *ModulesResolver) ResolveImport(importFile *ImportFile) (string, error) {
	resolver := m.Modules[importFile.Import]
	if resolver == nil {
		return "", nil
	}
	return filepath.Join(resolver.PacksDir, importFile.File), nil
}
