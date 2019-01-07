package cmd

import (
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/chats"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
)

func (o *CommonOptions) createChatProvider(chatConfig *config.ChatConfig) (chats.ChatProvider, error) {
	u := chatConfig.URL
	if u == "" {
		return nil, nil
	}
	authConfigSvc, err := o.createChatAuthConfigService()
	if err != nil {
		return nil, err
	}
	config := authConfigSvc.Config()

	server := config.GetOrCreateServer(u)
	userAuth, err := config.PickServerUserAuth(server, "user to access the chat service at "+u, o.BatchMode, "", o.In, o.Out, o.Err)
	if err != nil {
		return nil, err
	}
	return chats.CreateChatProvider(server.Kind, server, userAuth, o.BatchMode)
}

func (o *CommonOptions) createChatAuthConfigService() (auth.ConfigService, error) {
	secrets, err := o.LoadPipelineSecrets(kube.ValueKindChat, "")
	if err != nil {
		log.Warnf("The current user cannot query pipeline chat secrets: %s", err)
	}
	return o.CreateChatAuthConfigService(secrets)
}
