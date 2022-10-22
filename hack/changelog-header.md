## Linux

### amd64

```shell
# Download the archive and the cosign generated signature
curl -LO https://github.com/jenkins-x/jx/releases/download/{{.Version}}/jx-linux-amd64.tar.gz -LO https://github.com/jenkins-x/jx/releases/download/{{.Version}}/jx-linux-amd64.tar.gz.sig

# Install cosign: https://docs.sigstore.dev/cosign/installation
# Verify using cosign
cosign verify-blob --key https://raw.githubusercontent.com/jenkins-x/jx/main/jx.pub --signature jx-linux-amd64.tar.gz.sig jx-linux-amd64.tar.gz

tar -zxvf jx-linux-amd64.tar.gz
sudo mv jx /usr/local/bin
```

### arm

```shell
# Download the archive and the cosign generated signature
curl -LO https://github.com/jenkins-x/jx/releases/download/{{.Version}}/jx-linux-arm.tar.gz -LO https://github.com/jenkins-x/jx/releases/download/{{.Version}}/jx-linux-arm.tar.gz.sig

# Install cosign: https://docs.sigstore.dev/cosign/installation
# Verify using cosign
cosign verify-blob --key https://raw.githubusercontent.com/jenkins-x/jx/main/jx.pub --signature jx-linux-arm.tar.gz.sig jx-linux-arm.tar.gz

tar -zxvf jx-linux-arm.tar.gz
sudo mv jx /usr/local/bin
```

### arm64

```shell
# Download the archive and the cosign generated signature
curl -LO https://github.com/jenkins-x/jx/releases/download/{{.Version}}/jx-linux-arm64.tar.gz -LO https://github.com/jenkins-x/jx/releases/download/{{.Version}}/jx-linux-arm64.tar.gz.sig

# Install cosign: https://docs.sigstore.dev/cosign/installation
# Verify using cosign
cosign verify-blob --key https://raw.githubusercontent.com/jenkins-x/jx/main/jx.pub --signature jx-linux-arm64.tar.gz.sig jx-linux-arm64.tar.gz

tar -zxvf jx-linux-arm64.tar.gz
sudo mv jx /usr/local/bin
```

## macOS

### amd64

```shell
# Download the archive and the cosign generated signature
curl -LO https://github.com/jenkins-x/jx/releases/download/{{.Version}}/jx-darwin-amd64.tar.gz -LO https://github.com/jenkins-x/jx/releases/download/{{.Version}}/jx-darwin-amd64.tar.gz.sig

# Install cosign: https://docs.sigstore.dev/cosign/installation
# Verify using cosign
cosign verify-blob --key https://raw.githubusercontent.com/jenkins-x/jx/main/jx.pub --signature jx-darwin-amd64.tar.gz.sig jx-darwin-amd64.tar.gz

tar -zxvf jx-darwin-amd64.tar.gz
sudo mv jx /usr/local/bin
```

### arm64

```shell
# Download the archive and the cosign generated signature
curl -LO https://github.com/jenkins-x/jx/releases/download/{{.Version}}/jx-darwin-arm64.tar.gz -LO https://github.com/jenkins-x/jx/releases/download/{{.Version}}/jx-darwin-arm64.tar.gz.sig

# Install cosign: https://docs.sigstore.dev/cosign/installation
# Verify using cosign
cosign verify-blob --key https://raw.githubusercontent.com/jenkins-x/jx/main/jx.pub --signature jx-darwin-arm64.tar.gz.sig jx-darwin-arm64.tar.gz

tar -zxvf jx-darwin-arm64.tar.gz
sudo mv jx /usr/local/bin
```
