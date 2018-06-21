package cmd

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultChartRepo = "http://jenkins-x-chartmuseum:8080"
)

// StepHelmReleaseOptions contains the command line flags
type StepHelmReleaseOptions struct {
	StepHelmOptions
}

var (
	StepHelmReleaseLong = templates.LongDesc(`
		This pipeline step releases the Helm chart in the current directory
`)

	StepHelmReleaseExample = templates.Examples(`
		jx step helm release

`)
)

func NewCmdStepHelmRelease(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := StepHelmReleaseOptions{
		StepHelmOptions: StepHelmOptions{
			StepOptions: StepOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "release",
		Short:   "Releases the helm chart in the current directory",
		Aliases: []string{""},
		Long:    StepHelmReleaseLong,
		Example: StepHelmReleaseExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	options.addStepHelmFlags(cmd)
	return cmd
}

func (o *StepHelmReleaseOptions) Run() error {
	dir := o.Dir
	helmBinary, err := o.helmInitDependencyBuild(dir, o.defaultReleaseCharts())
	if err != nil {
		return err
	}

	err = o.runCommandVerboseAt(dir, helmBinary, "package", ".")
	if err != nil {
		return err
	}

	chartFile := filepath.Join(dir, "Chart.yaml")
	name, version, err := helm.LoadChartNameAndVersion(chartFile)
	if err != nil {
		return err
	}

	if name == "" {
		return fmt.Errorf("Could not find name in chart %s", chartFile)
	}
	if version == "" {
		return fmt.Errorf("Could not find version in chart %s", chartFile)
	}
	tarball := fmt.Sprintf("%s-%s.tgz", name, version)
	exists, err := util.FileExists(tarball)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("Generated helm file %s does not exist!", tarball)
	}
	defer os.Remove(tarball)

	chartRepo := o.releaseChartMuseumUrl()

	userName := os.Getenv("CHARTMUSEUM_CREDS_USR")
	password := os.Getenv("CHARTMUSEUM_CREDS_PSW")
	if userName == "" || password == "" {
		// lets try load them from the secret directly
		client, ns, err := o.KubeClient()
		if err != nil {
			return err
		}
		secret, err := client.CoreV1().Secrets(ns).Get(kube.SecretJenkinsChartMuseum, metav1.GetOptions{})
		if err != nil {
			log.Warnf("Could not load Secret %s in namespace %s: %s\n", kube.SecretJenkinsChartMuseum, ns, err)
		} else {
			if secret != nil && secret.Data != nil {
				if userName == "" {
					userName = string(secret.Data["BASIC_AUTH_USER"])
				}
				if password == "" {
					password = string(secret.Data["BASIC_AUTH_PASS"])
				}
			}
		}
	}
	if userName == "" {
		return fmt.Errorf("No enviroment variable $CHARTMUSEUM_CREDS_USR defined")
	}
	if password == "" {
		return fmt.Errorf("No enviroment variable CHARTMUSEUM_CREDS_PSW defined")
	}

	// post the tarball to the chart repository
	client := http.Client{}

	u := util.UrlJoin(chartRepo, "/api/charts")

	file, err := os.Open(tarball)
	if err != nil {
		return err
	}
	log.Infof("Uploading chart file %s to %s\n", util.ColorInfo(tarball), util.ColorInfo(u))
	req, err := http.NewRequest(http.MethodPost, u, bufio.NewReader(file))
	if err != nil {
		return err
	}
	req.SetBasicAuth(userName, password)
	req.Header.Set("Content-Type", "application/gzip")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	responseMessage := string(body)
	statusCode := res.StatusCode
	log.Infof("Received %d response: %s\n", statusCode, responseMessage)
	if statusCode >= 300 {
		return fmt.Errorf("Failed to post chart to %s due to response %d: %s", u, statusCode, responseMessage)
	}
	return nil
}
