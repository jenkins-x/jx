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

package ca

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	rsaBits  = 2048
	validFor = 48 * time.Hour
	caName   = "sonobuoy-ca"
)

var (
	pkixName = pkix.Name{
		Organization:       []string{"Heptio"},
		OrganizationalUnit: []string{"sonobuoy"},
		Country:            []string{"USA"},
		Locality:           []string{"Seattle"},
	}

	randReader = rand.Reader
)

// Authority represents a root certificate authority that can issues
// certificates to be used for Client certs.
// Sonobuoy issues every worker a client certificate
type Authority struct {
	sync.Mutex
	privKey    *ecdsa.PrivateKey
	cert       *x509.Certificate
	lastSerial *big.Int
}

// NewAuthority creates a new certificate authority. A new private key and root certificate will
// be generated but not returned.
func NewAuthority() (*Authority, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), randReader)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't generate private key")
	}
	auth := &Authority{
		privKey: privKey,
	}
	cert, err := auth.makeCert(privKey.Public(), func(cert *x509.Certificate) {
		cert.IsCA = true
		cert.KeyUsage = x509.KeyUsageCertSign
		cert.Subject.CommonName = caName
	})
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create certificate authority root certificate")
	}
	auth.cert = cert
	return auth, nil
}

// makeCert takes a public key and a function to mutate the certificate template with updated parameters
func (a *Authority) makeCert(pub crypto.PublicKey, mut func(*x509.Certificate)) (*x509.Certificate, error) {

	serialNumber := a.nextSerial()
	validFrom := time.Now()
	tmpl := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkixName,
		NotBefore:             validFrom,
		NotAfter:              validFrom.Add(validFor),
		KeyUsage:              0,
		ExtKeyUsage:           []x509.ExtKeyUsage{},
		BasicConstraintsValid: true,
	}
	mut(&tmpl)
	parent := a.cert
	// NewAuthority case
	if a.cert == nil {
		parent = &tmpl
	}

	newDERCert, err := x509.CreateCertificate(randReader, &tmpl, parent, pub, a.privKey)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make new certificate")
	}

	cert, err := x509.ParseCertificate(newDERCert)
	return cert, errors.Wrap(err, "couldn't re-parse created certificate")
}

func (a *Authority) makeLeafCert(mut func(*x509.Certificate)) (*tls.Certificate, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), randReader)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't generate private key")
	}

	cert, err := a.makeCert(privKey.Public(), mut)

	return &tls.Certificate{
		Certificate: [][]byte{cert.Raw, a.cert.Raw},
		PrivateKey:  privKey,
		Leaf:        cert,
	}, errors.Wrap(err, "couldn't make leaf cert")

}

func (a *Authority) nextSerial() *big.Int {
	a.Lock()
	defer a.Unlock()
	if a.lastSerial == nil {
		num := big.NewInt(1)
		a.lastSerial = num
		return num
	}
	// Make a copy
	return a.lastSerial.Add(a.lastSerial, big.NewInt(1))
}

// CACert is the root certificate of the CA.
func (a *Authority) CACert() *x509.Certificate {
	return a.cert
}

// CACertPool returns a CertPool prepopulated with the authority's certificate
func (a *Authority) CACertPool() *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AddCert(a.CACert())
	return pool
}

// ServerKeyPair makes a TLS server cert signed by our root CA. The returned certificate
// has a chain including the root CA cert.
func (a *Authority) ServerKeyPair(name string) (*tls.Certificate, error) {
	cert, err := a.makeLeafCert(func(cert *x509.Certificate) {
		cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		ip := net.ParseIP(name)
		if ip != nil {
			cert.IPAddresses = []net.IP{ip}
		} else {
			cert.DNSNames = []string{name}
		}
	})
	return cert, errors.Wrap(err, "couldn't make server certificate")
}

// MakeServerConfig makes a new server certificate, then returns a TLS config that uses it
// and will verify peer certificates
func (a *Authority) MakeServerConfig(name string) (*tls.Config, error) {
	cert, err := a.ServerKeyPair(name)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	pool.AddCert(a.cert)

	return &tls.Config{
		Certificates: []tls.Certificate{*cert},
		ServerName:   name,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
	}, nil
}

// ClientKeyPair makes a client cert signed by our root CA. The returned certificate
// has a chain including the root CA
func (a *Authority) ClientKeyPair(name string) (*tls.Certificate, error) {
	cert, err := a.makeLeafCert(func(cert *x509.Certificate) {
		cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		cert.Subject.CommonName = name
	})
	return cert, errors.Wrap(err, "couldn't make client certificate")
}
