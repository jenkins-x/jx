package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"fmt"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// StepGitOptions contains the command line flags
type StepGitOptions struct {
	StepOptions
}

// NewCmdStepGit Steps a command object for the "step" command
func NewCmdStepGit(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &StepGitOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "git",
		Short: "git [command]",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdStepGitCredentials(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *StepGitOptions) Run() error {
	return o.Cmd.Help()
}

func (o *StepGitOptions) findStagingRepoIds() ([]string, error) {
	answer := []string{}
	files, err := filepath.Glob(statingRepositoryProperties)
	if err != nil {
		return answer, err
	}
	for _, f := range files {
		data, err := ioutil.ReadFile(f)
		if err != nil {
			return answer, fmt.Errorf("Failed to load file %s: %s", f, err)
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, statingRepositoryIdPrefix) {
				id := strings.TrimSpace(strings.TrimPrefix(line, statingRepositoryIdPrefix))
				if id != "" {
					answer = append(answer, id)
				}
			}
		}
	}
	return answer, nil
}

func (o *StepGitOptions) dropRepositories(repoIds []string, message string) error {
	var answer error
	for _, repoId := range repoIds {
		err := o.dropRepository(repoId, message)
		if err != nil {
			o.warnf("Failed to drop repository %s: %s\n", util.ColorInfo(repoIds), util.ColorError(err))
			if answer == nil {
				answer = err
			}
		}
	}
	return answer
}

func (o *StepGitOptions) dropRepository(repoId string, message string) error {
	if repoId == "" {
		return nil
	}
	o.Printf("Dropping nexus release repository %s\n", util.ColorInfo(repoId))
	err := o.runCommand("mvn",
		"org.sonatype.plugins:nexus-staging-maven-plugin:1.6.5:rc-drop",
		"-DserverId=oss-sonatype-staging",
		"-DnexusUrl=https://oss.sonatype.org",
		"-DstagingRepositoryId="+repoId,
		"-Ddescription=\""+message+"\" -DstagingProgressTimeoutMinutes=60")
	if err != nil {
		o.warnf("Failed to drop repository %s due to: %s\n", repoId, err)
	} else {
		o.Printf("Dropped repository %s\n", util.ColorInfo(repoId))
	}
	return err
}

func (o *StepGitOptions) releaseRepository(repoId string) error {
	if repoId == "" {
		return nil
	}
	o.Printf("Releasing nexus release repository %s\n", util.ColorInfo(repoId))
	options := o
	err := options.runCommand("mvn",
		"org.sonatype.plugins:nexus-staging-maven-plugin:1.6.5:rc-release",
		"-DserverId=oss-sonatype-staging",
		"-DnexusUrl=https://oss.sonatype.org",
		"-DstagingRepositoryId="+repoId,
		"-Ddescription=\"Next release is ready\" -DstagingProgressTimeoutMinutes=60")
	if err != nil {
		o.warnf("Failed to release repository %s due to: %s\n", repoId, err)
	} else {
		o.Printf("Released repository %s\n", util.ColorInfo(repoId))
	}
	return err
}
