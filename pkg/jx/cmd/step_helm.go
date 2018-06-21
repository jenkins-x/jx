package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

const ()

// StepHelmOptions contains the command line flags
type StepHelmOptions struct {
	StepOptions

	Dir string
}

// NewCmdStepHelm Steps a command object for the "step" command
func NewCmdStepHelm(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &StepHelmOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "helm",
		Short: "helm [command]",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdStepHelmApply(f, out, errOut))
	cmd.AddCommand(NewCmdStepHelmBuild(f, out, errOut))
	cmd.AddCommand(NewCmdStepHelmRelease(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *StepHelmOptions) Run() error {
	return o.Cmd.Help()
}

func (o *StepHelmOptions) addStepHelmFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "The directory containing the helm chart to apply")
}

func (o *StepHelmOptions) findStagingRepoIds() ([]string, error) {
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

func (o *StepHelmOptions) dropRepositories(repoIds []string, message string) error {
	var answer error
	for _, repoId := range repoIds {
		err := o.dropRepository(repoId, message)
		if err != nil {
			log.Warnf("Failed to drop repository %s: %s\n", util.ColorInfo(repoIds), util.ColorError(err))
			if answer == nil {
				answer = err
			}
		}
	}
	return answer
}

func (o *StepHelmOptions) dropRepository(repoId string, message string) error {
	if repoId == "" {
		return nil
	}
	log.Infof("Dropping helm release repository %s\n", util.ColorInfo(repoId))
	err := o.runCommand("mvn",
		"org.sonatype.plugins:helm-staging-maven-plugin:1.6.5:rc-drop",
		"-DserverId=oss-sonatype-staging",
		"-DhelmUrl=https://oss.sonatype.org",
		"-DstagingRepositoryId="+repoId,
		"-Ddescription=\""+message+"\" -DstagingProgressTimeoutMinutes=60")
	if err != nil {
		log.Warnf("Failed to drop repository %s due to: %s\n", repoId, err)
	} else {
		log.Infof("Dropped repository %s\n", util.ColorInfo(repoId))
	}
	return err
}

func (o *StepHelmOptions) releaseRepository(repoId string) error {
	if repoId == "" {
		return nil
	}
	log.Infof("Releasing helm release repository %s\n", util.ColorInfo(repoId))
	options := o
	err := options.runCommand("mvn",
		"org.sonatype.plugins:helm-staging-maven-plugin:1.6.5:rc-release",
		"-DserverId=oss-sonatype-staging",
		"-DhelmUrl=https://oss.sonatype.org",
		"-DstagingRepositoryId="+repoId,
		"-Ddescription=\"Next release is ready\" -DstagingProgressTimeoutMinutes=60")
	if err != nil {
		log.Infof("Failed to release repository %s due to: %s\n", repoId, err)
	} else {
		log.Infof("Released repository %s\n", util.ColorInfo(repoId))
	}
	return err
}
