package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/local"
)

const (
	connectDesc = `This command creates a local environment for you to test your app. It will give you a localhost url that you can use to see your application working and it will print out logs from your application. This command must be run in the root of your application directory.
`
)

var (
	targetContainer string
	overridePorts   []string

	dryRun bool
)

type connectCmd struct {
	out      io.Writer
	logLines int64
}

func newConnectCmd(out io.Writer) *cobra.Command {
	var (
		cc                 = &connectCmd{out: out}
		runningEnvironment string
	)

	cmd := &cobra.Command{
		Use:   "connect",
		Short: "connect to your application locally",
		Long:  connectDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cc.run(runningEnvironment)
		},
	}

	f := cmd.Flags()
	f.Int64Var(&cc.logLines, "tail", 5, "lines of recent log lines to display")
	f.StringVarP(&runningEnvironment, environmentFlagName, environmentFlagShorthand, defaultDraftEnvironment(), environmentFlagUsage)
	f.StringVarP(&targetContainer, "container", "c", "", "name of the container to connect to")
	f.StringSliceVarP(&overridePorts, "override-port", "p", []string{}, "specify a local port to connect to, in the form <local>:<remote>")
	f.BoolVarP(&dryRun, "dry-run", "", false, "when this flag is used, draft connect will wait to find a ready pod then exit")

	return cmd
}

func (cn *connectCmd) run(runningEnvironment string) (err error) {
	deployedApp, err := local.DeployedApplication(draftToml, runningEnvironment)
	if err != nil {
		return err
	}

	client, config, err := getKubeClient(kubeContext)
	if err != nil {
		return err
	}

	var ports []string
	if len(deployedApp.OverridePorts) != 0 {
		ports = deployedApp.OverridePorts
	}

	// --override-port takes precedence
	if len(overridePorts) != 0 {
		ports = overridePorts
	}

	buildID, err := getLatestBuildID(deployedApp.Name)
	if err != nil {
		return err
	}

	connection, err := deployedApp.Connect(client, config, targetContainer, ports, buildID)
	if err != nil {
		return err
	}

	if dryRun {
		return
	}

	var connectionMessage = "Your connection is still active.\n"

	// output all local ports first - easier to spot
	for _, cc := range connection.ContainerConnections {
		for _, t := range cc.Tunnels {
			err = t.ForwardPort()
			if err != nil {
				return err
			}
			m := fmt.Sprintf("Connect to %v:%v on localhost:%#v\n", cc.ContainerName, t.Remote, t.Local)

			connectionMessage += m
			fmt.Fprintf(cn.out, m)
		}
	}

	for _, cc := range connection.ContainerConnections {
		readCloser, err := connection.RequestLogStream(deployedApp.Namespace, cc.ContainerName, cn.logLines)
		if err != nil {
			return err
		}

		defer readCloser.Close()
		go writeContainerLogs(cn.out, readCloser, cc.ContainerName)
	}

	stop := make(chan os.Signal, 2)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		os.Exit(0)
	}()

	for {
		time.Sleep(5 * time.Minute)
		fmt.Fprintf(cn.out, connectionMessage)
	}
}

func writeContainerLogs(out io.Writer, in io.ReadCloser, containerName string) error {
	b := bufio.NewReader(in)
	for {
		line, err := b.ReadString('\n')
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "[%v]: %v", containerName, line)
	}
}

func getLatestBuildID(appName string) (string, error) {
	h := draftpath.Home(homePath())
	files, err := ioutil.ReadDir(filepath.Join(h.Logs(), appName))
	if err != nil {
		return "", err
	}
	n := len(files)
	if n == 0 {
		return "", fmt.Errorf("could not find the latest build ID of your application. Try `draft up` first")
	}
	return files[n-1].Name(), nil
}
