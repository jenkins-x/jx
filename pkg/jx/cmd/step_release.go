package cmd

import (
	"fmt"
	"io"
	"os"
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

	return cmd
}

// Run implements this command
func (o *StepReleaseOptions) Run() error {
	err := o.runCommandVerbose("git", "config", "--global", "credential.helper", "store")
	if err != nil {
		return err
	}
	stepGitCredentialsOptions := &StepGitCredentialsOptions{
		StepOptions: o.StepOptions,
	}
	err = stepGitCredentialsOptions.Run()
	if err != nil {
		return fmt.Errorf("Failed to setup git credentials: %s", err)
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
	err = os.Setenv("DOCKER_REGISTRY", o.DockerRegistry)
	if err != nil {
		return err
	}
	err = os.Setenv("ORG", o.Organisation)
	if err != nil {
		return err
	}
	err = os.Setenv("APP_NAME", o.Application)
	if err != nil {
		return err
	}

	stepNextVersionOptions := &StepNextVersionOptions{
		StepOptions: o.StepOptions,
	}
	if o.isMaven() {
		stepNextVersionOptions.Filename = "pom.xml"
	} else if o.isNode() {
		stepNextVersionOptions.Filename = "package.json"
	} else {
		stepNextVersionOptions.UseGitTagOnly = true
	}
	err = stepNextVersionOptions.Run()
	if err != nil {
		return fmt.Errorf("Failed to create next version: %s", err)
	}
	o.Version = stepNextVersionOptions.NewVersion
	err = os.Setenv("VERSION", o.Version)
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
	err = stepTagOptions.Run()
	if err != nil {
		return err
	}

	err = o.buildSource()
	if err != nil {
		return err
	}
	err = o.runCommandVerbose("skaffold", "run", "-f", "skaffold.yaml")
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
		err = o.promote(chartsDir)
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

func (o *StepReleaseOptions) promote(dir string) error {
	stepChangelogOptions := &StepChangelogOptions{
		StepOptions: o.StepOptions,
		Dir:         dir,
	}
	err := stepChangelogOptions.Run()
	if err != nil {
		return fmt.Errorf("Failed to generate changelog: %s", err)
	}

	stepHelmReleaseOptions := &StepHelmReleaseOptions{
		StepHelmOptions: StepHelmOptions{
			StepOptions: o.StepOptions,
			Dir:         dir,
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
