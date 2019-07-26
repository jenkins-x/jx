package gits_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/pkg/tests"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

const (
	initialReadme       = "Cheesy!"
	commit1Readme       = "Yet more cheese!"
	commit2Contributing = "Even more cheese!"
	commit3License      = "It's cheesy!"
	contributing        = "CONTRIBUTING"
	readme              = "README"
	license             = "LICENSE"
)

func TestFetchAndMergeOneSHA(t *testing.T) {
	// This test only uses local repos, so it's safe to use real git
	env := prepareFetchAndMergeTests(t)
	defer env.Cleanup()
	// Test merging one commit
	err := gits.FetchAndMergeSHAs([]string{env.Sha1}, "master", env.BaseSha, "origin", env.LocalDir, env.Gitter)
	assert.NoError(t, err)
	readmeFile, err := ioutil.ReadFile(env.ReadmePath)
	assert.NoError(t, err)
	assert.Equal(t, commit1Readme, string(readmeFile))
}

func TestFetchAndMergeMultipleSHAs(t *testing.T) {
	// This test only uses local repos, so it's safe to use real git
	env := prepareFetchAndMergeTests(t)
	defer env.Cleanup()

	// Test merging two commit
	err := gits.FetchAndMergeSHAs([]string{env.Sha1, env.Sha2}, "master", env.BaseSha, "origin", env.LocalDir,
		env.Gitter)
	assert.NoError(t, err)
	localContributingPath := filepath.Join(env.LocalDir, contributing)
	readmeFile, err := ioutil.ReadFile(env.ReadmePath)
	assert.NoError(t, err)
	assert.Equal(t, commit1Readme, string(readmeFile))
	contributingFile, err := ioutil.ReadFile(localContributingPath)
	assert.NoError(t, err)
	assert.Equal(t, commit2Contributing, string(contributingFile))
}

func TestFetchAndMergeSHAAgainstNonHEADSHA(t *testing.T) {
	// This test only uses local repos, so it's safe to use real git
	env := prepareFetchAndMergeTests(t)
	defer env.Cleanup()

	// Test merging two commit
	err := gits.FetchAndMergeSHAs([]string{env.Sha3}, "master", env.Sha1, "origin", env.LocalDir,
		env.Gitter)
	assert.NoError(t, err)

	readmeFile, err := ioutil.ReadFile(env.ReadmePath)
	assert.NoError(t, err)
	assert.Equal(t, commit1Readme, string(readmeFile))

	localContributingPath := filepath.Join(env.LocalDir, contributing)
	_, err = os.Stat(localContributingPath)
	assert.True(t, os.IsNotExist(err))

	localLicensePath := filepath.Join(env.LocalDir, license)
	licenseFile, err := ioutil.ReadFile(localLicensePath)
	assert.NoError(t, err)
	assert.Equal(t, commit3License, string(licenseFile))
}

type FetchAndMergeTestEnv struct {
	Gitter     *gits.GitCLI
	BaseSha    string
	LocalDir   string
	Sha1       string
	Sha2       string
	Sha3       string
	ReadmePath string
	Cleanup    func()
}

func prepareFetchAndMergeTests(t *testing.T) FetchAndMergeTestEnv {
	gitter := gits.NewGitCLI()

	// Prepare a git repo to test - this is our "remote"
	remoteDir, err := ioutil.TempDir("", "remote")
	assert.NoError(t, err)
	err = gitter.Init(remoteDir)
	assert.NoError(t, err)

	readmePath := filepath.Join(remoteDir, readme)
	contributingPath := filepath.Join(remoteDir, contributing)
	licensePath := filepath.Join(remoteDir, license)
	err = ioutil.WriteFile(readmePath, []byte(initialReadme), 0600)
	assert.NoError(t, err)
	err = gitter.Add(remoteDir, readme)
	assert.NoError(t, err)
	err = gitter.CommitDir(remoteDir, "Initial Commit")
	assert.NoError(t, err)

	// Prepare another git repo, this is local repo
	localDir, err := ioutil.TempDir("", "local")
	assert.NoError(t, err)
	err = gitter.Init(localDir)
	assert.NoError(t, err)
	// Set up the remote
	err = gitter.AddRemote(localDir, "origin", remoteDir)
	assert.NoError(t, err)
	err = gitter.FetchBranch(localDir, "origin", "master")
	assert.NoError(t, err)
	err = gitter.Merge(localDir, "origin/master")
	assert.NoError(t, err)

	localReadmePath := filepath.Join(localDir, readme)
	readmeFile, err := ioutil.ReadFile(localReadmePath)
	assert.NoError(t, err)
	assert.Equal(t, initialReadme, string(readmeFile))
	baseSha, err := gitter.GetLatestCommitSha(localDir)
	assert.NoError(t, err)

	err = ioutil.WriteFile(readmePath, []byte(commit1Readme), 0600)
	assert.NoError(t, err)
	err = gitter.Add(remoteDir, readme)
	assert.NoError(t, err)
	err = gitter.CommitDir(remoteDir, "More Cheese")
	assert.NoError(t, err)
	sha1, err := gitter.GetLatestCommitSha(remoteDir)
	assert.NoError(t, err)

	err = ioutil.WriteFile(contributingPath, []byte(commit2Contributing), 0600)
	assert.NoError(t, err)
	err = gitter.Add(remoteDir, contributing)
	assert.NoError(t, err)
	err = gitter.CommitDir(remoteDir, "Even More Cheese")
	assert.NoError(t, err)
	sha2, err := gitter.GetLatestCommitSha(remoteDir)
	assert.NoError(t, err)

	// Put some commits on a branch
	branchNameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	branchName := branchNameUUID.String()
	err = gitter.CreateBranchFrom(remoteDir, branchName, baseSha)
	assert.NoError(t, err)
	err = gitter.Checkout(remoteDir, branchName)
	assert.NoError(t, err)

	err = ioutil.WriteFile(licensePath, []byte(commit3License), 0600)
	assert.NoError(t, err)
	err = gitter.Add(remoteDir, license)
	assert.NoError(t, err)
	err = gitter.CommitDir(remoteDir, "Even More Cheese")
	assert.NoError(t, err)
	sha3, err := gitter.GetLatestCommitSha(remoteDir)
	assert.NoError(t, err)

	return FetchAndMergeTestEnv{
		Gitter:     gitter,
		BaseSha:    baseSha,
		LocalDir:   localDir,
		Sha1:       sha1,
		Sha2:       sha2,
		Sha3:       sha3,
		ReadmePath: localReadmePath,
		Cleanup: func() {
			err := os.RemoveAll(localDir)
			assert.NoError(t, err)
			err = os.RemoveAll(remoteDir)
			assert.NoError(t, err)
		},
	}
}

func TestIsUnadvertisedObjectError(t *testing.T) {
	// Text copied from an error log
	err := errors.New("failed to clone three times it's likely things wont recover so lets kill the process after 3 attempts, last error: failed to fetch [pull/4042/head:PR-4042 3e1a943c00186c8aa364498201974c9ab734b353] from https://github.com/jenkins-x/jx.git in directory /tmp/git599291101: git output: error: Server does not allow request for unadvertised object 3e1a943c00186c8aa364498201974c9ab734b353: failed to run 'git fetch origin --depth=1 pull/4042/head:PR-4042 3e1a943c00186c8aa364498201974c9ab734b353' command in directory '/tmp/git599291101', output: 'error: Server does not allow request for unadvertised object 3e1a943c00186c8aa364498201974c9ab734b353'")
	assert.True(t, gits.IsUnadvertisedObjectError(err))
	err1 := errors.New("ipsum lorem")
	assert.False(t, gits.IsUnadvertisedObjectError(err1))
}

func TestForkAndPullRepo(t *testing.T) {
	type args struct {
		gitURL     string
		dir        string
		baseRef    string
		branchName string
		duplicate  bool
		provider   *gits.FakeProvider
		gitter     gits.Gitter
		initFn     func(args *args) error // initFn allows us to run some code at the start of the test
		cleanFn    func(args *args)
	}
	type test struct {
		name         string
		args         args
		dir          string
		baseRef      string
		upstreamInfo *gits.GitRepository
		forkInfo     *gits.GitRepository
		wantErr      bool
		postFn       func(args *args, test *test) error
	}
	tests := []test{
		{
			name: "noForkAndNewDir",
			args: args{
				gitter: gits.NewGitCLI(),
				initFn: func(args *args) error {
					acmeRepo, err := gits.NewFakeRepository("acme", "roadrunner", func(dir string) error {
						err := ioutil.WriteFile(filepath.Join(dir, "README"), []byte("Hello there!"), 0755)
						if err != nil {
							return errors.WithStack(err)
						}
						return nil
					}, args.gitter)
					assert.NoError(t, err)
					args.provider = gits.NewFakeProvider(acmeRepo)
					uuid, err := uuid.NewV4()
					assert.NoError(t, err)
					args.dir = filepath.Join(os.TempDir(), fmt.Sprintf("git-dir-%s", uuid.String()))
					return nil
				},
				cleanFn: func(args *args) {
					for _, o := range args.provider.Repositories {
						for _, r := range o {
							if r.BaseDir != "" {
								err := os.RemoveAll(r.BaseDir)
								assert.NoError(t, err)
							}
						}
					}
					err := os.RemoveAll(args.dir)
					assert.NoError(t, err)
				},
				gitURL:     fmt.Sprintf("https://fake.git/acme/roadrunner.git"),
				dir:        "",  // set by initFn
				provider:   nil, // set by initFn
				branchName: "master",
				baseRef:    "master",
			},
			baseRef: "master",
			upstreamInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/acme/roadrunner.git",
				HTMLURL:      "https://fake.git/acme/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "acme",
				Fork:         false,
			},
			postFn: func(args *args, test *test) error {
				// Need to dynamically set the directories as we make them up in the init
				test.dir = args.dir
				test.upstreamInfo.CloneURL = fmt.Sprintf("file://%s/acme", args.provider.Repositories["acme"][0].BaseDir)
				return nil
			},
		}, {
			name: "newForkAndNewDir",
			args: args{
				gitter: gits.NewGitCLI(),
				initFn: func(args *args) error {
					acmeRepo, err := gits.NewFakeRepository("acme", "roadrunner", func(dir string) error {
						err := ioutil.WriteFile(filepath.Join(dir, "README"), []byte("Hello there!"), 0755)
						if err != nil {
							return errors.WithStack(err)
						}
						return nil
					}, args.gitter)
					args.provider = gits.NewFakeProvider(acmeRepo)
					// Set the provider username to wile in order to create a fork
					args.provider.User.Username = "wile"
					uuid, err := uuid.NewV4()
					assert.NoError(t, err)
					args.dir = filepath.Join(os.TempDir(), fmt.Sprintf("git-dir-%s", uuid.String()))
					return nil
				},
				cleanFn: func(args *args) {
					for _, o := range args.provider.Repositories {
						for _, r := range o {
							if r.BaseDir != "" {
								err := os.RemoveAll(r.BaseDir)
								assert.NoError(t, err)
							}
						}
					}
					err := os.RemoveAll(args.dir)
					assert.NoError(t, err)
				},
				gitURL:     fmt.Sprintf("https://fake.git/acme/roadrunner.git"),
				dir:        "",  // set by initFn
				provider:   nil, // set by initFn
				branchName: "master",
				baseRef:    "master",
			},
			baseRef: "master",
			upstreamInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/acme/roadrunner.git",
				HTMLURL:      "https://fake.git/acme/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "acme",
				Fork:         false,
			},
			forkInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/wile/roadrunner.git",
				HTMLURL:      "https://fake.git/wile/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "wile",
				Project:      "wile",
				Fork:         true,
			},
			postFn: func(args *args, test *test) error {
				test.dir = args.dir //make sure we end up with the same dir we start with
				test.upstreamInfo.CloneURL = fmt.Sprintf("file://%s/acme", args.provider.Repositories["acme"][0].BaseDir)
				test.forkInfo.CloneURL = fmt.Sprintf("file://%s/wile", args.provider.Repositories["acme"][0].BaseDir)
				remotes, err := args.gitter.Remotes(args.dir)
				assert.NoError(t, err)
				assert.Len(t, remotes, 2)
				assert.Contains(t, remotes, "origin")
				assert.Contains(t, remotes, "upstream")
				return nil
			},
		},
		{
			name: "noForkAndExistingDir",
			args: args{
				gitter: gits.NewGitCLI(),
				initFn: func(args *args) error {
					acmeRepo, err := gits.NewFakeRepository("acme", "roadrunner", func(dir string) error {
						err := ioutil.WriteFile(filepath.Join(dir, "README"), []byte("Hello there!"), 0755)
						if err != nil {
							return errors.WithStack(err)
						}
						return nil
					}, args.gitter)
					assert.NoError(t, err)
					args.provider = gits.NewFakeProvider(acmeRepo)

					args.dir, err = ioutil.TempDir("", "")
					assert.NoError(t, err)
					return nil
				},
				cleanFn: func(args *args) {
					for _, o := range args.provider.Repositories {
						for _, r := range o {
							if r.BaseDir != "" {
								err := os.RemoveAll(r.BaseDir)
								assert.NoError(t, err)
							}
						}
					}
					err := os.RemoveAll(args.dir)
					assert.NoError(t, err)
				},
				gitURL:     fmt.Sprintf("https://fake.git/acme/roadrunner.git"),
				dir:        "",  // set by initFn
				provider:   nil, // set by initFn
				branchName: "master",
				baseRef:    "master",
			},
			baseRef: "master",
			upstreamInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/acme/roadrunner.git",
				HTMLURL:      "https://fake.git/acme/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "acme",
				Fork:         false,
			},
			postFn: func(args *args, test *test) error {
				// Need to dynamically set the directories as we make them up in the init
				test.dir = args.dir
				test.upstreamInfo.CloneURL = fmt.Sprintf("file://%s/acme", args.provider.Repositories["acme"][0].BaseDir)
				return nil
			},
		},
		{
			name: "newForkAndExistingDir",
			args: args{
				gitter: gits.NewGitCLI(),
				initFn: func(args *args) error {
					acmeRepo, err := gits.NewFakeRepository("acme", "roadrunner", func(dir string) error {
						err := ioutil.WriteFile(filepath.Join(dir, "README"), []byte("Hello there!"), 0755)
						if err != nil {
							return errors.WithStack(err)
						}
						return nil
					}, args.gitter)
					args.provider = gits.NewFakeProvider(acmeRepo)
					// Set the provider username to wile in order to create a fork
					args.provider.User.Username = "wile"

					args.dir, err = ioutil.TempDir("", "")
					assert.NoError(t, err)
					return nil
				},
				cleanFn: func(args *args) {
					for _, o := range args.provider.Repositories {
						for _, r := range o {
							if r.BaseDir != "" {
								err := os.RemoveAll(r.BaseDir)
								assert.NoError(t, err)
							}
						}
					}
					err := os.RemoveAll(args.dir)
					assert.NoError(t, err)
				},
				gitURL:     fmt.Sprintf("https://fake.git/acme/roadrunner.git"),
				dir:        "",  // set by initFn
				provider:   nil, // set by initFn
				branchName: "master",
				baseRef:    "master",
			},
			baseRef: "master",
			upstreamInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/acme/roadrunner.git",
				HTMLURL:      "https://fake.git/acme/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "acme",
				Fork:         false,
			},
			forkInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/wile/roadrunner.git",
				HTMLURL:      "https://fake.git/wile/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "wile",
				Project:      "wile",
				Fork:         true,
			},
			postFn: func(args *args, test *test) error {
				test.dir = args.dir //make sure we end up with the same dir we start with
				test.upstreamInfo.CloneURL = fmt.Sprintf("file://%s/acme", args.provider.Repositories["acme"][0].BaseDir)
				test.forkInfo.CloneURL = fmt.Sprintf("file://%s/wile", args.provider.Repositories["acme"][0].BaseDir)
				return nil
			},
		},
		{
			name: "existingForkAndNewDir",
			args: args{
				gitter: gits.NewGitCLI(),
				initFn: func(args *args) error {
					acmeRepo, err := gits.NewFakeRepository("acme", "roadrunner", func(dir string) error {
						err := ioutil.WriteFile(filepath.Join(dir, "README"), []byte("Hello there!"), 0655)
						if err != nil {
							return errors.WithStack(err)
						}
						return nil
					}, args.gitter)
					args.provider = gits.NewFakeProvider(acmeRepo)
					fork, err := args.provider.ForkRepository("acme", "roadrunner", "wile")
					assert.NoError(t, err)

					// Add a commit to the fork that isn't on the upstream to validate later. Let's use a temp clone and push it.
					dir, err := ioutil.TempDir("", "")
					assert.NoError(t, err)
					err = args.gitter.Clone(fork.CloneURL, dir)
					assert.NoError(t, err)
					err = ioutil.WriteFile(filepath.Join(dir, "CONTRIBUTING"), []byte("Welcome!"), 0655)
					assert.NoError(t, err)
					err = args.gitter.Add(dir, "CONTRIBUTING")
					assert.NoError(t, err)
					err = args.gitter.CommitDir(dir, "Second commit")
					assert.NoError(t, err)
					err = args.gitter.Push(dir)
					assert.NoError(t, err)

					// Set the provider username to wile in order to use the fork
					args.provider.User.Username = "wile"
					uuid, err := uuid.NewV4()
					assert.NoError(t, err)
					args.dir = filepath.Join(os.TempDir(), fmt.Sprintf("git-dir-%s", uuid.String()))
					return nil
				},
				cleanFn: func(args *args) {
					for _, o := range args.provider.Repositories {
						for _, r := range o {
							if r.BaseDir != "" {
								err := os.RemoveAll(r.BaseDir)
								assert.NoError(t, err)
							}
						}
					}
					err := os.RemoveAll(args.dir)
					assert.NoError(t, err)
				},
				gitURL:     fmt.Sprintf("https://fake.git/acme/roadrunner.git"),
				dir:        "",  // set by initFn
				provider:   nil, // set by initFn
				branchName: "master",
				baseRef:    "master",
			},
			baseRef: "master",
			upstreamInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/acme/roadrunner.git",
				HTMLURL:      "https://fake.git/acme/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "acme",
				Fork:         false,
			},
			forkInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/wile/roadrunner.git",
				HTMLURL:      "https://fake.git/wile/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "wile",
				Project:      "wile",
				Fork:         true,
			},
			postFn: func(args *args, test *test) error {
				test.dir = args.dir //make sure we end up with the same dir we start with
				test.upstreamInfo.CloneURL = fmt.Sprintf("file://%s/acme", args.provider.Repositories["acme"][0].BaseDir)
				test.forkInfo.CloneURL = fmt.Sprintf("file://%s/wile", args.provider.Repositories["acme"][0].BaseDir)
				_, gitConf, err := args.gitter.FindGitConfigDir(args.dir)
				assert.NoError(t, err)
				remotes, err := args.gitter.Remotes(args.dir)
				assert.NoError(t, err)
				assert.Len(t, remotes, 2)
				assert.Contains(t, remotes, "origin")
				assert.Contains(t, remotes, "upstream")
				originURL, err := args.gitter.DiscoverRemoteGitURL(gitConf)
				assert.NoError(t, err)
				upstreamURL, err := args.gitter.DiscoverUpstreamGitURL(gitConf)
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("file://%s/wile", args.provider.Repositories["acme"][0].BaseDir), originURL)
				assert.Equal(t, fmt.Sprintf("file://%s/acme", args.provider.Repositories["acme"][0].BaseDir), upstreamURL)
				assert.FileExists(t, filepath.Join(args.dir, "CONTRIBUTING"))
				return nil
			},
		},
		{
			name: "existingForkAndExistingDir",
			args: args{
				gitter: gits.NewGitCLI(),
				initFn: func(args *args) error {
					acmeRepo, err := gits.NewFakeRepository("acme", "roadrunner", func(dir string) error {
						err := ioutil.WriteFile(filepath.Join(dir, "README"), []byte("Hello there!"), 0655)
						if err != nil {
							return errors.WithStack(err)
						}
						return nil
					}, args.gitter)
					args.provider = gits.NewFakeProvider(acmeRepo)
					fork, err := args.provider.ForkRepository("acme", "roadrunner", "wile")
					assert.NoError(t, err)

					// Add a commit to the fork that isn't on the upstream to validate later. Let's use a temp clone and push it.
					dir, err := ioutil.TempDir("", "")
					assert.NoError(t, err)
					err = args.gitter.Clone(fork.CloneURL, dir)
					assert.NoError(t, err)
					err = ioutil.WriteFile(filepath.Join(dir, "CONTRIBUTING"), []byte("Welcome!"), 0655)
					assert.NoError(t, err)
					err = args.gitter.Add(dir, "CONTRIBUTING")
					assert.NoError(t, err)
					err = args.gitter.CommitDir(dir, "Second commit")
					assert.NoError(t, err)
					err = args.gitter.Push(dir)
					assert.NoError(t, err)

					// Set the provider username to wile in order to use the fork
					args.provider.User.Username = "wile"
					args.dir, err = ioutil.TempDir("", "")
					assert.NoError(t, err)
					return nil
				},
				cleanFn: func(args *args) {
					for _, o := range args.provider.Repositories {
						for _, r := range o {
							if r.BaseDir != "" {
								err := os.RemoveAll(r.BaseDir)
								assert.NoError(t, err)
							}
						}
					}
					err := os.RemoveAll(args.dir)
					assert.NoError(t, err)
				},
				gitURL:     fmt.Sprintf("https://fake.git/acme/roadrunner.git"),
				dir:        "",  // set by initFn
				provider:   nil, // set by initFn
				branchName: "master",
				baseRef:    "master",
			},
			baseRef: "master",
			upstreamInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/acme/roadrunner.git",
				HTMLURL:      "https://fake.git/acme/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "acme",
				Fork:         false,
			},
			forkInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/wile/roadrunner.git",
				HTMLURL:      "https://fake.git/wile/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "wile",
				Project:      "wile",
				Fork:         true,
			},
			postFn: func(args *args, test *test) error {
				test.dir = args.dir //make sure we end up with the same dir we start with
				test.upstreamInfo.CloneURL = fmt.Sprintf("file://%s/acme", args.provider.Repositories["acme"][0].BaseDir)
				test.forkInfo.CloneURL = fmt.Sprintf("file://%s/wile", args.provider.Repositories["acme"][0].BaseDir)
				_, gitConf, err := args.gitter.FindGitConfigDir(args.dir)
				assert.NoError(t, err)
				remotes, err := args.gitter.Remotes(args.dir)
				assert.NoError(t, err)
				assert.Len(t, remotes, 2)
				assert.Contains(t, remotes, "origin")
				assert.Contains(t, remotes, "upstream")
				originURL, err := args.gitter.DiscoverRemoteGitURL(gitConf)
				assert.NoError(t, err)
				upstreamURL, err := args.gitter.DiscoverUpstreamGitURL(gitConf)
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("file://%s/wile", args.provider.Repositories["acme"][0].BaseDir), originURL)
				assert.Equal(t, fmt.Sprintf("file://%s/acme", args.provider.Repositories["acme"][0].BaseDir), upstreamURL)
				assert.FileExists(t, filepath.Join(args.dir, "CONTRIBUTING"))
				return nil
			},
		}, {
			name: "existingForkAndExistingCheckout",
			args: args{
				gitter: gits.NewGitCLI(),
				initFn: func(args *args) error {
					acmeRepo, err := gits.NewFakeRepository("acme", "roadrunner", func(dir string) error {
						err := ioutil.WriteFile(filepath.Join(dir, "README"), []byte("Hello there!"), 0655)
						if err != nil {
							return errors.WithStack(err)
						}
						return nil
					}, args.gitter)
					args.provider = gits.NewFakeProvider(acmeRepo)
					fork, err := args.provider.ForkRepository("acme", "roadrunner", "wile")
					assert.NoError(t, err)

					// Add a commit to the fork that isn't on the upstream to validate later. Let's use a temp clone and push it.
					dir, err := ioutil.TempDir("", "")
					assert.NoError(t, err)
					err = args.gitter.Clone(fork.CloneURL, dir)
					assert.NoError(t, err)
					err = ioutil.WriteFile(filepath.Join(dir, "CONTRIBUTING"), []byte("Welcome!"), 0655)
					assert.NoError(t, err)
					err = args.gitter.Add(dir, "CONTRIBUTING")
					assert.NoError(t, err)
					err = args.gitter.CommitDir(dir, "Second commit")
					assert.NoError(t, err)
					err = args.gitter.Push(dir)
					assert.NoError(t, err)

					// Set the provider username to wile in order to use the fork
					args.provider.User.Username = "wile"

					// Let's checkout our fork
					args.dir, err = ioutil.TempDir("", "")
					assert.NoError(t, err)
					err = args.gitter.Clone(fork.CloneURL, args.dir)
					assert.NoError(t, err)

					return nil
				},
				cleanFn: func(args *args) {
					for _, o := range args.provider.Repositories {
						for _, r := range o {
							if r.BaseDir != "" {
								err := os.RemoveAll(r.BaseDir)
								assert.NoError(t, err)
							}
						}
					}
					err := os.RemoveAll(args.dir)
					assert.NoError(t, err)
				},
				gitURL:     fmt.Sprintf("https://fake.git/acme/roadrunner.git"),
				dir:        "",  // set by initFn
				provider:   nil, // set by initFn
				branchName: "master",
				baseRef:    "master",
			},
			baseRef: "master",
			upstreamInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/acme/roadrunner.git",
				HTMLURL:      "https://fake.git/acme/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "acme",
				Fork:         false,
			},
			forkInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/wile/roadrunner.git",
				HTMLURL:      "https://fake.git/wile/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "wile",
				Project:      "wile",
				Fork:         true,
			},
			postFn: func(args *args, test *test) error {
				test.dir = args.dir //make sure we end up with the same dir we start with
				test.upstreamInfo.CloneURL = fmt.Sprintf("file://%s/acme", args.provider.Repositories["acme"][0].BaseDir)
				test.forkInfo.CloneURL = fmt.Sprintf("file://%s/wile", args.provider.Repositories["acme"][0].BaseDir)
				_, gitConf, err := args.gitter.FindGitConfigDir(args.dir)
				assert.NoError(t, err)
				remotes, err := args.gitter.Remotes(args.dir)
				assert.NoError(t, err)
				assert.Len(t, remotes, 2)
				assert.Contains(t, remotes, "origin")
				assert.Contains(t, remotes, "upstream")
				originURL, err := args.gitter.DiscoverRemoteGitURL(gitConf)
				assert.NoError(t, err)
				upstreamURL, err := args.gitter.DiscoverUpstreamGitURL(gitConf)
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("file://%s/wile", args.provider.Repositories["acme"][0].BaseDir), originURL)
				assert.Equal(t, fmt.Sprintf("file://%s/acme", args.provider.Repositories["acme"][0].BaseDir), upstreamURL)
				assert.FileExists(t, filepath.Join(args.dir, "CONTRIBUTING"))
				return nil
			},
		}, {
			name: "existingForkAndExistingCheckoutWithDifferentBaseRef",
			args: args{
				gitter: gits.NewGitCLI(),
				initFn: func(args *args) error {
					acmeRepo, err := gits.NewFakeRepository("acme", "roadrunner", func(dir string) error {
						err := ioutil.WriteFile(filepath.Join(dir, "README"), []byte("Hello there!"), 0655)
						if err != nil {
							return errors.WithStack(err)
						}
						return nil
					}, args.gitter)
					args.provider = gits.NewFakeProvider(acmeRepo)
					fork, err := args.provider.ForkRepository("acme", "roadrunner", "wile")
					assert.NoError(t, err)

					// Add a commit to the upstream on a different branch to validate later. Let's use a temp clone and push it.
					dir, err := ioutil.TempDir("", "")
					assert.NoError(t, err)
					err = args.gitter.Clone(acmeRepo.GitRepo.CloneURL, dir)
					assert.NoError(t, err)
					err = ioutil.WriteFile(filepath.Join(dir, "CONTRIBUTING"), []byte("Welcome!"), 0655)
					assert.NoError(t, err)
					err = args.gitter.Add(dir, "CONTRIBUTING")
					assert.NoError(t, err)
					err = args.gitter.CommitDir(dir, "Second commit")
					assert.NoError(t, err)
					err = args.gitter.ForcePushBranch(dir, "HEAD", "other")
					assert.NoError(t, err)

					// Set the provider username to wile in order to use the fork
					args.provider.User.Username = "wile"

					// Let's checkout our fork
					args.dir, err = ioutil.TempDir("", "")
					assert.NoError(t, err)
					err = args.gitter.Clone(fork.CloneURL, args.dir)
					assert.NoError(t, err)

					return nil
				},
				cleanFn: func(args *args) {
					for _, o := range args.provider.Repositories {
						for _, r := range o {
							if r.BaseDir != "" {
								err := os.RemoveAll(r.BaseDir)
								assert.NoError(t, err)
							}
						}
					}
					err := os.RemoveAll(args.dir)
					assert.NoError(t, err)
				},
				gitURL:     fmt.Sprintf("https://fake.git/acme/roadrunner.git"),
				dir:        "",  // set by initFn
				provider:   nil, // set by initFn
				branchName: "master",
				baseRef:    "other",
			},
			baseRef: "other",
			upstreamInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/acme/roadrunner.git",
				HTMLURL:      "https://fake.git/acme/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "acme",
				Fork:         false,
			},
			forkInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/wile/roadrunner.git",
				HTMLURL:      "https://fake.git/wile/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "wile",
				Project:      "wile",
				Fork:         true,
			},
			postFn: func(args *args, test *test) error {
				test.dir = args.dir //make sure we end up with the same dir we start with
				test.upstreamInfo.CloneURL = fmt.Sprintf("file://%s/acme", args.provider.Repositories["acme"][0].BaseDir)
				test.forkInfo.CloneURL = fmt.Sprintf("file://%s/wile", args.provider.Repositories["acme"][0].BaseDir)
				_, gitConf, err := args.gitter.FindGitConfigDir(args.dir)
				assert.NoError(t, err)
				remotes, err := args.gitter.Remotes(args.dir)
				assert.NoError(t, err)
				assert.Len(t, remotes, 2)
				assert.Contains(t, remotes, "origin")
				assert.Contains(t, remotes, "upstream")
				originURL, err := args.gitter.DiscoverRemoteGitURL(gitConf)
				assert.NoError(t, err)
				upstreamURL, err := args.gitter.DiscoverUpstreamGitURL(gitConf)
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("file://%s/wile", args.provider.Repositories["acme"][0].BaseDir), originURL)
				assert.Equal(t, fmt.Sprintf("file://%s/acme", args.provider.Repositories["acme"][0].BaseDir), upstreamURL)
				assert.FileExists(t, filepath.Join(args.dir, "CONTRIBUTING"))
				return nil
			},
		}, {
			name: "existingForkAndExistingCheckoutWithLocalModifications",
			args: args{
				gitter: gits.NewGitCLI(),
				initFn: func(args *args) error {
					acmeRepo, err := gits.NewFakeRepository("acme", "roadrunner", func(dir string) error {
						err := ioutil.WriteFile(filepath.Join(dir, "README"), []byte("Hello there!"), 0655)
						if err != nil {
							return errors.WithStack(err)
						}
						return nil
					}, args.gitter)
					args.provider = gits.NewFakeProvider(acmeRepo)
					fork, err := args.provider.ForkRepository("acme", "roadrunner", "wile")
					assert.NoError(t, err)

					// Add a commit to the fork that isn't on the upstream to validate later. Let's use a temp clone and push it.
					dir, err := ioutil.TempDir("", "")
					assert.NoError(t, err)
					err = args.gitter.Clone(fork.CloneURL, dir)
					assert.NoError(t, err)
					err = ioutil.WriteFile(filepath.Join(dir, "CONTRIBUTING"), []byte("Welcome!"), 0655)
					assert.NoError(t, err)
					err = args.gitter.Add(dir, "CONTRIBUTING")
					assert.NoError(t, err)
					err = args.gitter.CommitDir(dir, "Second commit")
					assert.NoError(t, err)
					err = args.gitter.Push(dir)
					assert.NoError(t, err)

					// Set the provider username to wile in order to use the fork
					args.provider.User.Username = "wile"

					// Let's checkout our fork
					args.dir, err = ioutil.TempDir("", "")
					assert.NoError(t, err)
					err = args.gitter.Clone(fork.CloneURL, args.dir)
					assert.NoError(t, err)
					// Let's add some local modifications that don't conflict
					err = ioutil.WriteFile(filepath.Join(args.dir, "LICENSE"), []byte("TODO ;-)"), 0655)
					assert.NoError(t, err)

					return nil
				},
				cleanFn: func(args *args) {
					for _, o := range args.provider.Repositories {
						for _, r := range o {
							if r.BaseDir != "" {
								err := os.RemoveAll(r.BaseDir)
								assert.NoError(t, err)
							}
						}
					}
					err := os.RemoveAll(args.dir)
					assert.NoError(t, err)
				},
				gitURL:     fmt.Sprintf("https://fake.git/acme/roadrunner.git"),
				dir:        "",  // set by initFn
				provider:   nil, // set by initFn
				branchName: "master",
				baseRef:    "master",
			},
			baseRef: "master",
			upstreamInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/acme/roadrunner.git",
				HTMLURL:      "https://fake.git/acme/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "acme",
				Fork:         false,
			},
			forkInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/wile/roadrunner.git",
				HTMLURL:      "https://fake.git/wile/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "wile",
				Project:      "wile",
				Fork:         true,
			},
			postFn: func(args *args, test *test) error {
				test.dir = args.dir //make sure we end up with the same dir we start with
				test.upstreamInfo.CloneURL = fmt.Sprintf("file://%s/acme", args.provider.Repositories["acme"][0].BaseDir)
				test.forkInfo.CloneURL = fmt.Sprintf("file://%s/wile", args.provider.Repositories["acme"][0].BaseDir)
				_, gitConf, err := args.gitter.FindGitConfigDir(args.dir)
				assert.NoError(t, err)
				remotes, err := args.gitter.Remotes(args.dir)
				assert.NoError(t, err)
				assert.Len(t, remotes, 2)
				assert.Contains(t, remotes, "origin")
				assert.Contains(t, remotes, "upstream")
				originURL, err := args.gitter.DiscoverRemoteGitURL(gitConf)
				assert.NoError(t, err)
				upstreamURL, err := args.gitter.DiscoverUpstreamGitURL(gitConf)
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("file://%s/wile", args.provider.Repositories["acme"][0].BaseDir), originURL)
				assert.Equal(t, fmt.Sprintf("file://%s/acme", args.provider.Repositories["acme"][0].BaseDir), upstreamURL)
				assert.FileExists(t, filepath.Join(args.dir, "CONTRIBUTING"))
				assert.FileExists(t, filepath.Join(args.dir, "LICENSE"))
				return nil
			},
		}, {
			name: "existingForkAndExistingCheckoutWithNonConflictingLocalModifications",
			args: args{
				gitter: gits.NewGitCLI(),
				initFn: func(args *args) error {
					acmeRepo, err := gits.NewFakeRepository("acme", "roadrunner", func(dir string) error {
						err := ioutil.WriteFile(filepath.Join(dir, "README"), []byte("Hello there!"), 0655)
						if err != nil {
							return errors.WithStack(err)
						}
						return nil
					}, args.gitter)
					args.provider = gits.NewFakeProvider(acmeRepo)
					fork, err := args.provider.ForkRepository("acme", "roadrunner", "wile")
					assert.NoError(t, err)

					// Add a commit to the fork that isn't on the upstream to validate later. Let's use a temp clone and push it.
					dir, err := ioutil.TempDir("", "")
					assert.NoError(t, err)
					err = args.gitter.Clone(fork.CloneURL, dir)
					assert.NoError(t, err)
					err = ioutil.WriteFile(filepath.Join(dir, "CONTRIBUTING"), []byte("Welcome!"), 0655)
					assert.NoError(t, err)
					err = args.gitter.Add(dir, "CONTRIBUTING")
					assert.NoError(t, err)
					err = args.gitter.CommitDir(dir, "Second commit")
					assert.NoError(t, err)
					err = args.gitter.Push(dir)
					assert.NoError(t, err)

					// Set the provider username to wile in order to use the fork
					args.provider.User.Username = "wile"

					// Let's checkout our fork
					args.dir, err = ioutil.TempDir("", "")
					assert.NoError(t, err)
					err = args.gitter.Clone(fork.CloneURL, args.dir)
					assert.NoError(t, err)
					// Let's add some local modifications that don't conflict
					err = ioutil.WriteFile(filepath.Join(args.dir, "CONTRIBUTING"), []byte("TODO ;-)"), 0655)
					assert.NoError(t, err)

					return nil
				},
				cleanFn: func(args *args) {
					for _, o := range args.provider.Repositories {
						for _, r := range o {
							if r.BaseDir != "" {
								err := os.RemoveAll(r.BaseDir)
								assert.NoError(t, err)
							}
						}
					}
					err := os.RemoveAll(args.dir)
					assert.NoError(t, err)
				},
				gitURL:     fmt.Sprintf("https://fake.git/acme/roadrunner.git"),
				dir:        "",  // set by initFn
				provider:   nil, // set by initFn
				branchName: "master",
				baseRef:    "master",
			},
			baseRef: "master",
			upstreamInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/acme/roadrunner.git",
				HTMLURL:      "https://fake.git/acme/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "acme",
				Fork:         false,
			},
			forkInfo: &gits.GitRepository{
				Name:         "roadrunner",
				URL:          "https://fake.git/wile/roadrunner.git",
				HTMLURL:      "https://fake.git/wile/roadrunner",
				Scheme:       "https",
				Host:         "fake.git",
				Organisation: "wile",
				Project:      "wile",
				Fork:         true,
			},
			postFn: func(args *args, test *test) error {
				test.dir = args.dir //make sure we end up with the same dir we start with
				test.upstreamInfo.CloneURL = fmt.Sprintf("file://%s/acme", args.provider.Repositories["acme"][0].BaseDir)
				test.forkInfo.CloneURL = fmt.Sprintf("file://%s/wile", args.provider.Repositories["acme"][0].BaseDir)
				_, gitConf, err := args.gitter.FindGitConfigDir(args.dir)
				assert.NoError(t, err)
				remotes, err := args.gitter.Remotes(args.dir)
				assert.NoError(t, err)
				assert.Len(t, remotes, 2)
				assert.Contains(t, remotes, "origin")
				assert.Contains(t, remotes, "upstream")
				originURL, err := args.gitter.DiscoverRemoteGitURL(gitConf)
				assert.NoError(t, err)
				upstreamURL, err := args.gitter.DiscoverUpstreamGitURL(gitConf)
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("file://%s/wile", args.provider.Repositories["acme"][0].BaseDir), originURL)
				assert.Equal(t, fmt.Sprintf("file://%s/acme", args.provider.Repositories["acme"][0].BaseDir), upstreamURL)
				assert.FileExists(t, filepath.Join(args.dir, "CONTRIBUTING"))
				tests.AssertFileContains(t, filepath.Join(args.dir, "CONTRIBUTING"), "TODO ;-)")
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.initFn(&tt.args)
			assert.NoError(t, err)
			dir, baseRef, upstreamInfo, forkInfo, err := gits.ForkAndPullRepo(tt.args.gitURL, tt.args.dir, tt.args.baseRef, tt.args.branchName, tt.args.provider, tt.args.gitter, "")
			err2 := tt.postFn(&tt.args, &tt)
			assert.NoError(t, err2)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			//validate the returned data
			assert.Equal(t, tt.dir, dir)
			assert.Equal(t, tt.baseRef, baseRef)
			assert.Equal(t, tt.upstreamInfo, upstreamInfo)
			assert.Equal(t, tt.forkInfo, forkInfo)

			//validate the forked repo has the right files in it
			files, err := filepath.Glob(fmt.Sprintf("%s/README", dir))
			assert.NoError(t, err)
			assert.Len(t, files, 1)

			if len(files) == 1 {
				// validate the content is correct
				data, err := ioutil.ReadFile(files[0])
				assert.NoError(t, err)
				assert.Equal(t, []byte("Hello there!"), data)
			}
			tt.args.cleanFn(&tt.args)
		})
	}
}

func TestDuplicateGitRepoFromCommitsh(t *testing.T) {
	gitter := gits.NewGitCLI()
	originalRepo, err := gits.NewFakeRepository("acme", "roadrunner", func(dir string) error {
		err := ioutil.WriteFile(filepath.Join(dir, "README"), []byte("Hello!"), 0655)
		if err != nil {
			return errors.Wrapf(err, "writing README")
		}
		return nil
	}, gitter)
	assert.NoError(t, err)

	dir, err := ioutil.TempDir("", "")
	assert.NoError(t, err)
	err = gitter.Clone(originalRepo.GitRepo.CloneURL, dir)
	assert.NoError(t, err)

	err = gitter.CreateBranch(dir, "other")
	assert.NoError(t, err)

	err = gitter.Checkout(dir, "other")
	assert.NoError(t, err)

	err = ioutil.WriteFile(filepath.Join(dir, "LICENSE"), []byte("TODO"), 0655)
	assert.NoError(t, err)

	err = gitter.Add(dir, "LICENSE")
	assert.NoError(t, err)

	err = gitter.CommitDir(dir, "add license")
	assert.NoError(t, err)

	err = gitter.Push(dir)
	assert.NoError(t, err)

	err = gitter.CreateBranch(dir, "release")
	assert.NoError(t, err)

	err = gitter.Checkout(dir, "release")
	assert.NoError(t, err)

	err = ioutil.WriteFile(filepath.Join(dir, "CONTRIBUTING"), []byte("Welcome!"), 0655)
	assert.NoError(t, err)

	err = gitter.Add(dir, "CONTRIBUTING")
	assert.NoError(t, err)

	err = gitter.CommitDir(dir, "add contributing")
	assert.NoError(t, err)

	err = gitter.CreateTag(dir, "v1.0.0", "1.0.0")
	assert.NoError(t, err)

	err = gitter.Push(dir)
	assert.NoError(t, err)

	err = gitter.PushTag(dir, "v1.0.0")
	assert.NoError(t, err)
	type args struct {
		toOrg         string
		toName        string
		fromGitURL    string
		fromCommitish string
		toBranch      string
		gitter        gits.Gitter
	}
	tests := []struct {
		provider  *gits.FakeProvider
		name      string
		args      args
		want      *gits.GitRepository
		wantFiles map[string][]byte
		wantErr   bool
	}{
		{
			name: "sameOrg",
			args: args{
				toOrg:         "acme",
				toName:        "wile",
				fromGitURL:    "https://fake.git/acme/roadrunner.git",
				fromCommitish: "master",
				toBranch:      "master",
				gitter:        gitter,
			},
			want: &gits.GitRepository{
				Name:             "wile",
				AllowMergeCommit: false,
				HTMLURL:          "https://fake.git/acme/wile",
				CloneURL:         "",
				SSHURL:           "",
				Language:         "",
				Fork:             false,
				Stars:            0,
				URL:              "https://fake.git/acme/wile.git",
				Scheme:           "https",
				Host:             "fake.git",
				Organisation:     "acme",
				Project:          "",
				Private:          false,
			},
			wantErr: false,
			wantFiles: map[string][]byte{
				"README": []byte("Hello!"),
			},
		},
		{
			name: "differentOrg",
			args: args{
				toOrg:         "coyote",
				toName:        "wile",
				fromGitURL:    "https://fake.git/acme/roadrunner.git",
				fromCommitish: "master",
				toBranch:      "master",
				gitter:        gitter,
			},
			want: &gits.GitRepository{
				Name:             "wile",
				AllowMergeCommit: false,
				HTMLURL:          "https://fake.git/coyote/wile",
				CloneURL:         "",
				SSHURL:           "",
				Language:         "",
				Fork:             false,
				Stars:            0,
				URL:              "https://fake.git/coyote/wile.git",
				Scheme:           "https",
				Host:             "fake.git",
				Organisation:     "coyote",
				Project:          "",
				Private:          false,
			},
			wantErr: false,
			wantFiles: map[string][]byte{
				"README": []byte("Hello!"),
			},
		},
		{
			name: "tag",
			args: args{
				toOrg:         "coyote",
				toName:        "wile",
				fromGitURL:    "https://fake.git/acme/roadrunner.git",
				fromCommitish: "v1.0.0",
				toBranch:      "master",
				gitter:        gitter,
			},
			want: &gits.GitRepository{
				Name:             "wile",
				AllowMergeCommit: false,
				HTMLURL:          "https://fake.git/coyote/wile",
				CloneURL:         "",
				SSHURL:           "",
				Language:         "",
				Fork:             false,
				Stars:            0,
				URL:              "https://fake.git/coyote/wile.git",
				Scheme:           "https",
				Host:             "fake.git",
				Organisation:     "coyote",
				Project:          "",
				Private:          false,
			},
			wantErr: false,
			wantFiles: map[string][]byte{
				"README":       []byte("Hello!"),
				"CONTRIBUTING": []byte("Welcome!"),
			},
		}, {
			name: "branch",
			args: args{
				toOrg:         "coyote",
				toName:        "wile",
				fromGitURL:    "https://fake.git/acme/roadrunner.git",
				fromCommitish: "origin/other",
				toBranch:      "master",
				gitter:        gitter,
			},
			want: &gits.GitRepository{
				Name:             "wile",
				AllowMergeCommit: false,
				HTMLURL:          "https://fake.git/coyote/wile",
				CloneURL:         "",
				SSHURL:           "",
				Language:         "",
				Fork:             false,
				Stars:            0,
				URL:              "https://fake.git/coyote/wile.git",
				Scheme:           "https",
				Host:             "fake.git",
				Organisation:     "coyote",
				Project:          "",
				Private:          false,
			},
			wantErr: false,
			wantFiles: map[string][]byte{
				"README":  []byte("Hello!"),
				"LICENSE": []byte("TODO"),
			},
		}, {
			name: "destinationBranch",
			args: args{
				toOrg:         "coyote",
				toName:        "wile",
				fromGitURL:    "https://fake.git/acme/roadrunner.git",
				fromCommitish: "origin/other",
				toBranch:      "another",
				gitter:        gitter,
			},
			want: &gits.GitRepository{
				Name:             "wile",
				AllowMergeCommit: false,
				HTMLURL:          "https://fake.git/coyote/wile",
				CloneURL:         "",
				SSHURL:           "",
				Language:         "",
				Fork:             false,
				Stars:            0,
				URL:              "https://fake.git/coyote/wile.git",
				Scheme:           "https",
				Host:             "fake.git",
				Organisation:     "coyote",
				Project:          "",
				Private:          false,
			},
			wantErr: false,
			wantFiles: map[string][]byte{
				"README":  []byte("Hello!"),
				"LICENSE": []byte("TODO"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := gits.NewFakeProvider(originalRepo)
			provider.Gitter = gitter
			provider.CreateRepositoryAddFiles = func(dir string) error {
				err := ioutil.WriteFile(filepath.Join(dir, ".gitkeep"), []byte(""), 0655)
				assert.NoError(t, err)
				err = gitter.Add(dir, filepath.Join(dir, ".gitkeep"))
				assert.NoError(t, err)
				return nil
			}
			tt.provider = provider

			got, err := gits.DuplicateGitRepoFromCommitsh(tt.args.toOrg, tt.args.toName, tt.args.fromGitURL, tt.args.fromCommitish, tt.args.toBranch, tt.args.gitter, tt.provider)
			if tt.wantErr {
				assert.Error(t, err)
				return
			} else {
				assert.NoError(t, err)
				if err != nil {
					return
				}
			}
			baseDir := ""
			for _, r := range tt.provider.Repositories[got.Organisation] {
				if r.Name() == got.Name {
					baseDir = r.BaseDir
				}
			}
			tt.want.CloneURL = fmt.Sprintf("file://%s/%s", baseDir, got.Organisation)
			assert.Equal(t, tt.want, got)

			// Make a clone
			dir, err := ioutil.TempDir("", "")
			assert.NoError(t, err)
			err = gitter.Clone(got.CloneURL, dir)
			assert.NoError(t, err)

			err = gitter.FetchBranch(dir, "origin", tt.args.toBranch)
			assert.NoError(t, err)

			err = gitter.CheckoutRemoteBranch(dir, tt.args.toBranch)
			assert.NoError(t, err)

			for relPath, content := range tt.wantFiles {
				path := filepath.Join(dir, relPath)
				assert.FileExists(t, path)
				data, err := ioutil.ReadFile(path)
				assert.NoError(t, err)
				assert.Equal(t, content, data)
			}
		})
	}
}
