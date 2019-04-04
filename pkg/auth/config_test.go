package auth_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/stretchr/testify/assert"
)

func TestValidUser(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		user *auth.User
		err  bool
	}{
		"valid user with api token": {
			user: &auth.User{
				Username: "test",
				ApiToken: "test",
			},
			err: false,
		},
		"valid user with password": {
			user: &auth.User{
				Username: "test",
				Password: "test",
			},
			err: false,
		},
		"valid user with bearer token": {
			user: &auth.User{
				BearerToken: "test",
			},
			err: false,
		},
		"invalid user": {
			user: &auth.User{},
			err:  true,
		},
		"invalid user only with username": {
			user: &auth.User{
				Username: "test",
			},
			err: true,
		},
		"invalid user only with api token": {
			user: &auth.User{
				ApiToken: "test",
			},
			err: true,
		},
		"invalid user only with password": {
			user: &auth.User{
				Password: "test",
			},
			err: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.user.Valid()
			if tc.err {
				assert.Error(t, err, "user should be invalid")
			} else {
				assert.NoError(t, err, "user should be valid")
			}
		})
	}
}

func TestValidServer(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		server *auth.Server
		err    bool
	}{
		"valid server": {
			server: &auth.Server{
				URL:  "https://tests",
				Name: "test",
				Users: []*auth.User{
					{
						Username: "test",
						ApiToken: "test",
					},
				},
			},
			err: false,
		},
		"invalid server without users": {
			server: &auth.Server{
				URL:   "https://tests",
				Name:  "test",
				Users: []*auth.User{},
			},
			err: true,
		},
		"invalid server without URL": {
			server: &auth.Server{
				URL:  "",
				Name: "test",
				Users: []*auth.User{
					{
						Username: "test",
						ApiToken: "test",
					},
				},
			},
			err: true,
		},
		"invalid server with invalid user": {
			server: &auth.Server{
				URL:  "https://tests",
				Name: "test",
				Users: []*auth.User{
					{
						Username: "test",
					},
				},
			},
			err: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.server.Valid()
			if tc.err {
				assert.Error(t, err, "user should be invalid")
			} else {
				assert.NoError(t, err, "user should be valid")
			}
		})
	}
}

func TestPipelineUser(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		server *auth.Server
		want   auth.User
	}{
		"pipeline user": {
			server: &auth.Server{
				URL:  "https://test",
				Name: "test",
				Users: []*auth.User{
					{
						Username: "test1",
						ApiToken: "test",
						Kind:     auth.UserKindPipeline,
					},
					{
						Username: "test2",
						ApiToken: "test",
						Kind:     auth.UserKindLocal,
					},
				},
			},
			want: auth.User{
				Username: "test1",
				ApiToken: "test",
				Kind:     auth.UserKindPipeline,
			},
		},
		"pipeline user when no user available": {
			server: &auth.Server{
				URL:   "https://test",
				Name:  "test",
				Users: []*auth.User{},
			},
			want: auth.User{},
		},
		"pipeline user when no pipeline user available": {
			server: &auth.Server{
				URL:  "https://test",
				Name: "test",
				Users: []*auth.User{
					{
						Username: "test1",
						ApiToken: "test",
						Kind:     auth.UserKindLocal,
					},
					{
						Username: "test2",
						ApiToken: "test",
						Kind:     auth.UserKindLocal,
					},
				},
			},
			want: auth.User{
				Username: "test1",
				ApiToken: "test",
				Kind:     auth.UserKindLocal,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.server.PipelineUser()
			assert.Equal(t, tc.want, got)
		})
	}
}
