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

var defaultIgnores = []string{
	"templates/*",
}

//GenerateValues will generate a values.yaml file in dir. It scans all subdirectories for values.yaml files,
// and merges them into the values.yaml in the root directory,
// creating a nested key structure that matches the directory structure.
// Any keys used that match files with the same name in the directory (
// and have empty values) will be inlined as block scalars.
// Standard UNIX glob patterns can be passed to ignore directories.
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
		ignores = defaultIgnores
	}
	files := make(map[string]map[string]string)
	values := make(map[string]interface{})
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		rPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		// Check if should ignore the path
		if ignore, err := ignore(rPath, ignores); err != nil {
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

	for p, v := range values {
		// First, do file substitution - but only if any files were actually found
		if dirFiles := files[p]; dirFiles != nil && len(dirFiles) > 0 {
			err := recurse(v, dirFiles, "$")
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

func recurse(element interface{}, files map[string]string, jsonPath string) error {
	if e, ok := element.(map[string]interface{}); ok {
		for k, v := range e {
			if path, ok := files[k]; ok {
				if v == nil || util.IsZeroOfUnderlyingType(v) {
					// There is a filename in the directory structure that matches this key, and it has no value,
					// so we assign it
					b, err := ioutil.ReadFile(path)
					if err != nil {
						return err
					}
					e[k] = string(b)
				} else {
					return fmt.Errorf("value at %s must be empty but is %v", jsonPath, v)
				}
			} else {
				// keep on recursing
				jsonPath = fmt.Sprintf("%s.%s", jsonPath, k)
				err := recurse(v, files, jsonPath)
				if err != nil {
					return err
				}
			}
		}
	}
	// If it's not an object, we can't do much with it
	return nil
}

func ignore(path string, ignores []string) (bool, error) {
	for _, ignore := range ignores {
		if matched, err := filepath.Match(ignore, path); err != nil {
			return false, err
		} else if matched {
			return true, nil
		}
	}
	return false, nil
}
