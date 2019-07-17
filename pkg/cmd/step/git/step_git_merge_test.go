package git

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestStepGitMerge(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Step Git Merge Suite")
}

var _ = Describe("step git merge", func() {
	var (
		masterSha        string
		branchBSha       string
		branchCSha       string
		repoDir          string
		err              error
		testStdoutReader *os.File
		testStdoutWriter *os.File
		origStdout       *os.File
	)

	BeforeSuite(func() {
		// comment out to see logging output
		log.SetOutput(ioutil.Discard)
		_ = log.SetLevel("info")
	})

	BeforeEach(func() {
		By("capturing stdout")
		testStdoutReader, testStdoutWriter, _ = os.Pipe()
		origStdout = os.Stdout
		os.Stdout = testStdoutWriter
	})

	BeforeEach(func() {
		repoDir, err = ioutil.TempDir("", "jenkins-x-git-test-repo-")
		By(fmt.Sprintf("creating a test repository in '%s'", repoDir))
		Expect(err).NotTo(HaveOccurred())
		gits.GitCmd(Fail, repoDir, "init")

		By("adding single commit on master branch")
		gits.WriteFile(Fail, repoDir, "a.txt", "a")
		gits.Add(Fail, repoDir)
		masterSha = gits.Commit(Fail, repoDir, "a commit")

		By("creating branch 'b' and adding a commit")
		gits.Branch(Fail, repoDir, "b")
		gits.WriteFile(Fail, repoDir, "b.txt", "b")
		gits.Add(Fail, repoDir)
		branchBSha = gits.Commit(Fail, repoDir, "b commit")

		By("creating branch 'c' and adding a commit")
		gits.Checkout(Fail, repoDir, "master")
		gits.Branch(Fail, repoDir, "c")
		gits.WriteFile(Fail, repoDir, "c.txt", "c")
		gits.Add(Fail, repoDir)
		branchCSha = gits.Commit(Fail, repoDir, "c commit")

		By("checking out master")
		gits.Checkout(Fail, repoDir, "master")
	})

	AfterEach(func() {
		By("closing and resetting stdout")
		_ = testStdoutWriter.Close()
		os.Stdout = origStdout
		_ = os.RemoveAll(repoDir)
	})

	AfterEach(func() {
		By("deleting temp repo")
		_ = os.RemoveAll(repoDir)
	})

	Context("with command line options", func() {
		It("succeeds", func() {
			currentHeadSha := gits.HeadSha(Fail, repoDir)
			Expect(currentHeadSha).Should(Equal(masterSha))

			options := StepGitMergeOptions{
				StepOptions: opts.StepOptions{
					CommonOptions: &opts.CommonOptions{},
				},
				SHAs:       []string{branchBSha},
				Dir:        repoDir,
				BaseBranch: "master",
				BaseSHA:    masterSha,
			}

			err := options.Run()
			Expect(err).NotTo(HaveOccurred())

			currentHeadSha = gits.HeadSha(Fail, repoDir)
			Expect(currentHeadSha).Should(Equal(branchBSha))
		})
	})

	Context("with PULL_REFS", func() {
		BeforeEach(func() {
			err := os.Setenv("PULL_REFS", fmt.Sprintf("master:%s,b:%s", masterSha, branchBSha))
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err := os.Unsetenv("PULL_REFS")
			Expect(err).NotTo(HaveOccurred())
		})

		It("succeeds", func() {
			currentHeadSha := gits.HeadSha(Fail, repoDir)
			Expect(currentHeadSha).Should(Equal(masterSha))

			options := StepGitMergeOptions{
				StepOptions: opts.StepOptions{
					CommonOptions: &opts.CommonOptions{},
				},
				Dir: repoDir,
			}

			err := options.Run()
			Expect(err).NotTo(HaveOccurred())

			currentHeadSha = gits.HeadSha(Fail, repoDir)
			Expect(currentHeadSha).Should(Equal(branchBSha))
		})
	})

	Context("with multiple merge SHAs in PULL_REFS", func() {
		BeforeEach(func() {
			err := os.Setenv("PULL_REFS", fmt.Sprintf("master:%s,c:%s,b:%s", masterSha, branchCSha, branchBSha))
			Expect(err).NotTo(HaveOccurred())

		})

		AfterEach(func() {
			err := os.Unsetenv("PULL_REFS")
			Expect(err).NotTo(HaveOccurred())
		})

		It("merges all shas and creates a merge commit", func() {
			currentHeadSha := gits.HeadSha(Fail, repoDir)
			Expect(currentHeadSha).Should(Equal(masterSha))

			options := StepGitMergeOptions{
				StepOptions: opts.StepOptions{
					CommonOptions: &opts.CommonOptions{
						Verbose: true,
					},
				},
				Dir: repoDir,
			}

			err := options.Run()
			Expect(err).NotTo(HaveOccurred())

			stdout, err := readStdOut(testStdoutReader, testStdoutWriter)
			Expect(err).NotTo(HaveOccurred())

			logLines := strings.Split(stdout, "\n")
			logLines = deleteEmpty(logLines)
			Expect(len(logLines)).Should(Equal(4))
			Expect(strings.TrimSpace(logLines[0])).Should(Equal("MERGED SHA SUBJECT"))
			Expect(strings.TrimSpace(logLines[1])).Should(MatchRegexp(".* Merge commit '.*'"))
		})
	})

	Context("with no options and no PULL_REFS", func() {
		It("logs warning", func() {
			options := StepGitMergeOptions{
				StepOptions: opts.StepOptions{
					CommonOptions: &opts.CommonOptions{},
				},
				Dir: repoDir,
			}

			out := log.CaptureOutput(func() {
				err := options.Run()
				Expect(err).NotTo(HaveOccurred())

				currentHeadSha := gits.HeadSha(Fail, repoDir)
				Expect(currentHeadSha).Should(Equal(masterSha))
			})

			Expect(out).Should(ContainSubstring("no SHAs to merge"))
		})
	})
})

func readStdOut(r *os.File, w *os.File) (string, error) {
	err := w.Close()
	if err != nil {
		return "", err
	}
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func deleteEmpty(s []string) []string {
	var clean []string
	for _, str := range s {
		if str != "" {
			clean = append(clean, str)
		}
	}
	return clean
}
