/*
Copyright 2018 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/plugin/manifest"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
)

// Base is the struct that stores state for plugin drivers and contains helper methods.
type Base struct {
	Definition      plugin.Definition
	SessionID       string
	Namespace       string
	SonobuoyImage   string
	CleanedUp       bool
	ImagePullPolicy string
}

// TemplateData is all the fields available to plugin driver templates.
type TemplateData struct {
	PluginName        string
	ResultType        string
	SessionID         string
	Namespace         string
	SonobuoyImage     string
	ImagePullPolicy   string
	ProducerContainer string
	MasterAddress     string
	CACert            string
	SecretName        string
	ExtraVolumes      []string
}

// GetSessionID returns the session id associated with the plugin.
func (b *Base) GetSessionID() string {
	return b.SessionID
}

// GetName returns the name of this Job plugin.
func (b *Base) GetName() string {
	return b.Definition.Name
}

// GetSecretName gets a name for a secret based on the plugin name and session ID.
func (b *Base) GetSecretName() string {
	return fmt.Sprintf("sonobuoy-plugin-%s-%s", b.GetName(), b.GetSessionID())
}

// GetResultType returns the ResultType for this plugin (to adhere to plugin.Interface).
func (b *Base) GetResultType() string {
	return b.Definition.ResultType
}

//GetTemplateData fills a TemplateData struct with the passed in and state variables.
func (b *Base) GetTemplateData(masterAddress string, cert *tls.Certificate) (*TemplateData, error) {

	container, err := kuberuntime.Encode(manifest.Encoder, &b.Definition.Spec)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't reserialize container for job %q", b.Definition.Name)
	}

	volumes := make([]string, len(b.Definition.ExtraVolumes))
	for i, volume := range b.Definition.ExtraVolumes {
		enc, err := kuberuntime.Encode(manifest.Encoder, &volume)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't serialize extra volume")
		}
		volumes[i] = string(enc)
	}

	cacert := getCACertPEM(cert)

	return &TemplateData{
		PluginName:        b.Definition.Name,
		ResultType:        b.Definition.ResultType,
		SessionID:         b.SessionID,
		Namespace:         b.Namespace,
		SonobuoyImage:     b.SonobuoyImage,
		ImagePullPolicy:   b.ImagePullPolicy,
		ProducerContainer: string(container),
		MasterAddress:     masterAddress,
		CACert:            cacert,
		SecretName:        b.GetSecretName(),
		ExtraVolumes:      volumes,
	}, nil
}

// MakeTLSSecret makes a Kubernetes secret object for the given TLS certificate.
func (b *Base) MakeTLSSecret(cert *tls.Certificate) (*v1.Secret, error) {
	rsaKey, ok := cert.PrivateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key not ECDSA")
	}

	if len(cert.Certificate) <= 0 {
		return nil, errors.New("no certs in tls.certificate")
	}

	certDER := cert.Certificate[0]
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyPEM, err := getKeyPEM(rsaKey)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't PEM encode TLS key")
	}

	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.GetSecretName(),
			Namespace: b.Namespace,
		},
		Data: map[string][]byte{
			v1.TLSPrivateKeyKey: keyPEM,
			v1.TLSCertKey:       certPEM,
		},
		Type: v1.SecretTypeTLS,
	}, nil

}

// getCACertPEM extracts the CA cert from a tls.Certificate.
// If the provided Certificate has only one certificate in the chain, the CA
// will be the leaf cert.
func getCACertPEM(cert *tls.Certificate) string {
	cacert := ""
	if len(cert.Certificate) > 0 {
		caCertDER := cert.Certificate[len(cert.Certificate)-1]
		cacert = string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: caCertDER,
		}))
	}
	return cacert
}

// getKeyPEM turns an RSA Private Key into a PEM-encoded string
func getKeyPEM(key *ecdsa.PrivateKey) ([]byte, error) {
	derKEY, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: derKEY,
	}), nil
}
