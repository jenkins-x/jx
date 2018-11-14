package auth

const (
	DefaultWritePermissions = 0760
)

type AuthServer struct {
	URL   string
	Users []*UserAuth
	Name  string
	Kind  string

	CurrentUser string
}

type UserAuth struct {
	Username    string
	ApiToken    string
	BearerToken string
	Password    string `yaml:"password,omitempty"`
}

type AuthConfig struct {
	Servers []*AuthServer

	DefaultUsername  string
	CurrentServer    string
	PipeLineUsername string
	PipeLineServer   string
}

// FileBasedAuthConfigService is a service for handing the config of auth tokens
type FileBasedAuthConfigService struct {
	FileName string
	config   *AuthConfig
}
