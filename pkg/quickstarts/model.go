package quickstarts

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/AlecAivazis/survey.v1"
)

const (
	JenkinsXQuickstartsOwner = "jenkins-x-quickstarts"
)

// GitHubQuickstart returns a github based quickstart
func GitHubQuickstart(owner string, repo string, language string, framework string, tags ...string) *Quickstart {
	u := "https://github.com/" + owner + "/" + repo + "/archive/master.zip"

	return &Quickstart{
		ID:             owner + "/" + repo,
		Owner:          owner,
		Name:           repo,
		Language:       language,
		Framework:      framework,
		Tags:           tags,
		DownloadZipURL: u,
	}
}

func (q *Quickstart) SurveyName() string {
	if q.Owner == JenkinsXQuickstartsOwner {
		return q.Name
	}
	// TODO maybe make a nicer string?
	return q.ID
}

// Add adds the given quickstart to this mode. Returns true if it was added
func (m *QuickstartModel) Add(q *Quickstart) bool {
	if q != nil {
		id := q.ID
		if id != "" {
			m.Quickstarts[id] = q
			return true
		}
	}
	return false
}

func (f *QuickstartFilter) Matches(q *Quickstart) bool {
	text := f.Text
	if text != "" && !strings.Contains(q.ID, text) {
		return false
	}
	owner := strings.ToLower(f.Owner)
	if owner != "" && strings.ToLower(q.Owner) != owner {
		return false
	}
	language := strings.ToLower(f.Language)
	if owner != "" && strings.ToLower(q.Language) != language {
		return false
	}
	return true
}

// CreateSurvey creates a survey to query pick a quickstart
func (model *QuickstartModel) CreateSurvey(filter *QuickstartFilter) (*Quickstart, error) {
	quickstarts := model.Filter(filter)
	names := []string{}
	m := map[string]*Quickstart{}
	for _, q := range quickstarts {
		name := q.SurveyName()
		m[name] = q
		names = append(names, name)
	}
	sort.Strings(names)

	if len(names) == 0 {
		return nil, fmt.Errorf("No quickstarts match filter")
	}
	answer := ""
	if len(names) == 1 {
		answer = names[0]
	} else {
		prompt := &survey.Select{
			Message: "select the quickstart you wish to create",
			Options: names,
		}
		err := survey.AskOne(prompt, &answer, survey.Required)
		if err != nil {
			return nil, err
		}
	}
	return m[answer], nil
}

func (model *QuickstartModel) Filter(filter *QuickstartFilter) []*Quickstart {
	answer := []*Quickstart{}
	for _, q := range model.Quickstarts {
		if filter.Matches(q) {
			answer = append(answer, q)
		}
	}
	return answer
}
