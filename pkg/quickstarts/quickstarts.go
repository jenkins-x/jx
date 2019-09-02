package quickstarts

import (
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/versionstream"
)

func (q *Quickstart) SurveyName() string {
	if q.Owner == JenkinsXQuickstartsOwner {
		return q.Name
	}
	// TODO maybe make a nicer string?
	return q.ID
}

func (f *QuickstartFilter) Matches(q *Quickstart) bool {
	if strings.Contains(q.ID, "WIP-") {
		return false
	}
	text := f.Text
	if text != "" && !strings.Contains(q.ID, text) {
		return false
	}
	owner := strings.ToLower(f.Owner)
	if owner != "" && strings.ToLower(q.Owner) != owner {
		return false
	}
	language := strings.ToLower(f.Language)
	if language != "" && strings.ToLower(q.Language) != language {
		return false
	}
	framework := strings.ToLower(f.Framework)
	if framework != "" && strings.ToLower(q.Framework) != framework {
		return false
	}
	if !f.AllowML && util.StartsWith(q.Name, "ML-") {
		return false
	}
	return true
}

func (model *QuickstartModel) convertToQuickStart(from *versionstream.QuickStart, to *Quickstart) error {
	s := func(text string, override string) string {
		if override != "" {
			return override
		}
		return text
	}
	ss := func(texts []string, overrides []string) []string {
		answer := append([]string{}, texts...)
		for _, o := range overrides {
			if util.StringArrayIndex(answer, o) < 0 {
				answer = append(answer, o)
			}
		}
		return answer
	}

	to.ID = s(to.ID, from.ID)
	to.Owner = s(to.Owner, from.Owner)
	to.Name = s(to.Name, from.Name)
	to.DownloadZipURL = s(to.DownloadZipURL, from.DownloadZipURL)
	to.Framework = s(to.Framework, from.Framework)
	to.Language = s(to.Language, from.Language)
	to.Tags = ss(to.Tags, from.Tags)
	return nil
}
