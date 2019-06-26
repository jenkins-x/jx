package tests

import (
	"bytes"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	expect "github.com/Netflix/go-expect"
	"github.com/acarl005/stripansi"
	"github.com/hinshun/vt10x"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

var (
	defaultConsoleTimeout = 1 * time.Second
)

// IsDebugLog debug log?
func IsDebugLog() bool {
	return strings.ToLower(os.Getenv("JX_TEST_DEBUG")) == "true"
}

// Debugf debug format
func Debugf(message string, args ...interface{}) {
	if IsDebugLog() {
		log.Logger().Infof(message, args...)
	}
}

// Output returns the output to use for tests
func Output() terminal.FileWriter {
	if IsDebugLog() {
		return os.Stdout
	}
	return terminal.Stdio{}.Out
}

// TestShouldDisableMaven should disable maven
func TestShouldDisableMaven() bool {
	cmd := util.Command{
		Name: "mvn",
		Args: []string{"-v"},
	}
	_, err := cmd.RunWithoutRetry()
	return err != nil
}

// CreateAuthConfigService creates and returns a fixture ConfigService
func CreateAuthConfigService() auth.ConfigService {
	userAuth := auth.User{
		Username:    "jx-testing-user",
		ApiToken:    "someapitoken",
		BearerToken: "somebearertoken",
		Password:    "password",
	}
	authServer := auth.Server{
		Users:       []auth.User{userAuth},
		CurrentUser: userAuth.Username,
		URL:         "https://github.com",
		Kind:        gits.KindGitHub,
		Name:        "jx-testing-server",
	}
	authConfig := auth.Config{
		Servers:       []auth.Server{authServer},
		CurrentServer: authServer.URL,
	}
	authConfigSvc, err := auth.NewMemConfigService(authConfig)
	if err != nil {
		return nil
	}
	return authConfigSvc
}

//newTerminal Returns a fake terminal to test input and output.
func newTerminal(c *expect.Console) *terminal.Stdio {
	return &terminal.Stdio{
		In:  c.Tty(),
		Out: c.Tty(),
		Err: c.Tty(),
	}
}

// NewTerminal mock terminal to control stdin and stdout
func NewTerminal(t assert.TestingT, timeout *time.Duration) *ConsoleWrapper {
	buf := new(bytes.Buffer)
	if timeout == nil {
		timeout = &defaultConsoleTimeout
	}
	opts := []expect.ConsoleOpt{
		sendNoError(t),
		expect.WithStdout(buf),
		expect.WithDefaultTimeout(*timeout),
	}

	c, state, err := vt10x.NewVT10XConsole(opts...)
	if err != nil {
		panic(err)
	}
	return &ConsoleWrapper{
		tester:  t,
		console: c,
		state:   state,
		Stdio:   *newTerminal(c),
	}
}

// TestCloser closes io
func TestCloser(t *testing.T, closer io.Closer) {
	if err := closer.Close(); err != nil {
		t.Errorf("Close failed: %s", err)
		debug.PrintStack()
	}
}

func sendNoError(t assert.TestingT) expect.ConsoleOpt {
	return expect.WithSendObserver(
		func(msg string, n int, err error) {
			if err != nil {
				t.Errorf("Failed to send %q: %s\n%s", msg, err, string(debug.Stack()))
			}
			if len(msg) != n {
				t.Errorf("Only sent %d of %d bytes for %q\n%s", n, len(msg), msg, string(debug.Stack()))
			}
		},
	)
}

// SkipForWindows skips tests if they are running on Windows
// This is to be used for valid tests that just don't work on windows for whatever reason
func SkipForWindows(t *testing.T, reason string) {
	if runtime.GOOS == "windows" {
		t.Skipf("Test skipped on windows. Reason: %s", reason)
	}
}

// ExpectString does the same as the go-expect console.ExpectString method, but also reports failures to the testing object in a sensible format
func ExpectString(t *testing.T, console *expect.Console, s string) {
	out, err := console.ExpectString(s)
	assert.NoError(t, err, "Expected string: %q\nActual string: %q", s, stripansi.Strip(out))
}
