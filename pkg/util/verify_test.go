package util

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifySignature(t *testing.T) {
	// 1. Generate a key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	// 2. Create a dummy artifact
	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "artifact.txt")
	content := []byte("hello world")
	err = os.WriteFile(artifactPath, content, 0644)
	require.NoError(t, err)

	// 3. Sign the artifact
	h := sha256.New()
	h.Write(content)
	digest := h.Sum(nil)
	signature, err := privateKey.Sign(rand.Reader, digest, nil)
	require.NoError(t, err)

	signaturePath := filepath.Join(tmpDir, "artifact.txt.sig")
	err = os.WriteFile(signaturePath, []byte(base64.StdEncoding.EncodeToString(signature)), 0644)
	require.NoError(t, err)

	// 4. Test success case
	err = VerifySignature(artifactPath, signaturePath, pubPEM)
	assert.NoError(t, err)

	// 5. Test failure case (corrupted artifact)
	err = os.WriteFile(artifactPath, []byte("corrupted"), 0644)
	require.NoError(t, err)
	err = VerifySignature(artifactPath, signaturePath, pubPEM)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature")

	// 6. Test failure case (wrong key)
	otherKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	otherPubBytes, _ := x509.MarshalPKIXPublicKey(&otherKey.PublicKey)
	otherPubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: otherPubBytes})
	
	err = os.WriteFile(artifactPath, content, 0644) // restore content
	require.NoError(t, err)
	err = VerifySignature(artifactPath, signaturePath, otherPubPEM)
	assert.Error(t, err)
}
