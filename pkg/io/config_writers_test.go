package io_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/io"
	"github.com/stretchr/testify/assert"
)

func TestFileConfigWriter(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config auth.Config
		err    bool
	}{
		"write config to file": {
			config: auth.Config{
				Servers: []*auth.Server{
					&auth.Server{
						URL: "https://github.com",
						Users: []*auth.User{
							&auth.User{
								Username: "test",
								ApiToken: "test",
							},
						},
						Name: "GitHub",
						Kind: "github",
					},
				},
			},
			err: false,
		},
		"write empty config to file": {
			config: auth.Config{
				Servers: []*auth.Server{},
			},
			err: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			file, err := ioutil.TempFile("", "test-config")
			assert.NoError(t, err, "should create a temporary config file")
			defer os.Remove(file.Name())

			configWriter := io.NewFileConfigWriter(file.Name())
			err = configWriter.Write(&tc.config)
			if tc.err {
				assert.Error(t, err, "should write config into a file with an error")
			} else {
				assert.NoError(t, err, "should write config into a file without error")

				configReader := io.NewFileConfigReader(file.Name())
				config, err := configReader.Read()
				assert.NoError(t, err, "should read the written file without error")
				if config == nil {
					t.Fatal("should read a config object which is not nil")
				}
				assert.Equal(t, tc.config, *config)
			}
		})
	}
}
