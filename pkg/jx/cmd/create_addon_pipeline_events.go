package cmd

import (
	"io"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/kube/services"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"fmt"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
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
func NewCmdCreateAddonPipelineEvents(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateAddonPipelineEventsOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					In:      in,

					Out: out,
					Err: errOut,
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
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd, defaultPENamespace, defaultPEReleaseName, defaultPEVersion)

	cmd.Flags().StringVarP(&options.Password, "password", "p", "", "Password to access pipeline-events services such as Kibana and Elasticsearch.  Defaults to default Jenkins X admin password.")
	return cmd
}

// Run implements the command
func (o *CreateAddonPipelineEventsOptions) Run() error {

	if o.ReleaseName == "" {
		return util.MissingOption(optionRelease)
	}

	err := o.ensureHelm()
	if err != nil {
		return errors.Wrap(err, "failed to ensure that helm is present")
	}
	client, err := o.KubeClient()
	if err != nil {
		return err
	}

	devNamespace, _, err := kube.GetDevNamespace(client, o.currentNamespace)
	if err != nil {
		return fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
	}

	log.Infof("found dev namespace %s\n", devNamespace)

	setValues := strings.Split(o.SetValues, ",")
	err = o.installChart(o.ReleaseName, kube.ChartPipelineEvent, o.Version, o.Namespace, true, setValues, nil, "")
	if err != nil {
		return fmt.Errorf("elasticsearch deployment failed: %v", err)
	}

	log.Info("waiting for elasticsearch deployment to be ready, this can take a few minutes\n")

	err = kube.WaitForDeploymentToBeReady(client, esDeploymentName, o.Namespace, 10*time.Minute)
	if err != nil {
		return err
	}
	log.Info("waiting for kibana deployment to be ready, this can take a few minutes\n")

	err = kube.WaitForDeploymentToBeReady(client, kibanaDeploymentName, o.Namespace, 10*time.Minute)
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
	err = o.expose(devNamespace, o.Namespace, o.Password)
	if err != nil {
		return err
	}

	// get the external services URL
	kIng, err := services.GetServiceURLFromName(client, kibanaServiceName, o.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get external URL for service %s: %v", kibanaServiceName, err)
	}

	// get the external services URL
	esIng, err := services.GetServiceURLFromName(client, esServiceName, o.Namespace)
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

	_, err = client.CoreV1().Services(o.currentNamespace).Get(esServiceName, meta_v1.GetOptions{})
	if err != nil {
		// create a services link
		err = services.CreateServiceLink(client, o.currentNamespace, o.Namespace, esServiceName, esIng)
		if err != nil {
			return fmt.Errorf("failed creating a service link for %s in target namespace %s", esServiceName, o.Namespace)
		}
	}

	log.Successf("kibana is available and running %s\n", kIng)
	return nil
}
func (o *CreateAddonPipelineEventsOptions) addExposecontrollerAnnotations(serviceName string) error {
	client, err := o.KubeClient()
	if err != nil {
		return err
	}

	svc, err := client.CoreV1().Services(o.Namespace).Get(serviceName, meta_v1.GetOptions{})
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
		svc, err = client.CoreV1().Services(o.Namespace).Update(svc)
		if err != nil {
			return fmt.Errorf("failed to update service %s/%s", o.Namespace, serviceName)
		}
	}
	return nil
}
