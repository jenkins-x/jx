package gerrit

import (
	"encoding/json"
	"errors"
	"strconv"
)

// Number is a string representing a number. This type is only used in cases
// where the API being queried may return an inconsistent result.
type Number string

// String returns the string representing the current number.
func (n *Number) String() string {
	return string(*n)
}

// Int returns the current number as an integer
func (n *Number) Int() (int, error) {
	return strconv.Atoi(n.String())
}

// UnmarshalJSON will marshal the provided data into the current *Number struct.
func (n *Number) UnmarshalJSON(data []byte) error {
	// `data` is a number represented as a string (ex. "5").
	var stringNumber string
	if err := json.Unmarshal(data, &stringNumber); err == nil {
		*n = Number(stringNumber)
		return nil
	}

	// `data` is a number represented as an integer (ex. 5). Here
	// we're using json.Unmarshal to convert bytes -> number which
	// we then convert to our own Number type.
	var number int
	if err := json.Unmarshal(data, &number); err == nil {
		*n = Number(strconv.Itoa(number))
		return nil
	}
	return errors.New("cannot convert data to number")
}
