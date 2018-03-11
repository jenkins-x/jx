package main

import (
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/tests/util"
)

var answer = []string{}

var table = []TestUtil.TestTableEntry{
	{
		"standard", &survey.MultiSelect{
			Message: "What days do you prefer:",
			Options: []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
		}, &answer,
	},
	{
		"default (sunday, tuesday)", &survey.MultiSelect{
			Message: "What days do you prefer:",
			Options: []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
			Default: []string{"Sunday", "Tuesday"},
		}, &answer,
	},
	{
		"default not found", &survey.MultiSelect{
			Message: "What days do you prefer:",
			Options: []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
			Default: []string{"Sundayaa"},
		}, &answer,
	},
	{
		"no help - type ?", &survey.MultiSelect{
			Message: "What days do you prefer:",
			Options: []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
			Default: []string{"Sundayaa"},
		}, &answer,
	},
	{
		"can navigate with j/k", &survey.MultiSelect{
			Message: "What days do you prefer:",
			Options: []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
			Default: []string{"Sundayaa"},
		}, &answer,
	},
}

func main() {
	TestUtil.RunTable(table)
}
