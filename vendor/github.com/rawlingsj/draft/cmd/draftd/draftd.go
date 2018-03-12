// Copyright (c) Microsoft Corporation. All rights reserved.
//
// Licensed under the MIT license.

package main

import (
	"crypto/tls"
	"io"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/helm/pkg/tlsutil"
)

const (
	// tlsEnableEnvVar names the environment variable that enables TLS.
	tlsEnableEnvVar = "TILLER_TLS_ENABLE"
	// tlsVerifyEnvVar names the environment variable that enables
	// TLS, as well as certificate verification of the remote peer.
	tlsVerifyEnvVar = "TILLER_TLS_VERIFY"
	// tlsCertsEnvVar names the environment variable that points to
	// the directory where Draftd's TLS certificates are located.
	tlsCertsEnvVar = "TILLER_TLS_CERTS"
)

var (
	// flagDebug is a signal that the user wants additional output.
	flagDebug bool
	// tls flags and options
	tlsEnable  bool
	tlsVerify  bool
	keyFile    string
	certFile   string
	caCertFile string
)

var globalUsage = "The draft server."

func tlsEnableEnvVarDefault() bool { return os.Getenv(tlsEnableEnvVar) != "" }
func tlsVerifyEnvVarDefault() bool { return os.Getenv(tlsVerifyEnvVar) != "" }

func tlsDefaultsFromEnv(name string) (value string) {
	switch certsDir := os.Getenv(tlsCertsEnvVar); name {
	case "tls-key":
		return filepath.Join(certsDir, "tls.key")
	case "tls-cert":
		return filepath.Join(certsDir, "tls.crt")
	case "tls-ca-cert":
		return filepath.Join(certsDir, "ca.crt")
	}
	return ""
}

func tlsOptions() tlsutil.Options {
	opts := tlsutil.Options{CertFile: certFile, KeyFile: keyFile}
	if tlsVerify {
		opts.CaCertFile = caCertFile
		// We want to force the client to not only provide a cert, but to
		// provide a cert that we can validate.
		//
		// See: http://www.bite-code.com/2015/06/25/tls-mutual-auth-in-golang/
		opts.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return opts
}

func newRootCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "draftd",
		Short:        "The draft server.",
		Long:         globalUsage,
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if flagDebug {
				log.Printf("debug logging enabled")
				log.SetLevel(log.DebugLevel)
			}
		},
	}
	p := cmd.PersistentFlags()
	p.BoolVar(&flagDebug, "debug", false, "enable verbose output")

	cmd.AddCommand(
		newStartCmd(out),
		newVersionCmd(out),
	)

	return cmd
}

func main() {
	cmd := newRootCmd(os.Stdout)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
