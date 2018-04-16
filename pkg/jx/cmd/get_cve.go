package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"fmt"

	"github.com/jenkins-x/jx/pkg/cve"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// GetGitOptions the command line options
type GetCVEOptions struct {
	GetOptions
	App               string
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
	cmd.Flags().StringVarP(&o.App, "app", "", "", "Application name")
	cmd.Flags().StringVarP(&o.Version, "version", "", "", "Application version")
	cmd.Flags().StringVarP(&o.Env, "environment", "e", "", "The Environment to find running applications")
	cmd.Flags().StringVarP(&o.VulnerabilityType, "vulnerability-type", "t", "os", "Vulnerability Type")
}

// Run implements this command
func (o *GetCVEOptions) Run() error {
	server, auth, err := o.CommonOptions.getAddonAuthByKind("anchore-anchore-engine-core")
	if err != nil {
		return fmt.Errorf("error getting anchore engine auth details, %v", err)
	}

	p, err := cve.NewAnchoreProvider(server, auth)
	if err != nil {
		return fmt.Errorf("error creating anchore provider, %v", err)
	}
	table := o.CreateTable()
	image := "sha256:b9f03c3c4b196d46639bee0ec9cd0f6dbea8cc39d32767c8312f04317c3b18f4"
	err = p.GetImageVulnerabilityTable(&table, image)
	if err != nil {
		return fmt.Errorf("error getting vilnerability table for image %s: %v", image, err)
	}

	table.Render()
	return nil

}
