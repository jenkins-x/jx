package cmd

import (
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/chats"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/kube"
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

func (o *CommonOptions) CreateChatAuthConfigService() (auth.AuthConfigService, error) {
	secrets, err := o.Factory.LoadPipelineSecrets(kube.ValueKindChat, "")
	if err != nil {
		o.warnf("The current user cannot query pipeline chat secrets: %s", err)
	}
	return o.Factory.CreateChatAuthConfigService(secrets)
}
