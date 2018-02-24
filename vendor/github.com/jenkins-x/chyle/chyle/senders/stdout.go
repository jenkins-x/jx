package senders

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/antham/chyle/chyle/tmplh"
	"github.com/antham/chyle/chyle/types"
)

type stdoutConfig struct {
	FORMAT   string
	TEMPLATE string
}

// jSONStdout output commit payload as JSON on stdout
type jSONStdout struct {
	stdout io.Writer
}

func (j jSONStdout) Send(changelog *types.Changelog) error {
	return json.NewEncoder(j.stdout).Encode(changelog)
}

type templateStdout struct {
	stdout   io.Writer
	template string
}

func (t templateStdout) Send(changelog *types.Changelog) error {
	datas, err := tmplh.Build("stdout-template", t.template, changelog)

	if err != nil {
		return err
	}

	fmt.Fprint(t.stdout, datas)

	return nil
}

func newStdout(config stdoutConfig) Sender {
	if config.FORMAT == "json" {
		return newJSONStdout()
	}

	return newTemplateStdout(config.TEMPLATE)
}

func newJSONStdout() Sender {
	return jSONStdout{
		os.Stdout,
	}
}

func newTemplateStdout(template string) Sender {
	return templateStdout{
		os.Stdout,
		template,
	}
}
