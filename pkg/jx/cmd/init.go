package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

// InitOptions the flags for running init
type InitOptions struct {
	CommonOptions

	Flags    InitFlags
	Provider KubernetesProvider
}

type InitFlags struct {
}

var (
	initLong = templates.LongDesc(`
		This command installs the Jenkins-X platform on a connected kubernetes cluster
`)

	initExample = templates.Examples(`
		jx init
`)
)

// NewCmdGet creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a kubernetes cluster.
func NewCmdInit(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &InitOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Init Jenkins-X",
		Long:    initLong,
		Example: initExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	//cmd.Flags().StringP("git-provider", "", "GitHub", "Git provider, used to create tokens if not provided.  Supported providers: [GitHub]")

	// check if connected to a kube environment?

	return cmd
}

func (options *InitOptions) Run() error {

	return nil
}
