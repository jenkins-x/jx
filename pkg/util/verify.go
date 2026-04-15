package util

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
)

// DownloadFile downloads a file from a URL to a local destination.
func DownloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: status %s", url, resp.Status)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// GetFileAsString downloads a file from a URL and returns its content as a string.
func GetFileAsString(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download %s: status %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// VerifySignature verifies the signature of a file against a PEM-encoded public key.
// The signature is expected to be a Base64-encoded ASN.1 (DER) ECDSA signature.
func VerifySignature(artifactPath, signaturePath string, publicKeyPEM []byte) error {
	// Read the artifact content
	artifactBytes, err := os.ReadFile(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to read artifact file %s: %w", artifactPath, err)
	}

	// Read and decode the signature
	signatureEncoded, err := os.ReadFile(signaturePath)
	if err != nil {
		return fmt.Errorf("failed to read signature file %s: %w", signaturePath, err)
	}
	signatureBytes, err := base64.StdEncoding.DecodeString(string(signatureEncoded))
	if err != nil {
		// Try raw if base64 decoding fails
		signatureBytes = signatureEncoded
	}

	// Parse the public key
	block, _ := pem.Decode(publicKeyPEM)
	if block == nil {
		return fmt.Errorf("failed to decode public key PEM")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}
	pubKey, ok := pubInterface.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("public key is not an ECDSA key")
	}

	// Hash the artifact
	h := sha256.New()
	h.Write(artifactBytes)
	digest := h.Sum(nil)

	// Verify the signature
	if !ecdsa.VerifyASN1(pubKey, digest, signatureBytes) {
		return fmt.Errorf("invalid signature for artifact %s", artifactPath)
	}

	return nil
}
