package cmd

import (
	"fmt"
	"log"

	"github.com/fatih/color"
)

func failure(err error) {
	c := color.New(color.FgRed)
	if _, err := c.Fprintf(writer, "%s\n", err.Error()); err != nil {
		log.Fatal(err)
	}
}

func printWithNewLine(str string) {
	if _, err := fmt.Fprintf(writer, "%s\n", str); err != nil {
		log.Fatal(err)
	}
}
