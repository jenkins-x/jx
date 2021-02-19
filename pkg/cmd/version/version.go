package version

import (
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
)

// Build information. Populated at build-time.
var (
	Version   string
	Revision  string
	Branch    string
	BuildUser string
	BuildDate string
	GoVersion string
)

const (

	// TestVersion used in test cases for the current version if no
	// version can be found - such as if the version property is not properly
	// included in the go test flags
	TestVersion = "1.0.0-SNAPSHOT"
)

// ShowOptions the options for viewing running PRs
type Options struct {
	Verbose bool
	Quiet   bool
}

// NewCmdVersion creates a command object for the "version" command
func NewCmdVersion() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Displays the version of this command",
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&o.Quiet, "quiet", "q", false, "uses the quiet format of just outputting the version number only")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	v := GetVersion()
	if o.Quiet {
		log.Logger().Infof(v)
		return nil
	}
	log.Logger().Infof("version: %s", termcolor.ColorInfo(v))
	return nil
}

func GetVersion() string {
	if Version != "" {
		return Version
	}
	return TestVersion
}
