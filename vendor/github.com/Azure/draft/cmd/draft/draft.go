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

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/helm/pkg/helm"
	hpf "k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/tiller/environment"

	"github.com/Azure/draft/pkg/draft/draftpath"
)

const (
	homeEnvVar      = "DRAFT_HOME"
	hostEnvVar      = "HELM_HOST"
	namespaceEnvVar = "TILLER_NAMESPACE"
)

var (
	// flagDebug is a signal that the user wants additional output.
	flagDebug   bool
	kubeContext string
	// draftHome depicts the home directory where all Draft config is stored.
	draftHome string
	// tillerHost depicts where Tiller is hosted. This is used when the port forwarding logic by Kubernetes is unavailable.
	tillerHost string
	// tillerNamespace depicts which namespace Tiller is running in. This is used when Tiller was installed in a different namespace than kube-system.
	tillerNamespace string
	//rootCmd is the root command handling `draft`. It's used in other parts of package cmd to add/search the command tree.
	rootCmd *cobra.Command
	// globalConfig is the configuration stored in $DRAFT_HOME/config.toml
	globalConfig DraftConfig
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
		PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
			if flagDebug {
				log.SetLevel(log.DebugLevel)
			}
			os.Setenv(homeEnvVar, draftHome)
			globalConfig, err = ReadConfig()
			return
		},
	}
	p := cmd.PersistentFlags()
	p.StringVar(&draftHome, "home", defaultDraftHome(), "location of your Draft config. Overrides $DRAFT_HOME")
	p.BoolVar(&flagDebug, "debug", false, "enable verbose output")
	p.StringVar(&kubeContext, "kube-context", "", "name of the kubeconfig context to use when talking to Tiller")
	p.StringVar(&tillerNamespace, "tiller-namespace", defaultTillerNamespace(), "namespace where Tiller is running. This is used when Tiller was installed in a different namespace than kube-system. Overrides $TILLER_NAMESPACE")

	cmd.AddCommand(
		newConfigCmd(out),
		newCreateCmd(out),
		newHomeCmd(out),
		newInitCmd(out, in),
		newUpCmd(out),
		newVersionCmd(out),
		newPluginCmd(out),
		newConnectCmd(out),
		newDeleteCmd(out),
		newLogsCmd(out),
		newHistoryCmd(out),
	)

	// Find and add plugins
	loadPlugins(cmd, draftpath.Home(homePath()), out, in)

	return cmd
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

func defaultTillerNamespace() string {
	if namespace := os.Getenv(namespaceEnvVar); namespace != "" {
		return namespace
	}
	return environment.DefaultTillerNamespace
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
