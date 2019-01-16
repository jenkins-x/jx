package cmd

import "github.com/spf13/cobra"

//CommonDevPodOptions are common flags that are to be applied across all DevPod commands
type CommonDevPodOptions struct {
	Username string
}

func (o *CommonDevPodOptions) addCommonDevPodFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.Username, "username", "", "", "The username to create the DevPod. If not specified defaults to the current operating system user or $USER'")
}
