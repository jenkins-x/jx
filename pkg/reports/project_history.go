package reports

import (
	"fmt"
	"io/ioutil"

	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/yaml.v2"
)

type ProjectHistory struct {
	LastReportDate string           `json:"lastReportDate,omitempty"`
	Reports        []*ProjectReport `json:"reports,omitempty"`
	Contributors   []string         `json:"contributors,omitempty"`
	Committers     []string         `json:"committers,omitempty"`
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
	updateMetrics(&report.DownloadMetrics, &previous.DownloadMetrics, total)
	return report
}

func (h *ProjectHistory) IssueMetrics(reportDate string, total int) *ProjectReport {
	report := h.GetOrCreateReport(reportDate)
	previous := h.FindPreviousReport(reportDate)
	updateMetrics(&report.IssueMetrics, &previous.IssueMetrics, total)
	return report
}

func (h *ProjectHistory) PullRequestMetrics(reportDate string, total int) *ProjectReport {
	report := h.GetOrCreateReport(reportDate)
	previous := h.FindPreviousReport(reportDate)
	updateMetrics(&report.PullRequestMetrics, &previous.PullRequestMetrics, total)
	return report
}

func (h *ProjectHistory) CommitMetrics(reportDate string, total int) *ProjectReport {
	report := h.GetOrCreateReport(reportDate)
	previous := h.FindPreviousReport(reportDate)
	updateMetrics(&report.CommitMetrics, &previous.CommitMetrics, total)
	return report
}

func (h *ProjectHistory) NewCommitterMetrics(reportDate string, total int) *ProjectReport {
	report := h.GetOrCreateReport(reportDate)
	previous := h.FindPreviousReport(reportDate)
	updateMetrics(&report.NewCommitterMetrics, &previous.NewCommitterMetrics, total)
	return report
}

func (h *ProjectHistory) NewContributorMetrics(reportDate string, total int) *ProjectReport {
	report := h.GetOrCreateReport(reportDate)
	previous := h.FindPreviousReport(reportDate)
	updateMetrics(&report.NewContributorMetrics, &previous.NewContributorMetrics, total)
	return report
}

func (h *ProjectHistory) StarsMetrics(reportDate string, total int) *ProjectReport {
	report := h.GetOrCreateReport(reportDate)
	previous := h.FindPreviousReport(reportDate)
	updateMetrics(&report.StarsMetrics, &previous.StarsMetrics, total)
	return report
}

func updateMetrics(current *CountMetrics, previous *CountMetrics, total int) {
	current.Total += total
	previousTotal := 0
	if previous != nil {
		previousTotal = previous.Total
	}
	count := total - previousTotal
	current.Count = count
}

type CountMetrics struct {
	Count int `json:"count,omitempty"`
	Total int `json:"total,omitempty"`
}

type ProjectReport struct {
	ReportDate            string       `json:"reportDate,omitempty"`
	StarsMetrics          CountMetrics `json:"starsMetrics,omitempty"`
	DownloadMetrics       CountMetrics `json:"downloadMetrics,omitempty"`
	IssueMetrics          CountMetrics `json:"issueMetrics,omitempty"`
	PullRequestMetrics    CountMetrics `json:"pullRequestMetrics,omitempty"`
	CommitMetrics         CountMetrics `json:"commitMetrics,omitempty"`
	NewCommitterMetrics   CountMetrics `json:"newCommitterMetrics,omitempty"`
	NewContributorMetrics CountMetrics `json:"newContributorMetrics,omitempty"`
}

type ProjectHistoryService struct {
	FileName string
	history  *ProjectHistory
}

func NewProjectHistoryService(fileName string) (*ProjectHistoryService, *ProjectHistory, error) {
	svc := &ProjectHistoryService{
		FileName: fileName,
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
