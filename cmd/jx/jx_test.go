package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx/cmd/jx/app"
)

func TestSystem(t *testing.T) {
	systemTest := os.Getenv("SYSTEM_TEST") == "true"
	if systemTest {
		args := make([]string, 0)
		strippedArgs := make([]string, 0)
		for i, arg := range os.Args {
			if i > 0 && len(strippedArgs) > 0 && !strings.HasPrefix(arg, "-") && os.Args[i-1] == strippedArgs[len(strippedArgs)-1] && !strings.Contains(os.Args[i-1], "=") {
				// This is an argument for the previous string
				strippedArgs = append(strippedArgs, arg)
			} else if !strings.HasPrefix(arg, "-test.") {
				args = append(args, arg)
			} else {
				strippedArgs = append(strippedArgs, arg)
			}
		}
		fmt.Printf("This is a covered JX binary. Run with -test.coverprofile=mycover.out to generate coverage\n")
		fmt.Printf("Stripped arguments: %s\n", strings.Join(strippedArgs, ", "))
		fmt.Printf("Arguments passed to `jx` are: %s\n\n", strings.Join(args, ", "))
		// Purposefully ignore errors from app.Run as we are checking coverage
		err := app.Run(args)
		assert.NoError(t, err, "error executing jx")
	} else {
		if err := app.Run(nil); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
}
