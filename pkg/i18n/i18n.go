package i18n

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chai2010/gettext-go/gettext"
)

var knownTranslations = map[string][]string{
	"jx": {
		"default",
		"en_US",
		"zh_CN",
	},
	// only used for unit tests.
	"test": {
		"default",
		"en_US",
	},
}

func loadSystemLanguage() string {
	// Implements the following locale priority order: LC_ALL, LC_MESSAGES, LANG
	// Similarly to: https://www.gnu.org/software/gettext/manual/html_node/Locale-Environment-Variables.html
	langStr := os.Getenv("LC_ALL")
	if langStr == "" {
		langStr = os.Getenv("LC_MESSAGES")
	}
	if langStr == "" {
		langStr = os.Getenv("LANG")
	}
	if langStr == "" {
		return "default"
	}
	pieces := strings.Split(langStr, ".")
	if len(pieces) != 2 {
		return "default"
	}
	return pieces[0]
}

func findLanguage(root string, getLanguageFn func() string) string {
	langStr := getLanguageFn()

	translations := knownTranslations[root]
	for ix := range translations {
		if translations[ix] == langStr {
			return langStr
		}
	}
	return "default"
}

// LoadTranslations loads translation files. getLanguageFn should return a language
// string (e.g. 'en-US'). If getLanguageFn is nil, then the loadSystemLanguage function
// is used, which uses the 'LANG' environment variable.
func LoadTranslations(root string, getLanguageFn func() string) (err error) {
	if getLanguageFn == nil {
		getLanguageFn = loadSystemLanguage
	}

	langStr := findLanguage(root, getLanguageFn)
	translationFiles := []string{
		"jx/zh_CN/LC_MESSAGES/jx.po",
	}

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// Make sure to check the error on Close.
	var f io.Writer
	var data []byte
	for _, file := range translationFiles {
		filename := file
		if f, err = w.Create(file); err != nil {
			return
		}

		if data, err = Asset(filename); err != nil {
			return
		}

		if _, err = f.Write(data); err != nil {
			return
		}
	}

	if err = w.Close(); err == nil {
		gettext.BindTextdomain("jx", root+".zip", buf.Bytes())
		gettext.Textdomain("jx")
		gettext.SetLocale(langStr)
	}
	return
}

var i18nLoaded = false

// T translates a string, possibly substituting arguments into it along
// the way. If len(args) is > 0, args1 is assumed to be the plural value
// and plural translation is used.
func T(defaultValue string, args ...int) string {
	if !i18nLoaded {
		i18nLoaded = true
		if err := LoadTranslations("jx", nil); err != nil {
			fmt.Println(err)
		}
	}

	if len(args) == 0 {
		return gettext.PGettext("", defaultValue)
	}
	return fmt.Sprintf(gettext.PNGettext("", defaultValue, defaultValue+".plural", args[0]),
		args[0])
}
