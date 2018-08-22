package cmd

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
	"io"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"time"
	"golang.org/x/build/kubernetes/api"
)

// ControllerTeamOptions are the flags for the commands
type ControllerTeamOptions struct {
	ControllerOptions
	InstallOptions

	GitRepositoryOptions gits.GitRepositoryOptions
}

// NewCmdControllerTeam creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdControllerTeam(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ControllerTeamOptions{
		ControllerOptions: ControllerOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
		InstallOptions: createInstallOptions(f, out, errOut),
	}

	cmd := &cobra.Command{
		Use:   "team",
		Short: "Runs the team controller",
		Run: func(cmd *cobra.Command, args []string) {
			options.ControllerOptions.Cmd = cmd
			options.ControllerOptions.Args = args
			err := options.Run()
			CheckErr(err)
		},
		Aliases: []string{"team"},
	}

	options.ControllerOptions.addCommonFlags(cmd)
	options.InstallOptions.addInstallFlags(cmd, true)

	return cmd
}

// Run implements this command
func (o *ControllerTeamOptions) Run() error {
	err := o.ControllerOptions.registerTeamCRD()
	if err != nil {
		return err
	}

	jxClient, _, err := o.ControllerOptions.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	log.Infof("Watching for teams in all namespaces\n")

	stop := make(chan struct{})

	_, teamController := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(lo meta_v1.ListOptions) (runtime.Object, error) {
				return jxClient.JenkinsV1().Teams(api.NamespaceAll).List(lo)
			},
			WatchFunc: func(lo meta_v1.ListOptions) (watch.Interface, error) {
				return jxClient.JenkinsV1().Teams(api.NamespaceAll).Watch(lo)
			},
		},
		&v1.Team{},
		time.Minute*30,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				o.onTeamChange(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				o.onTeamChange(newObj)
			},
			DeleteFunc: func(obj interface{}) {
				//o.onEnvironmentChange(obj, jxClient, ns)
			},
		},
	)

	go teamController.Run(stop)

	// Wait forever
	select {}
}

func (o *ControllerTeamOptions) onTeamChange(obj interface{}) {
	team, ok := obj.(*v1.Team)
	if !ok {
		log.Infof("Object is not a Team %#v\n", obj)
		return
	}

	log.Infof("Found Team %#v\n", team)
}

