package git

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Sirupsen/logrus"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

var repo *git.Repository
var gitRepositoryPath = "testing-repository"

func setup() {
	path, err := os.Getwd()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	repo, err = git.PlainOpen(path + "/" + gitRepositoryPath)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getCommitFromRef(ref string) *object.Commit {
	cmd := exec.Command("git", "rev-parse", ref)
	cmd.Dir = gitRepositoryPath

	ID, err := cmd.Output()
	ID = ID[:len(ID)-1]

	if err != nil {
		logrus.WithField("ID", string(ID)).Fatal(err)
	}

	c, err := repo.CommitObject(plumbing.NewHash(string(ID)))

	if err != nil {
		logrus.WithField("ID", ID).Fatal(err)
	}

	return c
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

func TestResolveRef(t *testing.T) {
	type g struct {
		ref string
		f   func(*object.Commit, error)
	}

	tests := []g{
		{
			"HEAD",
			func(o *object.Commit, err error) {
				assert.NoError(t, err)
				assert.True(t, o.ID().String() == getCommitFromRef("HEAD").ID().String(), "Must resolve HEAD reference")
			},
		},
		{
			"test1",
			func(o *object.Commit, err error) {
				assert.NoError(t, err)
				assert.True(t, o.ID().String() == getCommitFromRef("test1").ID().String(), "Must resolve branch reference")
			},
		},
		{
			getCommitFromRef("test1").ID().String(),
			func(o *object.Commit, err error) {
				assert.NoError(t, err)
				assert.True(t, o.ID().String() == getCommitFromRef("test1").ID().String(), "Must resolve commit id")
			},
		},
		{
			"whatever",
			func(o *object.Commit, err error) {
				assert.Error(t, err)
				assert.EqualError(t, err, `reference "whatever" can't be found in git repository`)
			},
		},
	}

	for _, test := range tests {
		test.f(resolveRef(test.ref, repo))
	}
}

func TestResolveRefWithErrors(t *testing.T) {
	type g struct {
		ref  string
		repo *git.Repository
		f    func(*object.Commit, error)
	}

	tests := []g{
		{
			"whatever",
			repo,
			func(o *object.Commit, err error) {
				assert.Error(t, err)
				assert.EqualError(t, err, `reference "whatever" can't be found in git repository`)
			},
		},
	}

	for _, test := range tests {
		test.f(resolveRef(test.ref, test.repo))
	}
}

func TestFetchCommits(t *testing.T) {
	type g struct {
		path    string
		toRef   string
		fromRef string
		f       func(*[]object.Commit, error)
	}

	tests := []g{
		{
			gitRepositoryPath,
			getCommitFromRef("HEAD").ID().String(),
			getCommitFromRef("test").ID().String(),
			func(cs *[]object.Commit, err error) {
				assert.Error(t, err)
				assert.Regexp(t, `can't produce a diff between .*? and .*?, check your range is correct by running "git log .*?\.\..*?" command`, err.Error())
			},
		},
		{
			gitRepositoryPath,
			getCommitFromRef("HEAD~1").ID().String(),
			getCommitFromRef("HEAD~3").ID().String(),
			func(cs *[]object.Commit, err error) {
				assert.Error(t, err)
				assert.Regexp(t, `can't produce a diff between .*? and .*?, check your range is correct by running "git log .*?\.\..*?" command`, err.Error())
			},
		},
		{
			gitRepositoryPath,
			getCommitFromRef("HEAD~3").ID().String(),
			getCommitFromRef("test~2^2").ID().String(),
			func(cs *[]object.Commit, err error) {
				assert.NoError(t, err)
				assert.Len(t, *cs, 5)

				commitTests := []string{
					"Merge branch 'test2' into test1\n",
					"feat(file6) : new file 6\n\ncreate a new file 6\n",
					"feat(file5) : new file 5\n\ncreate a new file 5\n",
					"feat(file4) : new file 4\n\ncreate a new file 4\n",
					"feat(file3) : new file 3\n\ncreate a new file 3\n",
				}

				for i, c := range *cs {
					assert.Equal(t, commitTests[i], c.Message)
				}
			},
		},
		{
			gitRepositoryPath,
			getCommitFromRef("HEAD~4").ID().String(),
			getCommitFromRef("test~2^2^2").ID().String(),
			func(cs *[]object.Commit, err error) {
				assert.NoError(t, err)
				assert.Len(t, *cs, 5)

				commitTests := []string{
					"feat(file6) : new file 6\n\ncreate a new file 6\n",
					"feat(file5) : new file 5\n\ncreate a new file 5\n",
					"feat(file4) : new file 4\n\ncreate a new file 4\n",
					"feat(file3) : new file 3\n\ncreate a new file 3\n",
					"feat(file2) : new file 2\n\ncreate a new file 2\n",
				}

				for i, c := range *cs {
					assert.Equal(t, commitTests[i], c.Message)
				}
			},
		},
		{
			"whatever",
			getCommitFromRef("HEAD").ID().String(),
			getCommitFromRef("HEAD~1").ID().String(),
			func(cs *[]object.Commit, err error) {
				assert.EqualError(t, err, `check "whatever" is an existing git repository path`)
			},
		},
		{
			gitRepositoryPath,
			"whatever",
			getCommitFromRef("HEAD~1").ID().String(),
			func(cs *[]object.Commit, err error) {
				assert.EqualError(t, err, `reference "whatever" can't be found in git repository`)
			},
		},
		{
			gitRepositoryPath,
			getCommitFromRef("HEAD~1").ID().String(),
			"whatever",
			func(cs *[]object.Commit, err error) {
				assert.EqualError(t, err, `reference "whatever" can't be found in git repository`)
			},
		},
		{
			gitRepositoryPath,
			"HEAD",
			"HEAD",
			func(cs *[]object.Commit, err error) {
				assert.EqualError(t, err, `can't produce a diff between HEAD and HEAD, check your range is correct by running "git log HEAD..HEAD" command`)
			},
		},
	}

	for _, test := range tests {
		test.f(FetchCommits(test.path, test.toRef, test.fromRef))
	}
}

func TestShallowCloneProducesNoErrors(t *testing.T) {
	path := "shallow-repository-test"
	cmd := exec.Command("rm", "-rf", path)
	_, err := cmd.Output()

	assert.NoError(t, err)

	cmd = exec.Command("git", "clone", "--depth", "2", "https://github.com/octocat/Spoon-Knife.git", path)
	_, err = cmd.Output()

	assert.NoError(t, err)

	cmd = exec.Command("git", "rev-parse", "HEAD~1")
	cmd.Dir = path

	fromRef, err := cmd.Output()
	fromRef = fromRef[:len(fromRef)-1]

	assert.NoError(t, err, "Must extract HEAD~1")

	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = path

	toRef, err := cmd.Output()
	toRef = toRef[:len(toRef)-1]

	assert.NoError(t, err)

	commits, err := FetchCommits("shallow-repository-test", string(fromRef), string(toRef))

	assert.NoError(t, err)
	assert.Len(t, *commits, 1, "Must fetch commits in shallow clone")
}
