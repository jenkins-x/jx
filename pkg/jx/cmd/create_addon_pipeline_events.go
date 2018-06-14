package cmd

import (
	"io"
	"time"

	"github.com/spf13/cobra"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"fmt"

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
)

const (
	defaultPEName        = "pipeline-events"
	defaultPENamespace   = "pipeline-events"
	defaultPEReleaseName = "jx-pipeline-events"
	defaultPEVersion     = "0.0.11"
	kibanaServiceName    = "jx-pipeline-events-kibana"
	kibanaDeploymentName = "jx-pipeline-events-kibana"
	esDeploymentName     = "jx-pipeline-events-elasticsearch-client"
)

var (
	createAddonPipelineEventsLong = templates.LongDesc(`
		Creates the Jenkins X pipeline events addon
`)

	createAddonPipelineEventsExample = templates.Examples(`
		# Create the pipeline-events addon
		jx create addon pipeline-events

		# Create the pipeline-events addon in a custom namespace
		jx create addon pipeline-events -n mynamespace
	`)
)

// CreateAddonPipelineEventsOptions the options for the create spring command
type CreateAddonPipelineEventsOptions struct {
	CreateAddonOptions
	Password string
}

// NewCmdCreateAddonPipelineEvents creates a command object for the "create" command
func NewCmdCreateAddonPipelineEvents(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateAddonPipelineEventsOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "pipeline-events",
		Short:   "Create the pipeline events addon",
		Aliases: []string{"pe"},
		Long:    createAddonPipelineEventsLong,
		Example: createAddonPipelineEventsExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd, defaultPENamespace, defaultPEReleaseName)

	cmd.Flags().StringVarP(&options.Version, "version", "v", defaultPEVersion, "The version of the pipeline events chart to use")
	cmd.Flags().StringVarP(&options.Password, "password", "p", "", "Password to access pipeline-events services such as Kibana and Elasticsearch.  Defaults to default Jenkins X admin password.")
	return cmd
}

// Run implements the command
func (o *CreateAddonPipelineEventsOptions) Run() error {

	if o.ReleaseName == "" {
		return util.MissingOption(optionRelease)
	}

	_, _, err := o.KubeClient()
	if err != nil {
		return err
	}

	devNamespace, _, err := kube.GetDevNamespace(o.kubeClient, o.currentNamespace)
	if err != nil {
		return fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
	}

	log.Infof("found dev namespace %s\n", devNamespace)

	//values := []string{"globalConfig.users.admin.password=" + o.Password, "globalConfig.configDir=/anchore_service_dir"}
	err = o.installChart(o.ReleaseName, kube.ChartPipelineEvent, o.Version, o.Namespace, true, []string{})
	if err != nil {
		return fmt.Errorf("elasticsearch deployment failed: %v", err)
	}

	log.Info("waiting for elasticsearch deployment to be ready, this can take a few minutes\n")

	err = kube.WaitForDeploymentToBeReady(o.kubeClient, esDeploymentName, o.Namespace, 10*time.Minute)
	if err != nil {
		return err
	}
	log.Info("waiting for kibana deployment to be ready, this can take a few minutes\n")

	err = kube.WaitForDeploymentToBeReady(o.kubeClient, kibanaDeploymentName, o.Namespace, 10*time.Minute)
	if err != nil {
		return err
	}

	// annotate the kibana and elasticsearch services so exposecontroller can create an ingress rule
	err = o.addExposecontrollerAnnotations(kibanaServiceName)
	if err != nil {
		return err
	}

	esServiceName := kube.AddonServices[defaultPEName]
	err = o.addExposecontrollerAnnotations(esServiceName)
	if err != nil {
		return err
	}

	if o.Password == "" {
		o.Password, err = o.getDefaultAdminPassword(devNamespace)
		if err != nil {
			return err
		}
	}
	// create the ingress rule
	err = o.expose(devNamespace, o.Namespace, defaultPEReleaseName, o.Password)
	if err != nil {
		return err
	}

	// get the external service URL
	kIng, err := kube.GetServiceURLFromName(o.kubeClient, kibanaServiceName, o.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get external URL for service %s: %v", kibanaServiceName, err)
	}

	// get the external service URL
	esIng, err := kube.GetServiceURLFromName(o.kubeClient, esServiceName, o.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get external URL for service %s: %v", kibanaServiceName, err)
	}

	// create the local addonAuth.yaml file so `jx get cve` commands work
	tokenOptions := CreateTokenAddonOptions{
		Password: o.Password,
		Username: "admin",
		ServerFlags: ServerFlags{
			ServerURL:  esIng,
			ServerName: esDeploymentName,
		},
		Kind: kube.ValueKindPipelineEvent,
		CreateOptions: CreateOptions{
			CommonOptions: o.CommonOptions,
		},
	}
	err = tokenOptions.Run()
	if err != nil {
		return fmt.Errorf("failed to create addonAuth.yaml error: %v", err)
	}

	_, err = o.kubeClient.CoreV1().Services(o.currentNamespace).Get(esServiceName, meta_v1.GetOptions{})
	if err != nil {
		// create a service link
		err = kube.CreateServiceLink(o.kubeClient, o.currentNamespace, o.Namespace, esServiceName, esIng)
		if err != nil {
			return fmt.Errorf("failed creating a service link for %s in target namespace %s", esServiceName, o.Namespace)
		}
	}

	log.Successf("kibana is available and running %s\n", kIng)
	return nil
}
func (o *CreateAddonPipelineEventsOptions) addExposecontrollerAnnotations(serviceName string) error {

	svc, err := o.kubeClient.CoreV1().Services(o.Namespace).Get(serviceName, meta_v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get Service %s: %v", serviceName, err)
	}
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}

	annotationsUpdated := false
	if svc.Annotations[kube.AnnotationExpose] == "" {
		svc.Annotations[kube.AnnotationExpose] = "true"
		annotationsUpdated = true
	}
	if svc.Annotations[kube.AnnotationIngress] == "" {
		svc.Annotations[kube.AnnotationIngress] = "nginx.ingress.kubernetes.io/auth-type: basic\nnginx.ingress.kubernetes.io/auth-secret: jx-basic-auth"
		annotationsUpdated = true
	}
	if annotationsUpdated {
		svc, err = o.kubeClient.CoreV1().Services(o.Namespace).Update(svc)
		if err != nil {
			return fmt.Errorf("failed to update service %s/%s", o.Namespace, serviceName)
		}
	}
	return nil
}
