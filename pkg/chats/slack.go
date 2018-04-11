package chats

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/nlopes/slack"
)

type SlackChatProvider struct {
	SlackClient *slack.Client
	Server      *auth.AuthServer
	UserAuth    *auth.UserAuth
}

func CreateSlackChatProvider(server *auth.AuthServer, userAuth *auth.UserAuth, batchMode bool) (ChatProvider, error) {
	u := server.URL
	if u == "" {
		return nil, fmt.Errorf("No base URL for server!")
	}
	if userAuth == nil || userAuth.IsInvalid() || userAuth.ApiToken == "" {
		return nil, fmt.Errorf("No authentication found for Slack server %s", u)
	}
	slackClient := slack.New(userAuth.ApiToken)

	return &SlackChatProvider{
		SlackClient: slackClient,
		Server:      server,
		UserAuth:    userAuth,
	}, nil
}

func (c *SlackChatProvider) GetChannelMetrics(name string) (*ChannelMetrics, error) {
	metrics := &ChannelMetrics{
		Name: name,
	}
	ch, err := c.SlackClient.GetChannelInfo(name)
	if err != nil {
		return metrics, err
	}
	metrics.MemberCount = ch.NumMembers
	metrics.URL = util.UrlJoin(c.Server.URL, "messages", ch.ID)
	return metrics, nil
}
