package cmd

import (
	"io"

	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

// StepReportActivitiesOptions contains the command line flags
type StepReportActivitiesOptions struct {
	StepReportOptions
	Watch bool
}

var (
	StepReportActivitiesLong = templates.LongDesc(`
		This pipeline step reports activities to pluggable backends like ElasticSearch
`)

	StepReportActivitiesExample = templates.Examples(`
		jx step report activities
`)
)

func NewCmdStepReportActivities(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := StepReportActivitiesOptions{
		StepReportOptions: StepReportOptions{
			StepOptions: StepOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "activities",
		Short:   "Reports activities",
		Long:    StepReportActivitiesLong,
		Example: StepReportActivitiesExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&options.Watch, "watch", "w", false, "Whether to watch activities")
	return cmd
}

func (o *StepReportActivitiesOptions) Run() error {

	// look up services that we want to send events to using a label?

	// watch activities and send an event for each backend i.e elasticsearch
	f := o.Factory
	client, currentNs, err := f.CreateJXClient()
	if err != nil {
		return err
	}
	kubeClient, _, err := o.Factory.CreateClient()
	if err != nil {
		return err
	}
	ns, _, err := kube.GetDevNamespace(kubeClient, currentNs)
	if err != nil {
		return err
	}
	envList, err := client.JenkinsV1().Environments(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	kube.SortEnvironments(envList.Items)

	apisClient, err := f.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterPipelineActivityCRD(apisClient)
	if err != nil {
		return err
	}

	err = o.watchPipelineActivities(client, o.currentNamespace)
	if err != nil {
		return err
	}

	return nil
}

func (o *StepReportActivitiesOptions) watchPipelineActivities(jxClient *versioned.Clientset, ns string) error {

	activity := &v1.PipelineActivity{}
	listWatch := cache.NewListWatchFromClient(jxClient.JenkinsV1().RESTClient(), "pipelineactivities", ns, fields.Everything())
	kube.SortListWatchByName(listWatch)
	_, controller := cache.NewInformer(
		listWatch,
		activity,
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				// send to registered backends
				activity, ok := obj.(*v1.PipelineActivity)
				if !ok {
					o.Printf("Object is not a PipelineActivity %#v\n", obj)
					return
				}
				log.Infof("New activity added %s\n", activity.ObjectMeta.Name)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				activity, ok := newObj.(*v1.PipelineActivity)
				if !ok {
					o.Printf("Object is not a PipelineActivity %#v\n", newObj)
					return
				}
				log.Infof("Updated activity added %s\n", activity.ObjectMeta.Name)
			},
			DeleteFunc: func(obj interface{}) {
				// no need to send event
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	// Wait forever
	select {}
}
