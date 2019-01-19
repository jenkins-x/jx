package cmd

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/jenkins-x/jx/pkg/cloud/iks"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/surveyutils"
	"github.com/jenkins-x/jx/pkg/util"
	survey "gopkg.in/AlecAivazis/survey.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetDomain returns the domain name, trying to infer it either from various Kuberntes resources or cloud provider. If no domain
// can be determined, it will prompt to the user for a value.
func (o *CommonOptions) GetDomain(client kubernetes.Interface, domain string, provider string, ingressNamespace string, ingressService string, externalIP string) (string, error) {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	address := externalIP
	if address == "" {
		if provider == MINIKUBE {
			ip, err := o.getCommandOutput("", "minikube", "ip")
			if err != nil {
				return "", err
			}
			address = ip
		} else if provider == MINISHIFT {
			ip, err := o.getCommandOutput("", "minishift", "ip")
			if err != nil {
				return "", err
			}
			address = ip
		} else {
			info := util.ColorInfo
			log.Infof("Waiting to find the external host name of the ingress controller Service in namespace %s with name %s\n",
				info(ingressNamespace), info(ingressService))
			if provider == KUBERNETES {
				log.Infof("If you are installing Jenkins X on premise you may want to use the '--on-premise' flag or specify the '--external-ip' flags. See: %s\n",
					info("https://jenkins-x.io/getting-started/install-on-cluster/#installing-jenkins-x-on-premise"))
			}
			svc, err := client.CoreV1().Services(ingressNamespace).Get(ingressService, metav1.GetOptions{})
			if err != nil {
				return "", err
			}
			if svc != nil {
				for _, v := range svc.Status.LoadBalancer.Ingress {
					if v.IP != "" {
						address = v.IP
					} else if v.Hostname != "" {
						address = v.Hostname
					}
				}
			}
		}
	}
	defaultDomain := address

	if provider == AWS || provider == EKS {
		if domain != "" {
			err := amazon.RegisterAwsCustomDomain(domain, address)
			return domain, err
		}
		log.Infof("\nOn AWS we recommend using a custom DNS name to access services in your Kubernetes cluster to ensure you can use all of your Availability Zones\n")
		log.Infof("If you do not have a custom DNS name you can use yet, then you can register a new one here: %s\n\n",
			util.ColorInfo("https://console.aws.amazon.com/route53/home?#DomainRegistration:"))

		for {
			if util.Confirm("Would you like to register a wildcard DNS ALIAS to point at this ELB address? ", true,
				"When using AWS we need to use a wildcard DNS alias to point at the ELB host name so you can access services inside Jenkins X and in your Environments.", o.In, o.Out, o.Err) {
				customDomain := ""
				prompt := &survey.Input{
					Message: "Your custom DNS name: ",
					Help:    "Enter your custom domain that we can use to setup a Route 53 ALIAS record to point at the ELB host: " + address,
				}
				survey.AskOne(prompt, &customDomain, nil, surveyOpts)
				if customDomain != "" {
					err := amazon.RegisterAwsCustomDomain(customDomain, address)
					return customDomain, err
				}
			} else {
				break
			}
		}
	}

	if provider == IKS {
		if domain != "" {
			log.Infof("\nIBM Kubernetes Service will use provided domain. Ensure name is registered with DNS (ex. CIS) and pointing the cluster ingress IP: %s\n",
				util.ColorInfo(address))
			return domain, nil
		}
		clusterName, err := iks.GetClusterName()
		clusterRegion, err := iks.GetKubeClusterRegion(client)
		if err == nil && clusterName != "" && clusterRegion != "" {
			customDomain := clusterName + "." + clusterRegion + ".containers.appdomain.cloud"
			log.Infof("\nIBM Kubernetes Service will use the default cluster domain: ")
			log.Infof("%s\n", util.ColorInfo(customDomain))
			return customDomain, nil
		}
		log.Infof("ERROR getting IBM Kubernetes Service will use the default cluster domain:")
		log.Infof(err.Error())
	}

	if address != "" {
		addNip := true
		aip := net.ParseIP(address)
		if aip == nil {
			log.Infof("The Ingress address %s is not an IP address. We recommend we try resolve it to a public IP address and use that for the domain to access services externally.\n",
				util.ColorInfo(address))

			addressIP := ""
			if util.Confirm("Would you like wait and resolve this address to an IP address and use it for the domain?", true,
				"Should we convert "+address+" to an IP address so we can access resources externally", o.In, o.Out, o.Err) {

				log.Infof("Waiting for %s to be resolvable to an IP address...\n", util.ColorInfo(address))
				f := func() error {
					ips, err := net.LookupIP(address)
					if err == nil {
						for _, ip := range ips {
							t := ip.String()
							if t != "" && !ip.IsLoopback() {
								addressIP = t
								return nil
							}
						}
					}
					return fmt.Errorf("Address cannot be resolved yet %s", address)
				}
				o.retryQuiet(5*6, time.Second*10, f)
			}
			if addressIP == "" {
				addNip = false
				log.Infof("Still not managed to resolve address %s into an IP address. Please try figure out the domain by hand\n", address)
			} else {
				log.Infof("%s resolved to IP %s\n", util.ColorInfo(address), util.ColorInfo(addressIP))
				address = addressIP
			}
		}
		if addNip && !strings.HasSuffix(address, ".amazonaws.com") {
			defaultDomain = fmt.Sprintf("%s.nip.io", address)
		}
	}

	if domain == "" {
		if o.BatchMode {
			log.Successf("No domain flag provided so using default %s to generate Ingress rules", defaultDomain)
			return defaultDomain, nil
		}
		log.Successf("You can now configure a wildcard DNS pointing to the new Load Balancer address %s", address)
		log.Info("\nIf you do not have a custom domain setup yet, Ingress rules will be set for magic DNS nip.io.")
		log.Infof("\nOnce you have a custom domain ready, you can update with the command %s", util.ColorInfo("jx upgrade ingress --cluster"))

		log.Infof("\nIf you don't have a wildcard DNS setup then setup a DNS (A) record and point it at: %s then use the DNS domain in the next input...\n", address)

		if domain == "" {
			prompt := &survey.Input{
				Message: "Domain",
				Default: defaultDomain,
				Help:    "Enter your custom domain that is used to generate Ingress rules, defaults to the magic DNS nip.io",
			}
			survey.AskOne(prompt, &domain,
				survey.ComposeValidators(survey.Required, surveyutils.NoWhiteSpaceValidator()), surveyOpts)
		}
		if domain == "" {
			domain = defaultDomain
		}
	} else {
		if domain != defaultDomain {
			log.Successf("You can now configure your wildcard DNS %s to point to %s\n", domain, address)
		}
	}

	return domain, nil
}
