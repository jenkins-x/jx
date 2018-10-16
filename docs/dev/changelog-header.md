To install jx {{.Version}} see the [install guide](https://jenkins-x.io/getting-started/install/)

### Linux

```shell
curl -L https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-linux-amd64.tar.gz | tar xzv 
sudo mv jx /usr/local/bin
```

### macOS

```shell
brew tap jenkins-x/jx
brew install jx
```
