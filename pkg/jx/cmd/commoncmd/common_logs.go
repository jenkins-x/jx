package commoncmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

func (o *CommonOptions) TailLogs(ns string, pod string, containerName string) error {
	args := []string{"logs", "-n", ns, "-f"}
	if containerName != "" {
		args = append(args, "-c", containerName)
	}
	args = append(args, pod)
	name := "kubectl"
	e := exec.Command(name, args...)
	e.Stderr = o.Err
	stdout, _ := e.StdoutPipe()

	os.Setenv("PATH", util.PathWithBinary())
	err := e.Start()
	if err != nil {
		log.Errorf("Error: Command failed  %s %s\n", name, strings.Join(args, " "))
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		m := scanner.Text()
		fmt.Fprintln(o.Out, m)
		if m == "Finished: FAILURE" {
			os.Exit(1)
		}
	}
	e.Wait()
	return err
}
