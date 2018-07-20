package codeship

import "net/http"

// Authenticator is a strategy for authenticating with the API
type Authenticator interface {
	SetAuth(*http.Request)
}

// NewBasicAuth returns a new BasicAuth Authenticator
func NewBasicAuth(username, password string) *BasicAuth {
	return &BasicAuth{
		Username: username,
		Password: password,
	}
}

// BasicAuth is an Authenticator that implements basic auth
type BasicAuth struct {
	Username, Password string
}

// SetAuth implements Authenticator
func (a *BasicAuth) SetAuth(r *http.Request) {
	r.SetBasicAuth(a.Username, a.Password)
}
