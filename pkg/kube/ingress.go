package kube

import (
	"fmt"

	"strconv"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	IngressConfigConfigmap = "ingress-config"
	Domain                 = "domain"
	Email                  = "email"
	TLS                    = "tls"
	Issuer                 = "issuer"
	Exposer                = "exposer"
)

type IngressConfig struct {
	Email   string `structs:"email" yaml:"email" json:"email"`
	Domain  string `structs:"domain" yaml:"domain" json:"domain"`
	Issuer  string `structs:"issuer" yaml:"issuer" json:"issuer"`
	Exposer string `structs:"exposer" yaml:"exposer" json:"exposer"`
	TLS     bool   `structs:"tls" yaml:"tls" json:"tls"`
}

func GetIngress(client kubernetes.Interface, ns, name string) (string, error) {

	ing, err := client.ExtensionsV1beta1().Ingresses(ns).Get(name, meta_v1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get ingress rule %s. error: %v", name, err)
	}
	if ing == nil {
		return "", fmt.Errorf("failed to find ingress rule %s", name)
	}

	// default to the first rule
	if len(ing.Spec.Rules) > 0 {
		return ing.Spec.Rules[0].Host, nil
	}
	return "", fmt.Errorf("no hostname found for ingress rule %s", name)
}

func GetIngressConfig(c kubernetes.Interface, ns string) (IngressConfig, error) {

	var ic IngressConfig

	cm, err := c.CoreV1().ConfigMaps(ns).Get(IngressConfigConfigmap, meta_v1.GetOptions{})
	if err != nil {
		return ic, err
	}

	ic.Domain = cm.Data[Domain]
	ic.Email = cm.Data[Email]
	ic.Exposer = cm.Data[Exposer]
	ic.Issuer = cm.Data[Issuer]
	tls, exists := cm.Data[TLS]

	if exists {
		ic.TLS, err = strconv.ParseBool(tls)
		if err != nil {
			return ic, fmt.Errorf("failed to parse TLS string %s to bool from %s: %v", tls, IngressConfigConfigmap, err)
		}
	} else {
		ic.TLS = false
	}
	return ic, nil
}
