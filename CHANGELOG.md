### Linux

```shell
curl -L https://github.com/jenkins-x/jx/releases/download/v3.2.366/jx-linux-amd64.tar.gz | tar xzv 
sudo mv jx /usr/local/bin
```

### macOS

```shell
curl -L  https://github.com/jenkins-x/jx/releases/download/v3.2.366/jx-darwin-amd64.tar.gz | tar xzv
sudo mv jx /usr/local/bin
```

## Changes

### Tests

* set namespace to fix tests in GH actions (ankitm123)

### Chores

* deps: upgrade jenkins-x/jx3-pipeline-catalog to version 0.1.13 (jenkins-x-bot-test)
