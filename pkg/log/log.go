package log

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
)

// ColorInfo returns a new function that returns info-colorized (green) strings for the
// given arguments with fmt.Sprint().
var colorInfo = color.New(color.FgGreen).SprintFunc()

// ColorError returns a new function that returns error-colorized (red) strings for the
// given arguments with fmt.Sprint().
var colorError = color.New(color.FgRed).SprintFunc()

func init() {
	if isInCluster() {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{})
	}
}

func Infof(msg string, args ...interface{}) {
	logrus.Infof(msg, args...)
}

func Info(msg string) {
	logrus.Info(msg)
}

func Blank() {
	fmt.Println()
}

func Warnf(msg string, args ...interface{}) {
	logrus.Warnf(msg, args...)
}

func Warn(msg string) {
	logrus.Warnf(msg)
}

func Errorf(msg string, args ...interface{}) {
	logrus.Errorf(msg, args...)
}

func Error(msg string) {
	logrus.Error(msg)
}

func Fatalf(msg string, args ...interface{}) {
	logrus.Fatalf(msg, args)
}

func Fatal(msg string) {
	logrus.Fatal(msg)
}

func Success(msg string) {
	logrus.Info(colorInfo(msg))
}

func Successf(msg string, args ...interface{}) {
	logrus.Infof(colorInfo(msg), args)
}

func Failure(msg string) {
	logrus.Info(colorError(msg))
}

// function to tell if we are running incluster
func isInCluster() bool {
	_, err := rest.InClusterConfig()
	if err != nil {
		return false
	}
	return true
}
