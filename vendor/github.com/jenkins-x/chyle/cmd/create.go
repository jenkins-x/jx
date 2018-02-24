package cmd

import (
	"github.com/spf13/cobra"

	"github.com/antham/chyle/chyle"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new changelog",
	Long: `Create a new changelog according to what is defined as config.

Changelog creation follows this process :

1 - fetch commits
2 - filter relevant commits
3 - extract informations from commits fields and publish them to new fields
4 - enrich extracted datas with external apps
5 - publish datas`,
	Run: func(cmd *cobra.Command, args []string) {
		err := chyle.BuildChangelog(envTree)

		if err != nil {
			failure(err)

			exitError()
		}

		exitSuccess()
	},
}

func init() {
	RootCmd.AddCommand(createCmd)
}
