package cmd

import (
	"io"
	"fmt"
	"time"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"gopkg.in/yaml.v2"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	chgit "github.com/jenkins-x/chyle/chyle/git"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StepChangelogOptions contains the command line flags
type StepChangelogOptions struct {
	StepOptions

	PreviousRevision string
	CurrentRevision  string
}

var (
	StepChangelogLong = templates.LongDesc(`
		Generates a Changelog for the last tag

`)

	StepChangelogExample = templates.Examples(`
		jx step changelog

`)

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

	dir := "."
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

	o.Printf("Generating change log from git ref %s => %s\n", util.ColorInfo(previousRev), util.ColorInfo(currentRev))

	commits, err := chgit.FetchCommits(dir, previousRev, currentRev)
	if err != nil {
		return err
	}
	commitSummaries := []v1.CommitSummary{}

	if commits != nil {
		for _, commit := range *commits {
			o.Printf("%#v\n", commit)
			commitSummaries = append(commitSummaries, o.toCommitSummary(&commit))
		}
	}

	// TODO generate release name
	releaseName := ""
	release := &v1.Release{
		ObjectMeta: metav1.ObjectMeta{
			Name: releaseName,
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
			ResourceVersion:   "1",
			DeletionTimestamp: &metav1.Time{},
		},
		Spec: v1.ReleaseSpec{
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
	o.Printf("%s\n", string(data))
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
