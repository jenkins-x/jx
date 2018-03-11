package linguist

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Azure/draft/pkg/osutil"
	log "github.com/Sirupsen/logrus"
)

var (
	isIgnored                 func(string) bool
	isDetectedInGitAttributes func(filename string) string
)

// used for displaying results
type (
	// Language is the programming langage and the percentage on how sure linguist feels about its
	// decision.
	Language struct {
		Language string  `json:"language"`
		Percent  float64 `json:"percent"`
		// Color represents the color associated with the language in HTML hex notation.
		Color string `json:"color"`
	}
)

// sortableResult is a list or programming languages, sorted based on the likelihood of the
// primary programming language the application was written in.
type sortableResult []*Language

func (s sortableResult) Len() int {
	return len(s)
}

func (s sortableResult) Less(i, j int) bool {
	return s[i].Percent < s[j].Percent
}

func (s sortableResult) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func initLinguistAttributes(dir string) error {
	ignore := []string{}
	except := []string{}
	detected := make(map[string]string)

	gitignoreExists, err := osutil.Exists(filepath.Join(dir, ".gitignore"))
	if err != nil {
		return err
	}
	if gitignoreExists {
		log.Debugln("found .gitignore")

		f, err := os.Open(filepath.Join(dir, ".gitignore"))
		if err != nil {
			return err
		}
		defer f.Close()

		ignoreScanner := bufio.NewScanner(f)
		for ignoreScanner.Scan() {
			var isExcept bool
			path := strings.TrimSpace(ignoreScanner.Text())
			// if it's whitespace or a comment
			if len(path) == 0 || string(path[0]) == "#" {
				continue
			}
			if string(path[0]) == "!" {
				isExcept = true
				path = path[1:]
			}
			p := strings.Trim(path, string(filepath.Separator))
			if isExcept {
				except = append(except, p)
			} else {
				ignore = append(ignore, p)
			}
		}
		if err := ignoreScanner.Err(); err != nil {
			return fmt.Errorf("error reading .gitignore: %v", err)
		}
	}

	gitAttributesExists, err := osutil.Exists(filepath.Join(dir, ".gitattributes"))
	if err != nil {
		return err
	}
	if gitAttributesExists {
		log.Debugln("found .gitattributes")

		f, err := os.Open(filepath.Join(dir, ".gitattributes"))
		if err != nil {
			return err
		}
		defer f.Close()

		attributeScanner := bufio.NewScanner(f)
		var lineNumber int
		for attributeScanner.Scan() {
			lineNumber++
			line := strings.TrimSpace(attributeScanner.Text())
			words := strings.Fields(line)
			if len(words) != 2 {
				log.Printf("invalid line in .gitattributes at L%d: '%s'\n", lineNumber, line)
				continue
			}
			path := strings.Trim(words[0], string(filepath.Separator))
			attribute := words[1]
			if strings.HasPrefix(attribute, "linguist-documentation") || strings.HasPrefix(attribute, "linguist-vendored") || strings.HasPrefix(attribute, "linguist-generated") {
				if !strings.HasSuffix(strings.ToLower(attribute), "false") {
					ignore = append(except, path)
				}
			} else if strings.HasPrefix(attribute, "linguist-language") {
				attr := strings.Split(attribute, "=")
				if len(attr) != 2 {
					log.Printf("invalid line in .gitattributes at L%d: '%s'\n", lineNumber, line)
					continue
				}
				language := attr[1]
				detected[path] = language
			}
		}
		if err := attributeScanner.Err(); err != nil {
			return fmt.Errorf("error reading .gitattributes: %v", err)
		}
	}

	isIgnored = func(filename string) bool {
		for _, p := range ignore {
			if m, _ := filepath.Match(p, strings.TrimPrefix(filename, dir+string(filepath.Separator))); m {
				for _, e := range except {
					if m, _ := filepath.Match(e, strings.TrimPrefix(filename, dir+string(filepath.Separator))); m {
						return false
					}
				}
				return true
			}
		}
		return false
	}
	isDetectedInGitAttributes = func(filename string) string {
		for p, lang := range detected {
			if m, _ := filepath.Match(p, strings.TrimPrefix(filename, dir+string(filepath.Separator))); m {
				return lang
			}
		}
		return ""
	}
	return nil
}

// shoutouts to php
func fileGetContents(filename string) ([]byte, error) {
	log.Debugln("reading contents of", filename)

	// read only first 512 bytes of files
	contents := make([]byte, 512)
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	_, err = f.Read(contents)
	f.Close()
	if err != io.EOF {
		if err != nil {
			return nil, err
		}
	}
	return contents, nil
}

// ProcessDir walks through a directory and returns a list of sorted languages within that directory.
func ProcessDir(dirname string) ([]*Language, error) {
	var (
		langs     = make(map[string]int)
		totalSize int
	)
	if err := initLinguistAttributes(dirname); err != nil {
		return nil, err
	}
	exists, err := osutil.Exists(dirname)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, os.ErrNotExist
	}
	filepath.Walk(dirname, func(path string, file os.FileInfo, err error) error {
		size := int(file.Size())
		log.Debugln("with file: ", path)
		log.Debugln(path, "is", size, "bytes")
		if size == 0 {
			log.Debugln(path, "is empty file, skipping")
			return nil
		}
		if isIgnored(path) {
			log.Debugln(path, "is ignored, skipping")
			if file.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if file.IsDir() {
			if file.Name() == ".git" {
				log.Debugln(".git directory, skipping")
				return filepath.SkipDir
			}
		} else if (file.Mode() & os.ModeSymlink) == 0 {
			if ShouldIgnoreFilename(path) {
				log.Debugln(path, ": filename should be ignored, skipping")
				return nil
			}

			byGitAttr := isDetectedInGitAttributes(path)
			if byGitAttr != "" {
				log.Debugln(path, "got result by .gitattributes: ", byGitAttr)
				langs[byGitAttr] += size
				totalSize += size
				return nil
			}

			if byName := LanguageByFilename(path); byName != "" {
				log.Debugln(path, "got result by name: ", byName)
				langs[byName] += size
				totalSize += size
				return nil
			}

			contents, err := fileGetContents(path)
			if err != nil {
				return err
			}

			if ShouldIgnoreContents(contents) {
				log.Debugln(path, ": contents should be ignored, skipping")
				return nil
			}

			hints := LanguageHints(path)
			log.Debugf("%s got language hints: %#v\n", path, hints)
			byData := LanguageByContents(contents, hints)

			if byData != "" {
				log.Debugln(path, "got result by data: ", byData)
				langs[byData] += size
				totalSize += size
				return nil
			}

			log.Debugln(path, "got no result!!")
			langs["(unknown)"] += size
			totalSize += size
		}
		return nil
	})

	results := []*Language{}
	for lang, size := range langs {
		l := &Language{
			Language: lang,
			Percent:  (float64(size) / float64(totalSize)) * 100.0,
			Color:    LanguageColor(lang),
		}
		results = append(results, l)
		log.Debugf("language: %s percent: %f color: %s", l.Language, l.Percent, l.Color)
	}
	sort.Sort(sort.Reverse(sortableResult(results)))
	return results, nil
}

// Alias returns the language name for a given known alias.
//
// Occasionally linguist comes up with odd language names, or determines a Java app as a "Maven POM"
// app, which in essence is the same thing for Draft's intent.
func Alias(lang *Language) *Language {
	packAliases := map[string]string{
		"maven pom": "Java",
		"c#":        "csharp",
	}

	if alias, ok := packAliases[strings.ToLower(lang.Language)]; ok {
		lang.Language = alias
	}
	return lang
}
