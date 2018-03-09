package main

import (
	"fmt"
	"io"

	log "github.com/Sirupsen/logrus"
	docker "github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/tlsutil"

	"github.com/Azure/draft/pkg/draft"
	"github.com/Azure/draft/pkg/draftd/portforwarder"
	"github.com/Azure/draft/pkg/kube"
	"github.com/Azure/draft/pkg/storage/inprocess"
	cfgmaps "github.com/Azure/draft/pkg/storage/kube"
)

const startDesc = `
Starts the draft server.
`

type startCmd struct {
	out io.Writer
	// listenAddr is the address which the server will be listening on.
	listenAddr string
	// dockerAddr is the address which the docker engine listens on.
	dockerAddr string
	// dockerVersion is the API version of the docker engine. If unset, no version information is
	// sent to the engine, however it is strongly recommended by Docker to set this or the client
	// may break if the server is upgraded.
	dockerVersion string
	// retrieve docker engine information from environment
	dockerFromEnv bool
	// registryAuth is the authorization token used to push images up to the registry.
	registryAuth string
	// registryOrg is the organization (e.g. your DockerHub account) used to push images up to the registry.
	registryOrg string
	// registryURL is the URL of the registry (e.g. quay.io, docker.io, gcr.io)
	registryURL string
	// basedomain is the base domain used to construct the ingress host name to applications.
	basedomain string
	// tillerURI is the URI used to connect to tiller.
	tillerURI string
	// local allows draftd to run locally (for testing purposes).
	local bool
	// ingressEnabled sets whether we want to use ingress or not for applications
	ingressEnabled bool
	// storage engine draftd should use (default "inprocess").
	storageEngine string
}

func newStartCmd(out io.Writer) *cobra.Command {
	sc := &startCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "start",
		Short: "start the draft server",
		Long:  startDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return sc.run()
		},
	}

	f := cmd.Flags()
	f.StringVarP(&sc.listenAddr, "listen-addr", "l", "0.0.0.0:44135", "the address the server listens on")
	f.StringVarP(&sc.dockerAddr, "docker-addr", "", "unix:///var/run/docker.sock", "the address the docker engine listens on")
	f.StringVarP(&sc.dockerVersion, "docker-version", "", "", "the API version of the docker engine")
	f.BoolVarP(&sc.dockerFromEnv, "docker-from-env", "", true, "retrieve docker engine information from environment")
	f.StringVar(&sc.registryAuth, "registry-auth", "", "the authorization token used to push images up to the registry")
	f.StringVar(&sc.registryURL, "registry-url", "127.0.0.1:5000", "the URL of the registry (e.g. quay.io, docker.io, gcr.io)")
	f.StringVar(&sc.basedomain, "basedomain", "", "the base domain in which a wildcard DNS entry points to an ingress controller")
	f.StringVar(&sc.tillerURI, "tiller-uri", "tiller-deploy:44134", "the URI used to connect to tiller")
	f.StringVar(&sc.storageEngine, "storage", "inprocess", "storage engine draftd should use")
	f.BoolVarP(&sc.local, "local", "", false, "run draftd locally (uses local kubecfg)")
	f.BoolVarP(&sc.ingressEnabled, "ingress-enabled", "", false, "configure ingress")
	// add TLS flags
	f.BoolVar(&tlsEnable, "tls", tlsEnableEnvVarDefault(), "enable TLS")
	f.BoolVar(&tlsVerify, "tls-verify", tlsVerifyEnvVarDefault(), "enable TLS and verify remote certificate")
	f.StringVar(&keyFile, "tls-key", tlsDefaultsFromEnv("tls-key"), "path to TLS private key file")
	f.StringVar(&certFile, "tls-cert", tlsDefaultsFromEnv("tls-cert"), "path to TLS certificate file")
	f.StringVar(&caCertFile, "tls-ca-cert", tlsDefaultsFromEnv("tls-ca-cert"), "trust certificates signed by this CA")

	return cmd
}

func (c *startCmd) run() (err error) {
	cfg := &draft.ServerConfig{
		IngressEnabled: c.ingressEnabled,
		Basedomain:     c.basedomain,
		ListenAddr:     c.listenAddr,
		Registry: &draft.RegistryConfig{
			Auth: c.registryAuth,
			URL:  c.registryURL,
		},
	}
	if c.dockerFromEnv {
		if cfg.Docker, err = docker.NewEnvClient(); err != nil {
			return fmt.Errorf("failed to create docker env client: %v", err)
		}
	} else {
		if cfg.Docker, err = docker.NewClient(c.dockerAddr, c.dockerVersion, nil, nil); err != nil {
			return fmt.Errorf("failed to create docker client: %v", err)
		}
	}

	if c.local {
		if cfg.Kube, err = kube.GetOutOfClusterClient(); err != nil {
			return fmt.Errorf("failed to create out-of-cluster kubernetes client: %v", err)
		}
	} else {
		if cfg.Kube, err = kube.GetInClusterClient(); err != nil {
			return fmt.Errorf("failed to create in-cluster kubernetes client: %v", err)
		}
	}
	if tlsEnable || tlsVerify {
		tlscfg, err := tlsutil.ServerConfig(tlsOptions())
		if err != nil {
			return fmt.Errorf("failed to create server TLS configuration: %v", err)
		}
		cfg.UseTLS = true
		cfg.TLSConfig = tlscfg
	}
	switch c.storageEngine {
	case "configmaps":
		namespace := envOr(namespaceEnvVar, portforwarder.DefaultDraftNamespace)
		cfg.Storage = cfgmaps.NewConfigMaps(cfg.Kube.CoreV1().ConfigMaps(namespace))
	case "inprocess":
		cfg.Storage = inprocess.NewStore()
	default:
		return fmt.Errorf("unknown storage engine name provided: %q", c.storageEngine)
	}

	cfg.Helm = helm.NewClient(helm.Host(c.tillerURI))
	log.Printf("server is now listening at %s (tls=%t)", c.listenAddr, tlsEnable || tlsVerify)

	return draft.NewServer(cfg).Serve(context.Background())
}
