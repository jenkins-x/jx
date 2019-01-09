package util

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_getLatestReleaseFromGithubUsingHttpRedirect(t *testing.T) {
	t.Parallel()
	type args struct {
		githubOwner string
		githubRepo  string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"Happy Path",
			args{
				githubOwner: "org",
				githubRepo:  "repo",
			},
			"v1.2.3",
			false,
		},
		{
			"Redirection from one repo to another (redirects to happy path)",
			args{
				githubOwner: "redirect",
				githubRepo:  "repo",
			},
			"v1.2.3",
			false,
		},
		{
			"Not a redirect",
			args{
				githubOwner: "foo",
				githubRepo:  "bar",
			},
			"",
			true,
		},
		{
			"Redirect but no location header",
			args{
				githubOwner: "no",
				githubRepo:  "location",
			},
			"",
			true,
		},
		{
			"Bad Redirect - redirect to a page of an unknown format",
			args{
				githubOwner: "bad",
				githubRepo:  "location",
			},
			"",
			true,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Send response to be tested
		if req.URL.Path == "/org/repo/releases/latest" {
			rw.Header().Add("Location", "https://github.com/org/repo/releases/tag/v1.2.3")
			rw.WriteHeader(302)
		} else if req.URL.Path == "/redirect/repo/releases/latest" {
			// permanent Redirect to the happy path
			rw.Header().Add("Location", "/org/repo/releases/latest")
			rw.WriteHeader(301)
		} else if req.URL.Path == "/no/location/releases/latest" {
			// No location header in the response
			rw.WriteHeader(302)
		} else if req.URL.Path == "/bad/location/releases/latest" {
			rw.Header().Add("Location", "https://foo.bar")
			rw.WriteHeader(302)
		} else {
			_, _ = rw.Write([]byte(`This is not the redirect you are looking for`))
		}
	}))
	// Close the server when test finishes
	defer server.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getLatestReleaseFromHostUsingHttpRedirect(server.URL, tt.args.githubOwner, tt.args.githubRepo)
			if (err != nil) != tt.wantErr {
				t.Errorf("getLatestReleaseFromGithubUsingHttpRedirect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getLatestReleaseFromGithubUsingHttpRedirect() = %v, want %v", got, tt.want)
			}
		})
	}
}
