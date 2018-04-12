package reports

import (
	"fmt"
	"io/ioutil"

	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/yaml.v2"
)

type ProjectHistory struct {
	LastReportDate string           `yaml:"lastReportDate,omitempty"`
	Reports        []*ProjectReport `yaml:"reports,omitempty"`
	Contributors   []string         `yaml:"contributors,omitempty"`
	Committers     []string         `yaml:"committers,omitempty"`
}

type CountMetrics struct {
	Count int `yaml:"count,omitempty"`
	Total int `yaml:"total,omitempty"`
}

type ProjectReport struct {
	ReportDate            string       `yaml:"reportDate,omitempty"`
	StarsMetrics          CountMetrics `yaml:"starsMetrics,omitempty"`
	DownloadMetrics       CountMetrics `yaml:"downloadMetrics,omitempty"`
	IssueMetrics          CountMetrics `yaml:"issueMetrics,omitempty"`
	PullRequestMetrics    CountMetrics `yaml:"pullRequestMetrics,omitempty"`
	CommitMetrics         CountMetrics `yaml:"commitMetrics,omitempty"`
	NewCommitterMetrics   CountMetrics `yaml:"newCommitterMetrics,omitempty"`
	NewContributorMetrics CountMetrics `yaml:"newContributorMetrics,omitempty"`
	DeveloperChatMetrics  CountMetrics `yaml:"developerChatMetrics,omitempty"`
	UserChatMetrics       CountMetrics `yaml:"userChatMetrics,omitempty"`
}

func (h *ProjectHistory) GetOrCreateReport(reportDate string) *ProjectReport {
	report := h.FindReport(reportDate)
	if report == nil {
		report = &ProjectReport{
			ReportDate: reportDate,
		}
		h.Reports = append(h.Reports, report)
		h.LastReportDate = reportDate
	}
	return report
}

func (h *ProjectHistory) FindReport(reportDate string) *ProjectReport {
	for _, report := range h.Reports {
		if report.ReportDate == reportDate {
			return report
		}
	}
	return nil
}

func (h *ProjectHistory) FindPreviousReport(reportDate string) *ProjectReport {
	// TODO we should really do a date comparison but for noe we do posts in order anyway
	var previous = &ProjectReport{}
	for _, report := range h.Reports {
		if report.ReportDate == reportDate {
			return previous
		} else {
			previous = report
		}
	}
	return previous
}

func (h *ProjectHistory) DownloadMetrics(reportDate string, total int) *ProjectReport {
	report := h.GetOrCreateReport(reportDate)
	previous := h.FindPreviousReport(reportDate)
	updateMetricTotal(&report.DownloadMetrics, &previous.DownloadMetrics, total)
	return report
}

func (h *ProjectHistory) IssueMetrics(reportDate string, total int) *ProjectReport {
	report := h.GetOrCreateReport(reportDate)
	previous := h.FindPreviousReport(reportDate)
	addMetricCount(&report.IssueMetrics, &previous.IssueMetrics, total)
	return report
}

func (h *ProjectHistory) PullRequestMetrics(reportDate string, total int) *ProjectReport {
	report := h.GetOrCreateReport(reportDate)
	previous := h.FindPreviousReport(reportDate)
	addMetricCount(&report.PullRequestMetrics, &previous.PullRequestMetrics, total)
	return report
}

func (h *ProjectHistory) CommitMetrics(reportDate string, total int) *ProjectReport {
	report := h.GetOrCreateReport(reportDate)
	previous := h.FindPreviousReport(reportDate)
	addMetricCount(&report.CommitMetrics, &previous.CommitMetrics, total)
	return report
}

func (h *ProjectHistory) NewCommitterMetrics(reportDate string, total int) *ProjectReport {
	report := h.GetOrCreateReport(reportDate)
	previous := h.FindPreviousReport(reportDate)
	addMetricCount(&report.NewCommitterMetrics, &previous.NewCommitterMetrics, total)
	return report
}

func (h *ProjectHistory) NewContributorMetrics(reportDate string, total int) *ProjectReport {
	report := h.GetOrCreateReport(reportDate)
	previous := h.FindPreviousReport(reportDate)
	addMetricCount(&report.NewContributorMetrics, &previous.NewContributorMetrics, total)
	return report
}

func (h *ProjectHistory) StarsMetrics(reportDate string, total int) *ProjectReport {
	report := h.GetOrCreateReport(reportDate)
	previous := h.FindPreviousReport(reportDate)
	updateMetricTotal(&report.StarsMetrics, &previous.StarsMetrics, total)
	return report
}

func (h *ProjectHistory) DeveloperChatMetrics(reportDate string, total int) *ProjectReport {
	report := h.GetOrCreateReport(reportDate)
	previous := h.FindPreviousReport(reportDate)
	updateMetricTotal(&report.DeveloperChatMetrics, &previous.DeveloperChatMetrics, total)
	return report
}

func (h *ProjectHistory) UserChatMetrics(reportDate string, total int) *ProjectReport {
	report := h.GetOrCreateReport(reportDate)
	previous := h.FindPreviousReport(reportDate)
	updateMetricTotal(&report.UserChatMetrics, &previous.UserChatMetrics, total)
	return report
}

// addMetricCount adds a new metric value, such as number of commits in a release
func addMetricCount(current *CountMetrics, previous *CountMetrics, total int) {
	current.Count = total
	previousTotal := 0
	if previous != nil {
		previousTotal = previous.Total
	}
	current.Total = total + previousTotal
}

// updateMetricTotal takes the current total value and works out the incremental change
// since the last report. e.g. for updating the new number of stars or users in a chat room
func updateMetricTotal(current *CountMetrics, previous *CountMetrics, total int) {
	current.Total = total
	previousTotal := 0
	if previous != nil {
		previousTotal = previous.Total
	}
	count := total - previousTotal
	current.Count = count
}

type ProjectHistoryService struct {
	FileName string
	history  *ProjectHistory
}

func NewProjectHistoryService(fileName string) (*ProjectHistoryService, *ProjectHistory, error) {
	svc := &ProjectHistoryService{
		FileName: fileName,
		history:  &ProjectHistory{},
	}
	history, err := svc.LoadHistory()
	return svc, history, err
}

// LoadHistory loads the project history from disk if it exists
func (s *ProjectHistoryService) LoadHistory() (*ProjectHistory, error) {
	history := s.History()
	fileName := s.FileName
	if fileName != "" {
		exists, err := util.FileExists(fileName)
		if err != nil {
			return history, fmt.Errorf("Could not check if file exists %s due to %s", fileName, err)
		}
		if exists {
			data, err := ioutil.ReadFile(fileName)
			if err != nil {
				return history, fmt.Errorf("Failed to load file %s due to %s", fileName, err)
			}
			err = yaml.Unmarshal(data, history)
			if err != nil {
				return history, fmt.Errorf("Failed to unmarshal YAML file %s due to %s", fileName, err)
			}
		}
	}
	return history, nil
}

func (s *ProjectHistoryService) History() *ProjectHistory {
	if s.history == nil {
		s.history = &ProjectHistory{}
	}
	return s.history
}

// SaveHistory saves the history to disk
func (s *ProjectHistoryService) SaveHistory() error {
	fileName := s.FileName
	if fileName == "" {
		return fmt.Errorf("No filename defined!")
	}
	data, err := yaml.Marshal(s.history)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err == nil {
		fmt.Printf("Wrote Project History file %s\n", util.ColorInfo(fileName))
	}
	return err
}
