package main

import (
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/tests/util"
)

var answer = ""

var goodTable = []TestUtil.TestTableEntry{
	{
		"should open in editor", &survey.Editor{
			Message: "should open",
		}, &answer,
	},
	{
		"has help", &survey.Editor{
			Message: "press ? to see message",
			Help:    "Does this work?",
		}, &answer,
	},
	{
		"should not include the default value in the prompt", &survey.Editor{
			Message:     "the default value 'Hello World' should not include in the prompt",
			HideDefault: true,
			Default:     "Hello World",
		}, &answer,
	},
	{
		"should write the default value to the temporary file before the launch of the editor", &survey.Editor{
			Message:       "the default value 'Hello World' is written to the temporary file before the launch of the editor",
			AppendDefault: true,
			Default:       "Hello World",
		}, &answer,
	},
}

func main() {
	TestUtil.RunTable(goodTable)
}
