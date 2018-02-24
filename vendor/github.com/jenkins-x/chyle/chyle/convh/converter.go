package convh

import (
	"fmt"
	"strconv"
)

// parseBool replaces original ParseBool functions from strconv
// package to only convert "true" and "false" strings and nothing more
func parseBool(str string) (bool, error) {
	b, err := strconv.ParseBool(str)

	switch str {
	case "1", "t", "T", "TRUE", "True", "0", "f", "F", "FALSE", "False":
		return false, fmt.Errorf("can't convert %s to boolean", str)
	}

	return b, err
}

// GuessPrimitiveType extracts underlying primitive type from a string
// ,"true" will be translated as a boolean value for instance
func GuessPrimitiveType(str string) interface{} {
	if b, err := parseBool(str); err == nil {
		return b
	}

	if i, err := strconv.ParseInt(str, 10, 64); err == nil {
		return i
	}

	if f, err := strconv.ParseFloat(str, 64); err == nil {
		return f
	}

	return str
}

// ConvertToString extracts string value from an interface
func ConvertToString(value interface{}) (string, error) {
	switch v := value.(type) {
	case int:
		return strconv.Itoa(v), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("value can't be converted to string")
	}
}
