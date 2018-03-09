// Package tokenizer is a go port of https://github.com/github/linguist/blob/master/lib/linguist/tokenizer.rb
//
// in their words:
//
//  # Generic programming language tokenizer.
//  #
//  # Tokens are designed for use in the language bayes classifier.
//  # It strips any data strings or comments and preserves significant
//  # language symbols.
//
package tokenizer

import (
	"bufio"
	"bytes"
	"regexp"
)

var (
	// ByteLimit is the maximum input length for Tokenize()
	ByteLimit = 100000

	// StartLineComments turns string slices into their regexp slice counterparts
	// by this package's init() function.
	StartLineComments = []string{
		"\"", // Vim
		"%",  // Tex
	}
	// SingleLineComments turns string slices into their regexp slice counterparts
	// by this package's init() function.
	SingleLineComments = []string{
		"//", // C
		"--", // Ada, Haskell, AppleScript
		"#",  // Perl, Bash, Ruby
	}
	// MultiLineComments turns string slices into their regexp slice counterparts
	// by this package's init() function.
	MultiLineComments = [][]string{
		{"/*", "*/"},    // C
		{"<!--", "-->"}, // XML
		{"{-", "-}"},    // Haskell
		{"(*", "*)"},    // Coq
		{`"""`, `"""`},  // Python
		{"'''", "'''"},  // Python
		{"#`(", ")"},    // Perl6
	}
	startLineComment       []*regexp.Regexp
	beginSingleLineComment []*regexp.Regexp
	beginMultiLineComment  []*regexp.Regexp
	endMultiLineComment    []*regexp.Regexp
	stringRegexp           = regexp.MustCompile(`[^\\]*(["'` + "`])")
	numberRegexp           = regexp.MustCompile(`(0x[0-9a-f]([0-9a-f]|\.)*|\d(\d|\.)*)([uU][lL]{0,2}|([eE][-+]\d*)?[fFlL]*)`)
)

func init() {
	for _, st := range append(StartLineComments, SingleLineComments...) {
		startLineComment = append(startLineComment, regexp.MustCompile(`^\s*`+regexp.QuoteMeta(st)))
	}
	for _, sl := range SingleLineComments {
		beginSingleLineComment = append(beginSingleLineComment, regexp.MustCompile(regexp.QuoteMeta(sl)))
	}
	for _, ml := range MultiLineComments {
		beginMultiLineComment = append(beginMultiLineComment, regexp.MustCompile(regexp.QuoteMeta(ml[0])))
		endMultiLineComment = append(endMultiLineComment, regexp.MustCompile(regexp.QuoteMeta(ml[1])))
	}
}

// FindMultiLineComment compares a given token to the start of a multiline comment
// and if true, returns the bool with a regex. Otherwise false and nil.
func FindMultiLineComment(token []byte) (matched bool, terminator *regexp.Regexp) {
	for idx, re := range beginMultiLineComment {
		if re.Match(token) {
			return true, endMultiLineComment[idx]
		}
	}
	return false, nil
}

// Tokenize is a simple tokenizer that uses bufio.Scanner to process lines and individual words
// and matches them against regular expressions to filter out comments, strings, and numerals
// in a manner very similar to github's linguist (see https://github.com/github/linguist/blob/master/lib/linguist/tokenizer.rb)
//
// The intention is to merely retrieve significant tokens from a piece of source code
// in order to identify the programming language using statistical analysis
// and NOT to be used as any part of the process of compilation whatsoever.
//
// NOTE(tso): The tokens produced by this function may be of a dubious quality due to the approach taken.
// Feedback and alternate implementations welcome :)
func Tokenize(input []byte) (tokens []string) {
	if len(input) == 0 {
		return tokens
	}
	if len(input) >= ByteLimit {
		input = input[:ByteLimit]
	}

	var (
		mlStart     = false        // in a multiline comment
		mlEnd       *regexp.Regexp // closing token regexp
		stringStart = false        // in a string literal
		stringEnd   byte           // closing token byte to be found by the String regexp
	)

	buf := bytes.NewBuffer(input)
	scanlines := bufio.NewScanner(buf)
	scanlines.Split(bufio.ScanLines)

	// NOTE(tso): the use of goto here is probably interchangeable with continue
line:
	for scanlines.Scan() {
		ln := scanlines.Bytes()

		for _, re := range startLineComment {
			if re.Match(ln) {
				goto line
			}
		}

		// NOTE(tso): bufio.Scanner.Split(bufio.ScanWords) seems to just split on whitespace
		// this may yield inaccurate results where there is a lack of sufficient
		// whitespace for the approaches taken here, i.e. jumping straight to the
		// next word/line boundary.
		lnBuffer := bytes.NewBuffer(ln)
		scanwords := bufio.NewScanner(lnBuffer)
		scanwords.Split(bufio.ScanWords)
	word:
		for scanwords.Scan() {
			tokenBytes := scanwords.Bytes()
			tokenString := scanwords.Text()

			// find end of multi-line comment
			if mlStart {
				if mlEnd.Match(tokenBytes) {
					mlStart = false
					mlEnd = nil
				}
				goto word
			}

			// find end of string literal
			if stringStart {
				s := stringRegexp.FindSubmatch(tokenBytes)
				if s != nil && s[1][0] == stringEnd {
					stringStart = false
					stringEnd = 0
				}
				goto word
			}

			// find single-line comment
			for _, re := range beginSingleLineComment {
				if re.Match(tokenBytes) {
					goto line
				}
			}

			// find start of multi-line comment
			if matched, terminator := FindMultiLineComment(tokenBytes); matched {
				mlStart = true
				mlEnd = terminator
				goto word
			}

			// find start of string literal
			if s := stringRegexp.FindSubmatch(tokenBytes); s != nil {
				stringStart = true
				stringEnd = s[1][0]
				goto word
			}

			// find numeric literal
			if n := numberRegexp.Find(tokenBytes); n != nil {
				goto word
			}

			// add valid tokens to result set
			tokens = append(tokens, tokenString)
		}
	}
	return tokens
}
