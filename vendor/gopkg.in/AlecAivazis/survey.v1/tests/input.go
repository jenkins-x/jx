package main

import (
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/tests/util"
)

var val = ""

var table = []TestUtil.TestTableEntry{
	{
		"no default", &survey.Input{Message: "Hello world"}, &val,
	},
	{
		"default", &survey.Input{Message: "Hello world", Default: "default"}, &val,
	},
	{
		"no help, send '?'", &survey.Input{Message: "Hello world"}, &val,
	},
	{
		"Home, End Button test in random location", &survey.Input{Message: "Hello world"}, &val,
	},{
		"Delete and forward delete test at random location (test if screen overflows)", &survey.Input{Message: "Hello world"}, &val,
	},{
		"Moving around lines with left & right arrow keys", &survey.Input{Message: "Hello world"}, &val,
	},
}

func main() {
	TestUtil.RunTable(table)
}
