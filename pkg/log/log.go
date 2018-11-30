package log

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/fatih/color"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

func Infof(msg string, args ...interface{}) {
	Info(fmt.Sprintf(msg, args...))
}

func Info(msg string) {
	fmt.Fprint(terminal.NewAnsiStdout(os.Stdout), msg)
}

func Infoln(msg string) {
	fmt.Fprintln(terminal.NewAnsiStdout(os.Stdout), msg)
}

func Blank() {
	fmt.Println()
}

func Warnf(msg string, args ...interface{}) {
	Warn(fmt.Sprintf(msg, args...))
}

func Warn(msg string) {
	color.Yellow(msg)
}

func Errorf(msg string, args ...interface{}) {
	Error(fmt.Sprintf(msg, args...))
}

func Error(msg string) {
	color.Red(msg)
}

// Prints an error msg with a new line at the end
func Errorln(msg string) {
	Errorf("%v\n", msg)
}

func Fatalf(msg string, args ...interface{}) {
	Fatal(fmt.Sprintf(msg, args...))
}

func Fatal(msg string) {
	color.Red(msg)
}

func Success(msg string) {
	color.Green(msg)
}

func Successf(msg string, args ...interface{}) {
	Success(fmt.Sprintf(msg, args...))
}

func Failure(msg string) {
	color.Red(msg)
}

func Failuref(msg string, args ...interface{}) {
	Failure(fmt.Sprintf(msg, args...))
}

// AskForConfirmation uses Scanln to parse user input. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user. Typically, you should use fmt to print out a question
// before calling askForConfirmation. E.g. fmt.Println("WARNING: Are you sure? (yes/no)")
func AskForConfirmation(def bool) bool {
	var response string
	fmt.Scanln(&response)
	if len(response) == 0 {
		return def
	}
	okayResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	nokayResponses := []string{"n", "N", "no", "No", "NO"}
	if containsString(okayResponses, response) {
		return true
	} else if containsString(nokayResponses, response) {
		return false
	} else {
		Warn("Please type y or n & press enter: ")
		return AskForConfirmation(def)
	}
}

// posString returns the first index of element in slice.
// If slice does not contain element, returns -1.
func posString(slice []string, element string) int {
	for index, elem := range slice {
		if elem == element {
			return index
		}
	}
	return -1
}

// containsString returns true iff slice contains element
func containsString(slice []string, element string) bool {
	return !(posString(slice, element) == -1)
}

type SimpleLogFormatter struct {
}

func (f *SimpleLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(fmt.Sprintf(entry.Message) + "\n"), nil
}

func ConfigureLog(level string) {
	logrus.SetFormatter(&SimpleLogFormatter{})
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}
	logrus.SetLevel(lvl)
}
