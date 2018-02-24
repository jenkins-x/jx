package cmd

import (
	"fmt"

	"github.com/antham/chyle/prompt"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration prompt",
	Run: func(cmd *cobra.Command, args []string) {
		prompts := prompt.New(reader, writer)

		p := prompts.Run()

		printWithNewLine("")
		printWithNewLine("Generated configuration :")
		printWithNewLine("")

		for key, value := range (map[string]string)(p) {
			printWithNewLine(fmt.Sprintf(`export %s="%s"`, key, value))
		}
	},
}

func init() {
	RootCmd.AddCommand(configCmd)
}
