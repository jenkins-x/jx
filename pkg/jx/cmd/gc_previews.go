package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"strconv"

	"strings"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/log"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type GCPreviewsOptions struct {
	CommonOptions

	DisableImport bool
	OutDir        string
}

var (
	GCPreviewsLong = templates.LongDesc(`
		Garbage collect Jenkins X preview environments.  If a pull request is merged or closed the associated preview
		environment will be deleted.

`)

	GCPreviewsExample = templates.Examples(`
		jx garbage collect previews
		jx gc previews
`)
)

// NewCmd s a command object for the "step" command
func NewCmdGCPreviews(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GCPreviewsOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "previews",
		Short:   "garbage collection for preview environments",
		Long:    GCPreviewsLong,
		Example: GCPreviewsExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GCPreviewsOptions) Run() error {
	f := o.Factory
	client, currentNs, err := f.CreateJXClient()
	if err != nil {
		return err
	}

	// cannot use field selectors like `spec.kind=Preview` on CRDs so list all environments
	envs, err := client.JenkinsV1().Environments(currentNs).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(envs.Items) == 0 {
		// no preview environments found so lets return gracefully
		if o.Verbose {
			log.Info("no preview environments found\n")
		}
		return nil
	}

	for _, e := range envs.Items {
		if e.Spec.Kind == v1.EnvironmentKindTypePreview {
			gitInfo, err := gits.ParseGitURL(e.Spec.Source.URL)
			if err != nil {
				return err
			}
			// we need pull request info to include
			authConfigSvc, err := o.CreateGitAuthConfigService()
			if err != nil {
				return err
			}

			gitKind, err := o.GitServerKind(gitInfo)
			if err != nil {
				return err
			}

			gitProvider, err := gitInfo.CreateProvider(authConfigSvc, gitKind, o.Git())
			if err != nil {
				return err
			}
			prNum, err := strconv.Atoi(e.Spec.PreviewGitSpec.Name)
			if err != nil {
				log.Warn("Unable to convert PR " + e.Spec.PreviewGitSpec.Name + " to a number" + "\n")
			}
			pullRequest, err := gitProvider.GetPullRequest(gitInfo.Organisation, gitInfo, prNum)
			if err != nil {
				return err
			}

			lowerState := strings.ToLower(*pullRequest.State)

			if strings.HasPrefix(lowerState, "clos") {
				// lets delete the preview environment
				deleteOpts := DeleteEnvOptions{
					DeleteNamespace: true,
					CommonOptions:   o.CommonOptions,
				}
				deleteOpts.CommonOptions.Args = []string{e.Name}
				err = deleteOpts.Run()
				if err != nil {
					return fmt.Errorf("failed to delete preview environment %s: %v\n", e.Name, err)
				}
			}
		}
	}
	return nil
}
