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
	"testing"
)

func TestStepGitMerge(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Step Git Merge Suite")
}

var _ = Describe("step git merge", func() {
	var (
		masterSha string
		branchSha string
		repoDir   string
		err       error
	)

	BeforeEach(func() {
		repoDir, err = ioutil.TempDir("", "jenkins-x-git-test-repo-")
		if err != nil {
			Fail("unable to create test repo dir")
		}
		log.Logger().Debugf("created temporary git repo under %s", repoDir)
		gits.GitCmd(Fail, repoDir, "init")

		gits.WriteFile(Fail, repoDir, "a.txt", "a")
		gits.Add(Fail, repoDir)
		masterSha = gits.Commit(Fail, repoDir, "a commit")

		gits.Branch(Fail, repoDir, "foo")
		gits.WriteFile(Fail, repoDir, "b.txt", "b")
		gits.Add(Fail, repoDir)
		branchSha = gits.Commit(Fail, repoDir, "b commit")

		gits.Checkout(Fail, repoDir, "master")
	})

	AfterEach(func() {
		_ = os.RemoveAll(repoDir)
		log.Logger().Debugf("deleted temporary git repo under %s", repoDir)
	})

	Context("with command line options", func() {
		It("succeeds", func() {
			currentHeadSha := gits.HeadSha(Fail, repoDir)
			Expect(currentHeadSha).Should(Equal(masterSha))

			options := StepGitMergeOptions{
				StepOptions: opts.StepOptions{
					CommonOptions: &opts.CommonOptions{},
				},
				SHAs:       []string{branchSha},
				Dir:        repoDir,
				BaseBranch: "master",
				BaseSHA:    masterSha,
			}

			err := options.Run()
			Expect(err).NotTo(HaveOccurred())

			currentHeadSha = gits.HeadSha(Fail, repoDir)
			Expect(currentHeadSha).Should(Equal(branchSha))
		})
	})

	Context("with PULL_REFS", func() {
		BeforeEach(func() {
			err := os.Setenv("PULL_REFS", fmt.Sprintf("master:%s,foo:%s", masterSha, branchSha))
			if err != nil {
				Fail("unable to set PULL_REFS")
			}

		})

		AfterEach(func() {
			err := os.Unsetenv("PULL_REFS")
			if err != nil {
				Fail("unable to unset PULL_REFS")
			}
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
			Expect(currentHeadSha).Should(Equal(branchSha))
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
