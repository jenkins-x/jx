### Linux

```shell
curl -L https://github.com/jenkins-x/jx/releases/download/v3.2.353/jx-linux-amd64.tar.gz | tar xzv 
sudo mv jx /usr/local/bin
```

### macOS

```shell
curl -L  https://github.com/jenkins-x/jx/releases/download/v3.2.353/jx-darwin-amd64.tar.gz | tar xzv
sudo mv jx /usr/local/bin
```

## Changes

### Bug Fixes

* downgrade gitops version to 0.7.8 to fix versionstream tests (ankitm123)

### Chores

* deps: upgrade jenkins-x/jx to version 0.2.41 (jenkins-x-bot)
