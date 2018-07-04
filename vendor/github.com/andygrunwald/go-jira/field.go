package jira

// FieldService handles fields for the JIRA instance / API.
//
// JIRA API docs: https://developer.atlassian.com/cloud/jira/platform/rest/#api-Field
type FieldService struct {
	client *Client
}

// Field represents a field of a JIRA issue.
type Field struct {
	ID          string      `json:"id,omitempty" structs:"id,omitempty"`
	Key         string      `json:"key,omitempty" structs:"key,omitempty"`
	Name        string      `json:"name,omitempty" structs:"name,omitempty"`
	Custom      bool        `json:"custom,omitempty" structs:"custom,omitempty"`
	Navigable   bool        `json:"navigable,omitempty" structs:"navigable,omitempty"`
	Searchable  bool        `json:"searchable,omitempty" structs:"searchable,omitempty"`
	ClauseNames []string    `json:"clauseNames,omitempty" structs:"clauseNames,omitempty"`
	Schema      FieldSchema `json:"schema,omitempty" structs:"schema,omitempty"`
}

type FieldSchema struct {
	Type   string `json:"type,omitempty" structs:"type,omitempty"`
	System string `json:"system,omitempty" structs:"system,omitempty"`
}

// GetList gets all fields from JIRA
//
// JIRA API docs: https://developer.atlassian.com/cloud/jira/platform/rest/#api-api-2-field-get
func (s *FieldService) GetList() ([]Field, *Response, error) {
	apiEndpoint := "rest/api/2/field"
	req, err := s.client.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, nil, err
	}

	fieldList := []Field{}
	resp, err := s.client.Do(req, &fieldList)
	if err != nil {
		return nil, resp, NewJiraError(resp, err)
	}
	return fieldList, resp, nil
}
