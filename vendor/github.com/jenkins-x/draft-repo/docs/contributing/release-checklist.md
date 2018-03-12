# Release Checklist

**IMPORTANT**: If your experience deviates from this document, please document the changes to keep
it up-to-date.

## A Maintainer's Guide to Releasing Draft

So you're in charge of a new release for Draft? Cool. Here's what to do...

![TODO: Nothing](../img/nothing.png)

Just kidding! :trollface:

All releases will be of the form vX.Y.Z where X is the major version number, Y is the minor version
number and Z is the patch release number. This project strictly follows
[semantic versioning](http://semver.org/) so following this step is critical.

It is important to note that this document assumes that the git remote in your repository that
corresponds to "https://github.com/Azure/draft" is named "upstream". If yours is not (for example, if
you've chosen to name it "origin" or something similar instead), be sure to adjust the listed
snippets for your local environment accordingly. If you are not sure what your upstream remote is
named, use a command like `git remote -v` to find out.

If you don't have an upstream remote, you can add one easily using something like:

```
git remote add upstream git@github.com:Azure/draft.git
```

In this doc, we are going to reference a few environment variables as well, which you may want to
set for convenience. For major/minor releases, use the following:

```
export RELEASE_NAME=vX.Y.0
```

If you are creating a patch release, you may want to use the following instead:

```
export LATEST_PATCH_RELEASE=vX.Y.Z
export RELEASE_NAME=vX.Y.Z+1
```

## 1. Create the Release Branch

### Major/Minor Releases

Major releases are for new feature additions and behavioral changes *that break backwards compatibility*.
Minor releases are for new feature additions that do not break backwards compatibility. To create a
major or minor release, start by creating a `release-vX.Y.0` branch from master.

```
git fetch upstream
git checkout upstream/master
git checkout -b release-$RELEASE_NAME
```

This new branch is going to be the base for the release, which we are going to iterate upon later.

### Patch releases

Patch releases are a few critical cherry-picked fixes to existing releases. Start by creating a
`release-vX.Y.Z` branch from the latest patch release.

```
git fetch upstream --tags
git checkout $LATEST_PATCH_RELEASE
git checkout -b release-$RELEASE_NAME
```

From here, we can cherry-pick the commits we want to bring into the patch release:

```
# get the commits ids we want to cherry-pick
git log
# cherry-pick the commits starting from the oldest one, without including merge commits
git cherry-pick -x <commit-id>
git cherry-pick -x <commit-id>
```

This new branch is going to be the base for the release, which we are going to iterate upon later.

## 2. Change the Version Number in Git

Package `pkg/version` stores release-related information for Draft, including which version of
`draftd` is installed when running `draft init`. We want to change the `Release` field to the first
release candidate which we are releasing (more on that in step 5), along with a few chart-related
fields.

```
diff --git a/charts/draftd/Chart.yaml b/chart/draftd/Chart.yaml
index 954fa08..e3cf2d4 100644
--- a/chart/draftd/Chart.yaml
+++ b/chart/draftd/Chart.yaml
@@ -1,4 +1,4 @@
 name: draftd
 description: The Draft server
-version: canary
+version: v0.2.0-rc1
 apiVersion: v1
diff --git a/charts/draftd/values.yaml b/chart/draftd/values.yaml
index 3de6092..4c6314d 100644
--- a/chart/draftd/values.yaml
+++ b/chart/draftd/values.yaml
@@ -5,8 +5,8 @@ basedomain: example.com
 replicaCount: 1
 image:
   repository: microsoft/draft
-  tag: canary
-  pullPolicy: Always
+  tag: v0.2.0-rc1
+  pullPolicy: IfNotPresent
 debug: false
 service:
   http:
diff --git a/pkg/version/version.go b/pkg/version/version.go
index 4077fa4..b366797 100644
--- a/pkg/version/version.go
+++ b/pkg/version/version.go
@@ -21,7 +21,7 @@ var (
        // Increment major number for new feature additions and behavioral changes.
        // Increment minor number for bug fixes and performance enhancements.
        // Increment patch number for critical fixes to existing releases.
-       Release = "canary"
+       Release = "v0.2.0-rc1"

        // BuildMetadata is extra build time data
        BuildMetadata = ""
```

For patch releases, the old version number will be the latest patch release, so just bump the patch
number, incrementing Z by one and attach the release candidate version at the end.

We will want to keep this as a separate commit from the CHANGELOG which we will generate in the
next step, so let's commit our changes now.

```
git add .
git commit -m "bump version to $RELEASE_NAME-rc1"
```

## 3. Generate the CHANGELOG

Technically you can auto-generate a changelog based on the commits that occurred during a release
cycle, but it is usually more beneficial to the end-user if the changelog is hand-written by a
human being/marketing team/dog.

If you're releasing a major/minor release, listing notable user-facing features is usually
sufficient, listing the features in one of the four categories:

* Client
* Server
* Documentation
* Test Infrastructure (AKA Continuous Integration)

```
## vX.Y.0

### Client

* Implemented `draft up --set` [#139](https://github.com/Azure/draft/pull/139)

### Test Infrastructure

* Added drone.yml for CI automation [#128](https://github.com/Azure/draft/pull/128)
```

For patch releases, do the same, but make note of the symptoms and who is affected.

```
## vX.Y.Z

This is a bugfix release. Project repositories with a .dockerignore in the root directory were not
being properly parsed as intended. Users are encouraged to upgrade for the best experience.

### Client

* Fixed .dockerignore logic [#141](https://github.com/Azure/draft/pull/141)
```

Let's commit that now.

```
git add CHANGELOG.md
git commit -m "add $RELEASE_NAME CHANGELOG"
```

## 4. Commit and Push the Release Branch

In order for others to start testing, we can now push the release branch upstream and start the
test process.

```
git push upstream release-$RELEASE_NAME
```

If anyone is available, let others peer-review the branch before continuing to ensure that all the
proper changes have been made and all of the commits for the release are there.

## 5. Create a Release Candidate

Now that the release branch is out and ready, it is time to start creating and iterating on release
candidates.

```
git tag $RELEASE_NAME-rc1
git push upstream $RELEASE_NAME-rc1
```

CircleCI will automatically create a tagged release image and client binary to test with.

For testers, the process to start testing after CircleCI finishes building the artifacts involves
the following steps to grab the client from Azure Blob Storage:

linux/amd64, using /bin/bash:

    $ wget https://azuredraft.blob.core.windows.net/draft/draft-$RELEASE_NAME-rc1-linux-amd64.tar.gz

darwin/amd64, using Terminal.app:

    $ wget https://azuredraft.blob.core.windows.net/draft/draft-$RELEASE_NAME-rc1-darwin-amd64.tar.gz

windows/amd64, using PowerShell:

    PS C:\> $ReleaseName = "v0.2.0"
    PS C:\> Invoke-WebRequest -Uri "https://azuredraft.blob.core.windows.net/draft/draft-$RELEASE_NAME-rc1-windows-amd64.tar.gz" -OutFile "draft-$ReleaseName-rc1-windows-amd64.tar.gz"

Then, unpack and move the binary to somewhere on your $PATH, or move it somewhere and add it to
your $PATH (e.g. /usr/local/bin/helm for linux/macOS, C:\Program Files\helm\helm.exe for Windows).

## 6. Iterate on Successive Release Candidates

Spend several days explicitly investing time and resources to try and break Draft in every possible
way, documenting any findings pertinent to the release. This time should be spent testing and
finding ways in which the release might have caused various features or upgrade environments to
have issues, not coding. During this time, the release is in code freeze, and any additional code
changes will be pushed out to the next release.

During this phase, the release-$RELEASE_NAME branch will keep evolving as you will produce new
release candidates. The frequency of new candidates is up to the release manager: use your best
judgement taking into account the severity of reported issues, testers' availability, and the
release deadline date. Generally speaking, it is better to let a release roll over the deadline
than to ship a broken release.

Each time you'll want to produce a new release candidate, you will start by adding commits to the
branch by cherry-picking from master:

```
git cherry-pick -x <commit_id>
```

You will also want to update the release version number and the CHANGELOG as we did in steps 2 and
3 as separate commits.

After that, tag it and notify users of the new release candidate:

```
git tag $RELEASE_NAME-rc2
git push upstream $RELEASE_NAME-rc2
```

From here on just repeat this process, continuously testing until you're happy with the release
candidate.

## 7. Finalize the Release

When you're finally happy with the quality of a release candidate, you can move on and create the
real thing. First, you will want to change the release name we changed in step 2 back to the
official release name. Follow step 2 to make those changes then commit the change:

```
git add .
git commit -m "bump version to $RELEASE_NAME"
```

Double-check one last time to make sure eveything is in order, then finally push the release tag.

```
git tag $RELEASE_NAME
git push upstream $RELEASE_NAME
```

## 8. Push the CHANGELOG Commit to master

Now we need to push the CHANGELOG notes back to master. The "bump commit" can be discarded as the
master is just a rolling "canary" release.

```
git checkout master
git checkout -b changelog-$RELEASE_NAME
git cherry-pick -x <commit_id>
git push origin changelog-$RELEASE_NAME
```

Create a new pull request against master with this branch, then push that pretty green button to
merge it into master.

## 9. Bump the Homebrew forumla

And finally, we can update the [Homebrew formula version](https://github.com/Azure/homebrew-draft/blob/master/Formula/draft.rb#L4). Make sure to update the sha256 value with the value of

```shell
$ curl https://azuredraft.blob.core.windows.net/draft/draft-$RELEASE_NAME-darwin-amd64.tar.gz.sha256
```

## 10. Evangelize!

Congratulations! You're done. Go grab yourself a $DRINK_OF_CHOICE. You've earned it.

After enjoying a nice $DRINK_OF_CHOICE, go forth and announce the glad tidings of the new release
in Slack and on Twitter.

Optionally, write a blog post about the new release and showcase some of the new features on there!
