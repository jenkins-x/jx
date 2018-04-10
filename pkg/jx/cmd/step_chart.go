package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/issues"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/reports"
	"github.com/spf13/cobra"
)

const ()

var (
	stepChartLong = templates.LongDesc(`
		Generates charts for a project
`)

	stepChartExample = templates.Examples(`
		# create charts for the cuect
		jx step chart

			`)
)

// StepChartOptions contains the command line flags
type StepChartOptions struct {
	StepOptions

	FromDate             string
	ToDate               string
	Dir                  string
	BlogOutputDir        string
	BlogName             string
	CombineMinorReleases bool

	State StepChartState
}

type StepChartState struct {
	GitInfo        *gits.GitRepositoryInfo
	GitProvider    gits.GitProvider
	Tracker        issues.IssueProvider
	Release        *v1.Release
	BlogFileName   string
	Buffer         *bytes.Buffer
	Writer         *bufio.Writer
	HistoryService *reports.ProjectHistoryService
	History        *reports.ProjectHistory
}

// NewCmdStepChart Creates a new Command object
func NewCmdStepChart(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &StepChartOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "chart",
		Short:   "Creates charts for project metrics",
		Long:    stepChartLong,
		Example: stepChartExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "The directory to query to find the projects .git directory")
	cmd.Flags().StringVarP(&options.FromDate, "from-date", "f", "", "The date to create the charts from. Defaults to a week before the to date")
	cmd.Flags().StringVarP(&options.ToDate, "to-date", "t", "", "The date to query to")
	cmd.Flags().StringVarP(&options.BlogOutputDir, "blog-dir", "", "", "The Hugo-style blog source code to generate the charts into")
	cmd.Flags().StringVarP(&options.BlogName, "blog-name", "n", "", "The blog name")
	cmd.Flags().BoolVarP(&options.CombineMinorReleases, "combine-minor", "c", true, "If enabled lets combine minor releases together to simplify the charts")
	return cmd
}

// Run implements this command
func (o *StepChartOptions) Run() error {
	o.State = StepChartState{}
	var err error
	outDir := o.BlogOutputDir
	if outDir != "" {
		if o.BlogName == "" {
			t := time.Now()
			o.BlogName = "changes-" + strconv.Itoa(t.Day()) + "-" + strings.ToLower(t.Month().String()) + "-" + strconv.Itoa(t.Year())
		}
		historyFile := filepath.Join(o.BlogOutputDir, "data", "projectHistory.json")
		o.State.HistoryService, o.State.History, err = reports.NewProjectHistoryService(historyFile)
		if err != nil {
			return err
		}

		err = o.generateChangelog()
		if err != nil {
			return err
		}
	} else {
		gitInfo, gitProvider, tracker, err := o.createGitProvider(o.Dir)
		if err != nil {
			return err
		}
		if gitInfo == nil {
			return fmt.Errorf("Could not find a .git folder in the current directory so could not determine the current project")
		}
		o.State.GitInfo = gitInfo
		o.State.GitProvider = gitProvider
		o.State.Tracker = tracker
	}

	err = o.downloadsReport(o.State.GitProvider, o.State.GitInfo.Organisation, o.State.GitInfo.Name)
	if err != nil {
		return err
	}
	return o.addReportsToBlog()
}

func (o *StepChartOptions) downloadsReport(provider gits.GitProvider, owner string, repo string) error {
	releases, err := provider.ListReleases(owner, repo)
	if err != nil {
		return err
	}
	if len(releases) == 0 {
		o.warnf("No releases found for %s/%s/n", owner, repo)
		return nil
	}
	if o.CombineMinorReleases {
		releases = o.combineMinorReleases(releases)
	}
	history := o.State.History
	if history != nil {
		downloadCount := gits.ReleaseDownloadCount(releases)
		history.UpdateDownloadCount(o.ToDate, downloadCount)
	}

	report := o.createBarReport("downloads", "Version", "Downloads")

	for _, release := range releases {
		report.AddNumber(release.Name, release.DownloadCount)
	}
	report.Render()
	return nil
}

// createBarReport creates the new report instance
func (o *StepChartOptions) createBarReport(name string, legends ...string) reports.BarReport {
	outDir := o.BlogOutputDir
	if outDir != "" {
		blogName := o.BlogName
		if blogName == "" {
			t := time.Now()
			blogName = fmt.Sprintf("changes-%d-%s-%d", t.Day(), t.Month().String(), t.Year())
		}

		jsFileName := filepath.Join(outDir, "static", "news", blogName, name+".js")
		jsLinkURI := filepath.Join("/news", blogName, name+".js")
		state := &o.State
		if state.Buffer == nil {
			var buffer bytes.Buffer
			state.Buffer = &buffer
		}
		if state.Writer == nil {
			state.Writer = bufio.NewWriter(state.Buffer)
		}
		state.Buffer.WriteString(`

## ` + strings.Title(name) + `

`)
		return reports.NewBlogBarReport(name, state.Writer, jsFileName, jsLinkURI)
	}
	return reports.NewTableBarReport(o.CreateTable(), legends...)
}

func (options *StepChartOptions) combineMinorReleases(releases []*gits.GitRelease) []*gits.GitRelease {
	answer := []*gits.GitRelease{}
	m := map[string]*gits.GitRelease{}
	for _, release := range releases {
		name := release.Name
		if name != "" {
			idx := strings.LastIndex(name, ".")
			if idx > 0 {
				name = name[0:idx] + ".x"
			}
		}
		cur := m[name]
		if cur == nil {
			copy := *release
			copy.Name = name
			m[name] = &copy
			answer = append(answer, &copy)
		} else {
			cur.DownloadCount += release.DownloadCount
		}
	}
	return answer
}

func (o *StepChartOptions) generateChangelog() error {
	blogFile := filepath.Join(o.BlogOutputDir, "content", "news", o.BlogName+".md")
	previousDate := o.FromDate
	now := time.Now()
	if previousDate == "" {
		// default to 4 weeks ago
		t := now.Add(-time.Hour * 24 * 7 * 4)
		previousDate = gits.FormatDate(t)
		o.FromDate = previousDate
	}
	if o.ToDate == "" {
		o.ToDate = gits.FormatDate(now)
	}
	options := &StepChangelogOptions{
		StepOptions: o.StepOptions,
		Dir:         o.Dir,
		Version:     "Changes since " + previousDate,
		// TODO add time now and previous time
		PreviousDate:       previousDate,
		OutputMarkdownFile: blogFile,
	}
	err := options.Run()
	if err != nil {
		return err
	}
	state := &o.State
	output := &options.State
	state.GitInfo = output.GitInfo
	state.GitProvider = output.GitProvider
	state.Tracker = output.Tracker
	state.BlogFileName = blogFile
	return nil
}

func (o *StepChartOptions) addReportsToBlog() error {
	state := &o.State
	if state.BlogFileName != "" {
		data, err := ioutil.ReadFile(state.BlogFileName)
		if err != nil {
			return err
		}
		toDate := o.ToDate
		fromDate := o.FromDate
		prefix := `---
title: "Changes for ` + toDate + `"
date: 2017-03-19T18:36:00+02:00
description: "Change log for changes from ` + fromDate + ` to ` + toDate + `"
categories: [blog]
keywords: []
slug: "changes-` + strings.Replace(toDate, " ", "-", -1) + `"
aliases: []
author: jenkins-x-bot
---

## Changes for ` + toDate + `

This blog outlines the changes on the project from ` + fromDate + ` to ` + toDate + `.

` + o.createMetricsSummary() + `

[see more metrics](#metrics)

`
		postfix := ""
		if state.Writer != nil {
			state.Writer.Flush()
			postfix = `

## Metrics

` + state.Buffer.String()

		}
		changelog := strings.TrimSpace(string(data))
		changelog = strings.TrimPrefix(changelog, "## Changes")
		text := prefix + changelog + postfix
		err = ioutil.WriteFile(state.BlogFileName, []byte(text), DefaultWritePermissions)
		if err != nil {
			return err
		}
	}
	historyService := state.HistoryService
	if historyService != nil {
		return historyService.SaveHistory()
	}
	return nil
}

func (o *StepChartOptions) createMetricsSummary() string {
	var buffer bytes.Buffer
	out := bufio.NewWriter(&buffer)
	history := o.State.History
	if history != nil {
		toDate := o.ToDate
		report := history.FindReport(toDate)
		if report != nil {
			fmt.Fprintf(out, "Recent Downloads: **%d** Total Downloads: **%d**", report.DownloadCount, report.DownloadCount)
		} else {
			o.warnf("No report for date %s\n", toDate)
		}
	}
	out.Flush()
	return buffer.String()
}
