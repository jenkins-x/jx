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
		date:        "01-02-2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/setup/boot/")),
	},
	"upgrade platform": {
		replacement: "boot",
		date:        "01-02-2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/setup/boot/")),
	},
	"upgrade ingress": {
		replacement: "boot",
		date:        "01-02-2020",
		info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/setup/boot/")),
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
	if !cmd.HasSubCommands() {
		path := commandPath(cmd)
		if deprecation, ok := deprecatedCommands[path]; ok {
			cmd.Deprecated = deprecationMessage(deprecation)
		}
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
