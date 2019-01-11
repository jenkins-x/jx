package util

import (
	"encoding/hex"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// RegexpSplit splits a string into an array using the regexSep as a separator
func RegexpSplit(text string, regexSeperator string) []string {
	reg := regexp.MustCompile(regexSeperator)
	indexes := reg.FindAllStringIndex(text, -1)
	lastIdx := 0
	result := make([]string, len(indexes)+1)
	for i, element := range indexes {
		result[i] = text[lastIdx:element[0]]
		lastIdx = element[1]
	}
	result[len(indexes)] = text[lastIdx:]
	return result
}

// StringIndexes returns all the indices where the value occurs in the given string
func StringIndexes(text string, value string) []int {
	answer := []int{}
	t := text
	valueLen := len(value)
	offset := 0
	for {
		idx := strings.Index(t, value)
		if idx < 0 {
			break
		}
		answer = append(answer, idx+offset)
		offset += valueLen
		t = t[idx+valueLen:]
	}
	return answer
}

func StringArrayIndex(array []string, value string) int {
	for i, v := range array {
		if v == value {
			return i
		}
	}
	return -1
}

// StringArraysEqual returns true if the two string slices are equal
func StringArraysEqual(a1 []string, a2 []string) bool {
	if len(a1) != len(a2) {
		return false
	}
	for i := 0; i < len(a1); i++ {
		if a1[i] != a2[i] {
			return false
		}
	}
	return true
}

// FirstNotEmptyString returns the first non empty string or the empty string if none can be found
func FirstNotEmptyString(values ...string) string {
	if values != nil {
		for _, v := range values {
			if v != "" {
				return v
			}
		}
	}
	return ""
}

// SortedMapKeys returns the sorted keys of the given map
func SortedMapKeys(m map[string]string) []string {
	answer := []string{}
	for k, _ := range m {
		answer = append(answer, k)
	}
	sort.Strings(answer)
	return answer
}

func ReverseStrings(a []string) {
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
}

// StringArrayToLower returns a string slice with all the values converted to lower case
func StringArrayToLower(values []string) []string {
	answer := []string{}
	for _, v := range values {
		answer = append(answer, strings.ToLower(v))
	}
	return answer
}

// StringMatches returns true if the given text matches the includes/excludes lists
func StringMatchesAny(text string, includes []string, excludes []string) bool {
	for _, x := range excludes {
		if StringMatchesPattern(text, x) {
			return false
		}
	}
	if len(includes) == 0 {
		return true
	}
	for _, inc := range includes {
		if StringMatchesPattern(text, inc) {
			return true
		}
	}
	return false
}

// StringMatchesPattern returns true if the given text matches the includes/excludes lists
func StringMatchesPattern(text string, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(text, prefix)
	}
	return text == pattern
}


// StringsContaining if the filter is not empty return all the strings which contain the text
func StringsContaining(slice []string, filter string) []string {
	if filter == "" {
		return slice
	}
	answer := []string{}
	for _, text := range slice {
		if strings.Contains(text, filter) {
			answer = append(answer, text)
		}
	}
	return answer
}

// RandStringBytesMaskImprSrc returns a random hexadecimal string of length n.
func RandStringBytesMaskImprSrc(n int) (string, error) {
	src := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, (n+1)/2) // can be simplified to n/2 if n is always even

	if _, err := src.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b)[:n], nil
}

// DiffSlices compares the two slices and returns an array of items to delete from the old slice and a slice of
// new values to add to
func DiffSlices(oldSlice []string, newSlice []string) ([]string, []string) {
	toDelete := []string{}
	toInsert := []string{}

	for _, name := range oldSlice {
		if StringArrayIndex(newSlice, name) < 0 {
			toDelete = append(toInsert, name)
		}
	}
	for _, name := range newSlice {
		if StringArrayIndex(oldSlice, name) < 0 {
			toInsert = append(toInsert, name)
		}
	}
	return toDelete, toInsert
}

// ParseBool parses the boolean string. Returns false if the string is empty
func ParseBool(text string) (bool, error) {
	if text == "" {
		return false, nil
	}
	return strconv.ParseBool(text)
}

// CheckMark returns the check mark unicode character.
// We could configure this to use no color or avoid unicode using platform,  env vars or config?
func CheckMark() string {
	return "\u2705"
}

// RemoveStringFromSlice removes the first occurence of the specified string from a slice, if it exists and returns the result
func RemoveStringFromSlice(strings []string, toRemove string) []string {
	for i, str := range strings {
		if str == toRemove {
			return append(strings[:i], strings[i+1:]...)
		}
	}
	return strings
}
