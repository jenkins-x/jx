package cmd

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/version"
	"github.com/spf13/cobra"
	"path/filepath"
	"strings"
)

const (
	defaultRepoName            = "origin"
	defaultStableVersionBranch = "master"
)

// StepGetVersionChangeSetOptions contains the command line flags
type StepGetVersionChangeSetOptions struct {
	StepOptions
	VersionsDir        string
	VersionsRepository string
	TestingBranch      string
	StableBranch       string
	PR                 string
}

var (
	// StepGetVersionChangeSetLong command long description
	StepGetVersionChangeSetLong = templates.LongDesc(`
		This pipeline step generates environment variables from the differences of versions between jenkins-x-version branches

`)
	// StepGetVersionChangeSetExample command example
	StepGetVersionChangeSetExample = templates.Examples(`
		# This pipeline step generates environment variables from the differences of versions between jenkins-x-version PR21 and the master branch
		jx step get version-changeset --pr 21

        # This pipeline step generates environment variables from the differences of versions between jenkins-x-version PR21 and a branch called stuff
        jx step get version-changeset --stable-branch stuff --pr 21
`)
)

// NewCmdStepGetVersionChangeSet create the 'step git envs' command
func NewCmdStepGetVersionChangeSet(commonOpts *CommonOptions) *cobra.Command {
	options := StepGetVersionChangeSetOptions{
		StepOptions: StepOptions{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "version-changeset",
		Short:   "Creates environment variables from the differences of versions between jenkins-x-version branches",
		Long:    StepGetVersionChangeSetLong,
		Example: StepGetVersionChangeSetExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.VersionsRepository, "repo", "r", DefaultVersionsURL, "Jenkins X versions Git repo")
	cmd.Flags().StringVarP(&options.StableBranch, "stable-branch", "", defaultStableVersionBranch, "the versions git repository branch to compare against")
	cmd.Flags().StringVarP(&options.TestingBranch, "testing-branch", "", defaultStableVersionBranch, "the versions git repository branch to clone")
	cmd.Flags().StringVarP(&options.VersionsDir, "versions-dir", "", "", "the directory containing the versions repo")
	cmd.Flags().StringVarP(&options.PR, "pr", "", "", "the PR in the versions repository top clone")

	return cmd
}

// Run implements the command
func (o *StepGetVersionChangeSetOptions) Run() error {
	if o.VersionsDir == "" {
		versionDir, err := o.cloneJXVersionsRepo(o.VersionsRepository)
		if err != nil {
			return err
		}
		o.VersionsDir = versionDir
	}
	if o.PR != "" {
		o.TestingBranch = o.PR
		o.Git().FetchBranch(o.VersionsDir, defaultRepoName, "pull/"+o.PR+"/head:"+o.PR)
	}

	if o.StableBranch != defaultStableVersionBranch {
		o.Git().FetchBranch(o.VersionsDir, defaultRepoName, o.StableBranch+":"+o.StableBranch)
	}

	o.Git().Checkout(o.VersionsDir, o.TestingBranch)
	bulkChange, _ := o.Git().ListChangedFilesFromBranch(o.VersionsDir, o.StableBranch)
	changeSets := strings.Split(bulkChange, "\n")
	appUpdatedVersions := make([]string, 0)
	appPreviousVersions := make([]string, 0)
	for _, line := range changeSets {
		output := strings.Split(line, "\t")
		if len(output) == 2 {
			modifier := output[0]
			if modifier == "A" || modifier == "M" {
				file := output[1]
				if strings.HasSuffix(file, ".yml") {
					stableVersion, err := version.LoadStableVersionFile(filepath.Join(o.VersionsDir, file))
					if err == nil {
						if modifier == "M" {
							fileData, _ := o.Git().LoadFileFromBranch(o.VersionsDir, o.StableBranch, file)
							oldStableVersion, _ := version.LoadStableVersionFromData([]byte(fileData))
							oldApp := formatVersion(file, oldStableVersion)
							appPreviousVersions = append(appPreviousVersions, oldApp)
						}
						changedApp := formatVersion(file, stableVersion)
						appUpdatedVersions = append(appUpdatedVersions, changedApp)
					}
				}
			}

		}
	}
	updateEnv := strings.Join(appUpdatedVersions, ",")
	previousEnv := strings.Join(appPreviousVersions, ",")
	log.Infof("JX_CHANGED_VERSIONS=\"%s\"\n", updateEnv)
	log.Infof("JX_STABLE_VERSIONS=\"%s\"\n", previousEnv)
	return nil
}

func formatVersion(fileName string, stableVersion *version.StableVersion) string {
	formattedVersion := strings.Replace(fileName, string(version.KindChart)+"/", string(version.KindChart)+":", 1)
	formattedVersion = strings.Replace(formattedVersion, ".yml", "", 1)
	formattedVersion = formattedVersion + ":" + stableVersion.Version
	return formattedVersion
}
