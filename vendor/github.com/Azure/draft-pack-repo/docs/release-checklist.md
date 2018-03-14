# HOWTO: Release a new version of draft-pack-repo

1. Check out the source
1. Tag a release from master
1. Push the tag
1. Check out the tag
1. Run `VERSION=myversion make release-assets` to compile and build the release assets
1. Upload the assets to the Github Releases page for draft-pack-repo
1. Check out master again
1. Bump the version number in plugin.yaml
1. Commit and push to master

Full steps, in bash:

```bash
export VERSION=vX.Y.Z
git clone https://github.com/Azure/draft-pack-repo
cd draft-pack-repo
git tag $VERSION
git push origin $VERSION
git checkout $VERSION
make release-assets
git checkout master
$EDITOR plugin.yaml
git add plugin.yaml
git commit
git push origin master
```
