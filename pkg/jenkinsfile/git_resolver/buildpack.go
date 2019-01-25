package git_resolver

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/util"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// InitBuildPack initialises the build pack URL and git ref returning the packs dir or an error
func InitBuildPack(gitter gits.Gitter, packURL string, packRef string) (string, error) {
	u, err := url.Parse(strings.TrimSuffix(packURL, ".git"))
	if err != nil {
		return "", fmt.Errorf("Failed to parse build pack URL: %s: %s", packURL, err)
	}

	draftDir, err := util.DraftDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(draftDir, "packs", u.Host, u.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("Could not create %s: %s", dir, err)
	}

	err = gitter.CloneOrPull(packURL, dir)
	if err != nil {
		return "", err
	}
	if packRef != "master" && packRef != "" {
		err = gitter.CheckoutRemoteBranch(dir, packRef)
	}
	return filepath.Join(dir, "packs"), err
}
