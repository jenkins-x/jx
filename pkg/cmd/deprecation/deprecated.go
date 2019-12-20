package deprecation

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

// deprecatedCommands list of deprecated commands along with some more deprecation details
var deprecatedCommands map[string]deprecationInfo = map[string]deprecationInfo{
	"install": {
		replacement: "boot",
		date:        "Feb 1 2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/setup/boot/")),
	},
	"init": {
		replacement: "boot",
		date:        "Feb 1 2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/setup/boot/")),
	},
	"create terraform": {
		replacement: "boot",
		date:        "Feb 1 2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/setup/boot/")),
	},
	"create cluster minikube": {
		replacement: "minikube start",
		date:        "Feb 1 2020",
	},
	"create cluster minishift": {
		date: "Feb 1 2020",
	},
	"create cluster openshift": {
		date: "Feb 1 2020",
	},
	"create cluster icp": {
		date: "Feb 1 2020",
	},
	"create cluster oke": {
		date: "Feb 1 2020",
	},
	"create cluster kubernetes": {
		date: "Feb 1 2020",
	},
	"create cluster aws": {
		date: "Feb 1 2020",
	},
	"create post": {
		date: "Feb 1 2020",
	},
	"create step": {
		date: "Feb 1 2020",
		info: fmt.Sprintf("This will be renamed to %s command.", util.ColorStatus("edit pipeline")),
	},
	"create archetype": {
		replacement: "create quickstart",
		date:        "Feb 1 2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/first-project/create-quickstart/")),
	},
	"create micro": {
		replacement: "create quickstart",
		date:        "Feb 1 2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/first-project/create-quickstart/")),
	},
	"create lile": {
		replacement: "create quickstart",
		date:        "Feb 1 2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/first-project/create-quickstart/")),
	},
	"create camel": {
		replacement: "create quickstart",
		date:        "Feb 1 2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/first-project/create-quickstart/")),
	},
	"create jhipster": {
		replacement: "create quickstart",
		date:        "Feb 1 2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/first-project/create-quickstart/")),
	},
	"create spring": {
		replacement: "create quickstart",
		date:        "Feb 1 2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/first-project/create-quickstart/")),
	},
	"create addon knative-build": {
		date: "Feb 1 2020",
		info: "The knative-build is now replaced by Tekton pipeline",
	},
	"create etc-host": {
		date: "Feb 1 2020",
	},
	"create codeship": {
		date: "Feb 1 2020",
		info: "No longer needed",
	},
	"upgrade platform": {
		replacement: "upgrade boot",
		date:        "Feb 1 2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/setup/boot/")),
	},
	"upgrade cluster": {
		replacement: "boot",
		date:        "Feb 1 2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/setup/boot/")),
	},
	"update cluster": {
		replacement: "gcloud container cluster upgrade",
		date:        "Feb 1 2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://cloud.google.com/sdk/gcloud/")),
	},
	"upgrade ingress": {
		replacement: "boot",
		date:        "Feb 1 2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/setup/boot/")),
	},
	"upgrade extensions": {
		replacement: "upgrade apps",
		date:        "Feb 1 2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/managing-jx/common-tasks/upgrade-jx/#upgrading-apps")),
	},
	"get clusters": {
		replacement: "gcloud container clusters list",
		date:        "Feb 1 2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://cloud.google.com/sdk/gcloud/")),
	},
	"get workflows": {
		date: "Feb 1 2020",
	},
	"get aws": {
		date: "Feb 1 2020",
	},
	"get eks": {
		replacement: "eksctl get clusters",
		date:        "Feb 1 2020",
	},
	"get post": {
		date: "Feb 1 2020",
	},
	"delete aws": {
		date: "Feb 1 2020",
	},
	"delete eks": {
		replacement: "eksctl delete cluster",
		date:        "Feb 1 2020",
	},
	"delete post": {
		date: "Feb 1 2020",
	},
	"delete addon knative-build": {
		date: "Feb 1 2020",
		info: "The knative-build is now replaced by Tekton pipeline",
	},
	"delete extension": {
		date: "Feb 1 2020",
		info: "This is now replaced by apps",
	},
	"edit deploy": {
		date: "Feb 1 2020",
	},
	"edit extensionsrepository": {
		date: "Feb 1 2020",
		info: "This is now replaced by apps",
	},
	"step create jenkins": {
		date: "Feb 1 2020",
	},
	"step post": {
		date: "Feb 1 2020",
	},
	"step pre": {
		date: "Feb 1 2020",
	},
	"step credential": {
		date: "Feb 1 2020",
		info: "This replaced by Tekton's mounted secrets",
	},
	"step git credential": {
		date: "Apr 1 2020",
		info: "This replaced by Tekton's mounted secrets",
	},
	"step split monorepo": {
		date: "Feb 1 2020",
	},
	"step nexus drop": {
		date: "Feb 1 2020",
	},
	"step nexus release": {
		date: "Feb 1 2020",
	},
	"console": {
		replacement: "ui",
		date:        "Mar 1 2020",
		info:        "Classic Jenkins console will be replaced by Jenkins X UI app",
	},
	"controller workflow": {
		date: "Feb 1 2020",
	},
}

// deprecateInfo keeps some deprecation details related to a command
type deprecationInfo struct {
	replacement string
	date        string
	info        string
}

// DeprecateCommands runs recursively over all commands and set the deprecation message
// on every command defined the deprecated commands map.
func DeprecateCommands(cmd *cobra.Command) {
	path := commandPath(cmd)
	if deprecation, ok := deprecatedCommands[path]; ok {
		cmd.Deprecated = deprecationMessage(deprecation)
	}
	if !cmd.HasSubCommands() {
		return
	}
	for _, c := range cmd.Commands() {
		DeprecateCommands(c)
	}
}

func deprecationMessage(dep deprecationInfo) string {
	var date string
	if dep.date != "" {
		date = fmt.Sprintf("it will be removed on %s.", util.ColorInfo(dep.date))
	} else {
		date = "it will be soon removed."
	}
	var replacement string
	if dep.replacement != "" {
		replacement = fmt.Sprintf("We now highly recommend you use %s instead.", util.ColorInfo(dep.replacement))
	}
	msg := fmt.Sprintf("%s %s", date, replacement)
	if dep.info != "" {
		return fmt.Sprintf("%s %s", msg, dep.info)
	}
	return msg
}

func commandPath(cmd *cobra.Command) string {
	parentText := ""
	parent := cmd.Parent()
	if parent != nil {
		parentText = commandPath(parent)
		if parentText != "" {
			parentText += " "
		}
	}
	return strings.TrimPrefix(parentText, "jx ") + cmd.Name()
}
