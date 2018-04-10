package reports

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/jenkins-x/jx/pkg/util"
)

type ProjectHistory struct {
	LastReportDate string           `json:"lastReportDate,omitempty"`
	Reports        []*ProjectReport `json:"reports,omitempty"`
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
	var previous *ProjectReport
	for _, report := range h.Reports {
		if report.ReportDate == reportDate {
			return previous
		} else {
			previous = report
		}
	}
	return previous
}

func (h *ProjectHistory) UpdateDownloadCount(reportDate string, downloadCount int) *ProjectReport {
	report := h.GetOrCreateReport(reportDate)
	report.DownloadTotal += downloadCount

	previousCount := 0
	previous := h.FindPreviousReport(reportDate)
	if previous != nil {
		previousCount = previous.DownloadTotal
	}
	count := downloadCount - previousCount
	if count < 0 {
		count = 0
	}
	report.DownloadCount = count
	return report
}

type ProjectReport struct {
	ReportDate        string `json:"reportDate,omitempty"`
	DownloadCount     int    `json:"downloadCount,omitempty"`
	DownloadTotal     int    `json:"downloadTotal,omitempty"`
	IssueCount        int    `json:"issueCount,omitempty"`
	PullRequestCount  int    `json:"pullRequestCount,omitempty"`
	CommitCount       int    `json:"commitCount,omitempty"`
	NewCommitterCount int    `json:"newCommitterCount,omitempty"`
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
			err = json.Unmarshal(data, history)
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
	data, err := json.Marshal(s.history)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err == nil {
		fmt.Printf("Wrote Project History file %s\n", util.ColorInfo(fileName))
	}
	return err
}
