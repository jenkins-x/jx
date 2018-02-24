package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/antham/chyle/chyle"
	"github.com/antham/envh"
)

var envTree *envh.EnvTree
var debug bool

var writer io.Writer
var reader io.Reader

// RootCmd represents initial cobra command
var RootCmd = &cobra.Command{
	Use:   "chyle",
	Short: "Create a changelog from your commit history",
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		failure(err)
		exitError()
	}
}

func init() {
	reader = os.Stdin
	writer = os.Stdout

	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debugging")
}

func initConfig() {
	e, _ := envh.NewEnvTree("CHYLE", "_")

	envTree = &e

	chyle.EnableDebugging = debug
}
