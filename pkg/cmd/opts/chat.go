package opts

import (
	"github.com/jenkins-x/jx/pkg/chats"
	"github.com/jenkins-x/jx/pkg/config"
)

// CreateChatProvider creates a new chart provider from the given configuration
func (o *CommonOptions) CreateChatProvider(chatConfig *config.ChatConfig) (chats.ChatProvider, error) {
	url := chatConfig.URL
	if url == "" {
		return nil, nil
	}
	cs, err := o.CreateChatConfigService()
	if err != nil {
		return nil, err
	}
	cfg, err := cs.Config()
	if err != nil {
		return nil, err
	}
	server, err := cfg.GetServer(url)
	if err != nil {
		return nil, err
	}
	return chats.CreateChatProvider(server.Kind, server, o.BatchMode)
}
