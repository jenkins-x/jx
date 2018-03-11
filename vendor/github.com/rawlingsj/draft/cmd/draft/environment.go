package main

import (
	"os"

	"github.com/Azure/draft/pkg/draft/manifest"
)

const (
	environmentEnvVar        = "DRAFT_ENV"
	environmentFlagName      = "environment"
	environmentFlagShorthand = "e"
	environmentFlagUsage     = "the environment (development, staging, qa, etc) that draft will run under"
)

func defaultDraftEnvironment() string {
	env := os.Getenv(environmentEnvVar)
	if env == "" {
		env = manifest.DefaultEnvironmentName
	}
	return env
}
