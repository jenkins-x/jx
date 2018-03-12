package main

import (
	"fmt"
	"os"
	"testing"
)

func TestDefaultEnvironment(t *testing.T) {

	testCases := []struct {
		envVar   string
		expected string
	}{
		{envVar: "", expected: "development"},
		{envVar: "something", expected: "something"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("env-%s", tc.envVar), func(t *testing.T) {
			_ = os.Setenv(environmentEnvVar, tc.envVar)

			if result := defaultDraftEnvironment(); result != tc.expected {
				t.Errorf("Expected default environment %s, got %s", tc.expected, result)
			}
		})
	}
}
