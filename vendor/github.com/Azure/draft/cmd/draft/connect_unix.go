// +build !windows

package main

import (
	"fmt"
	"os"
)

func exportConnectEnv(exportEnv map[string]string) {
	for envVar, envVal := range exportEnv {
		fmt.Fprintf(os.Stdout, " export %s=%q\n", envVar, envVal)
	}
}
