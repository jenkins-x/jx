package operations

import (
	"bytes"
	"fmt"

	"github.com/blang/semver"
	"github.com/ghodss/yaml"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/cmd/opts"

	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/dependencymatrix"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/gits/releases"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/uuid"
)

// PullRequestOperation provides a way to execute a PullRequest operation using Git
type PullRequestOperation struct {
	*opts.CommonOptions
	GitURLs    []string
	SrcGitURL  string
	Base       string
	Component  string
	BranchName string
	Version    string
	DryRun     bool
}

// CreatePullRequest will fork (if needed) and pull a git repo, then perform the update, and finally create or update a
// PR for the change. Any open PR on the repo with the `updatebot` label will be updated.
func (o *PullRequestOperation) CreatePullRequest(kind string, update func(dir string, gitInfo *gits.GitRepository) ([]string, error)) (*gits.PullRequestInfo, error) {
	var result *gits.PullRequestInfo
	for _, gitURL := range o.GitURLs {
		dir, err := ioutil.TempDir("", "create-pr")
		if err != nil {
			return nil, err
		}
		provider, _, err := o.CreateGitProviderForURLWithoutKind(gitURL)
		if err != nil {
			return nil, errors.Wrapf(err, "creating git provider for directory %s", dir)
		}

		dir, _, upstreamInfo, forkInfo, err := gits.ForkAndPullRepo(gitURL, dir, o.Base, o.BranchName, provider, o.Git(), "")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to fork and pull %s", o.GitURLs)
		}

		gitInfo := upstreamInfo
		if forkInfo != nil {
			gitInfo = forkInfo
		}

		oldVersions, err := update(dir, gitInfo)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		nonSemantic := make([]string, 0)
		semantic := make([]semver.Version, 0)
		for _, v := range oldVersions {
			sv, err := semver.Parse(v)
			if err != nil {
				nonSemantic = append(nonSemantic, v)
			} else {
				semantic = append(semantic, sv)
			}
		}
		semver.Sort(semantic)
		sort.Strings(nonSemantic)
		dedupedSemantic := make([]string, 0)
		dedupedNonSemantic := make([]string, 0)
		previous := ""
		for _, v := range nonSemantic {
			if v != previous {
				dedupedNonSemantic = append(dedupedNonSemantic, v)
			}
			previous = v
		}
		previous = ""
		for _, v := range semantic {
			vStr := v.String()
			if vStr != previous {
				dedupedSemantic = append(dedupedSemantic, vStr)
			}
			previous = vStr
		}

		oldVersionsStr := strings.Join(dedupedSemantic, ", ")
		if len(dedupedNonSemantic) > 0 {
			if oldVersionsStr != "" {
				oldVersionsStr += " and "
			}
			oldVersionsStr = oldVersionsStr + strings.Join(dedupedNonSemantic, ", ")
		}

		commitMessage, details, updateDependency, assets, err := o.CreateDependencyUpdatePRDetails(kind, o.SrcGitURL, upstreamInfo, oldVersionsStr, o.Version, o.Component)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		var upstreamDependencyAsset *gits.GitReleaseAsset

		for _, asset := range assets {
			if asset.Name == dependencymatrix.DependencyUpdatesAssetName {
				upstreamDependencyAsset = &asset
				break
			}
		}

		err = dependencymatrix.UpdateDependencyMatrix(dir, updateDependency)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if upstreamDependencyAsset != nil {
			updatedPaths, err := addDependencyMatrixUpdatePaths(upstreamDependencyAsset, updateDependency)
			if err != nil {
				return nil, errors.Wrap(err, "adding dependency updates")
			}

			for _, d := range updatedPaths {
				err = dependencymatrix.UpdateDependencyMatrix(dir, d)
				if err != nil {
					return nil, errors.Wrapf(err, "updating dependency matrix with upstream dependency %+v", d)
				}
			}
		}

		labels := []string{"updatebot"}
		filter := &gits.PullRequestFilter{
			Labels: labels,
		}
		result, err = gits.PushRepoAndCreatePullRequest(dir, upstreamInfo, forkInfo, o.Base, details, filter, true, commitMessage, true, o.DryRun, o.Git(), provider, labels)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create PR for base %s and head branch %s from temp dir %s", o.Base, details.BranchName, dir)
		}
	}
	return result, nil
}

func addDependencyMatrixUpdatePaths(upstreamDependencyAsset *gits.GitReleaseAsset, updateDependency *v1.DependencyUpdate) ([]*v1.DependencyUpdate, error) {
	var upstreamUpdates dependencymatrix.DependencyUpdates
	var updates []*v1.DependencyUpdate
	resp, err := http.Get(upstreamDependencyAsset.BrowserDownloadURL)
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving dependency updates from %s", upstreamDependencyAsset.BrowserDownloadURL)
	}
	defer resp.Body.Close()

	// Write the body
	var b bytes.Buffer
	_, err = io.Copy(&b, resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving dependency updates from %s", upstreamDependencyAsset.BrowserDownloadURL)
	}
	err = yaml.Unmarshal(b.Bytes(), &upstreamUpdates)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshaling dependency updates from %s", upstreamDependencyAsset.BrowserDownloadURL)
	}
	for _, d := range upstreamUpdates.Updates {
		// Need to prepend a path element
		if d.Paths == nil {
			d.Paths = []v1.DependencyUpdatePath{
				{
					{
						Host:               updateDependency.Host,
						Owner:              updateDependency.Owner,
						Repo:               updateDependency.Repo,
						URL:                updateDependency.URL,
						Component:          updateDependency.Component,
						ToReleaseHTMLURL:   updateDependency.ToReleaseHTMLURL,
						ToVersion:          updateDependency.ToVersion,
						FromVersion:        updateDependency.FromVersion,
						FromReleaseName:    updateDependency.FromReleaseName,
						FromReleaseHTMLURL: updateDependency.FromReleaseHTMLURL,
						ToReleaseName:      updateDependency.ToReleaseName,
					},
				},
			}
		} else {
			for j, e := range d.Paths {
				if e == nil {
					d.Paths[j] = v1.DependencyUpdatePath{
						{
							Host:               updateDependency.Host,
							Owner:              updateDependency.Owner,
							Repo:               updateDependency.Repo,
							URL:                updateDependency.URL,
							Component:          updateDependency.Component,
							ToReleaseHTMLURL:   updateDependency.ToReleaseHTMLURL,
							ToVersion:          updateDependency.ToVersion,
							FromVersion:        updateDependency.FromVersion,
							FromReleaseName:    updateDependency.FromReleaseName,
							FromReleaseHTMLURL: updateDependency.FromReleaseHTMLURL,
							ToReleaseName:      updateDependency.ToReleaseName,
						},
					}
				}
				d.Paths[j] = append([]v1.DependencyUpdateDetails{
					{
						Host:               updateDependency.Host,
						Owner:              updateDependency.Owner,
						Repo:               updateDependency.Repo,
						URL:                updateDependency.URL,
						Component:          updateDependency.Component,
						ToReleaseHTMLURL:   updateDependency.ToReleaseHTMLURL,
						ToVersion:          updateDependency.ToVersion,
						FromVersion:        updateDependency.FromVersion,
						FromReleaseName:    updateDependency.FromReleaseName,
						FromReleaseHTMLURL: updateDependency.FromReleaseHTMLURL,
						ToReleaseName:      updateDependency.ToReleaseName,
					},
				}, e...)
			}
		}
		updates = append(updates, &d)
	}
	return updates, nil
}

// CreateDependencyUpdatePRDetails creates the PullRequestDetails for a pull request, taking the kind of change it is (an id)
// the srcRepoUrl for the repo that caused the change, the destRepo for the repo receiving the change, the fromVersion and the toVersion
func (o PullRequestOperation) CreateDependencyUpdatePRDetails(kind string, srcRepoURL string, destRepo *gits.GitRepository, fromVersion string, toVersion string, component string) (string, *gits.PullRequestDetails, *v1.DependencyUpdate, []gits.GitReleaseAsset, error) {
	var commitMessage, title, message strings.Builder
	commitMessage.WriteString("chore(deps): bump ")
	title.WriteString("chore(deps): bump ")
	message.WriteString("Update ")
	var update *v1.DependencyUpdate
	var assets []gits.GitReleaseAsset

	if srcRepoURL != "" {
		provider, srcRepo, err := o.CreateGitProviderForURLWithoutKind(srcRepoURL)
		if err != nil {
			return "", nil, nil, nil, errors.Wrapf(err, "creating git provider for %s", srcRepoURL)
		}
		update = &v1.DependencyUpdate{
			DependencyUpdateDetails: v1.DependencyUpdateDetails{
				Owner: srcRepo.Organisation,
				Repo:  srcRepo.Name,
				URL:   srcRepoURL,
			},
		}
		if srcRepo.Host != destRepo.Host {
			commitMessage.WriteString(srcRepoURL)
			title.WriteString(srcRepoURL)
			update.Host = srcRepo.Host
		} else {
			titleStr := fmt.Sprintf("%s/%s", srcRepo.Organisation, srcRepo.Name)
			commitMessage.WriteString(titleStr)
			title.WriteString(titleStr)
			update.Host = destRepo.Host
		}
		repoStr := fmt.Sprintf("[%s/%s](%s)", srcRepo.Organisation, srcRepo.Name, srcRepoURL)
		message.WriteString(repoStr)

		if component != "" {
			componentStr := fmt.Sprintf(":%s", component)
			commitMessage.WriteString(componentStr)
			title.WriteString(componentStr)
			message.WriteString(componentStr)
			update.Component = component
		}
		commitMessage.WriteString(" ")
		title.WriteString(" ")
		message.WriteString(" ")

		if fromVersion != "" {
			fromText := fmt.Sprintf("from %s ", fromVersion)
			commitMessage.WriteString(fromText)
			title.WriteString(fromText)
			update.FromVersion = fromVersion
			release, err := releases.GetRelease(fromVersion, srcRepo.Organisation, srcRepo.Name, provider)
			if err != nil {
				return "", nil, nil, nil, errors.Wrapf(err, "getting release from %s/%s", srcRepo.Organisation, srcRepo.Name)
			}
			if release != nil {
				message.WriteString(fmt.Sprintf("from [%s](%s) ", fromVersion, release.HTMLURL))
				update.FromReleaseName = release.Name
				update.FromReleaseHTMLURL = release.HTMLURL
			} else {
				message.WriteString(fmt.Sprintf("from %s ", fromVersion))
			}
		}
		if toVersion != "" {
			toText := fmt.Sprintf("to %s", toVersion)
			commitMessage.WriteString(toText)
			title.WriteString(toText)
			update.ToVersion = toVersion
			release, err := releases.GetRelease(toVersion, srcRepo.Organisation, srcRepo.Name, provider)
			if err != nil {
				return "", nil, nil, nil, errors.Wrapf(err, "getting release from %s/%s", srcRepo.Organisation, srcRepo.Name)
			}
			if release != nil {
				message.WriteString(fmt.Sprintf("to [%s](%s)", toVersion, release.HTMLURL))
				update.ToReleaseHTMLURL = release.HTMLURL
				update.ToReleaseName = release.Name
				if release.Assets != nil {
					assets = *release.Assets
				}
			} else {
				message.WriteString(fmt.Sprintf("to %s", toVersion))
			}
		}
	}
	message.WriteString(fmt.Sprintf("\n\nCommand run was `%s`", strings.Join(os.Args, " ")))
	commitMessage.WriteString(fmt.Sprintf("\n\nCommand run was `%s`", strings.Join(os.Args, " ")))
	return commitMessage.String(), &gits.PullRequestDetails{
		BranchName: fmt.Sprintf("bump-%s-version-%s", kind, string(uuid.NewUUID())),
		Title:      title.String(),
		Message:    message.String(),
	}, update, assets, nil
}
