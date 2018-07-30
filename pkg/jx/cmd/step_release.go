package cmd

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StepReleaseOptions contains the CLI arguments
type StepReleaseOptions struct {
	StepOptions

	DockerRegistry string
	Organisation   string
	Application    string
	Version        string
	GitUsername    string
	GitEmail       string
	Dir            string
	XdgConfigHome  string
}

// NewCmdStep Steps a command object for the "step" command
func NewCmdStepRelease(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &StepReleaseOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "release",
		Short: "performs a release on the current git repository",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.DockerRegistry, "docker-registry", "r", "", "the docker registry host or host:port to use. If not specified it is loaded from the `docker-registry` ConfigMap")
	cmd.Flags().StringVarP(&options.Organisation, "organisation", "o", "", "the docker organisation for the generated docker image")
	cmd.Flags().StringVarP(&options.Application, "application", "a", "", "the docker application image name")
	cmd.Flags().StringVarP(&options.GitUsername, "git-username", "u", "", "The git username to configure if there is none already setup")
	cmd.Flags().StringVarP(&options.GitEmail, "git-email", "e", "", "The git email address to configure if there is none already setup")
	cmd.Flags().StringVarP(&options.XdgConfigHome, "xdg-config-home", "", "/home/jenkins", "The home directory where git config is setup")

	return cmd
}

// Run implements this command
func (o *StepReleaseOptions) Run() error {

	err := o.runCommandVerbose("git", "config", "--global", "credential.helper", "store")
	if err != nil {
		return err
	}
	if o.XdgConfigHome != "" {
		if os.Getenv("XDG_CONFIG_HOME") == "" {
			err = o.Setenv("XDG_CONFIG_HOME", o.XdgConfigHome)
			if err != nil {
				return err
			}
		}
	}

	stepGitCredentialsOptions := &StepGitCredentialsOptions{
		StepOptions: o.StepOptions,
	}
	err = stepGitCredentialsOptions.Run()
	if err != nil {
		return fmt.Errorf("Failed to setup git credentials: %s", err)
	}
	dir := o.Dir
	gitUser, err := o.Git().Username(dir)
	if err != nil || gitUser == "" {
		gitUser = o.GitUsername
		if gitUser == "" {
			user, err := user.Current()
			if err == nil && user != nil {
				gitUser = user.Username
			}
		}
		if gitUser == "" {
			gitUser = "jenkins-x-bot"
		}
		err = o.Git().SetUsername(dir, gitUser)
		if err != nil {
			return fmt.Errorf("Failed to set git user %s: %s", gitUser, err)
		}
	}
	gitEmail, err := o.Git().Email(dir)
	if err != nil || gitEmail == "" {
		gitEmail = o.GitEmail
		if gitEmail == "" {
			gitEmail = "jenkins-x@googlegroups.com"
		}
		err = o.Git().SetEmail(dir, gitEmail)
		if err != nil {
			return fmt.Errorf("Failed to set git email %s: %s", gitEmail, err)
		}
	}

	if o.DockerRegistry == "" {
		o.DockerRegistry = os.Getenv("DOCKER_REGISTRY")
	}
	if o.Organisation == "" {
		o.Organisation = os.Getenv("ORG")
	}
	if o.Application == "" {
		o.Application = os.Getenv("APP_NAME")
	}
	if o.DockerRegistry == "" {
		o.DockerRegistry, err = o.loadDockerRegistry()
		if err != nil {
			return err
		}
	}
	if o.Organisation == "" || o.Application == "" {
		gitInfo, err := o.FindGitInfo("")
		if err != nil {
			return err
		}
		if o.Organisation == "" {
			o.Organisation = gitInfo.Organisation
		}
		if o.Application == "" {
			o.Application = gitInfo.Name
		}
	}
	err = o.Setenv("DOCKER_REGISTRY", o.DockerRegistry)
	if err != nil {
		return err
	}
	err = o.Setenv("ORG", o.Organisation)
	if err != nil {
		return err
	}
	err = o.Setenv("APP_NAME", o.Application)
	if err != nil {
		return err
	}

	stepNextVersionOptions := &StepNextVersionOptions{
		StepOptions: o.StepOptions,
	}
	if o.isNode() {
		stepNextVersionOptions.Filename = packagejson
		/*
			} else if o.isMaven() {
				stepNextVersionOptions.Filename = pomxml
		*/
	} else {
		stepNextVersionOptions.UseGitTagOnly = true
	}
	err = stepNextVersionOptions.Run()
	if err != nil {
		return fmt.Errorf("Failed to create next version: %s", err)
	}
	o.Version = stepNextVersionOptions.NewVersion
	err = o.Setenv("VERSION", o.Version)
	if err != nil {
		return err
	}

	err = o.updateVersionInSource()
	if err != nil {
		return fmt.Errorf("Failed to update version in source: %s", err)
	}

	stepTagOptions := &StepTagOptions{
		StepOptions: o.StepOptions,
	}
	stepTagOptions.Flags.Version = o.Version
	err = stepTagOptions.Run()
	if err != nil {
		return fmt.Errorf("Failed to tag source: %s", err)
	}

	err = o.buildSource()
	if err != nil {
		return err
	}
	err = o.runCommandVerbose("skaffold", "build", "-f", "skaffold.yaml")
	if err != nil {
		return fmt.Errorf("Failed to run skaffold: %s", err)
	}
	imageName := fmt.Sprintf("%s/%s/%s:%s", o.DockerRegistry, o.Organisation, o.Application, o.Version)

	stepPostBuildOptions := &StepPostBuildOptions{
		StepOptions:   o.StepOptions,
		FullImageName: imageName,
	}
	err = stepPostBuildOptions.Run()
	if err != nil {
		return fmt.Errorf("Failed to run post build step: %s", err)
	}

	// now lets promote from the charts dir...
	chartsDir := filepath.Join("charts", o.Application)
	exists, err := util.FileExists(chartsDir)
	if err != nil {
		return fmt.Errorf("Failed to find chart folder: %s", err)
	}
	if exists {
		err = o.releaseAndPromoteChart(chartsDir)
		if err != nil {
			return fmt.Errorf("Failed to promote: %s", err)
		}
	} else {
		log.Infof("No charts directory %s so not promoting\n", util.ColorInfo(chartsDir))
	}

	return nil
}

func (o *StepReleaseOptions) updateVersionInSource() error {
	if o.isMaven() {
		return o.runCommandVerbose("mvn", "versions:set", "-DnewVersion="+o.Version)
	}
	return nil
}

func (o *StepReleaseOptions) buildSource() error {
	if o.isMaven() {
		return o.runCommandVerbose("mvn", "clean", "deploy")
	}
	return nil

}

func (o *StepReleaseOptions) loadDockerRegistry() (string, error) {
	kubeClient, curNs, err := o.KubeClient()
	if err != nil {
		return "", err
	}
	ns, _, err := kube.GetDevNamespace(kubeClient, curNs)
	if err != nil {
		return "", err
	}

	configMapName := kube.ConfigMapJenkinsDockerRegistry
	cm, err := kubeClient.CoreV1().ConfigMaps(ns).Get(configMapName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("Could not find ConfigMap %s in namespace %s: %s", configMapName, ns, err)
	}
	if cm.Data != nil {
		dockerRegistry := cm.Data["docker.registry"]
		if dockerRegistry != "" {
			return dockerRegistry, nil
		}
	}
	return "", fmt.Errorf("Could not find the docker.registry property in the ConfigMap: %s", configMapName)
}

func (o *StepReleaseOptions) releaseAndPromoteChart(dir string) error {
	err := os.Chdir(dir)
	if err != nil {
		return fmt.Errorf("Failed to change to directory %s: %s", dir, err)
	}

	stepChangelogOptions := &StepChangelogOptions{
		StepOptions: o.StepOptions,
	}
	err = stepChangelogOptions.Run()
	if err != nil {
		return fmt.Errorf("Failed to generate changelog: %s", err)
	}

	stepHelmReleaseOptions := &StepHelmReleaseOptions{
		StepHelmOptions: StepHelmOptions{
			StepOptions: o.StepOptions,
		},
	}
	err = stepHelmReleaseOptions.Run()
	if err != nil {
		return fmt.Errorf("Failed to release helm chart: %s", err)
	}

	promoteOptions := PromoteOptions{
		CommonOptions: o.CommonOptions,
		AllAutomatic:  true,
		Timeout:       "1h",
		Version:       o.Version,
	}
	promoteOptions.BatchMode = true
	return promoteOptions.Run()
}

func (o *StepReleaseOptions) isMaven() bool {
	exists, err := util.FileExists("pom.xml")
	return exists && err == nil
}

func (o *StepReleaseOptions) isNode() bool {
	exists, err := util.FileExists("package.json")
	return exists && err == nil
}

func (o *StepReleaseOptions) Setenv(key string, value string) error {
	err := os.Setenv(key, value)
	if err != nil {
		return fmt.Errorf("Failed to set environment variable %s=%s: %s", key, value, err)
	}
	return nil
}
