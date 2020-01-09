package importcmd

import (
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

var info = util.ColorInfo

// ImportReporter an interface for reporting updates from the process
type ImportReporter interface {
	// UsingGitUserName report progress
	UsingGitUserName(username string)
	// PushedGitRepository report progress
	PushedGitRepository(url string)
	// GitRepositoryCreated report progress
	GitRepositoryCreated()
	// CreatedDevRepoPullRequest report progress
	CreatedDevRepoPullRequest(prURL string, devGitURL string)
	// CreatedProject report progress
	CreatedProject(genDir string)
	// GeneratedQuickStartAt report progress
	GeneratedQuickStartAt(genDir string)

	// Trace report generic trace message
	Trace(message string, options ...interface{})
}

var _ ImportReporter = &LogImportReporter{}

// LogImportReporter default implementation to log to the console
type LogImportReporter struct {
}

// Trace report generic trace message
func (r *LogImportReporter) Trace(message string, args ...interface{}) {
	log.Logger().Infof(message, args...)
}

// CreatedDevRepoPullRequest report progress
func (r *LogImportReporter) CreatedDevRepoPullRequest(prURL string, devGitURL string) {
	log.Logger().Infof("created pull request %s on the development git repository %s", info(prURL), info(devGitURL))
}

// GitRepositoryCreated report progress
func (r *LogImportReporter) GitRepositoryCreated() {
	log.Logger().Infof("\nGit repository created")
}

// UsingGitUserName report progress
func (r *LogImportReporter) UsingGitUserName(username string) {
	log.Logger().Infof("Using Git user name: %s", username)
}

// PushedGitRepository report progress
func (r *LogImportReporter) PushedGitRepository(repoURL string) {
	log.Logger().Infof("Pushed Git repository to %s\n", info(repoURL))
}

// CreatedProject report progress
func (r *LogImportReporter) CreatedProject(genDir string) {
	log.Logger().Infof("Created project at %s\n", util.ColorInfo(genDir))
}

// GeneratedQuickStartAt report progress
func (r *LogImportReporter) GeneratedQuickStartAt(genDir string) {
	log.Logger().Infof("Generated quickstart at %s", genDir)
}
