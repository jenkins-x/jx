package util

import "github.com/fatih/color"

var ColorInfo = color.New(color.FgGreen).SprintFunc()
var ColorStatus = color.New(color.FgBlue).SprintFunc()
var ColorWarning = color.New(color.FgYellow).SprintFunc()
var ColorError = color.New(color.FgRed).SprintFunc()
