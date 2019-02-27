package tekton

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/builds"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/tekton/syntax"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/knative/build-pipeline/pkg/apis/pipeline"
	tektonclient "github.com/knative/build-pipeline/pkg/client/clientset/versioned"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PipelineRunInfo provides information on a PipelineRun and its stages for use in getting logs and populating activity
type PipelineRunInfo struct {
	Name              string
	Organisation      string
	Repository        string
	Branch            string
	Build             string
	BuildNumber       int
	Pipeline          string
	PipelineRun       string
	LastCommitSHA     string
	LastCommitMessage string
	LastCommitURL     string
	GitURL            string
	GitInfo           *gits.GitRepository
	Stages            []*StageInfo
}

// StageInfo provides information on a particular stage, including its pod info or info on its nested stages
type StageInfo struct {
	// TODO: For now, we're not including git info - we're going to assume we have the same git info for the whole
	// pipeline.
	Name string

	// These fields will populated for all non-parent stages
	PodName        string
	Task           string
	TaskRun        string
	FirstStepImage string
	CreatedTime    time.Time
	Pod            *corev1.Pod

	// These fields will only be populated for appropriate parent stages
	Parallel []*StageInfo
	Stages   []*StageInfo

	// This field will be non-empty if this is a nested stage, containing a list of  the names of all its parent stages with the top-level parent first
	Parents []string
}

// GetStageNameIncludingParents constructs a full stage name including its parents, if they exist.
func (si *StageInfo) GetStageNameIncludingParents() string {
	if si.Name != "" {
		return strings.Join(append(si.Parents, si.Name), " / ")
	}
	return si.PodName
}

// PipelineRunInfoFilter allows specifying criteria on which to filter a list of PipelineRunInfos
type PipelineRunInfoFilter struct {
	Owner      string
	Repository string
	Branch     string
	Build      string
	Filter     string
	Pending    bool
}

func getPipelineStructureForPipelineRun(jxClient versioned.Interface, ns, prName string) (*v1.PipelineStructure, error) {
	var ps *v1.PipelineStructure

	// The PipelineStructure may not exist yet.
	f := func() error {
		// Get the PipelineStructure for this PipelineRun
		lookupPs, err := jxClient.JenkinsV1().PipelineStructures(ns).Get(prName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if lookupPs == nil {
			log.Infof("no PipelineStructure found yet for PipelineRun %s\n", util.ColorInfo(prName))
			return fmt.Errorf("No PipelineStructure found yet for PipelineRun %s", prName)
		}
		ps = lookupPs
		return nil
	}
	err := util.Retry(time.Minute*2, f)
	if err != nil {
		return nil, err
	}
	return ps, nil
}

func getBuildPodForPipelineRun(kubeClient kubernetes.Interface, ns, prName string, prStatus *duckv1alpha1.Condition) (*corev1.Pod, error) {
	var pod *corev1.Pod

	// The Pod may not exist yet.
	f := func() error {
		// Get the Pod for this PipelineRun
		podList, err := kubeClient.CoreV1().Pods(ns).List(metav1.ListOptions{
			LabelSelector: builds.LabelPipelineRunName + "=" + prName,
		})
		if err != nil {
			return err
		}

		// Only retry if there's no pod and the PipelineRun hasn't yet completed.
		if len(podList.Items) == 0 && prStatus != nil && prStatus.Status == corev1.ConditionUnknown {
			log.Infof("no Pod found yet for PipelineRun %s\n", util.ColorInfo(prName))
			return fmt.Errorf("No Pod found yet for PipelineRun %s", prName)
		}
		if len(podList.Items) == 1 {
			pod = &podList.Items[0]
		}
		return nil
	}
	err := util.Retry(time.Minute*2, f)
	if err != nil {
		return nil, err
	}
	return pod, nil
}

// GetBuild gets the build identifier
func (pri PipelineRunInfo) GetBuild() string {
	return pri.Build
}

// GetOrderedTaskStages gets all the stages in this pipeline which actually contain a Task, in rough execution order
// TODO: Handle parallelism better, where execution is not a straight line.
func (pri *PipelineRunInfo) GetOrderedTaskStages() []*StageInfo {
	var stages []*StageInfo

	for _, n := range pri.Stages {
		stages = append(stages, n.getOrderedTaskStagesForStage()...)
	}

	return stages
}

func (si *StageInfo) getOrderedTaskStagesForStage() []*StageInfo {
	// If this is a Task Stage, not a parent Stage, return itself
	if si.Task != "" {
		return []*StageInfo{si}
	}

	var stages []*StageInfo

	if len(si.Stages) > 0 {
		for _, n := range si.Stages {
			stages = append(stages, n.getOrderedTaskStagesForStage()...)
		}
	}

	if len(si.Parallel) > 0 {
		for _, n := range si.Parallel {
			stages = append(stages, n.getOrderedTaskStagesForStage()...)
		}
	}

	return stages
}

// CreatePipelineRunInfo looks up the PipelineRun for a given name and creates the PipelineRunInfo for it
func CreatePipelineRunInfo(kubeClient kubernetes.Interface, tektonClient tektonclient.Interface, jxClient versioned.Interface, ns, prName string) (*PipelineRunInfo, error) {
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

	pr, err := tektonClient.TektonV1alpha1().PipelineRuns(ns).Get(prName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, fmt.Sprintf("PipelineRun %s cannot be found", prName))
	}

	pri := &PipelineRunInfo{
		Name:        prName,
		PipelineRun: pr.Name,
		Pipeline:    pr.Spec.PipelineRef.Name,
	}

	var pod *corev1.Pod

	prStatus := pr.Status.GetCondition(duckv1alpha1.ConditionSucceeded)
	// TODO: Remove this when we unify generation
	if pr.Labels[syntax.LabelPipelineFromYaml] != "true" {
		pod, err = getBuildPodForPipelineRun(kubeClient, ns, prName, prStatus)
		if err != nil {
			return nil, errors.Wrapf(err, "Error finding Pod for PipelineRun %s", prName)
		}
		if pod == nil {
			if prStatus != nil && prStatus.Status == corev1.ConditionUnknown {
				return nil, fmt.Errorf("could not find Pod for PipelineRun %s", prName)
			}
			// The PipelineRun has completed and its pod(s) no longer exist, so just return nil in general.
			return nil, nil
		}

		si := &StageInfo{
			Task:    pod.Labels[builds.LabelTaskName],
			Pod:     pod,
			PodName: pod.Name,
			TaskRun: pod.Labels[builds.LabelTaskRunName],
		}

		si.CreatedTime = pod.CreationTimestamp.Time
		if len(pod.Spec.InitContainers) > 2 {
			si.FirstStepImage = pod.Spec.InitContainers[2].Image
		}

		pri.Stages = append(pri.Stages, si)
	} else {
		if err := pri.SetPodsForPipelineRun(kubeClient, tektonClient, jxClient, ns); err != nil {
			return nil, errors.Wrapf(err, "Failure populating stages and pods for PipelineRun %s", prName)
		}
	}

	pod = pri.FindFirstStagePod()

	if pod == nil {
		if prStatus != nil && prStatus.Status == corev1.ConditionUnknown {
			return nil, errors.New(fmt.Sprintf("Couldn't find a Stage with steps for PipelineRun %s", prName))
		}
		// Just return nil if the pipeline run is completed and its pods have been GCed
		return nil, nil
	}

	for _, initContainer := range pod.Spec.InitContainers {
		if strings.HasPrefix(initContainer.Name, "build-step-git-source") {
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
		build = builds.GetBuildNumberFromLabels(pr.Labels)
	}
	if build == "" {
		build = "1"
	}
	buildNumber, err := strconv.Atoi(build)
	if err != nil {
		buildNumber = 1
	}

	pri.Build = build
	pri.BuildNumber = buildNumber
	pri.Branch = branch
	if gitURL != "" {
		gitInfo, err := gits.ParseGitURL(gitURL)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to parse Git URL %s", gitURL)
		}
		if owner == "" {
			owner = gitInfo.Organisation
		}
		if repo == "" {
			repo = gitInfo.Name
		}
		pri.GitInfo = gitInfo
		pri.Pipeline = owner + "/" + repo + "/" + branch
		pri.Name = owner + "-" + repo + "-" + branch + "-" + build
		pri.Organisation = owner
		pri.Repository = repo
		pri.GitURL = gitURL
		pri.LastCommitMessage = lastCommitMessage
		pri.LastCommitSHA = lastCommitSha
		pri.LastCommitURL = lastCommitURL
	}
	return pri, nil
}

// SetPodsForPipelineRun populates the pods for all stages within its PipelineRunInfo
func (pri *PipelineRunInfo) SetPodsForPipelineRun(kubeClient kubernetes.Interface, tektonClient tektonclient.Interface, jxClient versioned.Interface, ns string) error {
	if pri.PipelineRun == "" {
		return errors.New("No PipelineRun specified")
	}
	ps, err := getPipelineStructureForPipelineRun(jxClient, ns, pri.PipelineRun)
	if err != nil {
		return err
	}
	if ps == nil {
		return errors.New(fmt.Sprintf("Could not find PipelineStructure for PipelineRun %s", pri.PipelineRun))
	}

	pscs := ps.GetAllStagesAndChildren()

	var firstTaskStage *StageInfo

	for _, psc := range pscs {
		pri.Stages = append(pri.Stages, stageAndChildrenToStageInfo(psc, []string{}))
	}

	for _, si := range pri.Stages {
		if firstTaskStage == nil {
			firstTaskStage = si
		}
		if err := si.SetPodsForStageInfo(kubeClient, tektonClient, ns, pri.PipelineRun); err != nil {
			return errors.Wrapf(err, "Couldn't populate Pods for Stages")
		}
	}

	return nil
}

// SetPodsForStageInfo populates the pods for a particular stage and/or its children
func (si *StageInfo) SetPodsForStageInfo(kubeClient kubernetes.Interface, tektonClient tektonclient.Interface, ns, prName string) error {
	if si.Task != "" {
		selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: map[string]string{
			pipeline.GroupName + pipeline.PipelineRunLabelKey: prName,
			syntax.LabelStageName:                             si.Name,
		}})
		if err != nil {
			return err
		}

		podList, err := kubeClient.CoreV1().Pods(ns).List(metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err != nil {
			return err
		}
		if len(podList.Items) == 0 {
			// TODO: Probably the pod just hasn't started yet, so return nil
			return nil
		}
		if len(podList.Items) > 1 {
			return errors.New(fmt.Sprintf("Too many Pods (%d) found for PipelineRun %s and Stage %s", len(podList.Items), prName, si.Name))
		}
		pod := podList.Items[0]
		si.PodName = pod.Name
		si.Task = pod.Labels[builds.LabelTaskName]
		si.TaskRun = pod.Labels[builds.LabelTaskRunName]
		si.Pod = &pod
		si.CreatedTime = pod.CreationTimestamp.Time
		if len(pod.Spec.InitContainers) > 2 {
			si.FirstStepImage = pod.Spec.InitContainers[2].Image
		}
	} else if len(si.Stages) > 0 {
		for _, child := range si.Stages {
			if err := child.SetPodsForStageInfo(kubeClient, tektonClient, ns, prName); err != nil {
				return err
			}
		}
	} else if len(si.Parallel) > 0 {
		for _, child := range si.Parallel {
			if err := child.SetPodsForStageInfo(kubeClient, tektonClient, ns, prName); err != nil {
				return err
			}
		}
	}

	return nil
}

// FindFirstStagePod finds the first stage in this pipeline run to have a pod, and then returns its pod
func (pri *PipelineRunInfo) FindFirstStagePod() *corev1.Pod {
	for _, s := range pri.Stages {
		found := s.findTaskStageInfo()
		if found != nil {
			return found.Pod
		}
	}
	return nil
}

// findTaskStageInfo gets the first stage that should actually have a pod created for it
func (si *StageInfo) findTaskStageInfo() *StageInfo {
	if si.Task != "" {
		return si
	}
	for _, s := range si.Parallel {
		child := s.findTaskStageInfo()
		if child != nil {
			return child
		}
	}
	for _, s := range si.Stages {
		child := s.findTaskStageInfo()
		if child != nil {
			return child
		}
	}

	return nil
}

// GetFullChildStageNames gets the fully qualified (i.e., with parents appended) names of each stage underneath this one.
func (si *StageInfo) GetFullChildStageNames(includeSelf bool) []string {
	if si.Task != "" && includeSelf {
		return []string{si.GetStageNameIncludingParents()}
	}

	var names []string
	for _, n := range si.Parallel {
		names = append(names, n.GetFullChildStageNames(true)...)
	}
	for _, n := range si.Stages {
		names = append(names, n.GetFullChildStageNames(true)...)
	}

	return names
}

func stageAndChildrenToStageInfo(psc *v1.PipelineStageAndChildren, parents []string) *StageInfo {
	si := &StageInfo{
		Name:    psc.Stage.Name,
		Parents: parents,
	}
	if psc.Stage.TaskRef != nil {
		si.Task = *psc.Stage.TaskRef
	}

	for _, s := range psc.Stages {
		si.Stages = append(si.Stages, stageAndChildrenToStageInfo(&s, append(parents, psc.Stage.Name)))
	}

	for _, s := range psc.Parallel {
		si.Parallel = append(si.Parallel, stageAndChildrenToStageInfo(&s, append(parents, psc.Stage.Name)))
	}

	return si
}

// PipelineRunMatches returns true if the pipeline run info matches the filter
func (o *PipelineRunInfoFilter) PipelineRunMatches(info *PipelineRunInfo) bool {
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
func (o *PipelineRunInfoFilter) BuildNumber() int {
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
func (pri *PipelineRunInfo) MatchesPipeline(activity *v1.PipelineActivity) bool {
	d := kube.CreatePipelineDetails(activity)
	if d == nil {
		return false
	}
	return d.GitOwner == pri.Organisation && d.GitRepository == pri.Repository && d.Build == pri.Build && strings.ToLower(d.BranchName) == strings.ToLower(pri.Branch)
}

// Status returns the build status
func (pri *PipelineRunInfo) Status() string {
	pod := pri.FindFirstStagePod()
	if pod == nil {
		return "No Pod"
	}
	return string(pod.Status.Phase)
}

// PipelineRunInfoOrder allows sorting of a slice of PipelineRunInfos
type PipelineRunInfoOrder []*PipelineRunInfo

func (a PipelineRunInfoOrder) Len() int      { return len(a) }
func (a PipelineRunInfoOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a PipelineRunInfoOrder) Less(i, j int) bool {
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

// SortPipelineRunInfos sorts a slice of PipelineRunInfos by their org, repo, branch, and build number
func SortPipelineRunInfos(pris []*PipelineRunInfo) {
	sort.Sort(PipelineRunInfoOrder(pris))
}
