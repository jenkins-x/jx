package cmd

import (
	"fmt"

	"github.com/jenkins-x/jx/v2/pkg/packages"

	"github.com/jenkins-x/jx/v2/pkg/util/system"

	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"

	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/health"
	version2 "github.com/jenkins-x/jx/v2/pkg/version"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/v2/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/spf13/cobra"
)

func getValidArgs() []string {
	return []string{"", "version", "status", "pvc", "pods", "ingresses", "secrets", "configmaps", "health"}
}

var jxDiagnoseExample = templates.Examples(fmt.Sprintf(`
	# To print diagnostic information about pods in n1 namespace
	jx diagnose pods -n n1
	Supported arguments to diagnose are %v
	
	Deprecated usage:
	# To print all information
	jx diagnose
	
	#To print specific resource information 
	jx diagnose --show=pods --show=version
`, getValidArgs()))

type DiagnoseOptions struct {
	*opts.CommonOptions
	Namespace string
	HelmTLS   bool
	//Deprecated
	Show   []string
	Runner util.Commander
}

func NewCmdDiagnose(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &DiagnoseOptions{
		CommonOptions: commonOpts,
	}
	cmd := &cobra.Command{
		Use:       "diagnose ARG",
		Short:     "Print diagnostic information about the Jenkins X installation",
		Example:   jxDiagnoseExample,
		ValidArgs: getValidArgs(),
		//Todo: This will be removed by cobra.ExactValidArgs(1) after --show flag is removed
		Args: cobra.OnlyValidArgs,
		Run: func(cmd *cobra.Command, args []string) {
			path, err := util.JXBinLocation()
			helper.CheckErr(err)
			runner := &util.Command{
				Name: fmt.Sprintf("%s/kubectl", path),
			}
			options.Runner = runner
			options.Cmd = cmd
			options.Args = args
			err = options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "The namespace to display the kube resources from. If left out, defaults to the current namespace")
	cmd.Flags().StringArrayVarP(&options.Show, "show", "", []string{"version", "status", "pvc", "pods", "ingresses", "secrets", "configmaps"}, "Determine what information to diagnose")
	cmd.Flags().BoolVarP(&options.HelmTLS, "helm-tls", "", false, "Whether to use TLS with helm")
	_ = cmd.Flags().MarkDeprecated("show", "use jx diagnose <object> instead. This will be removed on July 1, 2020")
	return cmd
}

func (o *DiagnoseOptions) Run() error {
	// Get the namespace to run the diagnostics in, and output it
	kubeClient, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return errors.Wrap(err, "failed to create kubeClient")
	}

	// Install kubectl
	err = packages.InstallKubectl(false)
	if err != nil {
		return err
	}

	log.Logger().Infof("Running in namespace: %s", util.ColorInfo(ns))
	_, table := o.GetPackageVersions(ns, o.HelmTLS)

	// os version
	osVersion, err := o.GetOsVersion()
	if err != nil {
		log.Logger().Warnf("Failed to get OS version: %s", err)
	} else {
		table.AddRow("Operating System", util.ColorInfo(osVersion))
	}

	table.Render()

	switch len(o.Args) {
	//ToDo: This wont be required after the deprecated --show flag is removed
	case 0:
		if o.showOption("version") {
			version := version2.GetVersion()
			_, err := fmt.Fprintln(o.CommonOptions.Out, util.ColorInfo(version))
			if err != nil {
				return err
			}
		}

		if o.showOption("status") {
			err := NewCmdStatus(o.CommonOptions).Execute()
			if err != nil {
				return err
			}
		}

		if o.showOption("pvc") {
			err := printStatus(o, "Kubernetes PVCs", "kubectl", "get", "pvc", "--namespace", ns)
			if err != nil {
				return err
			}
		}

		if o.showOption("pods") {
			err := printStatus(o, "Kubernetes Pods", "kubectl", "get", "po", "--namespace", ns)
			if err != nil {
				return err
			}
		}

		if o.showOption("ingresses") {
			err := printStatus(o, "Kubernetes Ingresses", "kubectl", "get", "ingress", "--namespace", ns)
			if err != nil {
				return err
			}
		}

		if o.showOption("secrets") {
			err := printStatus(o, "Kubernetes Secrets", "kubectl", "get", "secrets", "--namespace", ns)
			if err != nil {
				return err
			}
		}

		if o.showOption("configmaps") {
			err := printStatus(o, "Kubernetes Configmaps", "kubectl", "get", "configmaps", "--namespace", ns)
			if err != nil {
				return err
			}
		}

		if o.showOption("health") {
			err = health.Kuberhealthy(kubeClient, ns)
			if err != nil {
				return err
			}
		}
	case 1:
		if o.showInfo("version") {
			version := version2.GetVersion()
			_, err := fmt.Fprintln(o.CommonOptions.Out, util.ColorInfo(version))
			if err != nil {
				return err
			}
		}

		if o.showInfo("status") {
			err := NewCmdStatus(o.CommonOptions).Execute()
			if err != nil {
				return err
			}
		}

		if o.showInfo("pvc") {
			err := printStatus(o, "Kubernetes PVCs", "kubectl", "get", "pvc", "--namespace", ns)
			if err != nil {
				return err
			}
		}

		if o.showInfo("pods") {
			err := printStatus(o, "Kubernetes Pods", "kubectl", "get", "po", "--namespace", ns)
			if err != nil {
				return err
			}
		}

		if o.showInfo("ingresses") {
			err := printStatus(o, "Kubernetes Ingresses", "kubectl", "get", "ingress", "--namespace", ns)
			if err != nil {
				return err
			}
		}

		if o.showInfo("secrets") {
			err := printStatus(o, "Kubernetes Secrets", "kubectl", "get", "secrets", "--namespace", ns)
			if err != nil {
				return err
			}
		}

		if o.showInfo("configmaps") {
			err := printStatus(o, "Kubernetes Configmaps", "kubectl", "get", "configmaps", "--namespace", ns)
			if err != nil {
				return err
			}
		}

		if o.showInfo("health") {
			err = health.Kuberhealthy(kubeClient, ns)
			if err != nil {
				return err
			}
		}
	//ToDo: This wont be required after the deprecated --show flag is removed
	default:
		return errors.New("Only one argument is allowed")
	}

	log.Logger().Info("\nPlease visit https://jenkins-x.io/faq/issues/ for any known issues.")
	log.Logger().Info("\nFinished printing diagnostic information.")
	return nil
}

// Run the specified command (jx status, kubectl get po, etc) and print its output
func printStatus(o *DiagnoseOptions, header string, command string, options ...string) error {
	o.Runner.SetArgs(options)
	output, err := o.Runner.RunWithoutRetry()
	if err != nil {
		log.Logger().Errorf("Unable to get the %s", header)
		return err
	}
	// Print the output of the command, and add a little header at the top for formatting / readability
	log.Logger().Infof("\n%s:\n%s", header, util.ColorInfo(output))
	return nil
}

// Deprecated
func (o *DiagnoseOptions) showOption(e string) bool {
	for _, a := range o.Show {
		if a == e {
			return true
		}
	}
	return false
}

func (o *DiagnoseOptions) showInfo(e string) bool {
	for _, a := range o.Args {
		if a == e {
			return true
		}
	}
	return false
}

// GetOsVersion returns a human friendly string of the current OS
// in the case of an error this still returns a valid string for the details that can be found.
func (o *DiagnoseOptions) GetOsVersion() (string, error) {
	return system.GetOsVersion()
}
