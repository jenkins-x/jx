package jira

// PriorityService handles priorities for the JIRA instance / API.
//
// JIRA API docs: https://developer.atlassian.com/cloud/jira/platform/rest/#api-Priority
type PriorityService struct {
	client *Client
}

// Priority represents a priority of a JIRA issue.
// Typical types are "Normal", "Moderate", "Urgent", ...
type Priority struct {
	Self        string `json:"self,omitempty" structs:"self,omitempty"`
	IconURL     string `json:"iconUrl,omitempty" structs:"iconUrl,omitempty"`
	Name        string `json:"name,omitempty" structs:"name,omitempty"`
	ID          string `json:"id,omitempty" structs:"id,omitempty"`
	StatusColor string `json:"statusColor,omitempty" structs:"statusColor,omitempty"`
	Description string `json:"description,omitempty" structs:"description,omitempty"`
}

// GetList gets all priorities from JIRA
//
// JIRA API docs: https://developer.atlassian.com/cloud/jira/platform/rest/#api-api-2-priority-get
func (s *PriorityService) GetList() ([]Priority, *Response, error) {
	apiEndpoint := "rest/api/2/priority"
	req, err := s.client.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, nil, err
	}

	priorityList := []Priority{}
	resp, err := s.client.Do(req, &priorityList)
	if err != nil {
		return nil, resp, NewJiraError(resp, err)
	}
	return priorityList, resp, nil
}
