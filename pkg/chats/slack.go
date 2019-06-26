package chats

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/nlopes/slack"
)

type SlackChatProvider struct {
	slackClient *slack.Client
	server      auth.Server
}

func CreateSlackChatProvider(server auth.Server, batchMode bool) (ChatProvider, error) {
	u := server.URL
	if u == "" {
		return nil, fmt.Errorf("No base URL for server!")
	}
	user, err := server.GetCurrentUser()
	if err != nil {
		return nil, err
	}
	if user.IsInvalid() {
		return nil, fmt.Errorf("no authentication found for Slack server %s", u)
	}
	slackClient := slack.New(user.ApiToken)

	return &SlackChatProvider{
		slackClient: slackClient,
		server:      server,
	}, nil
}

func (c *SlackChatProvider) GetChannelMetrics(name string) (*ChannelMetrics, error) {
	metrics := &ChannelMetrics{
		Name: name,
	}
	name = strings.TrimPrefix(name, "#")
	id := name

	channels, err := c.slackClient.GetChannels(true)
	if err != nil {
		return metrics, err
	}
	for _, ch := range channels {
		log.Logger().Infof("Found channel %s with id %s", ch.Name, ch.ID)
		if ch.Name == name {
			id = ch.ID
			break
		}
	}
	info, err := c.slackClient.GetChannelInfo(id)
	if err != nil {
		return metrics, err
	}
	metrics.MemberCount = info.NumMembers
	metrics.ID = info.ID
	metrics.Name = info.Name
	metrics.Members = info.Members
	metrics.URL = util.UrlJoin(c.server.URL, "messages", info.ID)
	return metrics, nil
}
