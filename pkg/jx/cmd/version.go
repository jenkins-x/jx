package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/version"
)

func NewCmdVersion(f cmdutil.Factory, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Run: func(cmd *cobra.Command, args []string) {
			err := RunVersion(f, out, cmd)
			cmdutil.CheckErr(err)
		},
	}
	/*
		cmd.Flags().BoolP("client", "c", false, "Client version only (no server required).")
		cmd.Flags().BoolP("short", "", false, "Print just the version number.")
	*/
	cmd.Flags().MarkShorthandDeprecated("client", "please use --client instead.")
	return cmd
}

func RunVersion(f cmdutil.Factory, out io.Writer, cmd *cobra.Command) error {
	v := fmt.Sprintf("%#v", version.GetVersion())
	/*
		if cmdutil.GetFlagBool(cmd, "short") {
			v = version.Get().GitVersion
		}

		fmt.Fprintf(out, "Client Version: %s\n", v)
		if cmdutil.GetFlagBool(cmd, "client") {
			return nil
		}

		clientset, err := f.ClientSet()
		if err != nil {
			return err
		}

		serverVersion, err := clientset.Discovery().ServerVersion()
		if err != nil {
			return err
		}

		v = fmt.Sprintf("%#v", *serverVersion)
		if cmdutil.GetFlagBool(cmd, "short") {
			v = serverVersion.GitVersion
		}
	*/

	fmt.Fprintf(out, "Server Version: %s\n", v)
	return nil
}
