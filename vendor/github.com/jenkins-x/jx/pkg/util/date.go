package util

import (
	"fmt"
	"time"
)

const DateFormat = "January 2 2006"

func FormatDate(t time.Time) string {
	return fmt.Sprintf("%s %d %d", t.Month().String(), t.Day(), t.Year())
}

func ParseDate(dateText string) (time.Time, error) {
	return time.Parse(DateFormat, dateText)
}
