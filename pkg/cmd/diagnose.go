package cmd

import (
	"fmt"

	"github.com/jenkins-x/jx/v2/pkg/table"
	"github.com/jenkins-x/jx/v2/pkg/util/system"
	"k8s.io/client-go/kubernetes"

	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"

	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/cmd/version"
	"github.com/jenkins-x/jx/v2/pkg/health"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx-logging/pkg/log"
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

// DiagnoseOptions struct contains the diagnose flags
type DiagnoseOptions struct {
	*opts.CommonOptions
	Namespace string
	HelmTLS   bool
	//Deprecated
	Show   []string
	Runner util.Commander
}

//NewCmdDiagnose creates the diagnose command
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

// Run implements the diagnose command
func (o *DiagnoseOptions) Run() error {
	// Get the namespace to run the diagnostics in, and output it
	kubeClient, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return errors.Wrap(err, "failed to create kubeClient")
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

	switch len(o.Args) {
	//ToDo: This wont be required after the deprecated --show flag is removed
	case 0:
		err = runDeprecatedShow(o, table, kubeClient, ns)
		if err != nil {
			return err
		}
	case 1:
		if o.showInfo("version") {
			err := version.NewCmdVersion(o.CommonOptions).Execute()
			if err != nil {
				return err
			}
			// Render system level information
			table.Render()
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

//Deprecated
func runDeprecatedShow(o *DiagnoseOptions, table table.Table, kubeClient kubernetes.Interface, ns string) error {
	if o.showOption("version") {
		versionCmd := version.NewCmdVersion(o.CommonOptions)
		// Ugly hack to prevent --show flag from being passed to the version command
		versionCmd.SetArgs([]string{})
		err := versionCmd.Execute()
		if err != nil {
			return err
		}
		// Render system level information
		table.Render()
		return nil
	}

	if o.showOption("status") {
		statusCmd := NewCmdStatus(o.CommonOptions)
		// Ugly hack to prevent --show flag from being passed to the status command
		statusCmd.SetArgs([]string{})
		err := statusCmd.Execute()
		if err != nil {
			return err
		}
		return nil
	}

	if o.showOption("pvc") {
		err := printStatus(o, "Kubernetes PVCs", "kubectl", "get", "pvc", "--namespace", ns)
		if err != nil {
			return err
		}
		return nil
	}

	if o.showOption("pods") {
		err := printStatus(o, "Kubernetes Pods", "kubectl", "get", "po", "--namespace", ns)
		if err != nil {
			return err
		}
		return nil
	}

	if o.showOption("ingresses") {
		err := printStatus(o, "Kubernetes Ingresses", "kubectl", "get", "ingress", "--namespace", ns)
		if err != nil {
			return err
		}
		return nil
	}

	if o.showOption("secrets") {
		err := printStatus(o, "Kubernetes Secrets", "kubectl", "get", "secrets", "--namespace", ns)
		if err != nil {
			return err
		}
		return nil
	}

	if o.showOption("configmaps") {
		err := printStatus(o, "Kubernetes Configmaps", "kubectl", "get", "configmaps", "--namespace", ns)
		if err != nil {
			return err
		}
		return nil
	}

	if o.showOption("health") {
		err := health.Kuberhealthy(kubeClient, ns)
		if err != nil {
			return err
		}
		return nil
	}

	// Handle error if unsupported show option is passed
	return errors.New(fmt.Sprintf("Unsupported show option: %v", o.Show))
}
