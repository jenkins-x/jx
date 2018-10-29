package extensions

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	jenkinsv1client "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jenkins-x/jx/pkg/log"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/jenkins-x/jx/pkg/util"

	"github.com/spf13/cobra"
)

const (
	PluginCommandLabel = "jenkins.io/pluginCommand"
)

// pathVerifier receives a path and determines if it is valid or not
type PathVerifier interface {
	// Verify determines if a given path is valid
	Verify(path string) []error
}

type CommandOverrideVerifier struct {
	Root        *cobra.Command
	SeenPlugins map[string]string
}

// Verify implements PathVerifier and determines if a given path
// is valid depending on whether or not it overwrites an existing
// jx command path, or a previously seen plugin.
func (v *CommandOverrideVerifier) Verify(path string) []error {
	if v.Root == nil {
		return []error{fmt.Errorf("unable to verify path with nil root")}
	}

	// extract the plugin binary name
	segs := strings.Split(path, "/")
	binName := segs[len(segs)-1]

	cmdPath := strings.Split(binName, "-")
	if len(cmdPath) > 1 {
		// the first argument is always "jx" for a plugin binary
		cmdPath = cmdPath[1:]
	}

	errors := []error{}

	if isExec, err := IsExecutable(path); err == nil && !isExec {
		errors = append(errors, fmt.Errorf("warning: %s identified as a jx plugin, but it is not executable", path))
	} else if err != nil {
		errors = append(errors, fmt.Errorf("error: unable to identify %s as an executable file: %v", path, err))
	}

	if existingPath, ok := v.SeenPlugins[binName]; ok {
		errors = append(errors, fmt.Errorf("warning: %s is overshadowed by a similarly named plugin: %s", path, existingPath))
	} else {
		v.SeenPlugins[binName] = path
	}

	if cmd, _, err := v.Root.Find(cmdPath); err == nil {
		errors = append(errors, fmt.Errorf("warning: %s overwrites existing command: %q", binName, cmd.CommandPath()))
	}

	return errors
}

func IsExecutable(fullPath string) (bool, error) {
	info, err := os.Stat(fullPath)
	if err != nil {
		return false, err
	}

	if runtime.GOOS == "windows" {
		if strings.HasSuffix(info.Name(), ".exe") {
			return true, nil
		}
		return false, nil
	}

	if m := info.Mode(); !m.IsDir() && m&0111 != 0 {
		return true, nil
	}

	return false, nil
}

func FindPluginUrl(plugin jenkinsv1.PluginSpec) (string, error) {
	u := ""
	for _, binary := range plugin.Binaries {
		if strings.ToLower(runtime.GOOS) == strings.ToLower(binary.Goos) && strings.ToLower(runtime.
			GOARCH) == strings.ToLower(binary.Goarch) {
			u = binary.Url
		}
	}
	if u == "" {
		return "", fmt.Errorf("unable to locate binary for %s %s for %s", runtime.GOARCH, runtime.GOOS,
			plugin.SubCommand)
	}
	return u, nil
}

func EnsurePluginInstalled(plugin jenkinsv1.Plugin) (string, error) {
	pluginBinDir, err := util.PluginBinDir(plugin.ObjectMeta.Namespace)
	if err != nil {
		return "", err
	}
	path := filepath.Join(pluginBinDir, fmt.Sprintf("%s-%s", plugin.Spec.Name, plugin.Spec.Version))
	if _, err = os.Stat(path); os.IsNotExist(err) {
		u, err := FindPluginUrl(plugin.Spec)
		if err != nil {
			return "", err
		}
		log.Infof("Installing plugin %s version %s for command %s from %s\n", util.ColorInfo(plugin.Spec.Name),
			util.ColorInfo(plugin.Spec.Version), util.ColorInfo(fmt.Sprintf("jx %s", plugin.Spec.SubCommand)), util.ColorInfo(u))

		// Look for other versions to cleanup
		files, err := ioutil.ReadDir(pluginBinDir)
		if err != nil {
			return path, err
		}
		deleted := make([]string, 0)
		for _, f := range files {
			if strings.HasPrefix(f.Name(), plugin.Name) {
				err = os.Remove(filepath.Join(pluginBinDir, f.Name()))
				if err != nil {
					log.Warnf("Unable to delete old version of plugin %s installed at %s because %v\n", plugin.Name, f.Name(), err)
				} else {
					deleted = append(deleted, strings.TrimPrefix(f.Name(), fmt.Sprintf("%s-", plugin.Name)))
				}
			}
		}
		if len(deleted) > 0 {
			log.Infof("Deleted old plugin versions: %v\n", util.ColorInfo(deleted))
		}

		var httpClient = &http.Client{
			Timeout: time.Second * 10,
		}
		// Get the file
		pluginUrl, err := url.Parse(u)
		if err != nil {
			return "", err
		}
		filename := filepath.Base(pluginUrl.Path)
		tmpDir, err := ioutil.TempDir("", plugin.Spec.Name)
		defer func() {
			err := os.RemoveAll(tmpDir)
			if err != nil {
				log.Errorf("Error cleaning up tmpdir %s because %v\n", tmpDir, err)
			}
		}()
		if err != nil {
			return "", err
		}
		downloadFile := filepath.Join(tmpDir, filename)
		// Create the file
		out, err := os.Create(downloadFile)
		if err != nil {
			return path, err
		}
		defer out.Close()
		resp, err := httpClient.Get(u)
		if err != nil {
			return path, err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("unable to install plugin %s because %s getting %s", plugin.Name, resp.Status, u)
		}
		defer resp.Body.Close()

		// Write the body to file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return path, err
		}

		oldPath := downloadFile
		if strings.HasSuffix(filename, ".tar.gz") {
			err = util.UnTargz(downloadFile, tmpDir, make([]string, 0))
			if err != nil {
				return "", err
			}
			oldPath = filepath.Join(tmpDir, plugin.Spec.Name)
		}
		if strings.HasSuffix(filename, ".zip") {
			err = util.Unzip(downloadFile, tmpDir)
			if err != nil {
				return "", err
			}
			oldPath = filepath.Join(tmpDir, plugin.Spec.Name)
		}

		err = os.Rename(oldPath, path)
		if err != nil {
			return "", err
		}
		// Make the file executable
		err = os.Chmod(path, 0755)
		if err != nil {
			return path, err
		}
	}
	return path, nil
}

func ValidatePlugins(jxClient jenkinsv1client.Interface, ns string) error {
	// Validate installed plugins
	plugins, err := jxClient.JenkinsV1().Plugins(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	seenSubCommands := make(map[string][]jenkinsv1.Plugin, 0)
	for _, plugin := range plugins.Items {
		if _, ok := seenSubCommands[plugin.Spec.SubCommand]; !ok {
			seenSubCommands[plugin.Spec.SubCommand] = make([]jenkinsv1.Plugin, 0)
		}
		seenSubCommands[plugin.Spec.SubCommand] = append(seenSubCommands[plugin.Spec.SubCommand], plugin)
	}
	for subCommand, ps := range seenSubCommands {
		if len(ps) > 1 {
			log.Warnf("More than one extension has installed a plugin which will be called for jx %s. These extensions are:\n", util.ColorWarning(subCommand))
			for _, p := range ps {
				for _, o := range p.ObjectMeta.OwnerReferences {
					if o.Kind == "Extension" {
						log.Warnf("  %s\n", util.ColorWarning(o.Name))
					}
				}
			}
			log.Warnf("\nUnpredictable behavior will occur. Contact the extension authors and ask them to resolve the conflict.\n")
		}

	}
	return nil
}
