package builds

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	corev1 "k8s.io/api/core/v1"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type BuildPodInfo struct {
	PodName           string
	Name              string
	Organisation      string
	Repository        string
	Branch            string
	Build             string
	BuildNumber       int
	Pipeline          string
	LastCommitSHA     string
	LastCommitMessage string
	LastCommitURL     string
	GitURL            string
	FirstStepImage    string
	CreatedTime       time.Time
	GitInfo           *gits.GitRepository
	Pod               *corev1.Pod
}

type BuildPodInfoFilter struct {
	Owner      string
	Repository string
	Branch     string
	Build      string
	Filter     string
	Pending    bool
}

// CreateBuildPodInfo creates a BuildPodInfo from a Pod
func CreateBuildPodInfo(pod *corev1.Pod) *BuildPodInfo {
	branch := ""
	lastCommitSha := ""
	lastCommitMessage := ""
	lastCommitURL := ""
	owner := ""
	repo := ""
	build := ""
	shaRegexp, err := regexp.Compile("\b[a-z0-9]{40}\b")
	if err != nil {
		log.Warnf("Failed to compile regexp because %s", err)
	}
	gitURL := ""
	for _, initContainer := range pod.Spec.InitContainers {
		if initContainer.Name == "build-step-git-source" || initContainer.Name == "build-step-git-source-source" {
			args := initContainer.Args
			for i := 0; i <= len(args)-2; i += 2 {
				key := args[i]
				value := args[i+1]

				switch key {
				case "-url":
					gitURL = value
				case "-revision":
					if shaRegexp.MatchString(value) {
						lastCommitSha = value
					} else {
						branch = value
					}
				}
			}
		}
		var pullPullSha, pullBaseSha string
		for _, v := range initContainer.Env {
			if v.Value == "" {
				continue
			}
			if v.Name == "PULL_PULL_SHA" {
				pullPullSha = v.Value
			}
			if v.Name == "PULL_BASE_SHA" {
				pullBaseSha = v.Value
			}
			if v.Name == "BRANCH_NAME" {
				branch = v.Value
			}
			if v.Name == "REPO_OWNER" {
				owner = v.Value
			}
			if v.Name == "REPO_NAME" {
				repo = v.Value
			}
			if v.Name == "JX_BUILD_NUMBER" {
				build = v.Value
			}
			if v.Name == "SOURCE_URL" && gitURL == "" {
				gitURL = v.Value
			}
		}
		if branch == "" {
			for _, v := range initContainer.Env {
				if v.Name == "PULL_BASE_REF" {
					build = v.Value
				}
			}
		}
		if build == "" {
			for _, v := range initContainer.Env {
				if v.Name == "BUILD_NUMBER" || v.Name == "BUILD_ID" {
					build = v.Value
				}
			}
		}
		if lastCommitSha == "" && pullPullSha != "" {
			lastCommitSha = pullPullSha
		}
		if lastCommitSha == "" && pullBaseSha != "" {
			lastCommitSha = pullBaseSha
		}
	}
	if build == "" {
		build = GetBuildNumberFromLabels(pod.Labels)
	}
	if build == "" {
		build = "1"
	}
	if branch == "" {
		branch = "master"
	}
	buildNumber, err := strconv.Atoi(build)
	if err != nil {
		buildNumber = 1
	}
	answer := &BuildPodInfo{
		Pod:               pod,
		PodName:           pod.Name,
		Build:             build,
		BuildNumber:       buildNumber,
		Branch:            branch,
		GitURL:            gitURL,
		LastCommitSHA:     lastCommitSha,
		LastCommitMessage: lastCommitMessage,
		LastCommitURL:     lastCommitURL,
		CreatedTime:       pod.CreationTimestamp.Time,
	}
	if len(pod.Spec.InitContainers) > 2 {
		answer.FirstStepImage = pod.Spec.InitContainers[2].Image
	}

	if gitURL != "" {
		gitInfo, err := gits.ParseGitURL(gitURL)
		if err != nil {
			log.Warnf("Failed to parse Git URL %s: %s", gitURL, err)
			return nil
		}
		if owner == "" {
			owner = gitInfo.Organisation
		}
		if repo == "" {
			repo = gitInfo.Name
		}
		answer.GitInfo = gitInfo
		answer.Pipeline = owner + "/" + repo + "/" + branch
		answer.Name = owner + "-" + repo + "-" + branch + "-" + build
	}
	answer.Organisation = owner
	answer.Repository = repo
	return answer
}

// BuildMatches returns true if the build info matches the filter
func (o *BuildPodInfoFilter) BuildMatches(info *BuildPodInfo) bool {
	if o.Owner != "" && o.Owner != info.Organisation {
		return false
	}
	if o.Repository != "" && o.Repository != info.Repository {
		return false
	}
	if o.Branch != "" && strings.ToLower(o.Branch) != strings.ToLower(info.Branch) {
		return false
	}
	if o.Build != "" && o.Build != info.Build {
		return false
	}
	if o.Filter != "" && !strings.Contains(info.Name, o.Filter) {
		return false
	}
	if o.Pending {
		status := info.Status()
		if status != "Pending" && status != "Running" {
			return false
		}
	}
	return true
}

// BuildNumber returns the integer build number filter if specified
func (o *BuildPodInfoFilter) BuildNumber() int {
	text := o.Build
	if text != "" {
		answer, err := strconv.Atoi(text)
		if err != nil {
			return answer
		}
	}
	return 0
}

// MatchesPipeline returns true if this build info matches the given pipeline
func (b *BuildPodInfo) MatchesPipeline(activity *v1.PipelineActivity) bool {
	d := kube.CreatePipelineDetails(activity)
	if d == nil {
		return false
	}
	return d.GitOwner == b.Organisation && d.GitRepository == b.Repository && d.Build == b.Build && strings.ToLower(d.BranchName) == strings.ToLower(b.Branch)
}

// Status returns the build status
func (b *BuildPodInfo) Status() string {
	pod := b.Pod
	if pod == nil {
		return "No Pod"
	}
	return string(pod.Status.Phase)
}

type BuildPodInfoOrder []*BuildPodInfo

func (a BuildPodInfoOrder) Len() int      { return len(a) }
func (a BuildPodInfoOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a BuildPodInfoOrder) Less(i, j int) bool {
	b1 := a[i]
	b2 := a[j]
	if b1.Organisation != b2.Organisation {
		return b1.Organisation < b2.Organisation
	}
	if b1.Repository != b2.Repository {
		return b1.Repository < b2.Repository
	}
	if b1.Branch != b2.Branch {
		return b1.Branch < b2.Branch
	}
	return b1.BuildNumber > b2.BuildNumber
}

func SortBuildPodInfos(buildPodInfos []*BuildPodInfo) {
	sort.Sort(BuildPodInfoOrder(buildPodInfos))
}
