package main

import (
	"os"

	"github.com/jenkins-x/jx/v2/cmd/codegen/app"
)

func main() {
	if err := app.Run(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
