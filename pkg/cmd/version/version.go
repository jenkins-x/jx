package version

import (
	"fmt"
	"io"
	"os"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Build information. Populated at build-time.
type buildInfo struct {
	Version      string
	Revision     string
	Branch       string
	GitTreeState string
	BuildDate    string
	GoVersion    string
}

// Build information. Populated at build-time.
var (
	Version      string
	Revision     string
	BuildDate    string
	GoVersion    string
	Branch       string
	GitTreeState string
)

const (

	// TestVersion used in test cases for the current version if no
	// version can be found - such as if the version property is not properly
	// included in the go test flags.
	TestVersion = "3.2.238"

	// TestRevision can be used in tests if no revision is passed in the test flags
	TestRevision = "04b628f48"

	// TestBranch can be used in tests if no tree state is passed in the test flags
	TestBranch = "main"

	// TestTreeState can be used in tests if no tree state is passed in the test flags
	TestTreeState = "clean"

	// TestBuildDate can be used in tests if no build date is passed in the test flags
	TestBuildDate = "2022-05-31T14:51:38Z"

	// TestGoVersion can be used in tests if no version is passed in the test flags
	TestGoVersion = "1.17.8"
)

// ShowOptions the options for viewing running PRs
type Options struct {
	Verbose bool
	Quiet   bool
	Short   bool
	Out     io.Writer
}

// NewCmdVersion creates a command object for the "version" command
func NewCmdVersion() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Displays the version of this command",
		Run: func(_ *cobra.Command, _ []string) {
			o.run()
		},
	}
	cmd.Flags().BoolVarP(&o.Quiet, "quiet", "q", false, "uses the quiet format of just outputting the version number only")
	cmd.Flags().BoolVarP(&o.Short, "short", "s", false, "uses the short format of just outputting the version number only")
	return cmd, o
}

// Run implements the command
func (o *Options) run() {
	v := getBuildInfo()
	if o.Out == nil {
		o.Out = os.Stdout
	}
	if o.Quiet {
		fmt.Fprintln(o.Out, "The --quit, -q flag is being deprecated from JX on Oct 2022\nUse --short, -s instead")
		fmt.Fprintf(o.Out, "%s\n", v.Version)
		return
	}
	if o.Short {
		fmt.Fprintf(o.Out, "%s\n", v.Version)
		return
	}
	fmt.Fprintf(o.Out, "version: %s\n", v.Version)
	fmt.Fprintf(o.Out, "shaCommit: %s\n", v.Revision)
	fmt.Fprintf(o.Out, "buildDate: %s\n", v.BuildDate)
	fmt.Fprintf(o.Out, "goVersion: %s\n", v.GoVersion)
	fmt.Fprintf(o.Out, "branch: %s\n", v.Branch)
	fmt.Fprintf(o.Out, "gitTreeState: %s\n", v.GitTreeState)
}

func getBuildInfo() buildInfo {
	return buildInfo{
		Version:      getVersion(),
		Revision:     getCommitSha(),
		Branch:       getBranch(),
		GitTreeState: getTreeState(),
		BuildDate:    getBuildDate(),
		GoVersion:    getGoVersion(),
	}
}

func getVersion() string {
	if Version != "" {
		return Version
	}
	return TestVersion
}

func getGoVersion() string {
	if GoVersion != "" {
		return GoVersion
	}
	return TestGoVersion
}

func getCommitSha() string {
	if Revision != "" {
		return Revision
	}
	return TestRevision
}

func getBuildDate() string {
	if BuildDate != "" {
		return BuildDate
	}
	return TestBuildDate
}

func getBranch() string {
	if Branch != "" {
		return Branch
	}
	return TestBranch
}

func getTreeState() string {
	if GitTreeState != "" {
		return GitTreeState
	}
	return TestTreeState
}

func GetSemverVersion() (semver.Version, error) {
	text := getVersion()
	v, err := semver.Make(text)
	if err != nil {
		return v, errors.Wrapf(err, "failed to parse version %s", text)
	}
	return v, nil
}
