package kube

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
)

// TailLogs will tail the logs for the pod in ns with containerName,
// returning when the logs are complete. It writes to errOut and out.
func TailLogs(ns string, pod string, containerName string, errOut io.Writer, out io.Writer) error {
	args := []string{"logs", "-n", ns, "-f"}
	if containerName != "" {
		args = append(args, "-c", containerName)
	}
	args = append(args, pod)
	name := "kubectl"
	e := exec.Command(name, args...)
	e.Stderr = errOut
	stdout, _ := e.StdoutPipe()

	err := os.Setenv("PATH", util.PathWithBinary())
	if err != nil {
		return errors.Wrap(err, "failed to set PATH env variable")
	}
	err = e.Start()
	if err != nil {
		log.Logger().Errorf("Error: Command failed  %s %s", name, strings.Join(args, " "))
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		m := scanner.Text()
		_, err = fmt.Fprintln(out, m)
		if err != nil {
			return err
		}
		if m == "Finished: FAILURE" {
			os.Exit(1)
		}
	}
	err = e.Wait()
	if err != nil {
		return err
	}
	return nil
}
