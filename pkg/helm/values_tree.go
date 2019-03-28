package helm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"

	"github.com/ghodss/yaml"

	"github.com/jenkins-x/jx/pkg/log"
)

//DefaultValuesTreeIgnores is the default set of ignored files for collapsing the values tree which are used if
// ignores is nil
var DefaultValuesTreeIgnores = []string{
	"templates/*",
}

//GenerateValues will generate a values.yaml file in dir. It scans all subdirectories for values.yaml files,
// and merges them into the values.yaml in the root directory,
// creating a nested key structure that matches the directory structure.
// Any keys used that match files with the same name in the directory (
// and have empty values) will be inlined as block scalars.
// Standard UNIX glob patterns can be passed to IgnoreFile directories.
func GenerateValues(dir string, ignores []string, verbose bool) ([]byte, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	} else if os.IsNotExist(err) {
		return nil, fmt.Errorf("%s does not exist", dir)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}
	if ignores == nil {
		ignores = DefaultValuesTreeIgnores
	}
	files := make(map[string]map[string]string)
	values := make(map[string]interface{})
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		rPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		// Check if should IgnoreFile the path
		if ignore, err := util.IgnoreFile(rPath, ignores); err != nil {
			return err
		} else if !ignore {
			rDir, file := filepath.Split(rPath)
			// For the root dir we just consider directories (which the walk func does for us)
			if rDir != "" {
				// If it's values.yaml, then read and parse it
				if file == "values.yaml" {
					b, err := ioutil.ReadFile(path)
					if err != nil {
						return err
					}
					v := make(map[string]interface{})

					err = yaml.Unmarshal(b, &v)
					if err != nil {
						return err
					}
					values[rDir] = v
				} else {
					// for other files, just store a reference
					if _, ok := files[rDir]; !ok {
						files[rDir] = make(map[string]string)
					}
					files[rDir][file] = path
				}
			}
		} else {
			if verbose {
				log.Infof("Ignoring %s\n", rPath)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Load the root values.yaml
	rootValuesFileName := filepath.Join(dir, ValuesFileName)
	rootValues, err := LoadValuesFile(rootValuesFileName)
	if err != nil {
		return nil, err
	}

	// externalFileHandler is used to read any inline any files that match into the values.yaml
	externalFileHandler := func(path string, element map[string]interface{}, key string) error {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		element[key] = string(b)
		return nil
	}
	for p, v := range values {
		// First, do file substitution - but only if any files were actually found
		if dirFiles := files[p]; dirFiles != nil && len(dirFiles) > 0 {
			err := HandleExternalFileRefs(v, dirFiles, "", externalFileHandler)
			if err != nil {
				return nil, err
			}
		}

		// Then, merge the values to the root file
		keys := strings.Split(strings.TrimSuffix(p, "/"), string(os.PathSeparator))
		x := rootValues
		jsonPath := "$"
		for i, k := range keys {
			k = strings.TrimSuffix(k, "/")
			jsonPath = fmt.Sprintf("%s.%s", jsonPath, k)
			v1, ok1 := x[k]
			if i < len(keys)-1 {
				// Create the nested file object structure
				if !ok1 {
					// Easy, just create the nested object!
					new := make(map[string]interface{})
					x[k] = new
					x = new
				} else {
					// Need to do a type check
					v2, ok2 := v1.(map[string]interface{})

					if !ok2 {
						return nil, fmt.Errorf("%s is not an associative array", jsonPath)
					}
					x = v2
				}
			} else {
				// Apply
				x[k] = v
			}
		}
	}
	return yaml.Marshal(rootValues)
}

// HandleExternalFileRefs recursively scans the element map structure,
// looking for nested maps. If it finds keys that match any key-value pair in possibles it will call the handler.
// The jsonPath is used for referencing the path in the map structure when reporting errors.
func HandleExternalFileRefs(element interface{}, possibles map[string]string, jsonPath string,
	handler func(path string, element map[string]interface{}, key string) error) error {
	if jsonPath == "" {
		// set zero value
		jsonPath = "$"
	}
	if e, ok := element.(map[string]interface{}); ok {
		for k, v := range e {
			if paths, ok := possibles[k]; ok {
				if v == nil || util.IsZeroOfUnderlyingType(v) {
					// There is a filename in the directory structure that matches this key, and it has no value,
					// so we handle it
					err := handler(paths, e, k)
					if err != nil {
						return err
					}
				} else {
					return fmt.Errorf("value at %s must be empty but is %v", jsonPath, v)
				}
			} else {
				// keep on recursing
				jsonPath = fmt.Sprintf("%s.%s", jsonPath, k)
				err := HandleExternalFileRefs(v, possibles, jsonPath, handler)
				if err != nil {
					return err
				}
			}
		}
	}
	// If it's not an object, we can't do much with it
	return nil
}
