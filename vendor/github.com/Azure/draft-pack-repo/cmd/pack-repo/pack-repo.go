// Copyright (c) Microsoft Corporation. All rights reserved.
//
// Licensed under the MIT license.

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

const homeEnvVar = "DRAFT_HOME"

var (
	flagDebug   bool
	globalUsage = `The Draft pack repository plugin.
`
)

func newRootCmd(out io.Writer, in io.Reader) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "pack-repo",
		Short:        "the Draft pack repository plugin",
		Long:         globalUsage,
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if flagDebug {
				log.SetLevel(log.DebugLevel)
			}
		},
	}

	p := cmd.PersistentFlags()
	p.BoolVar(&flagDebug, "debug", false, "enable verbose output")

	cmd.AddCommand(
		newAddCmd(out),
		newListCmd(out),
		newRemoveCmd(out, in),
		newUpdateCmd(out),
		newVersionCmd(out),
	)
	return cmd
}

func homePath() string {
	return os.Getenv(homeEnvVar)
}

func debug(format string, args ...interface{}) {
	if flagDebug {
		format = fmt.Sprintf("[debug] %s\n", format)
		fmt.Printf(format, args...)
	}
}

func main() {
	cmd := newRootCmd(os.Stdout, os.Stdin)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func validateArgs(args, expectedArgs []string) error {
	if len(args) != len(expectedArgs) {
		return fmt.Errorf("This command needs %v argument(s): %v", len(expectedArgs), expectedArgs)
	}
	return nil
}

func promptYesOrNo(msg string, defaultToYes bool, input io.Reader, output io.Writer) (bool, error) {
	promptMessage := fmt.Sprintf("%s (y/N): ", msg)
	if defaultToYes {
		promptMessage = fmt.Sprintf("%s (Y/n): ", msg)
	}
	for {
		fmt.Fprint(output, promptMessage)
		reader := bufio.NewReader(input)
		answer, err := reader.ReadString('\n')
		if err != nil {
			return false, fmt.Errorf("Could not read input: %s", err)
		}
		answer = strings.ToLower(strings.TrimSpace(answer))
		switch answer {
		case "":
			return defaultToYes, nil
		case "y":
			return true, nil
		case "n":
			return false, nil
		default:
			fmt.Fprintln(output, "Please enter y or n.")
			continue
		}
	}
}
