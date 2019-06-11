package start

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"

	gojenkins "github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/prow"

	"github.com/spf13/cobra"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	build "github.com/knative/build/pkg/apis/build/v1alpha1"
	v1 "k8s.io/api/core/v1"
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
	*opts.CommonOptions

	Output          string
	Tail            bool
	Filter          string
	JenkinsSelector opts.JenkinsSelectorOptions

	Jobs map[string]gojenkins.Job

	ProwOptions prow.Options
}

var (
	startPipelineLong = templates.LongDesc(`
		Starts the pipeline build.

`)

	startPipelineExample = templates.Examples(`
		# Start a pipeline
		jx start pipeline foo

		# Select the pipeline to start
		jx start pipeline

		# Select the pipeline to start and tail the log
		jx start pipeline -t
	`)
)

// NewCmdStartPipeline creates the command
func NewCmdStartPipeline(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StartPipelineOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:     "pipeline [flags]",
		Short:   "Starts one or more pipelines",
		Long:    startPipelineLong,
		Example: startPipelineExample,
		Aliases: []string{"pipe", "pipeline", "build", "run"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.Tail, "tail", "t", false, "Tails the build log to the current terminal")
	cmd.Flags().StringVarP(&options.Filter, "filter", "f", "", "Filters all the available jobs by those that contain the given text")
	options.JenkinsSelector.AddFlags(cmd)

	return cmd
}

// Run implements this command
func (o *StartPipelineOptions) Run() error {
	kubeClient, currentNamespace, err := o.KubeClientAndNamespace()
	if err != nil {
		return err
	}
	_, _, err = o.JXClient()
	if err != nil {
		return err
	}

	isProw, err := o.IsProw()
	if err != nil {
		return err
	}
	args := o.Args
	names := []string{}
	o.ProwOptions = prow.Options{
		KubeClient: kubeClient,
		NS:         currentNamespace,
	}
	if o.JenkinsSelector.IsCustom() {
		isProw = false
	}
	if len(args) == 0 {
		if isProw {
			names, err = o.ProwOptions.GetReleaseJobs()
			if err != nil {
				return err
			}
			names = util.StringsContaining(names, o.Filter)
		} else {
			jobMap, err := o.GetJenkinsJobs(&o.JenkinsSelector, o.Filter)
			if err != nil {
				return err
			}
			o.Jobs = jobMap

			for k := range o.Jobs {
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
	settings, err := o.TeamSettings()
	if err != nil {
		return err
	}
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
	agent := prowjobv1.KnativeBuildAgent
	if settings.GetProwEngine() == jenkinsv1.ProwEngineTypeTekton {
		agent = prow.TektonAgent
	}
	jobSpec := prowjobv1.ProwJobSpec{
		BuildSpec: postSubmitJob.BuildSpec,
		Agent:     agent,
	}
	jobSpec.Type = prowjobv1.PostsubmitJob

	//todo needs to change when we add support for multiple git providers with Prow
	sourceURL := fmt.Sprintf("https://github.com/%s/%s.git", org, repo)
	sourceSpec := &build.SourceSpec{
		Git: &build.GitSourceSpec{
			Url:      sourceURL,
			Revision: branch,
		},
	}
	if jobSpec.BuildSpec != nil {
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
	}

	p := prow.NewProwJob(jobSpec, nil)
	p.Status = prowjobv1.ProwJobStatus{
		State: prowjobv1.PendingState,
	}
	p.Spec.Refs = &prowjobv1.Refs{
		BaseRef: branch,
		Org:     org,
		Repo:    repo,
	}

	client, currentNamespace, err := o.KubeClientAndNamespace()
	if err != nil {
		return err
	}
	_, err = prow.CreateProwJob(client, currentNamespace, p)
	return err
}

func (o *StartPipelineOptions) startJenkinsJob(name string) error {
	job := o.Jobs[name]

	jenkinsClient, err := o.CreateCustomJenkinsClient(&o.JenkinsSelector)
	if err != nil {
		return err
	}
	job.Url = jenkins.SwitchJenkinsBaseURL(job.Url, jenkinsClient.BaseURL())

	// ignore errors as it could be there's no last build yet
	previous, _ := jenkinsClient.GetLastBuild(job)

	params := url.Values{}
	err = jenkinsClient.Build(job, params)
	if err != nil {
		return err
	}

	i := 0
	for {
		last, err := jenkinsClient.GetLastBuild(job)

		// lets ignore the first query in case there's no build yet
		if i > 0 && err != nil {
			return err
		}
		i++

		if last.Number != previous.Number {
			last.Url = jenkins.SwitchJenkinsBaseURL(last.Url, jenkinsClient.BaseURL())

			log.Logger().Infof("Started build of %s at %s", util.ColorInfo(name), util.ColorInfo(last.Url))
			log.Logger().Infof("%s %s", util.ColorStatus("view the log at:"), util.ColorInfo(util.UrlJoin(last.Url, "/console")))
			if o.Tail {
				return o.TailJenkinsBuildLog(&o.JenkinsSelector, name, &last)
			}
			return nil
		}
		time.Sleep(time.Second)
	}
}
