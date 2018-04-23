package cmd

import (
	"io"

	"fmt"

	"os"

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/spf13/cobra"
)

// StepPostBuildOptions contains the command line flags
type StepPostBuildOptions struct {
	StepOptions

	OutputFile string
}

var (
	StepPostBuildLong = templates.LongDesc(`
		This pipeline step performs post build actions such as CVE analysis
`)

	StepPostBuildExample = templates.Examples(`
		jx step post build
`)
)

func NewCmdStepPostBuild(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := StepGitCredentialsOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "build",
		Short:   "Performs post build actions in a pipeline",
		Long:    StepPostBuildLong,
		Example: StepPostBuildExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	return cmd
}

func (o *StepPostBuildOptions) Run() error {

	_, _, err := o.KubeClient()
	if err != nil {
		return fmt.Errorf("error connecting to kubernetes cluster: %v", err)
	}

	// let's try and add image to CVE provider
	err = o.addImageCVEProvider()
	if err != nil {
		return fmt.Errorf("error adding image to CVE provider: %v", err)
	}

	return nil
}
func (o *StepPostBuildOptions) addImageCVEProvider() error {

	isRunning, err := kube.IsDeploymentRunning(o.kubeClient, AnchoreDeploymentName, o.currentNamespace)
	if err != nil {
		return err
	}

	if !isRunning {
		log.Infof("no CVE provider running in the current %s namespace so skip adding image to be analysed", o.currentNamespace)
		return nil
	}

	cveProviderHost := os.Getenv("JENKINS_X_DOCKER_REGISTRY_SERVICE_HOST")
	if cveProviderHost == "" {
		return fmt.Errorf("no JENKINS_X_DOCKER_REGISTRY_SERVICE_HOST env var found")
	}
	cveProviderPort := os.Getenv("JENKINS_X_DOCKER_REGISTRY_SERVICE_PORT")
	if cveProviderPort == "" {
		return fmt.Errorf("no JENKINS_X_DOCKER_REGISTRY_SERVICE_PORT env var found")
	}

	appName := os.Getenv("APP_NAME")
	if appName == "" {
		return fmt.Errorf("no APP_NAME env var found")
	}

	org := os.Getenv("ORG")
	if org == "" {
		return fmt.Errorf("no ORG env var found")
	}

	version := os.Getenv("VERSION")
	if version == "" {
		return fmt.Errorf("no ORG env var found")
	}

	fullImageName := fmt.Sprintf("%s:%s/%s/%s:%s", cveProviderHost, cveProviderPort, org, appName, version)
	log.Infof("adding image %s to CVE provider\n", fullImageName)

	err = o.runCommand("anchore-cli", "image", "add", fullImageName)
	if err != nil {
		return fmt.Errorf("failed to add image %s to anchore engine: %v\n", fullImageName, err)
	}
	// todo get response and use image id to annotate pods when doing a helm install / upgrade
	return nil
}
