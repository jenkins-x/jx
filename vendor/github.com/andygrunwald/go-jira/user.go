package jira

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// UserService handles users for the JIRA instance / API.
//
// JIRA API docs: https://docs.atlassian.com/jira/REST/cloud/#api/2/user
type UserService struct {
	client *Client
}

// User represents a JIRA user.
type User struct {
	Self            string     `json:"self,omitempty" structs:"self,omitempty"`
	Name            string     `json:"name,omitempty" structs:"name,omitempty"`
	Password        string     `json:"-"`
	Key             string     `json:"key,omitempty" structs:"key,omitempty"`
	EmailAddress    string     `json:"emailAddress,omitempty" structs:"emailAddress,omitempty"`
	AvatarUrls      AvatarUrls `json:"avatarUrls,omitempty" structs:"avatarUrls,omitempty"`
	DisplayName     string     `json:"displayName,omitempty" structs:"displayName,omitempty"`
	Active          bool       `json:"active,omitempty" structs:"active,omitempty"`
	TimeZone        string     `json:"timeZone,omitempty" structs:"timeZone,omitempty"`
	ApplicationKeys []string   `json:"applicationKeys,omitempty" structs:"applicationKeys,omitempty"`
}

// UserGroup represents the group list
type UserGroup struct {
	Self string `json:"self,omitempty" structs:"self,omitempty"`
	Name string `json:"name,omitempty" structs:"name,omitempty"`
}

// Get gets user info from JIRA
//
// JIRA API docs: https://docs.atlassian.com/jira/REST/cloud/#api/2/user-getUser
func (s *UserService) Get(username string) (*User, *Response, error) {
	apiEndpoint := fmt.Sprintf("/rest/api/2/user?username=%s", username)
	req, err := s.client.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, nil, err
	}

	user := new(User)
	resp, err := s.client.Do(req, user)
	if err != nil {
		return nil, resp, NewJiraError(resp, err)
	}
	return user, resp, nil
}

// Create creates an user in JIRA.
//
// JIRA API docs: https://docs.atlassian.com/jira/REST/cloud/#api/2/user-createUser
func (s *UserService) Create(user *User) (*User, *Response, error) {
	apiEndpoint := "/rest/api/2/user"
	req, err := s.client.NewRequest("POST", apiEndpoint, user)
	if err != nil {
		return nil, nil, err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil {
		return nil, resp, err
	}

	responseUser := new(User)
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		e := fmt.Errorf("Could not read the returned data")
		return nil, resp, NewJiraError(resp, e)
	}
	err = json.Unmarshal(data, responseUser)
	if err != nil {
		e := fmt.Errorf("Could not unmarshall the data into struct")
		return nil, resp, NewJiraError(resp, e)
	}
	return responseUser, resp, nil
}

// GetGroups returns the groups which the user belongs to
//
// JIRA API docs: https://docs.atlassian.com/jira/REST/cloud/#api/2/user-getUserGroups
func (s *UserService) GetGroups(username string) (*[]UserGroup, *Response, error) {
	apiEndpoint := fmt.Sprintf("/rest/api/2/user/groups?username=%s", username)
	req, err := s.client.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, nil, err
	}

	userGroups := new([]UserGroup)
	resp, err := s.client.Do(req, userGroups)
	if err != nil {
		return nil, resp, NewJiraError(resp, err)
	}
	return userGroups, resp, nil
}

// Get information about the current logged-in user
//
// JIRA API docs: https://developer.atlassian.com/cloud/jira/platform/rest/#api-api-2-myself-get
func (s *UserService) GetSelf() (*User, *Response, error) {
	const apiEndpoint = "rest/api/2/myself"
	req, err := s.client.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, nil, err
	}
	var user User
	resp, err := s.client.Do(req, &user)
	if err != nil {
		return nil, resp, NewJiraError(resp, err)
	}
	return &user, resp, nil
}

// Find searches for user info from JIRA:
// It can find users by email, username or name
//
// JIRA API docs: https://docs.atlassian.com/jira/REST/cloud/#api/2/user-findUsers
func (s *UserService) Find(property string) ([]User, *Response, error) {
	apiEndpoint := fmt.Sprintf("/rest/api/2/user/search?username=%s", property)
	req, err := s.client.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, nil, err
	}

	users := []User{}
	resp, err := s.client.Do(req, &users)
	if err != nil {
		return nil, resp, NewJiraError(resp, err)
	}
	return users, resp, nil
}
