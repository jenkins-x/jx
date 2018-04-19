package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"fmt"

	"github.com/jenkins-x/jx/pkg/cve"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
)

// GetGitOptions the command line options
type GetCVEOptions struct {
	GetOptions
	ImageName         string
	ImageID           string
	Version           string
	Env               string
	VulnerabilityType string
}

var (
	getCVELong = templates.LongDesc(`
		Display Common Vulnerabilities and Exposures (CVEs)

`)

	getCVEExample = templates.Examples(`
		# List all Common Vulnerabilities and Exposures (CVEs)

		jx get cve # using current dir as the context for app name
		jx get cve --app foo
		jx get cve --app foo --version 1.0.0
		jx get cve --app foo --env staging
		jx get cve --env staging
	`)
)

// NewCmdGetCVE creates the command
func NewCmdGetCVE(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetCVEOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "cve [flags]",
		Short:   "Display Common Vulnerabilities and Exposures (CVEs)",
		Long:    getCVELong,
		Example: getCVEExample,
		Aliases: []string{"cves"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addGetCVEFlags(cmd)

	return cmd
}

func (o *GetCVEOptions) addGetCVEFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.ImageName, "image-name", "", "", "Full image name e.g. jenkinsxio/nexus ")
	cmd.Flags().StringVarP(&o.ImageID, "image-id", "", "", "Image ID in CVE engine if already known")
	cmd.Flags().StringVarP(&o.Version, "version", "", "", "Version or tag e.g. 0.0.1")
	cmd.Flags().StringVarP(&o.Env, "environment", "e", "", "The Environment to find running applications")

}

// Run implements this command
func (o *GetCVEOptions) Run() error {

	_, _, err := o.KubeClient()
	if err != nil {
		return fmt.Errorf("cannot connect to kubernetes cluster: %v", err)
	}

	err = o.ensureCVEProviderRunning()
	if err != nil {
		return fmt.Errorf("no CVE provider running, have you tried `jx create addon anchore` %v", err)
	}

	server, auth, err := o.CommonOptions.getAddonAuthByKind("anchore-anchore-engine-core")
	if err != nil {
		return fmt.Errorf("error getting anchore engine auth details, %v", err)
	}

	p, err := cve.NewAnchoreProvider(server, auth)
	if err != nil {
		return fmt.Errorf("error creating anchore provider, %v", err)
	}
	table := o.CreateTable()
	table.AddRow("Name", "Version", "Severity", "Vulnerability", "URL", "Package", "Fix")

	query := cve.CVEQuery{
		ImageID:     o.ImageID,
		ImageName:   o.ImageName,
		Environment: o.Env,
		Vesion:      o.Version,
	}

	err = p.GetImageVulnerabilityTable(&table, query)
	if err != nil {
		return fmt.Errorf("error getting vulnerability table for image %s: %v", query.ImageID, err)
	}

	table.Render()
	return nil
}

func (o *GetCVEOptions) ensureCVEProviderRunning() error {
	isRunning, err := kube.IsDeploymentRunning(o.kubeClient, "anchore-anchore-engine-core", o.currentNamespace)
	if err != nil {
		return err
	}
	if isRunning {
		return nil
	}

	// TODO ask if user wants to intall a CVE provider addon

	return nil
}
