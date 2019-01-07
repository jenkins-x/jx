package cmd

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/prow"
	"k8s.io/api/core/v1"
	"k8s.io/test-infra/prow/kube"
	"k8s.io/test-infra/prow/pjutil"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	build "github.com/knative/build/pkg/apis/build/v1alpha1"
	prowjobv1 "k8s.io/test-infra/prow/apis/prowjobs/v1"
)

const (
	repoOwnerEnv   = "REPO_OWNER"
	repoNameEnv    = "REPO_NAME"
	jmbrBranchName = "BRANCH_NAME"
	jmbrSourceURL  = "SOURCE_URL"
)

// StartPipelineOptions contains the command line options
type StartPipelineOptions struct {
	GetOptions

	Tail   bool
	Filter string

	Jobs map[string]gojenkins.Job

	ProwOptions prow.Options
}

var (
	start_pipeline_long = templates.LongDesc(`
		Starts the pipeline build.

`)

	start_pipeline_example = templates.Examples(`
		# Start a pipeline
		jx start pipeline foo

		# Select the pipeline to start
		jx start pipeline

		# Select the pipeline to start and tail the log
		jx start pipeline -t
	`)
)

// NewCmdStartPipeline creates the command
func NewCmdStartPipeline(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StartPipelineOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,

				Out: out,
				Err: errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "pipeline [flags]",
		Short:   "Starts one or more pipelines",
		Long:    start_pipeline_long,
		Example: start_pipeline_example,
		Aliases: []string{"pipe", "pipeline", "build", "run"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.Tail, "tail", "t", false, "Tails the build log to the current terminal")
	cmd.Flags().StringVarP(&options.Filter, "filter", "f", "", "Filters all the available jobs by those that contain the given text")

	return cmd
}

// Run implements this command
func (o *StartPipelineOptions) Run() error {
	_, err := o.KubeClient()
	if err != nil {
		return err
	}
	_, _, err = o.JXClient()
	if err != nil {
		return err
	}

	isProw, err := o.isProw()
	if err != nil {
		return err
	}
	args := o.Args
	names := []string{}
	o.ProwOptions = prow.Options{
		NS: o.currentNamespace,
	}
	if len(args) == 0 {
		if isProw {
			names, err = o.ProwOptions.GetReleaseJobs()
			if err != nil {
				return err
			}
		} else {
			jobMap, err := o.getJobMap(o.Filter)
			if err != nil {
				return err
			}
			o.Jobs = jobMap

			for k, _ := range o.Jobs {
				names = append(names, k)
			}
		}
		if len(names) == 0 {
			return errors.New("no jobs found to trigger")
		}
		sort.Strings(names)

		defaultName := ""
		for _, n := range names {
			if strings.HasSuffix(n, "/master") {
				defaultName = n
				break
			}
		}
		name, err := util.PickNameWithDefault(names, "Which pipeline do you want to start: ", defaultName, "", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
		args = []string{name}
	}
	for _, a := range args {
		if isProw {
			err = o.createProwJob(a)
			if err != nil {
				return err
			}
		} else {
			err = o.startJenkinsJob(a)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *StartPipelineOptions) createProwJob(jobname string) error {
	parts := strings.Split(jobname, "/")
	if len(parts) != 3 {
		return fmt.Errorf("job name [%s] does not match org/repo/branch format", jobname)
	}
	org := parts[0]
	repo := parts[1]
	branch := parts[2]

	postSubmitJob, err := o.ProwOptions.GetPostSubmitJob(org, repo, branch)
	if err != nil {
		return err
	}
	jobSpec := kube.ProwJobSpec{
		BuildSpec: postSubmitJob.BuildSpec,
		Agent:     prowjobv1.KnativeBuildAgent,
	}

	//todo needs to change when we add support for multiple git providers with Prow
	sourceURL := fmt.Sprintf("https://github.com/%s/%s.git", org, repo)
	sourceSpec := &build.SourceSpec{
		Git: &build.GitSourceSpec{
			Url:      sourceURL,
			Revision: branch,
		},
	}
	jobSpec.BuildSpec.Source = sourceSpec
	env := map[string]string{}

	// enrich with jenkins multi branch plugin env vars
	env[jmbrBranchName] = branch
	env[jmbrSourceURL] = jobSpec.BuildSpec.Source.Git.Url
	env[repoOwnerEnv] = org
	env[repoNameEnv] = repo

	for i, step := range jobSpec.BuildSpec.Steps {
		if len(step.Env) == 0 {

			step.Env = []v1.EnvVar{}
		}
		for k, v := range env {
			e := v1.EnvVar{
				Name:  k,
				Value: v,
			}
			jobSpec.BuildSpec.Steps[i].Env = append(jobSpec.BuildSpec.Steps[i].Env, e)
		}
	}
	if jobSpec.BuildSpec.Template != nil {
		if len(jobSpec.BuildSpec.Template.Env) == 0 {

			jobSpec.BuildSpec.Template.Env = []v1.EnvVar{}
		}
		for k, v := range env {
			e := v1.EnvVar{
				Name:  k,
				Value: v,
			}
			jobSpec.BuildSpec.Template.Env = append(jobSpec.BuildSpec.Template.Env, e)
		}
	}

	p := pjutil.NewProwJob(jobSpec, nil)
	p.Status = kube.ProwJobStatus{
		State: kube.PendingState,
	}
	p.Spec.Refs = &kube.Refs{
		BaseRef: branch,
		Org:     org,
		Repo:    repo,
	}

	client, err := o.KubeClient()
	if err != nil {
		return err
	}
	_, err = prow.CreateProwJob(client, o.currentNamespace, p)
	return err
}

func (o *StartPipelineOptions) startJenkinsJob(name string) error {
	job := o.Jobs[name]
	jenkins, err := o.JenkinsClient()
	if err != nil {
		return err
	}

	// ignore errors as it could be there's no last build yet
	previous, _ := jenkins.GetLastBuild(job)

	params := url.Values{}
	err = jenkins.Build(job, params)
	if err != nil {
		return err
	}

	i := 0
	for {
		last, err := jenkins.GetLastBuild(job)

		// lets ignore the first query in case there's no build yet
		if i > 0 && err != nil {
			return err
		}
		i++

		if last.Number != previous.Number {
			log.Infof("Started build of %s at %s\n", util.ColorInfo(name), util.ColorInfo(last.Url))
			log.Infof("%s %s\n", util.ColorStatus("view the log at:"), util.ColorInfo(util.UrlJoin(last.Url, "/console")))
			if o.Tail {
				return o.tailBuild(name, &last)
			}
			return nil
		}
		time.Sleep(time.Second)
	}
}

func jobName(prefix string, j *gojenkins.Job) string {
	name := j.FullName
	if name == "" {
		name = j.Name
	}
	if prefix != "" {
		name = prefix + "/" + name
	}
	return name
}

func IsPipeline(j *gojenkins.Job) bool {
	return strings.Contains(j.Class, "Job")
}
