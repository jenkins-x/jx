package report

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube/naming"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/cobra"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	stepReportImageVersionLong    = templates.LongDesc(`Creates a report of a set of package versions. This command is typically used inside images to determine what tools are inside.`)
	stepReportImageVersionExample = templates.Examples(`
`)
)

// ImageVersionReport the report
type ImageVersionReport struct {
	Versions []Pair `json:"versions,omitempty"`
	Failures []Pair `json:"failures,omitempty"`
}

// StepReportImageVersionOptions contains the command line flags and other helper objects
type StepReportImageVersionOptions struct {
	StepReportOptions
	FileName       string
	VersionsDir    string
	GitURL         string
	GitBranch      string
	ServiceAccount string
	Folder         string
	BackoffLimit   int32

	Report ImageVersionReport
}

// NewCmdStepReportImageVersion Creates a new Command object
func NewCmdStepReportImageVersion(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepReportImageVersionOptions{
		StepReportOptions: StepReportOptions{
			StepOptions: step.StepOptions{
				CommonOptions: commonOpts,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "image versions",
		Short:   "Creates a report of a set of package versions",
		Aliases: []string{"iv"},
		Long:    stepReportImageVersionLong,
		Example: stepReportImageVersionExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	options.StepReportOptions.AddReportFlags(cmd)

	cmd.Flags().StringVarP(&options.FileName, "name", "n", "", "The name of the file to generate")
	cmd.Flags().StringVarP(&options.VersionsDir, "dir", "d", "", "The dir of the version stream. If not specified it the version stream is cloned")
	cmd.Flags().StringVarP(&options.ServiceAccount, "service-account", "", "tekton-bot", "The Kubernetes ServiceAccount to use to run the Job")
	cmd.Flags().Int32VarP(&options.BackoffLimit, "backoff-limit", "l", int32(2), "The backoff limit: how many times to retry the job before considering it failed) to run in the Job")
	cmd.Flags().StringVarP(&options.GitURL, "git-url", "", "", "The git URL of the project to store the results")
	cmd.Flags().StringVarP(&options.GitBranch, "branch", "", "", "The git branch to store the results")
	cmd.Flags().StringVarP(&options.Folder, "folder", "", "reports/imageVersions", "The folder to put the reports inside")
	return cmd
}

// Run generates the report
func (o *StepReportImageVersionOptions) Run() error {
	if o.VersionsDir == "" {
		resolver, err := o.GetVersionResolver()
		if err != nil {
			return err
		}
		o.VersionsDir = resolver.VersionsDir
	}
	dir := filepath.Join(o.VersionsDir, "docker")
	exists, err := util.DirExists(dir)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("directory does not exist %s", dir)
	}

	report := &o.Report
	err = o.generateReport(dir)
	if err != nil {
		return err
	}
	return o.OutputReport(report, o.FileName, o.OutputDir)
}

func (o *StepReportImageVersionOptions) generateReport(imagesDir string) error {
	m, err := LoadImageMap(imagesDir)
	if err != nil {
		return err
	}

	if len(m.Images) == 0 {
		return fmt.Errorf("no images found")
	}

	kubeClient, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}

	id, err := uuid.NewV4()
	if err != nil {
		return err
	}

	containers := []corev1.Container{}
	counter := 0
	for image, versions := range m.Images {
		for version, source := range versions {
			log.Logger().Infof("processing image %s version %s from %s", image, version, source)

			counter++
			name := "c" + strconv.Itoa(counter)
			containers = append(containers, o.createImageVersionContainer(name, image, version, source))
		}
	}

	name := naming.ToValidName("jx-report-image-version-" + id.String())
	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Now(),
				},
				Spec: corev1.PodSpec{
					Containers:         containers,
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: o.ServiceAccount,
				},
			},
			BackoffLimit: &o.BackoffLimit,
		},
	}
	_, err = kubeClient.BatchV1().Jobs(ns).Create(job)
	if err != nil {
		data, err := yaml.Marshal(job)
		if err == nil {
			log.Logger().Warnf("failed to create job %s %s", err.Error(), string(data))
		}
		return errors.Wrapf(err, "failed to create job %s", job.Name)
	} else {
		log.Logger().Infof("created Job %s", util.ColorInfo(job.Name))
	}

	// wait for job to complete?
	return nil
}

func (o *StepReportImageVersionOptions) createImageVersionContainer(name string, image string, version string, source string) corev1.Container {
	// TODO
	//fullImage := image + ":" + version
	fullImage := "gcr.io/jenkinsxio/builder-go:0.0.0-SNAPSHOT-PR-5365-3"

	path := filepath.Join(o.Folder, image+"-"+version)
	args := fmt.Sprintf(` --to-path="%s"`, path)
	if o.GitURL != "" {
		args += " --git-url " + o.GitURL
	}
	if o.GitBranch != "" {
		args += " --git-branch " + o.GitBranch
	}
	commands := "jx step report version -n versions.yml;\njx step stash -c reports -p versions.yml" + args

	envVars := []corev1.EnvVar{}
	if o.GitURL != "" {
		commands = "jx step git credentials;\n" + commands
		envVars = append(envVars, corev1.EnvVar{
			Name:  "XDG_CONFIG_HOME",
			Value: "/workspace/xdg_config",
		})
		envVars = append(envVars, corev1.EnvVar{
			Name:  "GIT_AUTHOR_NAME",
			Value: "jenkins-x-bot",
		})
		envVars = append(envVars, corev1.EnvVar{
			Name:  "GIT_AUTHOR_EMAIL",
			Value: "jenkins-x@googlegroups.com",
		})
		envVars = append(envVars, corev1.EnvVar{
			Name:  "GIT_COMMITTER_NAME",
			Value: "jenkins-x-bot",
		})
		envVars = append(envVars, corev1.EnvVar{
			Name:  "GIT_COMMITTER_EMAIL",
			Value: "jenkins-x@googlegroups.com",
		})
	}
	return corev1.Container{
		Name:    name,
		Image:   fullImage,
		Command: []string{"/bin/sh", "-c"},
		Args:    []string{commands},
		Env:     envVars,
	}
}

func (o *StepReportImageVersionOptions) getPackageVersion(name string) (string, error) {
	args := []string{"version"}
	switch name {
	case "jx":
		args = []string{"--version"}
	case "kubectl":
		args = append(args, "--client", "--short")
	case "helm":
		args = append(args, "--client", "--short")
	case "helm3":
		args = append(args, "--short")
	}
	version, err := o.GetCommandOutput("", name, args...)

	// lets trim non-numeric prefixes such as for `git version` returning `git version 1.2.3`
	idxs := numberRegex.FindStringIndex(version)
	if len(idxs) > 0 {
		return version[idxs[0]:], err
	}
	return version, err
}
