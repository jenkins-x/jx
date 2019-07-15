package gitresolver

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
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

	err = ensureBranchTracksOrigin(dir, packRef, gitter)
	if err != nil {
		return "", errors.Wrapf(err, "there was a problem ensuring the branch %s has tracking info", packRef)
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

func ensureBranchTracksOrigin(dir string, packRef string, gitter gits.Gitter) error {
	empty, err := util.IsEmpty(dir)
	if err != nil {
		return errors.Wrapf(err, "there was a problem checking if %s is empty", dir)
	}

	// The repository is cloned, before the pull, we have to make sure we fetch & checkout <packRef> and we are tracking origin/<packRef>
	// This is due to a bug happening on old clones done by the old cloning func
	if !empty {
		err := gitter.FetchBranch(dir, "origin", packRef)
		if err != nil {
			return err
		}
		err = gitter.Checkout(dir, packRef)
		if err != nil {
			return err
		}
		err = gitter.SetUpstreamTo(dir, packRef)
		if err != nil {
			return errors.Wrapf(err, "there was a problem setting upstream to remote branch origin/%s", packRef)
		}
	}

	return nil
}
