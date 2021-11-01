package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-api/v4/pkg/util"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/jenkins-x/jx/pkg/plugins"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/util/homedir"

	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
)

type Options struct {
	options.BaseOptions
	CommandRunner cmdrunner.CommandRunner
	BrowserPath   string
	Host          string
	Port          int
	OctantArgs    []string
}

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Views the Jenkins X UI (octant).`)

	cmdExample = templates.Examples(`
		# open the UI
		jx ui
        # To pass arguments to octant
  		jx ui -o --namespace=jx -o -v -o --browser-path="/#/jx/pipelines"
`)
)

// NewCmdUI opens the octant UI
func NewCmdUI() (*cobra.Command, *Options) {
	o := &Options{}
	cmd := &cobra.Command{
		Use:     "ui",
		Short:   "Views the Jenkins X UI (octant)",
		Long:    cmdLong,
		Example: cmdExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&o.BrowserPath, "browser-path", "p", "/#/jx/pipelines-recent", "The browser path inside octant to open")
	cmd.Flags().StringVarP(&o.Host, "host", "", "", "The host to listen on")
	cmd.Flags().IntVarP(&o.Port, "port", "", 0, "The port for octant to listen on")
	cmd.Flags().StringSliceVarP(&o.OctantArgs, "octant-args", "o", []string{"--namespace=jx"}, "Extra arguments passed to the octant binary")
	o.BaseOptions.AddBaseFlags(cmd)
	return cmd, o
}

func (o *Options) Run() error {
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.QuietCommandRunner
	}
	// lets find the octant binary...
	octantBin, err := plugins.GetOctantBinary("")
	if err != nil {
		return errors.Wrap(err, "failed to download the octant binary")
	}

	err = VerifyOctantPlugins(o.CommandRunner)
	if err != nil {
		return errors.Wrap(err, "failed to download the Jenkins X octant plugins")
	}

	args := append([]string{"--browser-path", o.BrowserPath}, o.OctantArgs...)

	if o.Port != 0 || o.Host != "" {
		if o.Port == 0 {
			o.Port = 8080
		}
		if o.Host == "" {
			o.Host = "localhost"
		}
		args = append(args, "--listener-addr", fmt.Sprintf("%s:%d", o.Host, o.Port))
	}

	c := &cmdrunner.Command{
		Name: octantBin,
		Args: args,
		Out:  os.Stdout,
		Err:  os.Stderr,
		In:   os.Stdin,
	}
	_, err = o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to start octant via: %s", c.CLI())
	}
	return nil
}

func VerifyOctantPlugins(runner cmdrunner.CommandRunner) error {
	err := VerifyOctantPluginVersion(runner, "octant-jx", plugins.OctantJXVersion, func() (string, error) {
		return plugins.GetOctantJXBinary(plugins.OctantJXVersion)
	})
	if err != nil {
		return err
	}
	return VerifyOctantPluginVersion(runner, "octant-jxo", plugins.OctantJXVersion, func() (string, error) {
		return plugins.GetOctantJXOBinary(plugins.OctantJXVersion)
	})
}

func VerifyOctantPluginVersion(runner cmdrunner.CommandRunner, pluginName, requiredVersion string, fn func() (string, error)) error {
	pluginDir := OctantPluginsDir()
	octantJxBin := filepath.Join(pluginDir, pluginName)

	version := ""
	exists, err := util.FileExists(octantJxBin)
	if err != nil {
		return errors.Wrapf(err, "failed to check if file exists %s", octantJxBin)
	}

	if exists {
		c := &cmdrunner.Command{
			Name: octantJxBin,
			Args: []string{"version"},
		}
		// lets try check the version
		out, err := runner(c)
		if err != nil {
			log.Logger().Warnf("failed to run command %s version: %s", octantJxBin, err.Error())
		} else {
			version = strings.TrimSpace(out)
		}
	}

	if version == requiredVersion {
		log.Logger().Debugf("the %s plugin is already on version %s", info(pluginName), info(version))
		return nil
	}
	if version != "" {
		log.Logger().Infof("the %s plugin %s is being upgraded to %s", info(pluginName), info(version), info(requiredVersion))
	}
	jxBin, err := fn()
	if err != nil {
		return err
	}

	err = os.MkdirAll(pluginDir, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create octant plugin directory %s", pluginDir)
	}
	err = files.CopyFile(jxBin, octantJxBin)
	if err != nil {
		return errors.Wrapf(err, "failed to copy new plugin version %s to %s", jxBin, octantJxBin)
	}

	log.Logger().Debugf("updated plugin file %s", info(octantJxBin))
	return nil
}

// OctantPluginsDir returns the location of octant plugins
func OctantPluginsDir() string {
	return filepath.Join(homedir.HomeDir(), ".config", "octant", "plugins")
}
