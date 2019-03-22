package main


import (
	"github.com/jenkins-x/jx/cmd/codegen/app"
	"os"
)

func main() {
	if err := app.Run(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
