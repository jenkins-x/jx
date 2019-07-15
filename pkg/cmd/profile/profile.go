package profile

import (
	"errors"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
)

const (
	// DefaultProfileFile location of profle config
	DefaultProfileFile = "profile.yaml"
	// OpenSourceProfile constant for OSS profile
	OpenSourceProfile = "oss"
	// CloudBeesProfile constant for CloudBees profile
	CloudBeesProfile = "cloudbees"
)

// Profile contains the command line options
type Profile struct {
	*opts.CommonOptions
}

// JxProfile contains the jx profile info
type JxProfile struct {
	InstallType string
}

var (
	profileLong = templates.LongDesc(`
		Sets the profile for the jx install
`)

	profileExample = templates.Examples(`
		# Sets the profile for the jx install to cloudbees
		jx profile cloudbees

        # Set the profile for the jx install to open source
		jx profile oss
	`)
)

// NewCmdProfile creates the command object
func NewCmdProfile(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &Profile{
		commonOpts,
	}

	cmd := &cobra.Command{
		Use:     "profile TYPE",
		Short:   "Set your jx profile",
		Long:    profileLong,
		Example: profileExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	return cmd
}

// Run implements this command
func (o *Profile) Run() error {
	if len(o.Args) < 1 {
		return errors.New("Please specify a valid profile of cloudbees or oss ")
	}
	activatedProfle := OpenSourceProfile
	if o.Args[0] == CloudBeesProfile {
		activatedProfle = CloudBeesProfile
	}
	jxHome, err := util.ConfigDir()
	if err != nil {
		return err
	}
	profileSettingsFile := filepath.Join(jxHome, DefaultProfileFile)
	jxProfle := JxProfile{
		InstallType: activatedProfle,
	}
	data, err := yaml.Marshal(jxProfle)
	if err == nil {
		err = ioutil.WriteFile(profileSettingsFile, data, util.DefaultWritePermissions)
		if activatedProfle == CloudBeesProfile {
			log.Logger().Info("Activating the CloudBees Jenkins X Distribution")
		} else {
			log.Logger().Info("Activating the Jenkins X Profile")
		}

	}

	return err
}
