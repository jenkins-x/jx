package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube/pki"
	"github.com/jenkins-x/jx/pkg/kube/services"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	survey "gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	upgradeIngressLong = templates.LongDesc(`
		Upgrades the Jenkins X Ingress rules
`)

	upgradeIngressExample = templates.Examples(`
		# Upgrades the Jenkins X Ingress rules
		jx upgrade ingress
	`)
)

const (
	CertManagerDeployment = "cert-manager"
	CertManagerNamespace  = "cert-manager"
	Exposecontroller      = "exposecontroller"

	certsIssuedReadyTimeout = 5 * time.Minute
)

// UpgradeIngressOptions the options for the create spring command
type UpgradeIngressOptions struct {
	CreateOptions

	SkipCertManager     bool
	Cluster             bool
	Force               bool
	Namespaces          []string
	Version             string
	TargetNamespaces    []string
	Services            []string
	SkipResourcesUpdate bool
	WaitForCerts        bool

	IngressConfig kube.IngressConfig
}

// NewCmdUpgradeIngress defines the command
func NewCmdUpgradeIngress(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	commonOptions := CommonOptions{
		Factory: f,
		In:      in,
		Out:     out,
		Err:     errOut,
	}

	options := &UpgradeIngressOptions{
		CreateOptions: CreateOptions{
			CommonOptions: commonOptions,
		},
	}

	cmd := &cobra.Command{
		Use:     "ingress",
		Short:   "Upgrades Ingress rules",
		Aliases: []string{"ing"},
		Long:    upgradeIngressLong,
		Example: upgradeIngressExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addFlags(cmd)
	options.addCommonFlags(cmd)

	return cmd
}

func (o *UpgradeIngressOptions) addFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&o.Cluster, "cluster", "", false, "Enable cluster wide Ingress upgrade")
	cmd.Flags().StringArrayVarP(&o.Namespaces, "namespaces", "", []string{}, "Namespaces to upgrade")
	cmd.Flags().BoolVarP(&o.SkipCertManager, "skip-certmanager", "", false, "Skips cert-manager installation")
	cmd.Flags().StringArrayVarP(&o.Services, "services", "", []string{}, "Services to upgrade")
	cmd.Flags().BoolVarP(&o.SkipResourcesUpdate, "skip-resources-update", "", false, "Skips the update of jx related resources such as webhook or Jenkins URL")
	cmd.Flags().BoolVarP(&o.Force, "force", "", false, "Forces upgrades of all webooks even if ingress URL has not changed")
	cmd.Flags().BoolVarP(&o.WaitForCerts, "wait-for-certs", "", false, "Waits for TLS certs to be issued by cert-manager")
}

// Run implements the command
func (o *UpgradeIngressOptions) Run() error {
	client, err := o.KubeClient()
	if err != nil {
		return fmt.Errorf("cannot connect to Kubernetes cluster: %v", err)
	}

	o.devNamespace, _, err = kube.GetDevNamespace(client, o.currentNamespace)
	if err != nil {
		return errors.Wrap(err, "getting the dev namesapce")
	}
	previousWebHookEndpoint := ""
	if !o.SkipResourcesUpdate {
		previousWebHookEndpoint, err = o.GetWebHookEndpoint()
		if err != nil {
			return errors.Wrap(err, "getting the webhook endpoint")
		}
	}

	// if existing ingress exist in the namespaces ask do you want to delete them?
	ingressToDelete, err := o.getExistingIngressRules()
	if err != nil {
		return errors.Wrap(err, "getting the existing ingress rules")
	}

	// wizard to ask for config values
	err = o.confirmExposecontrollerConfig()
	if err != nil {
		return errors.Wrap(err, "configure exposecontroller")
	}

	// confirm values
	if !o.BatchMode {
		if !util.Confirm(fmt.Sprintf("Using config values %v, ok?", o.IngressConfig), true, "", o.In, o.Out, o.Err) {
			log.Infof("Terminating\n")
			return nil
		}
	}

	// save details to a configmap
	_, err = kube.SaveAsConfigMap(client, kube.ConfigMapIngressConfig, o.devNamespace, o.IngressConfig)
	if err != nil {
		return errors.Wrap(err, "saving ingress config into a configmap")
	}

	// ensure cert-manager is installed and add the cert-manager to all services which are going to be exposed
	var services []*v1.Service
	if o.IngressConfig.TLS {
		err = o.ensureCertmanagerSetup()
		if err != nil {
			return errors.Wrap(err, "ensure cert-manager setup")
		}

		// annotate any service that has expose=true with correct cert-manager staging / prod annotation
		services, err = o.AnnotateExposedServicesWithCertManager(o.Services...)
		if err != nil {
			return errors.Wrap(err, "annotating the exposed service with cert-manager")
		}
	}

	err = o.CleanServiceAnnotations(o.Services...)
	if err != nil {
		return errors.Wrap(err, "cleaning service annotations")
	}

	// remove the ingress resource in order to allow the ingress-controller to recreate them
	for name, namespace := range ingressToDelete {
		log.Infof("Deleting ingress %s/%s\n", namespace, name)
		err := client.ExtensionsV1beta1().Ingresses(namespace).Delete(name, &metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("cannot delete ingress rule %s in namespace %s: %v", name, namespace, err)
		}
	}

	// start watching and collecting ready certificates
	var notReadyCertsCh <-chan map[pki.Certificate]bool
	ctx, cancel := context.WithTimeout(context.Background(), certsIssuedReadyTimeout)
	defer cancel()
	if o.IngressConfig.TLS && o.WaitForCerts {
		certsCh, err := o.watchReadyCertificates(ctx)
		if err != nil {
			return errors.Wrap(err, "start watching ready certificates")
		}
		notReadyCertsCh = o.startCollectingReadyCertificates(ctx, services, certsCh)
	}

	// run the expose-controller to create the ingress rules
	err = o.createIngressRules()
	if err != nil {
		return errors.Wrap(err, "creating the ingress rules")
	}

	// update all resource dependent to the ingress endpoints
	if !o.SkipResourcesUpdate {
		o.updateResources(previousWebHookEndpoint)
	}

	log.Success("Ingress rules recreated\n")

	if o.IngressConfig.TLS {
		if o.WaitForCerts {
			log.Infoln("Waiting for TLS certificates to be issued...")
			select {
			case certs := <-notReadyCertsCh:
				cancel()
				if len(certs) == 0 {
					log.Success("All TLS certificates are ready\n")
				} else {
					log.Warn("Following TLS certificates are not ready:\n")
					for cert := range certs {
						log.Warnf("%s\n", cert)
					}
					return errors.New("not all TLS certificates are ready")
				}
			case <-ctx.Done():
				log.Warn("Timeout reached while waiting for TLS certificates to be ready\n")
			}
		} else {
			log.Warn("It can take around 5 minutes for Cert Manager to get certificates from Lets Encrypt and update Ingress rules\n")
			log.Info("Use the following commands to diagnose any issues:\n")
			log.Infof("jx logs %s -n %s\n", CertManagerDeployment, CertManagerNamespace)
			log.Info("kubectl describe certificates\n")
			log.Info("kubectl describe issuers\n\n")
		}
	}

	return nil
}

func (o *UpgradeIngressOptions) watchReadyCertificates(ctx context.Context) (<-chan pki.Certificate, error) {
	client, err := o.CreateCertManagerClient()
	if err != nil {
		return nil, errors.Wrap(err, "creating the cert-manager client")
	}

	// watch certificates across all namesapces
	namespace := ""
	certsCh, err := pki.WatchCertificatesIssuedReady(ctx, client, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "start watching certificates")
	}
	return certsCh, nil
}

func (o *UpgradeIngressOptions) startCollectingReadyCertificates(ctx context.Context, services []*v1.Service,
	certsCh <-chan pki.Certificate) <-chan map[pki.Certificate]bool {
	resultCh := make(chan map[pki.Certificate]bool)
	go func() {
		certs := pki.ToCertificates(services)
		certsMap := make(map[pki.Certificate]bool)
		for _, cert := range certs {
			certsMap[cert] = true
		}

		for {
			select {
			case cert := <-certsCh:
				delete(certsMap, cert)
				// check if all expected certificates are received
				if len(certsMap) == 0 {
					// send a map with no certificates to indicate success
					resultCh <- certsMap
					return
				}
			case <-ctx.Done():
				// send the current state of the certificates map
				resultCh <- certsMap
				return
			}
		}
	}()
	return resultCh
}

func (o *UpgradeIngressOptions) updateResources(previousWebHookEndpoint string) error {
	_, _, err := o.JXClient()
	if err != nil {
		return errors.Wrap(err, "failed to get jxclient")
	}

	isProwEnabled, err := o.isProw()
	if err != nil {
		return errors.Wrap(err, "checking if is prow")
	}

	if !isProwEnabled {
		err = o.updateJenkinsURL(o.TargetNamespaces)
		if err != nil {
			return errors.Wrap(err, "upgrade jenkins URL")
		}
	}

	updatedWebHookEndpoint, err := o.GetWebHookEndpoint()
	if err != nil {
		return errors.Wrap(err, "retrieving the webhook endpoint")
	}

	log.Infof("Previous webhook endpoint %s\n", previousWebHookEndpoint)
	log.Infof("Updated webhook endpoint %s\n", updatedWebHookEndpoint)
	updateWebHooks := true
	if !o.BatchMode {
		if !util.Confirm("Do you want to update all existing webhooks?", true, "", o.In, o.Out, o.Err) {
			updateWebHooks = false
		}
	}

	if updateWebHooks {
		o.updateWebHooks(previousWebHookEndpoint, updatedWebHookEndpoint)
	}
	return nil
}

func (o *UpgradeIngressOptions) isIngressForServices(ingress *v1beta1.Ingress) bool {
	services := o.Services
	if len(services) == 0 {
		// allow all ingresses if no services filter is defined
		return true
	}
	rules := ingress.Spec.Rules
	for _, rule := range rules {
		http := rule.IngressRuleValue.HTTP
		if http == nil {
			continue
		}
		for _, path := range http.Paths {
			service := path.Backend.ServiceName
			i := util.StringArrayIndex(services, service)
			if i >= 0 {
				return true
			}
		}
	}
	return false
}

func (o *UpgradeIngressOptions) getExistingIngressRules() (map[string]string, error) {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	existingIngressNames := map[string]string{}
	client, err := o.KubeClient()
	if err != nil {
		return existingIngressNames, err
	}
	var confirmMessage string
	if o.Cluster {
		confirmMessage = "Existing ingress rules found in the cluster.  Confirm to delete all and recreate them"

		ings, err := client.ExtensionsV1beta1().Ingresses("").List(metav1.ListOptions{})
		if err != nil {
			return existingIngressNames, fmt.Errorf("cannot list all ingresses in cluster: %v", err)
		}
		for _, i := range ings.Items {
			if i.Annotations[services.ExposeGeneratedByAnnotation] == Exposecontroller {
				if o.isIngressForServices(&i) {
					existingIngressNames[i.Name] = i.Namespace
				}
			}
		}

		nsList, err := client.CoreV1().Namespaces().List(metav1.ListOptions{})
		for _, n := range nsList.Items {
			o.TargetNamespaces = append(o.TargetNamespaces, n.Name)
		}

	} else if len(o.Namespaces) > 0 {
		confirmMessage = fmt.Sprintf("Existing ingress rules found in namespaces %v namespace.  Confirm to delete and recreate them", o.Namespaces)
		// loop round each
		for _, n := range o.Namespaces {
			ings, err := client.ExtensionsV1beta1().Ingresses(n).List(metav1.ListOptions{})
			if err != nil {
				return existingIngressNames, fmt.Errorf("cannot list all ingresses in cluster: %v", err)
			}
			for _, i := range ings.Items {
				if i.Annotations[services.ExposeGeneratedByAnnotation] == Exposecontroller {
					if o.isIngressForServices(&i) {
						existingIngressNames[i.Name] = i.Namespace
					}
				}
			}
			o.TargetNamespaces = append(o.TargetNamespaces, n)
		}
	} else {
		confirmMessage = "Existing ingress rules found in current namespace.  Confirm to delete and recreate them"
		// fall back to current ns only
		log.Infof("looking for existing ingress rules in current namespace %s\n", o.currentNamespace)

		ings, err := client.ExtensionsV1beta1().Ingresses(o.currentNamespace).List(metav1.ListOptions{})
		if err != nil {
			return existingIngressNames, fmt.Errorf("cannot list all ingresses in cluster: %v", err)
		}
		for _, i := range ings.Items {
			if i.Annotations[services.ExposeGeneratedByAnnotation] == Exposecontroller {
				if o.isIngressForServices(&i) {
					existingIngressNames[i.Name] = i.Namespace
				}
			}
		}
		o.TargetNamespaces = append(o.TargetNamespaces, o.currentNamespace)
	}

	if len(existingIngressNames) == 0 {
		return existingIngressNames, nil
	}

	confirm := &survey.Confirm{
		Message: confirmMessage,
		Default: true,
	}
	flag := true
	err = survey.AskOne(confirm, &flag, nil, surveyOpts)
	if err != nil {
		return existingIngressNames, err
	}
	if !flag {
		return existingIngressNames, errors.New("Not able to automatically delete existing ingress rules.  Either delete manually or change the scope the command should run in")
	}

	return existingIngressNames, nil
}

func (o *UpgradeIngressOptions) confirmExposecontrollerConfig() error {
	// get current ingress config to use as existing defaults
	client, err := o.KubeClient()
	if err != nil {
		return err
	}
	devNamespace, _, err := kube.GetDevNamespace(client, o.currentNamespace)
	if err != nil {
		return fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
	}

	// Overwrites the ingress config with the values from config map only if this config map exists
	ic, err := kube.GetIngressConfig(client, devNamespace)
	if err == nil {
		o.IngressConfig = ic
	}

	o.IngressConfig.Exposer, err = util.PickNameWithDefault([]string{"Ingress", "Route"}, "Expose type", o.IngressConfig.Exposer, "", o.In, o.Out, o.Err)
	if err != nil {
		return err
	}

	o.IngressConfig.Domain, err = util.PickValue("Domain:", o.IngressConfig.Domain, true, "", o.In, o.Out, o.Err)
	if err != nil {
		return err
	}

	if !strings.HasSuffix(o.IngressConfig.Domain, "nip.io") {

		o.IngressConfig.TLS = util.Confirm("If your network is publicly available would you like to enable cluster wide TLS?", true, "Enables cert-manager and configures TLS with signed certificates from LetsEncrypt", o.In, o.Out, o.Err)

		if o.IngressConfig.TLS {
			log.Infof("If testing LetsEncrypt you should use staging as you may be rate limited using production.")
			clusterIssuer, err := util.PickNameWithDefault([]string{"staging", "production"}, "Use LetsEncrypt staging or production?", "production", "", o.In, o.Out, o.Err)
			// if the cluster issuer is production the string needed by letsencrypt is prod
			if clusterIssuer == "production" {
				clusterIssuer = "prod"
			}
			if err != nil {
				return err
			}
			o.IngressConfig.Issuer = "letsencrypt-" + clusterIssuer

			if o.IngressConfig.Email == "" {
				email1, err := o.getCommandOutput("", "git", "config", "user.email")
				if err != nil {
					return err
				}

				o.IngressConfig.Email = strings.TrimSpace(email1)
			}

			o.IngressConfig.Email, err = util.PickValue("Email address to register with LetsEncrypt:", o.IngressConfig.Email, true, "", o.In, o.Out, o.Err)
			if err != nil {
				return err
			}
		}
	}

	return nil

}
func (o *UpgradeIngressOptions) createIngressRules() error {
	client, err := o.KubeClient()
	if err != nil {
		return err
	}
	devNamespace, _, err := kube.GetDevNamespace(client, o.currentNamespace)
	if err != nil {
		return fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
	}
	for _, n := range o.TargetNamespaces {
		o.CleanExposecontrollerReources(n)

		err := pki.CleanCertSecrets(client, n)
		if err != nil {
			return err
		}

		err = kube.CleanCertmanagerResources(client, n, o.IngressConfig)
		if err != nil {
			return err
		}

		err = o.runExposecontroller(devNamespace, n, o.IngressConfig, o.Services...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *UpgradeIngressOptions) ensureCertmanagerSetup() error {
	if !o.SkipCertManager {
		return o.ensureCertmanager()
	}
	return nil
}

// AnnotateExposedServicesWithCertManager annotates exposed services with cert manager
func (o *UpgradeIngressOptions) AnnotateExposedServicesWithCertManager(svcs ...string) ([]*v1.Service, error) {
	result := make([]*v1.Service, 0)
	client, err := o.KubeClient()
	if err != nil {
		return result, err
	}
	for _, n := range o.TargetNamespaces {
		issuer := o.IngressConfig.Issuer
		if issuer == "" {
			return result, fmt.Errorf("no issuer was configured for cert manager")
		}
		services, err := services.AnnotateServicesWithCertManagerIssuer(client, n, issuer, svcs...)
		if err != nil {
			return result, err
		}
		result = append(result, services...)
	}
	return result, nil
}

// CleanServiceAnnotations cleans service annotations
func (o *UpgradeIngressOptions) CleanServiceAnnotations(svcs ...string) error {
	client, err := o.KubeClient()
	if err != nil {
		return err
	}
	for _, n := range o.TargetNamespaces {
		err := services.CleanServiceAnnotations(client, n, svcs...)
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *UpgradeIngressOptions) updateWebHooks(oldHookEndpoint string, newHookEndpoint string) error {
	if oldHookEndpoint == newHookEndpoint && !o.Force {
		log.Infof("Webhook URL unchanged. Use %s to force updating\n", util.ColorInfo("--force"))
		return nil
	}

	log.Infof("Updating all webHooks from %s to %s\n", util.ColorInfo(oldHookEndpoint), util.ColorInfo(newHookEndpoint))

	updateWebHook := UpdateWebhooksOptions{
		CommonOptions: o.CommonOptions,
	}

	authConfigService, err := o.CreateGitAuthConfigService()
	if err != nil {
		return errors.Wrap(err, "failed to create git auth service")
	}

	gitServer := authConfigService.Config().CurrentServer
	git, err := o.gitProviderForGitServerURL(gitServer, "github")
	if err != nil {
		return errors.Wrap(err, "unable to determine git provider")
	}

	// organisation
	organisation, err := gits.PickOrganisation(git, "", o.In, o.Out, o.Err)
	if err != nil {
		return errors.Wrap(err, "unable to determine git provider")
	}

	updateWebHook.PreviousHookUrl = oldHookEndpoint
	updateWebHook.Org = organisation
	updateWebHook.DryRun = false

	return updateWebHook.Run()
}
