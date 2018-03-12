package linguist

import (
	"bufio"
	"bytes"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ghodss/yaml"
)

var (
	vendorRE *regexp.Regexp
	doxRE    *regexp.Regexp

	extensions   = map[string][]string{}
	filenames    = map[string][]string{}
	interpreters = map[string][]string{}
	colors       = map[string]string{}

	shebangRE       = regexp.MustCompile(`^#!\s*(\S+)(?:\s+(\S+))?.*`)
	scriptVersionRE = regexp.MustCompile(`((?:\d+\.?)+)`)
)

func init() {
	var regexps []string
	bytes := []byte(files["data/vendor.yml"])
	if err := yaml.Unmarshal(bytes, &regexps); err != nil {
		log.Fatal(err)
		return
	}
	vendorRE = regexp.MustCompile(strings.Join(regexps, "|"))

	var moreregex []string
	bytes = []byte(files["data/documentation.yml"])
	if err := yaml.Unmarshal(bytes, &moreregex); err != nil {
		log.Fatal(err)
		return
	}
	doxRE = regexp.MustCompile(strings.Join(moreregex, "|"))

	type language struct {
		Extensions   []string `yaml:"extensions,omitempty"`
		Filenames    []string `yaml:"filenames,omitempty"`
		Interpreters []string `yaml:"interpreters,omitempty"`
		Color        string   `yaml:"color,omitempty"`
	}
	languages := map[string]*language{}

	bytes = []byte(files["data/languages.yml"])
	if err := yaml.Unmarshal(bytes, &languages); err != nil {
		log.Fatal(err)
	}

	for n, l := range languages {
		for _, e := range l.Extensions {
			extensions[e] = append(extensions[e], n)
		}
		for _, f := range l.Filenames {
			filenames[f] = append(filenames[f], n)
		}
		for _, i := range l.Interpreters {
			interpreters[i] = append(interpreters[i], n)
		}
		colors[n] = l.Color
	}
}

// LanguageColor is a convenience function that returns the color associated
// with the language, in HTML Hex notation (e.g. "#123ABC")
// from the languages.yml file provided by https://github.com/github/linguist
//
// Returns the empty string if there is no associated color for the language.
func LanguageColor(language string) string {
	if c, ok := colors[language]; ok {
		return c
	}
	return ""
}

// LanguageByFilename attempts to determine the language of a source file based solely on
// common naming conventions and file extensions
// from the languages.yml file provided by https://github.com/github/linguist
//
// Returns the empty string in ambiguous or unrecognized cases.
func LanguageByFilename(filename string) string {
	if l := filenames[filename]; len(l) == 1 {
		return l[0]
	}
	ext := filepath.Ext(filename)
	if ext != "" {
		if l := extensions[ext]; len(l) == 1 {
			return l[0]
		}
	}
	return ""
}

// LanguageHints attempts to detect all possible languages of a source file based solely on
// common naming conventions and file extensions
// from the languages.yml file provided by https://github.com/github/linguist
//
// Intended to be used with LanguageByContents.
//
// May return an empty slice.
func LanguageHints(filename string) (hints []string) {
	if l, ok := filenames[filename]; ok {
		hints = append(hints, l...)
	}
	if ext := filepath.Ext(filename); ext != "" {
		if l, ok := extensions[ext]; ok {
			hints = append(hints, l...)
		}
	}
	return hints
}

// LanguageByContents attempts to detect the language of a source file based on its
// contents and a slice of hints to the possible answer.
//
// Obtain hints with LanguageHints()
//
// Returns the empty string a language could not be determined.
func LanguageByContents(contents []byte, hints []string) string {
	interpreter := detectInterpreter(contents)
	if interpreter != "" {
		if l := interpreters[interpreter]; len(l) == 1 {
			return l[0]
		}
	}
	return Analyse(contents, hints)
}

func detectInterpreter(contents []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(contents))
	scanner.Scan()
	line := scanner.Text()
	m := shebangRE.FindStringSubmatch(line)
	if m == nil || len(m) != 3 {
		return ""
	}
	base := filepath.Base(m[1])
	if base == "env" && m[2] != "" {
		base = m[2]
	}
	// Strip suffixed version number.
	return scriptVersionRE.ReplaceAllString(base, "")
}

// ShouldIgnoreFilename checks if filename should not be passed to LanguageByFilename.
//
// (this simply calls IsVendored and IsDocumentation)
func ShouldIgnoreFilename(filename string) bool {
	vendored := IsVendored(filename)
	documentation := IsDocumentation(filename)
	return vendored || documentation
	// return IsVendored(filename) || IsDocumentation(filename)
}

// ShouldIgnoreContents checks if contents should not be passed to LangugeByContents.
//
// (this simply calls IsBinary)
func ShouldIgnoreContents(contents []byte) bool {
	return IsBinary(contents)
}

// IsVendored checks if path contains a filename commonly belonging to configuration files.
func IsVendored(path string) bool {
	return vendorRE.MatchString(path)
}

// IsDocumentation checks if path contains a filename commonly belonging to documentation.
func IsDocumentation(path string) bool {
	return doxRE.MatchString(path)
}

// IsBinary checks contents for known character escape codes which
// frequently show up in binary files but rarely (if ever) in text.
//
// Use this check before using LanguageFromContents to reduce likelihood
// of passing binary data into it which can cause inaccurate results.
func IsBinary(contents []byte) bool {
	// NOTE(tso): preliminary testing on this method of checking for binary
	// contents were promising, having fed a document consisting of all
	// utf-8 codepoints from 0000 to FFFF with satisfactory results. Thanks
	// to robpike.io/cmd/unicode:
	// ```
	// unicode -c $(seq 0 65535 | xargs printf "%04x ") | tr -d '\n' > unicode_test
	// ```
	//
	// However, the intentional presence of character escape codes to throw
	// this function off is entirely possible, as is, potentially, a binary
	// file consisting entirely of the 4 exceptions to the rule for the first
	// 512 bytes. It is also possible that more character escape codes need
	// to be added.
	//
	// Further analysis and real world testing of this is required.
	for n, b := range contents {
		if n >= 512 {
			break
		}
		if b < 32 {
			switch b {
			case 0:
				fallthrough
			case 9:
				fallthrough
			case 10:
				fallthrough
			case 13:
				continue
			default:
				return true
			}
		}
	}
	return false
}
