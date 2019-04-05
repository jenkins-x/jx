package cmd

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/version"
)

// VersionResolver
type VersionResolver struct {
	VersionsDir string
}

// CreateVersionResolver creates a new VersionResolver service
func (o *CommonOptions) CreateVersionResolver(repo string) (*VersionResolver, error) {
	versionsDir, err := o.cloneJXVersionsRepo(repo)
	if err != nil {
		return nil, err
	}
	return &VersionResolver{
		VersionsDir: versionsDir,
	}, nil
}

// ResolveDockerImage ensures the given docker image has a valid version if there is one in the version stream
func (v *VersionResolver) ResolveDockerImage(image string) (string, error) {
	// lets check if we already have a version
	path := strings.SplitN(image, ":", 2)
	if len(path) == 2 && path[1] != "" {
		return image, nil
	}
	info, err := version.LoadStableVersion(v.VersionsDir, version.KindDocker, image)
	if err != nil {
		return image, err
	}
	if info.Version == "" {
		// lets check if there is a docker.io prefix and if so lets try fetch without the docker prefix
		prefix := "docker.io/"
		if strings.HasPrefix(image, prefix) {
			image = strings.TrimPrefix(image, prefix)
			info, err = version.LoadStableVersion(v.VersionsDir, version.KindDocker, image)
			if err != nil {
				return image, err
			}
		}
	}
	if info.Version == "" {
		log.Warnf("could not find a stable version of docker image: %s from %s\nFor background see: https://jenkins-x.io/architecture/version-stream/\n", image, v.VersionsDir)
		log.Infof("Please lock this version down via the command: %s\n", util.ColorInfo(fmt.Sprintf("jx step create version pr -k docker -n %s -v 1.2.3\n", image)))
		return image, nil
	}
	prefix := strings.TrimSuffix(strings.TrimSpace(image), ":")
	return prefix + ":" + info.Version, nil
}

// StableVersion returns the stable version of the given kind name
func (v *VersionResolver) StableVersion(kind version.VersionKind, name string) (*version.StableVersion, error) {
	return version.LoadStableVersion(v.VersionsDir, kind, name)
}

// StableVersionNumber returns the stable version number of the given kind name
func (v *VersionResolver) StableVersionNumber(kind version.VersionKind, name string) (string, error) {
	return version.LoadStableVersionNumber(v.VersionsDir, kind, name)
}

// getVersionNumber returns the version number for the given kind and name or blank string if there is no locked version
func (o *CommonOptions) getVersionNumber(kind version.VersionKind, name, repo string) (string, error) {
	versioner, err := o.CreateVersionResolver(repo)
	if err != nil {
		return "", err
	}
	return versioner.StableVersionNumber(kind, name)
}
