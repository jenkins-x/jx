package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/draft/manifest"
	"github.com/Azure/draft/pkg/draft/pack"
	"github.com/Azure/draft/pkg/draft/pack/repo"
	"github.com/Azure/draft/pkg/linguist"
	"github.com/Azure/draft/pkg/osutil"
)

const (
	draftToml  = "draft.toml"
	createDesc = `This command transforms the local directory to be deployable via 'draft up'.
`
)

// ErrNoLanguageDetected is raised when `draft create` does not detect source
// code for linguist to classify, or if there are no packs available for the detected languages.
var ErrNoLanguageDetected = errors.New("no languages were detected")

type createCmd struct {
	appName        string
	out            io.Writer
	pack           string
	home           draftpath.Home
	dest           string
	repositoryName string
}

func newCreateCmd(out io.Writer) *cobra.Command {
	cc := &createCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "create [path]",
		Short: "transform the local directory to be deployable to Kubernetes",
		Long:  createDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				cc.dest = args[0]
			}
			return cc.run()
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			// Docker naming convention doesn't allow upper case names for repositories.
			cc.normalizeApplicationName()
		},
	}

	cc.home = draftpath.Home(homePath())

	f := cmd.Flags()
	f.StringVarP(&cc.appName, "app", "a", "", "name of the Helm release. By default, this is a randomly generated name")
	f.StringVarP(&cc.pack, "pack", "p", "", "the named Draft starter pack to scaffold the app with")

	return cmd
}

func (c *createCmd) run() error {
	var err error
	mfest := manifest.New()

	if c.appName != "" {
		mfest.Environments[manifest.DefaultEnvironmentName].Name = c.appName
	}

	chartExists, err := osutil.Exists(filepath.Join(c.dest, pack.ChartsDir))
	if err != nil {
		return fmt.Errorf("there was an error checking if charts/ exists: %v", err)
	}
	if chartExists {
		// chart dir already exists, so we just tell the user that we are happily skipping the
		// process.
		fmt.Fprintln(c.out, "--> chart directory charts/ already exists. Ready to sail!")
		return nil
	}

	if c.pack != "" {
		// --pack was explicitly defined, so we can just lazily use that here. No detection required.
		packsFound, err := pack.Find(c.home.Packs(), c.pack)
		if err != nil {
			return err
		}
		if len(packsFound) == 0 {
			return fmt.Errorf("No packs found with name %s", c.pack)

		} else if len(packsFound) == 1 {
			packSrc := packsFound[0]
			if err = pack.CreateFrom(c.dest, packSrc); err != nil {
				return err
			}

		} else {
			return fmt.Errorf("Multiple packs named %s found: %v", c.pack, packsFound)
		}

	} else {
		// pack detection time
		packPath, err := doPackDetection(c.home, c.out)
		if err != nil {
			return err
		}
		err = pack.CreateFrom(c.dest, packPath)
		if err != nil {
			return err
		}
	}
	tomlFile := filepath.Join(c.dest, draftToml)
	draftToml, err := os.OpenFile(tomlFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer draftToml.Close()

	if err := toml.NewEncoder(draftToml).Encode(mfest); err != nil {
		return fmt.Errorf("could not write metadata to draft.toml: %v", err)
	}

	ignoreFile := filepath.Join(c.dest, ignoreFileName)
	if _, err := os.Stat(ignoreFile); os.IsNotExist(err) {
		d1 := []byte("*.swp\n*.tmp\n*.temp\n.git*\n")
		if err := ioutil.WriteFile(ignoreFile, d1, 0644); err != nil {
			return err
		}
	}

	fmt.Fprintln(c.out, "--> Ready to sail")
	return nil
}

func (c *createCmd) normalizeApplicationName() {
	if c.appName == "" {
		return
	}

	nameIsUpperCase := false
	for _, char := range c.appName {
		if unicode.IsUpper(char) {
			nameIsUpperCase = true
			break
		}
	}

	if !nameIsUpperCase {
		return
	}

	lowerCaseName := strings.ToLower(c.appName)
	fmt.Fprintf(
		c.out,
		"--> Application %s will be renamed to %s for docker compatibility\n",
		c.appName,
		lowerCaseName,
	)
	c.appName = lowerCaseName
}

// doPackDetection performs pack detection across all the packs available in $(draft home)/packs in
// alphabetical order, returning the pack dirpath and any errors that occurred during the pack detection.
func doPackDetection(home draftpath.Home, out io.Writer) (string, error) {
	langs, err := linguist.ProcessDir(".")
	log.Debugf("linguist.ProcessDir('.') result:\n\nError: %v", err)
	if err != nil {
		return "", fmt.Errorf("there was an error detecting the language: %s", err)
	}
	for _, lang := range langs {
		log.Debugf("%s:\t%f (%s)", lang.Language, lang.Percent, lang.Color)
	}
	if len(langs) == 0 {
		return "", ErrNoLanguageDetected
	}
	for _, lang := range langs {
		detectedLang := linguist.Alias(lang)
		fmt.Fprintf(out, "--> Draft detected %s (%f%%)\n", detectedLang.Language, detectedLang.Percent)
		for _, repository := range repo.FindRepositories(home.Packs()) {
			packDir := filepath.Join(repository.Dir, repo.PackDirName)
			packs, err := ioutil.ReadDir(packDir)
			if err != nil {
				return "", fmt.Errorf("there was an error reading %s: %v", packDir, err)
			}
			for _, file := range packs {
				if file.IsDir() {
					if strings.Compare(strings.ToLower(detectedLang.Language), strings.ToLower(file.Name())) == 0 {
						packPath := filepath.Join(packDir, file.Name())
						log.Debugf("pack path: %s", packPath)
						return packPath, nil
					}
				}
			}
		}
		fmt.Fprintf(out, "--> Could not find a pack for %s. Trying to find the next likely language match...\n", detectedLang.Language)
	}
	return "", ErrNoLanguageDetected
}
