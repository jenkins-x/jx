### Linux

```shell
curl -L https://github.com/jenkins-x/jx/releases/download/v3.2.364/jx-linux-amd64.tar.gz | tar xzv 
sudo mv jx /usr/local/bin
```

### macOS

```shell
curl -L  https://github.com/jenkins-x/jx/releases/download/v3.2.364/jx-darwin-amd64.tar.gz | tar xzv
sudo mv jx /usr/local/bin
```

## Changes

### Code Refactoring

* add unit tests (ankitm123)

### Tests

* comment dashboard test to unblock other PRs (ankitm123)
* add code cov integration (ankitm123)

### Chores

* add codecov status badge (ankitm123)
