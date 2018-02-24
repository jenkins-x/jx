package chyle

import (
	"log"
	"os"

	"github.com/antham/chyle/chyle/config"
	"github.com/antham/chyle/chyle/git"

	"github.com/antham/envh"
)

var logger *log.Logger

func init() {
	logger = log.New(os.Stdout, "CHYLE - ", log.Ldate|log.Ltime)
}

// EnableDebugging activates step logging
var EnableDebugging = false

// BuildChangelog creates a changelog from defined configuration
func BuildChangelog(envConfig *envh.EnvTree) error {
	conf, err := config.Create(envConfig)

	if err != nil {
		return err
	}

	if EnableDebugging {
		config.Debug(conf, logger)
	}

	commits, err := git.FetchCommits(conf.GIT.REPOSITORY.PATH, conf.GIT.REFERENCE.FROM, conf.GIT.REFERENCE.TO)

	if err != nil {
		return err
	}

	return proceed(newProcess(conf), commits)
}
