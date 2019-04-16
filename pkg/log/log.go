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

// Debugf debug logging with arguments
func Debugf(msg string, args ...interface{}) {
	logrus.Debugf(msg, args...)
}

// Debug debug logging
func Debug(msg string) {
	logrus.Debug(msg)
}

// Infof info logging with arguments
func Infof(msg string, args ...interface{}) {
	logrus.Infof(msg, args...)
}

// Info info logging
func Info(msg string) {
	logrus.Info(msg)
}

// Blank prints a blank line
func Blank() {
	fmt.Println()
}

// Warnf warning logging with arguments
func Warnf(msg string, args ...interface{}) {
	logrus.Warnf(msg, args...)
}

// Warn warning logging
func Warn(msg string) {
	logrus.Warnf(msg)
}

// Errorf warning logging with arguments
func Errorf(msg string, args ...interface{}) {
	logrus.Errorf(msg, args...)
}

// Error warning logging
func Error(msg string) {
	logrus.Error(msg)
}

// Fatalf logging with arguments
func Fatalf(msg string, args ...interface{}) {
	logrus.Fatalf(msg, args...)
}

// Fatal logging
func Fatal(msg string) {
	logrus.Fatal(msg)
}

// Success grean logging
func Success(msg string) {
	logrus.Info(colorInfo(msg))
}

// Successf grean logging with arguments
func Successf(msg string, args ...interface{}) {
	logrus.Infof(colorInfo(msg), args...)
}

// Failure red logging
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
