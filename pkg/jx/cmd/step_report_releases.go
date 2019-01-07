package cmd

import (
	"io"
	"time"

	"fmt"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	pe "github.com/jenkins-x/jx/pkg/pipeline_events"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

// StepReportReleasesOptions contains the command line flags
type StepReportReleasesOptions struct {
	StepReportOptions
	Watch bool
	pe.PipelineEventsProvider
}

var (
	StepReportReleasesLong = templates.LongDesc(`
		This pipeline step reports releases to pluggable backends like ElasticSearch
`)

	StepReportReleasesExample = templates.Examples(`
		jx step report Releases
`)
)

func NewCmdStepReportReleases(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := StepReportReleasesOptions{
		StepReportOptions: StepReportOptions{
			StepOptions: StepOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					In:      in,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "releases",
		Short:   "Reports Releases",
		Long:    StepReportReleasesLong,
		Example: StepReportReleasesExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&options.Watch, "watch", "w", false, "Whether to watch Releases")
	options.addCommonFlags(cmd)
	return cmd
}

func (o *StepReportReleasesOptions) Run() error {

	// look up services that we want to send events to using a label?

	// watch Releases and send an event for each backend i.e elasticsearch
	_, _, err := o.KubeClient()
	if err != nil {
		return fmt.Errorf("cannot connect to Kubernetes cluster: %v", err)
	}

	jxClient, _, err := o.JXClient()
	if err != nil {
		return fmt.Errorf("cannot create jx client: %v", err)
	}

	apisClient, err := o.ApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterReleaseCRD(apisClient)
	if err != nil {
		return err
	}

	esServiceName := kube.AddonServices[defaultPEName]
	externalURL, err := o.ensureAddonServiceAvailable(esServiceName)
	if err != nil {
		log.Warnf("no %s service found, are you in your teams dev environment?  Type `jx env` to switch.\n", esServiceName)
		return fmt.Errorf("try running `jx create addon pipeline-events` in your teams dev environment: %v", err)
	}

	server, auth, err := o.CommonOptions.getAddonAuthByKind(kube.ValueKindPipelineEvent, externalURL)
	if err != nil {
		return fmt.Errorf("error getting %s auth details, %v", kube.ValueKindPipelineEvent, err)
	}

	o.PipelineEventsProvider, err = pe.NewElasticsearchProvider(server, auth)
	if err != nil {
		return fmt.Errorf("error creating elasticsearch provider, %v", err)
	}

	if o.Watch {
		err = o.watchPipelineReleases(jxClient, o.currentNamespace)
		if err != nil {
			return err
		}
	}

	err = o.getPipelineReleases(jxClient, o.currentNamespace)
	if err != nil {
		return err
	}

	return nil
}

func (o *StepReportReleasesOptions) getPipelineReleases(jxClient versioned.Interface, ns string) error {
	releases, err := jxClient.JenkinsV1().Releases(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, r := range releases.Items {
		err := o.PipelineEventsProvider.SendRelease(&r)
		if err != nil {
			log.Errorf("%v\n", err)
			return err
		}
	}
	return nil
}

func (o *StepReportReleasesOptions) watchPipelineReleases(jxClient versioned.Interface, ns string) error {

	release := &v1.Release{}
	listWatch := cache.NewListWatchFromClient(jxClient.JenkinsV1().RESTClient(), "releases", metav1.NamespaceAll, fields.Everything())
	kube.SortListWatchByName(listWatch)
	_, controller := cache.NewInformer(
		listWatch,
		release,
		time.Hour*24,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				// send to registered backends
				release, ok := obj.(*v1.Release)
				if !ok {
					log.Errorf("Object is not a Release %#v\n", obj)
					return
				}
				log.Infof("New release added %s\n", release.ObjectMeta.Name)
				err := o.PipelineEventsProvider.SendRelease(release)
				if err != nil {
					log.Errorf("%v\n", err)
					return
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				release, ok := newObj.(*v1.Release)
				if !ok {
					log.Errorf("Object is not a Release %#v\n", newObj)
					return
				}
				log.Infof("Updated release added %s\n", release.ObjectMeta.Name)

				err := o.PipelineEventsProvider.SendRelease(release)
				if err != nil {
					log.Errorf("%v\n", err)
					return
				}
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
