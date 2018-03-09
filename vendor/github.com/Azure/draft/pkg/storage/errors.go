package storage

import (
	"fmt"
)

// NewErrAppStorageNotFound returns a formatted error specifying the storage
// for application specified by appName does not exist.
func NewErrAppStorageNotFound(appName string) error {
	return fmt.Errorf("application storage for %q not found", appName)
}

// NewErrAppStorageExists returns a formatted error specifying the storage
// for application specified by appName already exists.
func NewErrAppStorageExists(appName string) error {
	return fmt.Errorf("application storage for %q already exists", appName)
}

// NewErrAppBuildNotFound returns a formatted error specifying the storage
// object for build with buildID does not exist.
func NewErrAppBuildNotFound(appName, buildID string) error {
	return fmt.Errorf("application %q build storage with ID %q not found", appName, buildID)
}

// NewErrAppBuildExists returns a formatted error specifying the storage
// object for build with buildID already exists.
func NewErrAppBuildExists(appName, buildID string) error {
	return fmt.Errorf("application %q build storage with ID %q already exists", appName, buildID)
}
