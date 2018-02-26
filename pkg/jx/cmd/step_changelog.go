package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	/*
		"k8s.io/apimachinery/pkg/util/yaml"
	*/
	"github.com/ghodss/yaml"
	chgit "github.com/jenkins-x/chyle/chyle/git"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StepChangelogOptions contains the command line flags
type StepChangelogOptions struct {
	StepOptions

	PreviousRevision string
	CurrentRevision  string
	TemplatesDir     string
	ReleaseYamlFile  string
	CrdYamlFile      string
	Dir      string
	OverwriteCRD     bool
	GenerateCRD      bool
}

const (
	ReleaseName = `{{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}`

	SpecName    = `{{ .Chart.Name }}`
	SpecVersion = `{{ .Chart.Version }}`

	ReleaseCrdYaml = `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: 2018-02-24T14:56:33Z
  name: releases.jenkins.io
  resourceVersion: "557150"
  selfLink: /apis/apiextensions.k8s.io/v1beta1/customresourcedefinitions/releases.jenkins.io
  uid: e77f4e08-1972-11e8-988e-42010a8401df
spec:
  group: jenkins.io
  names:
    kind: Release
    listKind: ReleaseList
    plural: releases
    shortNames:
    - rel
    singular: release
  scope: Namespaced
  version: v1`
)

var (
	StepChangelogLong = templates.LongDesc(`
		Generates a Changelog for the last tag

`)

	StepChangelogExample = templates.Examples(`
		jx step changelog

`)
)

func NewCmdStepChangelog(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := StepChangelogOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "changelog",
		Short:   "Creates a changelog for a git tag",
		Aliases: []string{"changes"},
		Long:    StepChangelogLong,
		Example: StepChangelogExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.PreviousRevision, "previous-rev", "p", "", "the previous tag revision")
	cmd.Flags().StringVarP(&options.CurrentRevision, "rev", "r", "", "the current tag revision")
	cmd.Flags().StringVarP(&options.TemplatesDir, "templates-dir", "t", "", "the directory containing the helm chart templates to generate the resources")
	cmd.Flags().StringVarP(&options.ReleaseYamlFile, "release-yaml-file", "", "release.yaml", "the name of the file to generate the Release YAML")
	cmd.Flags().StringVarP(&options.CrdYamlFile, "crd-yaml-file", "", "release-crd.yaml", "the name of the file to generate the Release CustomResourceDefinition YAML")
	cmd.Flags().StringVarP(&options.Dir, "dir", "", "", "The directory of the git repository. Defaults to the current working directory")
	cmd.Flags().BoolVarP(&options.OverwriteCRD, "overwrite", "o", false, "overwrites the Release CRD YAML file if it exists")
	cmd.Flags().BoolVarP(&options.GenerateCRD, "crd", "c", false, "Generate the CRD in the chart")
	return cmd
}

func (o *StepChangelogOptions) Run() error {
	apisClient, err := o.Factory.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterPipelineActivityCRD(apisClient)
	if err != nil {
		return err
	}

	err = kube.RegisterReleaseCRD(apisClient)
	if err != nil {
		return err
	}

	dir := o.Dir
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	previousRev := o.PreviousRevision
	if previousRev == "" {
		previousRev, err = gits.GetPreviousGitTagSHA(dir)
		if err != nil {
			return err
		}
	}
	currentRev := o.CurrentRevision
	if currentRev == "" {
		currentRev, err = gits.GetCurrentGitTagSHA(dir)
		if err != nil {
			return err
		}
	}

	templatesDir := o.TemplatesDir
	if templatesDir == "" {
		chartFile, err := o.FindHelmChart()
		if err != nil {
			return fmt.Errorf("Could not find helm chart %s", err)
		}
		path, _ := filepath.Split(chartFile)
		templatesDir = filepath.Join(path, "templates")
	}
	err = os.MkdirAll(templatesDir, DefaultWritePermissions)
	if err != nil {
		return fmt.Errorf("Failed to create the templates directory %s due to %s", templatesDir, err)
	}

	o.Printf("Generating change log from git ref %s => %s\n", util.ColorInfo(previousRev), util.ColorInfo(currentRev))

	gitDir, _, err := gits.FindGitConfigDir(dir)
	if err != nil {
		return err
	}
	if gitDir == "" {
		o.warnf("No git directory could be found from dir %s\n", dir)
		return nil
	}

	commits, err := chgit.FetchCommits(gitDir, previousRev, currentRev)
	if err != nil {
		return err
	}
	commitSummaries := []v1.CommitSummary{}

	if commits != nil {
		for _, commit := range *commits {
			commitSummaries = append(commitSummaries, o.toCommitSummary(&commit))
		}
	}

	// TODO generate release name
	release := &v1.Release{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Release",
			APIVersion: jenkinsio.GroupAndVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ReleaseName,
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
			ResourceVersion:   "1",
			DeletionTimestamp: &metav1.Time{},
		},
		Spec: v1.ReleaseSpec{
			Name:    SpecName,
			Version: SpecVersion,
			Commits: commitSummaries,
		},
	}
	data, err := yaml.Marshal(release)
	if err != nil {
		return err
	}
	if data == nil {
		return fmt.Errorf("Could not marshal release to yaml")
	}
	releaseFile := filepath.Join(templatesDir, o.ReleaseYamlFile)
	crdFile := filepath.Join(templatesDir, o.CrdYamlFile)
	err = ioutil.WriteFile(releaseFile, data, DefaultWritePermissions)
	if err != nil {
		return fmt.Errorf("Failed to save Release YAML file %s: %s", releaseFile, err)
	}
	exists, err := util.FileExists(crdFile)
	if err != nil {
		return fmt.Errorf("Failed to check for CRD YAML file %s: %s", crdFile, err)
	}
	o.Printf("generated: %s\n", util.ColorInfo(releaseFile))
	if o.GenerateCRD && (o.OverwriteCRD || !exists) {
		err = ioutil.WriteFile(crdFile, []byte(ReleaseCrdYaml), DefaultWritePermissions)
		if err != nil {
			return fmt.Errorf("Failed to save Release CRD YAML file %s: %s", crdFile, err)
		}
		o.Printf("generated: %s\n", util.ColorInfo(crdFile))
	}
	return nil
}

func (o *StepChangelogOptions) toCommitSummary(commit *object.Commit) v1.CommitSummary {
	// TODO
	url := ""
	sha := commit.Hash.String()
	return v1.CommitSummary{
		Message:   commit.Message,
		URL:       url,
		SHA:       sha,
		Author:    o.toUserDetails(commit.Author),
		Committer: o.toUserDetails(commit.Committer),
	}
}
func (o *StepChangelogOptions) toUserDetails(signature object.Signature) *v1.UserDetails {
	// TODO
	login := ""
	return &v1.UserDetails{
		Login: login,
		Name:  signature.Name,
		Email: signature.Email,
		CreationTimestamp: &metav1.Time{
			Time: signature.When,
		},
	}
}
