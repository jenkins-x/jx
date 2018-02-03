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
}

type AuthConfig struct {
	Servers []*AuthServer

	DefaultUsername string
	CurrentServer   string
}

type AuthConfigService struct {
	FileName string
	config   AuthConfig
}
