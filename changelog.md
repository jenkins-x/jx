### Linux

```shell
curl -L https://github.com/jenkins-x/jx/releases/download/v3.2.344/jx-linux-amd64.tar.gz | tar xzv 
sudo mv jx /usr/local/bin
```

### macOS

```shell
curl -L  https://github.com/jenkins-x/jx/releases/download/v3.2.344/jx-darwin-amd64.tar.gz | tar xzv
sudo mv jx /usr/local/bin
```

## Changes

### Bug Fixes

* Make command ctx an alias for plugin jx-context (Mårten Svantesson)

### Chores

* Included githubactions in the dependabot config (naveen)
