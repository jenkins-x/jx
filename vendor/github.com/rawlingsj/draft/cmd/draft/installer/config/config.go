package config

import (
	b64 "encoding/base64"
	"fmt"
	"io/ioutil"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"strconv"
)

const chartConfigTpl = `
ingress:
  enabled: %s
  basedomain: %s
registry:
  url: %s
  authtoken: %s
imageOverride: %s
`

const chartConfigTLSTpl = `
tls:
  enable: %s
  verify: %s
  key: %s
  cert: %s
  cacert: %s
`

type DraftConfig struct {
	Basedomain   string
	Image        string
	Ingress      bool
	RegistryURL  string
	RegistryAuth string

	//
	// TLS configurations for Draftd
	//

	// EnableTLS instructs Draftd to serve with TLS enabled.
	//
	// Implied by VerifyTLS. If set the TLSKey and TLSCert are required.
	EnableTLS bool
	// VerifyTLS instructs Draftd to serve with TLS enabled verify remote certificates.
	//
	// If set TLSKey, TLSCert, TLSCaCert are required.
	VerifyTLS bool
	// TLSKeyFile identifies the file containing the pem encoded TLS private
	// key Draftd should use.
	//
	// Required and valid if and only if EnableTLS or VerifyTLS is set.
	TLSKeyFile string
	// TLSCertFile identifies the file containing the pem encoded TLS
	// certificate Draftd should use.
	//
	// Required and valid if and only if EnableTLS or VerifyTLS is set.
	TLSCertFile string
	// TLSCaCertFile identifies the file containing the pem encoded TLS CA
	// certificate Draftd should use to verify remotes certificates.
	//
	// Required and valid if and only if VerifyTLS is set.
	TLSCaCertFile string
}

// FromClientConfig reads a kubernetes client config, searching for information that may indicate
// this is a minikube/Azure Container Services/Google Container Engine cluster and return
// configuration optimized for that cloud, as well as the cloud provider name.
// Currently only supports minikube
func FromClientConfig(config clientcmd.ClientConfig) (*DraftConfig, string, error) {
	var cloudProviderName string

	rawConfig, err := config.RawConfig()
	if err != nil {
		return nil, "", err
	}

	draftConfig := &DraftConfig{}

	if rawConfig.CurrentContext == "minikube" {
		// we imply that the user has installed the registry addon
		draftConfig.RegistryURL = "$(REGISTRY_SERVICE_HOST)"
		draftConfig.RegistryAuth = "e30K"
		draftConfig.Basedomain = "k8s.local"
		cloudProviderName = rawConfig.CurrentContext
	}

	return draftConfig, cloudProviderName, nil
}

func (cfg *DraftConfig) WithTLS() bool { return cfg.EnableTLS || cfg.VerifyTLS }

// String returns the string representation of a DraftConfig.
func (cfg *DraftConfig) String() string {
	cfgstr := fmt.Sprintf(
		chartConfigTpl,
		strconv.FormatBool(cfg.Ingress),
		cfg.Basedomain,
		cfg.RegistryURL,
		cfg.RegistryAuth,
		cfg.Image,
	)
	if cfg.WithTLS() {
		cfgstr += addChartConfigTLSTpl(cfg)
	}
	return cfgstr
}

func addChartConfigTLSTpl(cfg *DraftConfig) string {
	return fmt.Sprintf(chartConfigTLSTpl,
		strconv.FormatBool(cfg.EnableTLS),
		strconv.FormatBool(cfg.VerifyTLS),
		b64.StdEncoding.EncodeToString(readfile(cfg.TLSKeyFile)),
		b64.StdEncoding.EncodeToString(readfile(cfg.TLSCertFile)),
		b64.StdEncoding.EncodeToString(readfile(cfg.TLSCaCertFile)),
	)
}

func readfile(path string) []byte {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: reading file %q: %v\n", path, err)
		os.Exit(2)
	}
	return b
}
