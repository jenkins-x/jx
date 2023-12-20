## Linux

### amd64

```shell
# Download the archive and the cosign generated signature
curl -LO https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-linux-amd64.tar.gz -LO https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-linux-amd64.tar.gz.sig -LO https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-linux-amd64.tar.gz.pem

# Install cosign: https://docs.sigstore.dev/cosign/installation
# Verify using cosign
COSIGN_EXPERIMENTAL=1 cosign verify-blob --certificate jx-linux-amd64.tar.gz.pem --signature jx-linux-amd64.tar.gz.sig jx-linux-amd64.tar.gz

tar -zxvf jx-linux-amd64.tar.gz
sudo mv jx /usr/local/bin
```

### arm

```shell
# Download the archive and the cosign generated signature
curl -LO https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-linux-arm.tar.gz -LO https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-linux-arm.tar.gz.sig -LO https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-linux-arm.tar.gz.pem

# Install cosign: https://docs.sigstore.dev/cosign/installation
# Verify using cosign
COSIGN_EXPERIMENTAL=1 cosign verify-blob --certificate jx-linux-arm.tar.gz.pem --signature jx-linux-arm.tar.gz.sig jx-linux-arm.tar.gz
tar -zxvf jx-linux-arm.tar.gz
sudo mv jx /usr/local/bin
```

### arm64

```shell
# Download the archive and the cosign generated signature
curl -LO https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-linux-arm64.tar.gz -LO https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-linux-arm64.tar.gz.sig -LO https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-linux-arm64.tar.gz.pem

# Install cosign: https://docs.sigstore.dev/cosign/installation
# Verify using cosign
COSIGN_EXPERIMENTAL=1 cosign verify-blob --certificate jx-linux-arm64.tar.gz.pem --signature jx-linux-arm64.tar.gz.sig jx-linux-arm64.tar.gz

tar -zxvf jx-linux-arm64.tar.gz
sudo mv jx /usr/local/bin
```

## macOS

### amd64

```shell
# Download the archive and the cosign generated signature
curl -LO https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-darwin-amd64.tar.gz -LO https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-darwin-amd64.tar.gz.sig -LO https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-darwin-amd64.tar.gz.pem

# Install cosign: https://docs.sigstore.dev/cosign/installation
# Verify using cosign
COSIGN_EXPERIMENTAL=1 cosign verify-blob --certificate jx-darwin-amd64.tar.gz.pem --signature jx-darwin-amd64.tar.gz.sig jx-darwin-amd64.tar.gz

tar -zxvf jx-darwin-amd64.tar.gz
sudo mv jx /usr/local/bin
```

### arm64

```shell
# Download the archive and the cosign generated signature
curl -LO https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-darwin-arm64.tar.gz -LO https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-darwin-arm64.tar.gz.sig -LO https://github.com/jenkins-x/jx/releases/download/v{{.Version}}/jx-darwin-arm64.tar.gz.pem

# Install cosign: https://docs.sigstore.dev/cosign/installation
# Verify using cosign
COSIGN_EXPERIMENTAL=1 cosign verify-blob --certificate jx-darwin-arm64.tar.gz.pem --signature jx-darwin-arm64.tar.gz.sig jx-darwin-arm64.tar.gz

tar -zxvf jx-darwin-arm64.tar.gz
sudo mv jx /usr/local/bin
```
