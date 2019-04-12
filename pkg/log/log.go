package log

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/fatih/color"
)

func Blank() {
	logrus.Info()
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

type SimpleLogFormatter struct {
}

func (f *SimpleLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(fmt.Sprintf(entry.Message) + "\n"), nil
}

// ConfigureLog configures the logs level
func ConfigureLog(level string) {
	// Set warning level when the level is empty to avoid an unexpected exit
	if level == "" {
		level = "warn"
	}
	logrus.SetFormatter(&SimpleLogFormatter{})
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		logrus.Info(err.Error())
		os.Exit(-1)
	}
	logrus.SetLevel(lvl)
}
