package cmd

import (
	"github.com/jenkins-x/jx/pkg/chats"
	"github.com/jenkins-x/jx/pkg/config"
)

func (o *CommonOptions) createChatProvider(chatConfig *config.ChatConfig) (chats.ChatProvider, error) {
	u := chatConfig.URL
	if u == "" {
		return nil, nil
	}
	authConfigSvc, err := o.CreateChatAuthConfigService()
	if err != nil {
		return nil, err
	}
	config := authConfigSvc.Config()

	server := config.GetOrCreateServer(u)
	userAuth, err := config.PickServerUserAuth(server, "user to access the chat service at "+u, o.BatchMode)
	if err != nil {
		return nil, err
	}
	return chats.CreateChatProvider(server.Kind, server, userAuth, o.BatchMode)
}
