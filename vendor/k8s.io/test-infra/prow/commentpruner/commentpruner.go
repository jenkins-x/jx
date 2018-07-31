/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package commentpruner facilitates efficiently deleting bot comments as a reaction to webhook events.
package commentpruner

import (
	"sync"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/github"
)

type githubClient interface {
	BotName() (string, error)
	ListIssueComments(org, repo string, number int) ([]github.IssueComment, error)
	DeleteComment(org, repo string, id int) error
}

// EventClient is a struct that provides bot comment deletion for an event related to an issue.
// A single client instance should be created for each event and shared by all consumers of the event.
// The client fetches the comments only once and filters that list repeatedly to find bot comments to
// delete. This avoids using lots of API tokens when fetching comments for each handler that wants
// to delete comments. (An HTTP cache only partially helps with this because deletions modify the
// list of comments so the next call requires GH to send the resource again.)
type EventClient struct {
	org    string
	repo   string
	number int

	ghc githubClient
	log *logrus.Entry

	once     sync.Once
	lock     sync.Mutex
	comments []github.IssueComment
}

// NewEventClient creates an EventClient struct. This should be used once per webhook event.
func NewEventClient(ghc githubClient, log *logrus.Entry, org, repo string, number int) *EventClient {
	return &EventClient{
		org:    org,
		repo:   repo,
		number: number,

		ghc: ghc,
		log: log,
	}
}

// PruneComments fetches issue comments if they have not yet been fetched for this webhook event
// and then deletes any bot comments indicated by the func 'shouldPrune'.
func (c *EventClient) PruneComments(shouldPrune func(github.IssueComment) bool) {
	c.once.Do(func() {
		botName, err := c.ghc.BotName()
		if err != nil {
			c.log.WithError(err).Error("failed to get the bot's name. Pruning will consider all comments.")
		}
		comments, err := c.ghc.ListIssueComments(c.org, c.repo, c.number)
		if err != nil {
			c.log.WithError(err).Errorf("failed to list comments for %s/%s#%d", c.org, c.repo, c.number)
		}
		if botName != "" {
			for _, comment := range comments {
				if comment.User.Login == botName {
					c.comments = append(c.comments, comment)
				}
			}
		}
	})

	c.lock.Lock()
	defer c.lock.Unlock()

	var remaining []github.IssueComment
	for _, comment := range c.comments {
		removed := false
		if shouldPrune(comment) {
			if err := c.ghc.DeleteComment(c.org, c.repo, comment.ID); err != nil {
				c.log.WithError(err).Errorf("failed to delete stale comment with ID '%d'", comment.ID)
			} else {
				removed = true
			}
		}
		if !removed {
			remaining = append(remaining, comment)
		}
	}
	c.comments = remaining
}
