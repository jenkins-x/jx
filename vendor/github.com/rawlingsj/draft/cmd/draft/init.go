package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"

	"github.com/Azure/draft/cmd/draft/installer"
	installerConfig "github.com/Azure/draft/cmd/draft/installer/config"
	"github.com/Azure/draft/pkg/draft/draftpath"
)

const (
	initDesc = `
This command installs the server side component of Draft onto your
Kubernetes Cluster and sets up local configuration in $DRAFT_HOME (default ~/.draft/)

To set up just a local environment, use '--client-only'. That will configure
$DRAFT_HOME, but not attempt to connect to a remote cluster and install the Draft
deployment.

To dump information about the Draft chart, combine the '--dry-run' and '--debug' flags.
`
)

type initCmd struct {
	clientOnly     bool
	dryRun         bool
	out            io.Writer
	in             io.Reader
	home           draftpath.Home
	autoAccept     bool
	helmClient     *helm.Client
	ingressEnabled bool
	image          string
}

func newInitCmd(out io.Writer, in io.Reader) *cobra.Command {
	i := &initCmd{
		out: out,
		in:  in,
	}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "initialize Draft on both client and server",
		Long:  initDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New("This command does not accept arguments")
			}
			i.home = draftpath.Home(homePath())
			return i.run()
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&i.clientOnly, "client-only", "c", false, "install local configuration, but skip remote configuration")
	f.BoolVarP(&i.ingressEnabled, "ingress-enabled", "", false, "configure ingress")
	f.BoolVar(&i.autoAccept, "auto-accept", false, "automatically accept configuration defaults (if detected). It will still prompt for information if this is set to true and no cloud provider was found")
	f.BoolVar(&i.dryRun, "dry-run", false, "go through all the steps without actually installing anything. Mostly used along with --debug for debugging purposes.")
	f.StringVarP(&i.image, "draftd-image", "i", "", "override Draftd image")
	// flags for bootstrapping draftd with tls
	f.BoolVar(&tlsEnable, "draftd-tls", false, "install Draftd with TLS enabled")
	f.BoolVar(&tlsVerify, "draftd-tls-verify", false, "install Draftd with TLS enabled and to verify remote certificates")
	f.StringVar(&tlsKeyFile, "draftd-tls-key", "", "path to TLS key file to install with Draftd")
	f.StringVar(&tlsCertFile, "draftd-tls-cert", "", "path to TLS certificate file to install with Draftd")
	f.StringVar(&tlsCaCertFile, "tls-ca-cert", "", "path to CA root certificate")

	return cmd
}

// tlsOptions sanitizes the tls flags as well as checks for the existence of
// required tls files indicate by those flags, if any.
func (i *initCmd) tlsOptions(cfg *installerConfig.DraftConfig) error {
	cfg.EnableTLS = tlsEnable || tlsVerify
	cfg.VerifyTLS = tlsVerify
	if cfg.EnableTLS {
		missing := func(file string) bool {
			_, err := os.Stat(file)
			return os.IsNotExist(err)
		}
		if cfg.TLSKeyFile = tlsKeyFile; cfg.TLSKeyFile == "" || missing(cfg.TLSKeyFile) {
			return errors.New("missing required TLS key file")
		}
		if cfg.TLSCertFile = tlsCertFile; cfg.TLSCertFile == "" || missing(cfg.TLSCertFile) {
			return errors.New("missing required TLS certificate file")
		}
		if cfg.VerifyTLS {
			if cfg.TLSCaCertFile = tlsCaCertFile; cfg.TLSCaCertFile == "" || missing(cfg.TLSCaCertFile) {
				return errors.New("missing required TLS CA file")
			}
		}
	}
	return nil
}

// runInit initializes local config and installs Draft to Kubernetes Cluster
func (i *initCmd) run() error {

	if !i.dryRun {
		if err := i.setupDraftHome(); err != nil {
			return err
		}
	}
	fmt.Fprintf(i.out, "$DRAFT_HOME has been configured at %s.\n", draftHome)

	if !i.clientOnly {
		client, clientConfig, err := getKubeClient(kubeContext)
		if err != nil {
			return fmt.Errorf("Could not get a kube client: %s", err)
		}
		restClientConfig, err := clientConfig.ClientConfig()
		if err != nil {
			return fmt.Errorf("Could not retrieve client config from the kube client: %s", err)
		}

		i.helmClient, err = setupHelm(client, restClientConfig, draftNamespace)
		if err != nil {
			return err
		}

		draftConfig, cloudProvider, err := installerConfig.FromClientConfig(clientConfig)
		if err != nil {
			return fmt.Errorf("Could not generate chart config from kube client config: %s", err)
		}

		if cloudProvider == "minikube" {
			fmt.Fprintf(i.out, "\nDraft detected that you are using %s as your cloud provider. AWESOME!\n", cloudProvider)

			if !i.autoAccept {
				fmt.Fprint(i.out, "Is it okay to use the registry addon in minikube to store your application images?\nIf not, we will prompt you for information on the registry you'd like to push your application images to during development. [Y/n] ")
				reader := bufio.NewReader(i.in)
				text, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("Could not read input: %s", err)
				}
				text = strings.TrimSpace(text)
				if text == "" || strings.ToLower(text) == "y" {
					i.autoAccept = true
				}
			}
		}

		draftConfig.Ingress = i.ingressEnabled
		draftConfig.Image = i.image

		if !i.autoAccept || cloudProvider == "" {
			// prompt for missing information
			fmt.Fprintf(i.out, "\nIn order to configure Draft, we need a bit more information...\n\n")

			reader := bufio.NewReader(i.in)
			if err := setupContainerRegistry(i.out, reader, draftConfig); err != nil {
				return err
			}

			if err := setupBasedomain(i.out, reader, i.ingressEnabled, draftConfig); err != nil {
				return err
			}

		}

		if err := i.tlsOptions(draftConfig); err != nil {
			return err
		}

		log.Debugf("raw chart config: %s", draftConfig)

		if !i.dryRun {
			// attempt to purge the old release, but log errors to --debug
			if err := installer.Uninstall(i.helmClient); err != nil {
				log.Debugf("error uninstalling Draft: %s", err)
			}
			if err := installer.Install(i.helmClient, draftNamespace, draftConfig); err != nil {
				return fmt.Errorf("error installing Draft: %s", err)
			}
		}
		fmt.Fprintln(i.out, "Draft has been installed into your Kubernetes Cluster.")
	} else {
		fmt.Fprintln(i.out, "Skipped installing Draft's server side component in Kubernetes due to 'client-only' flag having been set")
	}

	fmt.Fprintln(i.out, "Happy Sailing!")
	return nil
}

func (i *initCmd) setupDraftHome() error {
	ensureFuncs := []func() error{
		i.ensureDirectories,
		i.ensurePlugins,
		i.ensurePacks,
	}

	for _, funct := range ensureFuncs {
		if err := funct(); err != nil {
			return err
		}
	}

	return nil
}

func setupTillerConnection(client kubernetes.Interface, restClientConfig *restclient.Config, namespace string) (*kube.Tunnel, error) {
	tunnel, err := portforwarder.New(namespace, client, restClientConfig)
	if err != nil {
		return nil, fmt.Errorf("Could not get a connection to tiller: %s\nPlease ensure you have run `helm init`", err)
	}

	return tunnel, err
}

func setupBasedomain(out io.Writer, reader *bufio.Reader, ingress bool, draftConfig *installerConfig.DraftConfig) error {
	if ingress {
		fmt.Fprint(out, "4. Enter your top-level domain for ingress (e.g. draft.example.com): ")
		basedomain, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("Could not read input: %s", err)
		}
		draftConfig.Basedomain = strings.TrimSpace(basedomain)
	} else {
		draftConfig.Basedomain = ""
	}

	return nil
}

func setupHelm(kubeClient *kubernetes.Clientset, restClientConfig *restclient.Config, namespace string) (*helm.Client, error) {
	tunnel, err := setupTillerConnection(kubeClient, restClientConfig, namespace)
	if err != nil {
		return nil, err
	}

	return helm.NewClient(helm.Host(fmt.Sprintf("localhost:%d", tunnel.Local))), nil
}

func setupContainerRegistry(out io.Writer, reader *bufio.Reader, draftConfig *installerConfig.DraftConfig) error {
	fmt.Fprint(out, "1. Enter your Docker registry URL (e.g. docker.io/myuser, quay.io/myuser, myregistry.azurecr.io): ")
	registryURL, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("Could not read input: %s", err)
	}
	draftConfig.RegistryURL = strings.TrimSpace(registryURL)

	fmt.Fprint(out, "2. Enter your username: ")
	dockerUser, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("Could not read input: %s", err)
	}
	dockerUser = strings.TrimSpace(dockerUser)
	fmt.Fprint(out, "3. Enter your password: ")
	// NOTE(bacongobbler): casting syscall.Stdin here to an int is intentional here as on
	// Windows, syscall.Stdin is a Handler, which is of type uintptr.
	dockerPass, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println("")
	if err != nil {
		return fmt.Errorf("Could not read input: %s", err)
	}

	registryAuth := base64.StdEncoding.EncodeToString(
		[]byte(fmt.Sprintf(
			`{"username":"%s","password":"%s"}`,
			dockerUser,
			dockerPass)))

	draftConfig.RegistryAuth = registryAuth
	return nil
}
