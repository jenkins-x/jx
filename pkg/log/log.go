package log

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
)

// colorStatus returns a new function that returns status-colorized (cyan) strings for the
// given arguments with fmt.Sprint().
var colorStatus = color.New(color.FgCyan).SprintFunc()

// colorWarn returns a new function that returns status-colorized (yellow) strings for the
// given arguments with fmt.Sprint().
var colorWarn = color.New(color.FgYellow).SprintFunc()

// colorInfo returns a new function that returns info-colorized (green) strings for the
// given arguments with fmt.Sprint().
var colorInfo = color.New(color.FgGreen).SprintFunc()

// colorError returns a new function that returns error-colorized (red) strings for the
// given arguments with fmt.Sprint().
var colorError = color.New(color.FgRed).SprintFunc()

// FormatLayoutType the layout kind
type FormatLayoutType string

const (
	// FormatLayoutJSON uses JSON layout
	FormatLayoutJSON FormatLayoutType = "json"

	// FormatLayoutText uses classic colorful Jenkins X layout
	FormatLayoutText FormatLayoutType = "text"
)

func init() {
	format := os.Getenv("JX_LOG_FORMAT")
	if format == "json" {
		SetFormatter(FormatLayoutJSON)
	} else {
		SetFormatter(FormatLayoutText)
	}
}

// SetFormatter sets the logrus format to use either text or JSON formatting
func SetFormatter(layout FormatLayoutType) {
	switch layout {
	case FormatLayoutJSON:
		logrus.SetFormatter(&logrus.JSONFormatter{})
	default:
		logrus.SetFormatter(NewJenkinsXTextFormat())
	}
}

// JenkinsXTextFormat lets use a custom text format
type JenkinsXTextFormat struct {
	ShowInfoLevel   bool
	ShowTimestamp   bool
	TimestampFormat string
}

// NewJenkinsXTextFormat creates the default Jenkins X text formatter
func NewJenkinsXTextFormat() *JenkinsXTextFormat {
	return &JenkinsXTextFormat{
		ShowInfoLevel:   false,
		ShowTimestamp:   false,
		TimestampFormat: "2006-01-02 15:04:05",
	}
}

// Format formats the log statement
func (f *JenkinsXTextFormat) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer

	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	level := strings.ToUpper(entry.Level.String())
	switch level {
	case "INFO":
		if f.ShowInfoLevel {
			b.WriteString(colorStatus(level))
			b.WriteString(": ")
		}
	case "WARN":
		b.WriteString(colorWarn(level))
		b.WriteString(": ")
	case "DEBUG":
		b.WriteString(colorStatus(level))
		b.WriteString(": ")
	default:
		b.WriteString(colorError(level))
		b.WriteString(": ")
	}
	if f.ShowTimestamp {
		b.WriteString(entry.Time.Format(f.TimestampFormat))
		b.WriteString(" - ")
	}

	b.WriteString(entry.Message)

	/*    if len(entry.Data) > 0 {
	          b.WriteString(" || ")
	      }
	      for key, value := range entry.Data {
	          b.WriteString(key)
	          b.WriteByte('=')
	          b.WriteByte('{')
	          fmt.Fprint(b, value)
	          b.WriteString("}, ")
	      }

	*/

	if !strings.HasSuffix(entry.Message, "\n") {
		b.WriteByte('\n')
	}
	return b.Bytes(), nil
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
