// Copyright (c) Microsoft Corporation. All rights reserved.
//
// Licensed under the MIT license.

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/helm/pkg/helm"
	hpf "k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/tlsutil"

	"github.com/Azure/draft/pkg/draft"
	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/draftd/portforwarder"
)

const (
	hostEnvVar      = "DRAFT_HOST"
	homeEnvVar      = "DRAFT_HOME"
	namespaceEnvVar = "DRAFT_NAMESPACE"
)

var (
	// flagDebug is a signal that the user wants additional output.
	flagDebug   bool
	kubeContext string
	// draftdTunnel is a tunnelled connection used to send requests to Draftd.
	// TODO refactor out this global var
	draftdTunnel *kube.Tunnel
	// draftHome depicts the home directory where all Draft config is stored.
	draftHome string
	// draftHost depicts where the Draftd server is hosted. This is used when the port forwarding logic by Kubernetes is unavailable.
	draftHost string
	// draftNamespace depicts which namespace the Draftd server is running in. This is used when Draftd was installed in a different namespace than kube-system.
	draftNamespace string

	//rootCmd is the root command handling `draft`. It's used in other parts of package cmd to add/search the command tree.
	rootCmd *cobra.Command

	// tls flags, options, and defaults
	tlsCaCertFile string // path to TLS CA certificate file
	tlsCertFile   string // path to TLS certificate file
	tlsKeyFile    string // path to TLS key file
	tlsVerify     bool   // enable TLS and verify remote certificates
	tlsEnable     bool   // enable TLS

	tlsCaCertDefault = "$DRAFT_HOME/ca.pem"
	tlsCertDefault   = "$DRAFT_HOME/cert.pem"
	tlsKeyDefault    = "$DRAFT_HOME/key.pem"
)

var globalUsage = `The application deployment tool for Kubernetes.
`

func init() {
	rootCmd = newRootCmd(os.Stdout, os.Stdin)
}

func newRootCmd(out io.Writer, in io.Reader) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "draft",
		Short:        globalUsage,
		Long:         globalUsage,
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if flagDebug {
				log.SetLevel(log.DebugLevel)
			}
			os.Setenv(homeEnvVar, draftHome)
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			teardown()
		},
	}
	p := cmd.PersistentFlags()
	p.StringVar(&draftHome, "home", defaultDraftHome(), "location of your Draft config. Overrides $DRAFT_HOME")
	p.BoolVar(&flagDebug, "debug", false, "enable verbose output")
	p.StringVar(&kubeContext, "kube-context", "", "name of the kubeconfig context to use")
	p.StringVar(&draftHost, "host", defaultDraftHost(), "address of Draftd. This is used when the port forwarding feature by Kubernetes is unavailable. Overrides $DRAFT_HOST")
	p.StringVar(&draftNamespace, "draft-namespace", defaultDraftNamespace(), "namespace where Draftd is running. This is used when Draftd was installed in a different namespace than kube-system. Overrides $DRAFT_NAMESPACE")

	cmd.AddCommand(
		newCreateCmd(out),
		newHomeCmd(out),
		newInitCmd(out, in),
		addFlagsTLS(newUpCmd(out)),
		addFlagsTLS(newVersionCmd(out)),
		newPluginCmd(out),
		newConnectCmd(out),
		newDeleteCmd(out),
		newLogsCmd(out),
	)

	// Find and add plugins
	loadPlugins(cmd, draftpath.Home(homePath()), out, in)

	return cmd
}

func setupConnection(c *cobra.Command, args []string) error {
	if draftHost == "" {
		client, config, err := getKubeClient(kubeContext)
		if err != nil {
			return err
		}
		tunnel, err := portforwarder.New(client, config, draftNamespace)
		if err != nil {
			return err
		}

		draftHost = fmt.Sprintf("127.0.0.1:%d", tunnel.Local)
		log.Debugf("Created tunnel using local port: '%d'", tunnel.Local)
	}

	log.Debugf("SERVER: %q", draftHost)
	return nil
}

func teardown() {
	if draftdTunnel != nil {
		draftdTunnel.Close()
	}
}

func ensureDraftClient(client *draft.Client) *draft.Client {
	if client != nil {
		return client
	}
	return newClient()
}

func newClient() *draft.Client {
	cfg := &draft.ClientConfig{ServerAddr: draftHost, Stdout: os.Stdout, Stderr: os.Stderr}
	if tlsVerify || tlsEnable {
		if tlsCaCertFile == tlsCaCertDefault {
			tlsCaCertFile = os.ExpandEnv(tlsCaCertDefault)
		}
		if tlsCertFile == tlsCertDefault {
			tlsCertFile = os.ExpandEnv(tlsCertDefault)
		}
		if tlsKeyFile == tlsKeyDefault {
			tlsKeyFile = os.ExpandEnv(tlsKeyDefault)
		}
		debug("Key=%q, Cert=%q, CA=%q\n", tlsKeyFile, tlsCertFile, tlsCaCertFile)
		tlsopts := tlsutil.Options{
			InsecureSkipVerify: true,
			CertFile:           tlsCertFile,
			KeyFile:            tlsKeyFile,
		}
		if tlsVerify {
			tlsopts.InsecureSkipVerify = false
			tlsopts.CaCertFile = tlsCaCertFile
		}
		tlscfg, err := tlsutil.ClientConfig(tlsopts)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		cfg.UseTLS = true
		cfg.TLSConfig = tlscfg
	}
	return draft.NewClient(cfg)
}

func defaultDraftHost() string {
	return os.Getenv(hostEnvVar)
}

func defaultDraftNamespace() string {
	if namespace := os.Getenv(namespaceEnvVar); namespace != "" {
		return namespace
	}
	return portforwarder.DefaultDraftNamespace
}

func defaultDraftHome() string {
	if home := os.Getenv(homeEnvVar); home != "" {
		return home
	}

	homeEnvPath := os.Getenv("HOME")
	if homeEnvPath == "" && runtime.GOOS == "windows" {
		homeEnvPath = os.Getenv("USERPROFILE")
	}

	return filepath.Join(homeEnvPath, ".draft")
}

func homePath() string {
	return os.ExpandEnv(draftHome)
}

// configForContext creates a Kubernetes REST client configuration for a given kubeconfig context.
func configForContext(context string) (clientcmd.ClientConfig, *rest.Config, error) {
	clientConfig := kube.GetConfig(context)
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("could not get Kubernetes config for context %q: %s", context, err)
	}
	return clientConfig, config, nil
}

// getKubeClient creates a Kubernetes config and client for a given kubeconfig context.
func getKubeClient(context string) (kubernetes.Interface, *rest.Config, error) {
	_, config, err := configForContext(context)
	if err != nil {
		return nil, nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get Kubernetes client: %s", err)
	}
	return client, config, nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func debug(format string, args ...interface{}) {
	if flagDebug {
		format = fmt.Sprintf("[debug] %s\n", format)
		fmt.Printf(format, args...)
	}
}

func validateArgs(args, expectedArgs []string) error {
	if len(args) != len(expectedArgs) {
		return fmt.Errorf("This command needs %v argument(s): %v", len(expectedArgs), expectedArgs)
	}
	return nil
}

// addFlagsTLS adds the flags for supporting client side TLS to the
// helm command (only those that invoke communicate to Draftd.)
func addFlagsTLS(cmd *cobra.Command) *cobra.Command {
	// add flags
	cmd.Flags().StringVar(&tlsCaCertFile, "tls-ca-cert", tlsCaCertDefault, "path to TLS CA certificate file")
	cmd.Flags().StringVar(&tlsCertFile, "tls-cert", tlsCertDefault, "path to TLS certificate file")
	cmd.Flags().StringVar(&tlsKeyFile, "tls-key", tlsKeyDefault, "path to TLS key file")
	cmd.Flags().BoolVar(&tlsVerify, "tls-verify", false, "enable TLS for request and verify remote")
	cmd.Flags().BoolVar(&tlsEnable, "tls", false, "enable TLS for request")
	return cmd
}

func setupHelm(kubeClient kubernetes.Interface, config *rest.Config, namespace string) (helm.Interface, error) {
	tunnel, err := setupTillerConnection(kubeClient, config, namespace)
	if err != nil {
		return nil, err
	}

	return helm.NewClient(helm.Host(fmt.Sprintf("127.0.0.1:%d", tunnel.Local))), nil
}

func setupTillerConnection(client kubernetes.Interface, config *rest.Config, namespace string) (*kube.Tunnel, error) {
	tunnel, err := hpf.New(namespace, client, config)
	if err != nil {
		return nil, fmt.Errorf("Could not get a connection to tiller: %s\nPlease ensure you have run `helm init`", err)
	}

	return tunnel, err
}
