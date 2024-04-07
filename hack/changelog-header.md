## Linux

### amd64

```shell
curl -L https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-linux-amd64.tar.gz | tar xzv
sudo mv jx /usr/local/bin
```

### arm

```shell
curl -L https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-linux-arm.tar.gz | tar xzv
sudo mv jx /usr/local/bin
```

### arm64

```shell
curl -L https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-linux-arm64.tar.gz | tar xzv
sudo mv jx /usr/local/bin
```

## macOS

### Using homebrew

```shell
brew install --no-quarantine --cask jenkins-x/jx/jx
```

### Using curl

#### amd64

```shell
curl -L https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-darwin-amd64.tar.gz | tar xzv
sudo mv jx /usr/local/bin
```

#### arm64

```shell
curl -L https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-darwin-arm64.tar.gz | tar xzv
sudo mv jx /usr/local/bin
```
