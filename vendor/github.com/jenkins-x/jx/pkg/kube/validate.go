package kube

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	OptionName      = "name"
	OptionNamespace = "namespace"
)

func ValidateSubDomain(val interface{}) error {
	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("Expected some text!")
	}
	if strings.TrimSpace(str) == "" {
		return fmt.Errorf("Value is required")
	}
	errors := validation.IsDNS1123Subdomain(str)
	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "/n"))
	}
	return nil
}

func ValidateName(val interface{}) error {
	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("Expected some text!")
	}
	if strings.TrimSpace(str) == "" {
		return fmt.Errorf("Value is required")
	}
	errors := validation.IsDNS1123Label(str)
	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "/n"))
	}
	return nil
}

func ValidSubDomainOption(option string, value string) error {
	if value != "" {
		errors := validation.IsDNS1123Subdomain(value)
		if len(errors) > 0 {
			return util.InvalidOptionf(option, value, strings.Join(errors, "/n"))
		}
	}
	return nil
}

func ValidNameOption(option string, value string) error {
	if value != "" {
		errors := validation.IsDNS1123Label(value)
		if len(errors) > 0 {
			return util.InvalidOptionf(option, value, strings.Join(errors, "/n"))
		}
	}
	return nil
}

func ValidateEnvironmentDoesNotExist(jxClient *versioned.Clientset, ns string, str string) error {
	if str != "" {
		_, err := jxClient.JenkinsV1().Environments(ns).Get(str, metav1.GetOptions{})
		if err == nil {
			return fmt.Errorf("Environment %s already exists!", str)
		}
	}
	return nil
}
