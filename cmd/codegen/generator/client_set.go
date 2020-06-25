package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/v2/cmd/codegen/util"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/ast/astutil"
)

const (
	basePath       = "k8s.io/code-generator/cmd"
	defaultVersion = "kubernetes-1.11.3"
)

var (
	// AllGenerators is a list of all the generators provide by kubernetes code-generator
	allGenerators = map[string]string{
		"clientset": "client-gen",
		"defaulter": "defaulter-gen",
		"listers":   "lister-gen",
		"informers": "informer-gen",
		"deepcopy":  "deepcopy-gen",
	}
)

// InstallCodeGenerators installs client-gen from the kubernetes-incubator/reference-docs repository.
func InstallCodeGenerators(version string, gopath string) error {
	if version == "" {
		version = defaultVersion
	}
	path := fmt.Sprintf("%s/...", basePath)
	util.AppLogger().Infof("installing %s version %s into %s", path, version, gopath)
	err := util.GoGet(path, version, gopath, true, false, false)
	if err != nil {
		return err
	}

	return nil
}

// GenerateClient runs the generators specified. It looks at the specified groupsWithVersions in inputPackage and generates to outputPackage (
//// relative to the module outputBase). A boilerplateFile is written to the top of any generated files.
func GenerateClient(generators []string, groupsWithVersions []string, inputPackage string, outputPackage string,
	relativePackage string, outputBase string, boilerplateFile string, gopath string, semVer string) error {
	for _, generatorName := range generators {
		if generator, ok := allGenerators[generatorName]; ok {
			switch generatorName {
			case "clientset":
				err := generateWithOutputPackage(generator,
					generatorName,
					groupsWithVersions,
					inputPackage,
					outputPackage,
					outputBase,
					boilerplateFile,
					gopath,
					"--clientset-name",
					"versioned")
				if err != nil {
					return err
				}
				if semVer != "" {
					oldPkg := filepath.Join(outputPackage, generatorName)
					csDir := filepath.Join(outputBase, oldPkg)
					svPkg := strings.ReplaceAll(oldPkg, fmt.Sprintf("/%s", relativePackage), fmt.Sprintf("/%s/%s", semVer, relativePackage))
					err = fixClientImportsForSemVer(csDir, oldPkg, svPkg)
					if err != nil {
						return errors.Wrapf(err, "updating clientset imports for semver")
					}
				}
			case "deepcopy":
				err := defaultGenerate(generator,
					generatorName,
					groupsWithVersions,
					inputPackage,
					outputPackage,
					outputBase,
					boilerplateFile,
					gopath,
					"-O",
					"zz_generated.deepcopy",
					"--bounding-dirs",
					inputPackage)
				if err != nil {
					return err
				}
				// If we have a semver, copy the semver's pkg directory contents to the normal pkg and delete the semver directory
				if semVer != "" {
					wd, err := os.Getwd()
					if err != nil {
						return errors.Wrapf(err, "getting working directory")
					}
					pkgs, err := packagesForGroupsWithVersions(inputPackage, groupsWithVersions)
					if err != nil {
						return err
					}
					for _, p := range pkgs {
						svFile := filepath.Join(outputBase, inputPackage, p, "zz_generated.deepcopy.go")
						pkgFile := strings.ReplaceAll(svFile, fmt.Sprintf("/%s/", semVer), "/")
						exists, err := util.FileExists(svFile)
						if err != nil {
							return errors.Wrapf(err, "checking if file %s exists", svFile)
						}
						if exists {
							err = util.CopyFile(svFile, pkgFile)
							if err != nil {
								return errors.Wrapf(err, "copying %s to %s", svFile, pkgFile)
							}
						}
					}
					err = os.RemoveAll(filepath.Join(wd, semVer))
				}
			case "informers":
				basePkg := outputPackage
				if semVer != "" {
					basePkg = strings.ReplaceAll(basePkg, fmt.Sprintf("/%s", relativePackage), fmt.Sprintf("/%s/%s", semVer, relativePackage))
				}
				err := generateWithOutputPackage(generator,
					generatorName,
					groupsWithVersions,
					inputPackage,
					outputPackage,
					outputBase,
					boilerplateFile,
					gopath,
					"--versioned-clientset-package",
					fmt.Sprintf("%s/clientset/versioned", basePkg),
					"--listers-package",
					fmt.Sprintf("%s/listers", basePkg))
				if err != nil {
					return err
				}
				if semVer != "" {
					oldPkg := filepath.Join(outputPackage, generatorName)
					infDir := filepath.Join(outputBase, oldPkg)
					svPkg := strings.ReplaceAll(oldPkg, fmt.Sprintf("/%s", relativePackage), fmt.Sprintf("/%s/%s", semVer, relativePackage))
					err = fixClientImportsForSemVer(infDir, oldPkg, svPkg)
					if err != nil {
						return errors.Wrapf(err, "updating informer imports for semver")
					}
				}
			default:
				err := generateWithOutputPackage(generator, generatorName, groupsWithVersions, inputPackage,
					outputPackage, outputBase, boilerplateFile, gopath)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.Errorf("no generator %s available", generatorName)
		}
	}
	return nil
}

// GetBoilerplateFile is responsible for canonicalizing the name of the boilerplate file
func GetBoilerplateFile(fileName string) (string, error) {
	if fileName != "" {
		if _, err := os.Stat(fileName); err != nil && !os.IsNotExist(err) {
			return "", errors.Wrapf(err, "error reading boilerplate file %s", fileName)
		} else if err == nil {
			abs, err := filepath.Abs(fileName)
			if err == nil {
				fileName = abs
			} else {
				util.AppLogger().Errorf("error converting %s to absolute path so leaving as is %v", fileName, err)
			}
		}
	}
	return fileName, nil
}

func generateWithOutputPackage(generator string, name string, groupsWithVersions []string,
	inputPackage string, outputPackage string, outputBase string, boilerplateFile string, gopath string, args ...string) error {
	args = append(args, "--output-package", fmt.Sprintf("%s/%s", outputPackage, name))
	return defaultGenerate(generator, name, groupsWithVersions, inputPackage, outputPackage, outputBase,
		boilerplateFile, gopath, args...)
}

func fixClientImportsForSemVer(clientDir string, oldPackage string, semVerPackage string) error {
	return filepath.Walk(clientDir, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".go") {
			fs := token.NewFileSet()
			f, err := parser.ParseFile(fs, path, nil, parser.ParseComments)
			if err != nil {
				return errors.Wrapf(err, "parsing %s", path)
			}
			importsToReplace := make(map[string]string)

			for _, imp := range f.Imports {
				existingImp := strings.Replace(imp.Path.Value, `"`, ``, 2)
				if strings.HasPrefix(existingImp, oldPackage) {
					importsToReplace[existingImp] = strings.ReplaceAll(existingImp, oldPackage, semVerPackage)
				}
			}
			if len(importsToReplace) > 0 {
				for oldPkg, newPkg := range importsToReplace {
					astutil.RewriteImport(fs, f, oldPkg, newPkg)
				}
				// Sort the imports
				ast.SortImports(fs, f)
				var buf bytes.Buffer
				err = format.Node(&buf, fs, f)
				if err != nil {
					return errors.Wrapf(err, "convert AST to []byte for %s", path)
				}
				err = ioutil.WriteFile(path, buf.Bytes(), 0600)
				if err != nil {
					return errors.Wrapf(err, "writing %s", path)
				}
			}
		}
		return nil
	})
}
