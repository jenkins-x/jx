package gits_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx/pkg/log"

	"github.com/jenkins-x/jx/pkg/gits"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGitCLI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Git CLI Test Suite")
}

var _ = Describe("Git CLI", func() {
	var (
		git                *gits.GitCLI
		repoDir            string
		err                error
		origAuthor         string
		origAuthorEmail    string
		origCommitter      string
		origCommitterEmail string
	)

	BeforeSuite(func() {
		// comment out to see logging output
		//log.SetOutput(ioutil.Discard)
		_ = log.SetLevel("info")

		origCommitter = os.Getenv("GIT_COMMITTER_NAME")
		_ = os.Setenv("GIT_COMMITTER_NAME", "test-committer")
		origCommitterEmail = os.Getenv("GIT_COMMITTER_EMAIL")
		_ = os.Setenv("GIT_COMMITTER_EMAIL", "test-committer@acme.com")

		origAuthor = os.Getenv("GIT_AUTHOR_NAME")
		_ = os.Setenv("GIT_AUTHOR_NAME", "test-author")
		origAuthorEmail = os.Getenv("GIT_AUTHOR_EMAIL")
		_ = os.Setenv("GIT_AUTHOR_EMAIL", "test-author@acme.com")

		git = &gits.GitCLI{}
	})

	AfterSuite(func() {
		if origCommitter != "" {
			_ = os.Setenv("GIT_COMMITTER_NAME", origCommitter)
		} else {
			_ = os.Unsetenv("GIT_COMMITTER_NAME")
		}

		if origCommitterEmail != "" {
			_ = os.Setenv("GIT_COMMITTER_EMAIL", origCommitterEmail)
		} else {
			_ = os.Unsetenv("GIT_COMMITTER_EMAIL")
		}

		if origAuthor != "" {
			_ = os.Setenv("GIT_AUTHOR_NAME", origAuthor)
		} else {
			_ = os.Unsetenv("GIT_AUTHOR_NAME")
		}

		if origAuthorEmail != "" {
			_ = os.Setenv("GIT_AUTHOR_EMAIL", origAuthorEmail)
		} else {
			_ = os.Unsetenv("GIT_AUTHOR_EMAIL")
		}
	})

	BeforeEach(func() {
		repoDir, err = ioutil.TempDir("", "jenkins-x-git-test-repo-")
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("creating a test repository in '%s'", repoDir))
		gits.GitCmd(Fail, repoDir, "init")
	})

	AfterEach(func() {
		By("deleting temp repo")
		_ = os.RemoveAll(repoDir)
	})

	Describe("#ConvertToValidBranchName", func() {
		It("converts a string into a valid git branch name", func() {
			testCases := []struct {
				input    string
				expected string
			}{
				{
					"testing-thingy", "testing-thingy",
				},
				{
					"testing-thingy/", "testing-thingy",
				},
				{
					"testing-thingy.lock", "testing-thingy",
				},
				{
					"foo bar", "foo_bar",
				},
				{
					"foo\t ~bar", "foo_bar",
				},
			}
			for _, data := range testCases {
				actual := git.ConvertToValidBranchName(data.input)
				Expect(actual).Should(Equal(data.expected))
			}
		})
	})

	Describe("#GetCommits", func() {
		var (
			commitASha string
			commitBSha string
			commitCSha string
			commitDSha string
			commitESha string
		)

		BeforeEach(func() {
			By("adding commit a on master branch")
			gits.WriteFile(Fail, repoDir, "a.txt", "a")
			gits.Add(Fail, repoDir)
			commitASha = gits.Commit(Fail, repoDir, "commit a")

			By("creating branch 'b' and adding a commit")
			gits.Branch(Fail, repoDir, "b")
			gits.WriteFile(Fail, repoDir, "b.txt", "b")
			gits.Add(Fail, repoDir)
			commitBSha = gits.Commit(Fail, repoDir, "commit b")

			By("creating branch 'c' and adding a commit")
			gits.Checkout(Fail, repoDir, "master")
			gits.Branch(Fail, repoDir, "c")
			gits.WriteFile(Fail, repoDir, "c.txt", "c")
			gits.Add(Fail, repoDir)
			commitCSha = gits.Commit(Fail, repoDir, "commit c")

			gits.Checkout(Fail, repoDir, "master")
			By("adding commit d on master branch")
			gits.WriteFile(Fail, repoDir, "d.txt", "d")
			gits.Add(Fail, repoDir)
			commitDSha = gits.Commit(Fail, repoDir, "commit d")

			By("adding commit e on master branch")
			gits.WriteFile(Fail, repoDir, "e.txt", "e")
			gits.Add(Fail, repoDir)
			commitESha = gits.Commit(Fail, repoDir, "commit e")
		})

		It("returns all commits in range", func() {
			commits, err := git.GetCommits(repoDir, commitASha, commitESha)
			Expect(err).NotTo(HaveOccurred())
			Expect(commits).Should(HaveLen(2))
			Expect(commits[0].SHA).Should(Equal(commitESha))
			Expect(commits[0].Message).Should(Equal("commit e"))
			Expect(commits[1].SHA).Should(Equal(commitDSha))
			Expect(commits[1].Message).Should(Equal("commit d"))
		})

		It("returns author and committer", func() {
			commits, err := git.GetCommits(repoDir, commitASha, commitDSha)
			Expect(err).NotTo(HaveOccurred())
			Expect(commits).Should(HaveLen(1))
			Expect(commits[0].Author.Name).Should(Equal("test-author"))
			Expect(commits[0].Author.Email).Should(Equal("test-author@acme.com"))
			Expect(commits[0].Committer.Name).Should(Equal("test-committer"))
			Expect(commits[0].Committer.Email).Should(Equal("test-committer@acme.com"))
		})

		It("returns merge commits", func() {
			gits.Merge(Fail, repoDir, commitBSha, commitCSha)
			Expect(err).NotTo(HaveOccurred())

			commits, err := git.GetCommits(repoDir, commitESha, "HEAD")
			Expect(err).NotTo(HaveOccurred())
			Expect(commits).Should(HaveLen(3))
			Expect(commits[0].Message).Should(ContainSubstring("Merge commit"))
		})
	})

	Describe("#GetLatestCommitSha", func() {
		Context("when there is no commit", func() {
			Specify("an error is returned", func() {
				_, err := git.GetLatestCommitSha(repoDir)
				Expect(err).ShouldNot(BeNil())
				// TODO Currently the error message is returned which seems odd. Should be empty string imo.
				//Expect(sha).Should(BeEmpty())
			})
		})

		Context("when there are commits", func() {
			BeforeEach(func() {
				gits.WriteFile(Fail, repoDir, "a.txt", "foo")
				gits.Add(Fail, repoDir)
				gits.Commit(Fail, repoDir, "first commit")
			})

			Specify("the sha of the last commit is returned", func() {
				sha, err := git.GetLatestCommitSha(repoDir)
				Expect(err).Should(BeNil())
				Expect(sha).Should(Equal(gits.ReadRef(Fail, repoDir, "refs/heads/master")))
			})
		})
	})

	Describe("Get version tags", func() {
		var (
			tag2SHA string
			tag3SHA string
		)

		Context("when tags are on master", func() {
			BeforeEach(func() {
				gits.WriteFile(Fail, repoDir, "a.txt", "a")
				gits.Add(Fail, repoDir)
				gits.Commit(Fail, repoDir, "first commit")
				gits.Tag(Fail, repoDir, "v0.0.1", "version 0.0.1")

				gits.WriteFile(Fail, repoDir, "b.txt", "b")
				gits.Add(Fail, repoDir)
				gits.Commit(Fail, repoDir, "second commit")
				tag2SHA = gits.Tag(Fail, repoDir, "v0.0.2", "version 0.0.2")

				gits.WriteFile(Fail, repoDir, "c.txt", "c")
				gits.Add(Fail, repoDir)
				gits.Commit(Fail, repoDir, "third commit")
				tag3SHA = gits.Tag(Fail, repoDir, "v0.0.3", "version 0.0.3")
			})

			It("#GetCurrentGitTagSHA", func() {
				sha, tag, err := git.GetCurrentGitTagSHA(repoDir)
				Expect(err).Should(BeNil())
				Expect(sha).Should(Equal(tag3SHA))
				Expect(tag).Should(Equal("v0.0.3"))
			})

			It("#GetPreviousGitTagSHA", func() {
				sha, tag, err := git.GetPreviousGitTagSHA(repoDir)
				Expect(err).Should(BeNil())
				Expect(sha).Should(Equal(tag2SHA))
				Expect(tag).Should(Equal("v0.0.2"))
			})
		})

		Context("when tags are made on release branches", func() {
			BeforeEach(func() {
				By("creating commit on master")
				gits.WriteFile(Fail, repoDir, "a.txt", "a")
				gits.Add(Fail, repoDir)
				gits.Commit(Fail, repoDir, "first master commit")

				By("creating first release branch")
				gits.Branch(Fail, repoDir, "release_0_0_1")
				gits.WriteFile(Fail, repoDir, "VERSION", "0.0.1")
				gits.Add(Fail, repoDir)
				gits.Commit(Fail, repoDir, "adding version")
				gits.Tag(Fail, repoDir, "v0.0.1", "version 0.0.1")

				By("creating commit on master")
				gits.Checkout(Fail, repoDir, "master")
				gits.WriteFile(Fail, repoDir, "b.txt", "b")
				gits.Add(Fail, repoDir)
				gits.Commit(Fail, repoDir, "second master commit")

				By("creating second release branch")
				gits.Branch(Fail, repoDir, "release_0_0_2")
				gits.WriteFile(Fail, repoDir, "VERSION", "0.0.2")
				gits.Add(Fail, repoDir)
				gits.Commit(Fail, repoDir, "adding version")
				tag2SHA = gits.Tag(Fail, repoDir, "v0.0.2", "version 0.0.2")

				By("creating commit on master")
				gits.Checkout(Fail, repoDir, "master")
				gits.WriteFile(Fail, repoDir, "c.txt", "c")
				gits.Add(Fail, repoDir)
				gits.Commit(Fail, repoDir, "third master commit")

				By("creating third release branch")
				gits.Branch(Fail, repoDir, "release_0_0_3")
				gits.WriteFile(Fail, repoDir, "VERSION", "0.0.3")
				gits.Add(Fail, repoDir)
				gits.Commit(Fail, repoDir, "adding version")
				tag3SHA = gits.Tag(Fail, repoDir, "v0.0.3", "version 0.0.3")
			})

			It("#GetCurrentGitTagSHA", func() {
				sha, tag, err := git.GetCurrentGitTagSHA(repoDir)
				Expect(err).Should(BeNil())
				Expect(sha).Should(Equal(tag3SHA))
				Expect(tag).Should(Equal("v0.0.3"))
			})

			It("#GetPreviousGitTagSHA", func() {
				sha, tag, err := git.GetPreviousGitTagSHA(repoDir)
				Expect(err).Should(BeNil())
				Expect(sha).Should(Equal(tag2SHA))
				Expect(tag).Should(Equal("v0.0.2"))
			})
		})

		Context("when tags are made in detached HEAD mode", func() {
			BeforeEach(func() {
				By("creating commit on master")
				gits.WriteFile(Fail, repoDir, "a.txt", "a")
				gits.Add(Fail, repoDir)
				gits.Commit(Fail, repoDir, "first master commit")

				By("detaching HEAD and creating first release")
				gits.DetachHead(Fail, repoDir)
				gits.WriteFile(Fail, repoDir, "VERSION", "0.0.1")
				gits.Add(Fail, repoDir)
				gits.Commit(Fail, repoDir, "adding version")
				gits.Tag(Fail, repoDir, "v0.0.1", "version 0.0.1")

				By("creating commit on master")
				gits.Checkout(Fail, repoDir, "master")
				gits.WriteFile(Fail, repoDir, "b.txt", "b")
				gits.Add(Fail, repoDir)
				gits.Commit(Fail, repoDir, "second master commit")

				By("detaching HEAD and creating second release")
				gits.DetachHead(Fail, repoDir)
				gits.WriteFile(Fail, repoDir, "VERSION", "0.0.2")
				gits.Add(Fail, repoDir)
				gits.Commit(Fail, repoDir, "adding version")
				tag2SHA = gits.Tag(Fail, repoDir, "v0.0.2", "version 0.0.2")

				By("creating commit on master")
				gits.Checkout(Fail, repoDir, "master")
				gits.WriteFile(Fail, repoDir, "c.txt", "c")
				gits.Add(Fail, repoDir)
				gits.Commit(Fail, repoDir, "third master commit")

				By("detaching HEAD and creating second release")
				gits.DetachHead(Fail, repoDir)
				gits.WriteFile(Fail, repoDir, "VERSION", "0.0.3")
				gits.Add(Fail, repoDir)
				gits.Commit(Fail, repoDir, "adding version")
				tag3SHA = gits.Tag(Fail, repoDir, "v0.0.3", "version 0.0.3")
			})

			It("#GetCurrentGitTagSHA", func() {
				sha, tag, err := git.GetCurrentGitTagSHA(repoDir)
				Expect(err).Should(BeNil())
				Expect(sha).Should(Equal(tag3SHA))
				Expect(tag).Should(Equal("v0.0.3"))
			})

			It("#GetPreviousGitTagSHA", func() {
				sha, tag, err := git.GetPreviousGitTagSHA(repoDir)
				Expect(err).Should(BeNil())
				Expect(sha).Should(Equal(tag2SHA))
				Expect(tag).Should(Equal("v0.0.2"))
			})
		})
	})
})

func TestTags(t *testing.T) {
	gitter := gits.NewGitCLI()
	dir, err := ioutil.TempDir("", "")
	assert.NoError(t, err)
	err = gitter.Init(dir)
	assert.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(dir, "README.md"), []byte("Hello"), 0655)
	assert.NoError(t, err)
	err = gitter.Add(dir, "README.md")
	assert.NoError(t, err)
	err = gitter.CommitDir(dir, "commit 1")
	assert.NoError(t, err)
	err = gitter.CreateTag(dir, "v0.0.1", "version 0.0.1")
	assert.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(dir, "README.md"), []byte("Hello again"), 0655)
	assert.NoError(t, err)
	err = gitter.Add(dir, "README.md")
	assert.NoError(t, err)
	err = gitter.CommitDir(dir, "commit 3")
	assert.NoError(t, err)
	err = gitter.CreateTag(dir, "v0.0.2", "version 0.0.2")
	assert.NoError(t, err)
	tags, err := gitter.Tags(dir)
	assert.NoError(t, err)
	assert.Len(t, tags, 2)
	assert.Contains(t, tags, "v0.0.1")
	assert.Contains(t, tags, "v0.0.2")
}
