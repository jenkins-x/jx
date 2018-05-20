package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SyncOptions struct {
	CommonOptions

	Container   string
	Namespace   string
	Pod         string
	Dir         string
	RemoteDir   string
	Daemon      bool
	Reload      bool
	NoKsyncInit bool
	WatchOnly   bool

	stopCh chan struct{}
}

var (
	sync_long = templates.LongDesc(`
		Synchronises your local files to a DevPod so you an build and test your code easily on the cloud

		For more documentation see: [http://jenkins-x.io/developing/devpods/](http://jenkins-x.io/developing/devpods/)

`)

	sync_example = templates.Examples(`
		# Starts synchonizing the current directory files to the users DevPod
		jx sync 
`)

	defaultStignoreFile = `.git
bin
build
`
)

func NewCmdSync(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &SyncOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "sync",
		Short:   "Synchronises your local files to a devpod",
		Long:    sync_long,
		Example: sync_example,
		Aliases: []string{"log"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Container, "container", "c", "", "The name of the container to log")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "the namespace to look for the Deployment. Defaults to the current namespace")
	cmd.Flags().StringVarP(&options.Pod, "pod", "p", "", "the pod name to use")
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "The directory to watch. Defaults to the current directory")
	cmd.Flags().StringVarP(&options.RemoteDir, "remote-dir", "r", "", "The remote directory in the DevPod to sync")
	cmd.Flags().BoolVarP(&options.Daemon, "daemon", "", false, "Runs ksync in a background daemon")
	cmd.Flags().BoolVarP(&options.Reload, "reload", "", false, "Should we reload the remote container on file changes?")
	cmd.Flags().BoolVarP(&options.WatchOnly, "watch-only", "", false, "Should we just run the `ksync watch` command only")
	cmd.Flags().BoolVarP(&options.NoKsyncInit, "no-init", "", false, "Disables the use of 'ksync init' to ensure we have initialised ksync")
	return cmd
}

func (o *SyncOptions) Run() error {
	client, curNs, err := o.KubeClient()
	if err != nil {
		return err
	}
	ns, _, err := kube.GetDevNamespace(client, curNs)
	if err != nil {
		return err
	}
	u, err := user.Current()
	if err != nil {
		return err
	}
	_, err = o.installKSync()
	if err != nil {
		return err
	}

	if !o.NoKsyncInit {
		flag, err := kube.IsDaemonSetExists(client, "ksync", "kube-system")
		if !flag || err != nil {
			o.Printf("Initialising ksync\n")
			err = o.runCommandInteractive(true, "ksync", "init", "--upgrade")
			if err != nil {
				return err
			}
		}
	}

	args := []string{"watch"}
	if o.Daemon {
		args = append(args, "--daemon")
	}
	cmd := exec.Command("ksync", args...)
	cmd.Stdout = o.Out
	cmd.Stderr = o.Out
	err = cmd.Start()
	if err != nil {
		return err
	}

	o.Printf("Started the ksync watch\n")
	time.Sleep(1 * time.Second)

	state := cmd.ProcessState
	if state != nil && state.Exited() {
		return fmt.Errorf("ksync watch terminated")
	}

	if !o.WatchOnly {
		name := o.Pod
		username := u.Username
		names, pods, err := kube.GetPods(client, ns, username)
		if err != nil {
			return err
		}
		info := util.ColorInfo
		if len(names) == 0 {
			return fmt.Errorf("There are no DevPods for user %s in namespace %s. You can create one via: %s\n", info(username), info(ns), info("jx create devpod"))
		}

		if name == "" {
			n, err := util.PickName(names, "Pick Pod:")
			if err != nil {
				return err
			}
			name = n
		} else if util.StringArrayIndex(names, name) < 0 {
			return util.InvalidOption(optionLabel, name, names)
		}

		if name == "" {
			return fmt.Errorf("No pod found for namespace %s with name %s", ns, name)
		}

		dir := o.Dir
		if dir == "" {
			dir, err = os.Getwd()
			if err != nil {
				return err
			}
		}
		remoteDir := o.RemoteDir
		if remoteDir == "" {
			pod := pods[name]
			if pod == nil {
				return fmt.Errorf("Pod %s does not exist!", name)
			}
			ann := pod.Annotations
			if ann != nil {
				remoteDir = ann[kube.AnnotationWorkingDir]
			}
			if remoteDir == "" {
				o.warnf("Missing annotation %s on pod %s", kube.AnnotationWorkingDir, name)
				remoteDir = "/code"
			}
		}

		err = o.CreateKsync(client, ns, name, dir, remoteDir, username)
		if err != nil {
			o.killWatchProcess(cmd)
			return err
		}
	}

	return cmd.Wait()
}

// CreateKsync removes the exiting ksync if it already exists then create a new ksync of the given name
func (o *SyncOptions) CreateKsync(client *kubernetes.Clientset, ns string, name string, dir string, remoteDir string, username string) error {
	info := util.ColorInfo
	o.Printf("synchronizing directory %s to DevPod %s path %s\n", info(dir), info(name), info(remoteDir))

	ignoreFile := filepath.Join(dir, ".stignore")
	exists, err := util.FileExists(ignoreFile)
	if err != nil {
		return err
	}
	if !exists {
		err = ioutil.WriteFile(ignoreFile, []byte(defaultStignoreFile), DefaultWritePermissions)
		if err != nil {
			return err
		}
	}
	matchLabels := map[string]string{
		kube.LabelPodTemplate:    name,
		kube.LabelDevPodUsername: username,
	}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: matchLabels})
	if err != nil {
		return err
	}

	_, pods, err := kube.GetPodsWithLabels(client, ns, selector.String())
	if err != nil {
		return err
	}

	deleteNames := []string{}
	err = o.retry(5, time.Second, func() error {
		text, err := o.getCommandOutput(dir, "ksync", "get")
		if err == nil {
			for i, line := range strings.Split(text, "\n") {
				if i > 1 {
					cols := strings.Split(strings.TrimSpace(line), " ")
					if len(cols) > 0 {
						n := strings.TrimSpace(cols[0])
						if n == name || pods[n] == nil {
							if util.StringArrayIndex(deleteNames, n) < 0 && n != "starting" {
								deleteNames = append(deleteNames, n)
							}
						}
					}
				}
			}
		}
		return err
	})
	if err != nil {
		o.warnf("Failed to get from ksync daemon: %s\n", err)
	}

	reload := "--reload=false"
	if o.Reload {
		reload = "--reload=true"
	}

	for _, n := range deleteNames {
		// ignore results as we may not have a spec yet for this name
		o.Printf("Removing old ksync %s\n", n)

		o.runCommand("ksync", "delete", n)
	}

	time.Sleep(1 * time.Second)

	return o.runCommand("ksync", "create", "--name", name, "-l", "jenkins.io/devpod="+name, reload, "-n", ns, dir, remoteDir)
}

func (o *SyncOptions) killWatchProcess(cmd *exec.Cmd) {
	if err := cmd.Process.Kill(); err != nil {
		o.warnf("failed to kill 'ksync watch' process: %s\n", err)
	}
}
