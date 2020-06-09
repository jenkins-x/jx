package deprecation

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/spf13/cobra"
)

// DeprecatedCommands list of deprecated commands along with some more deprecation details
var DeprecatedCommands = map[string]DeprecationInfo{
	"install": {
		Replacement: "jx boot",
		Date:        "Jun 1 2020",
		Info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/setup/boot/")),
	},
	"init": {
		Replacement: "jx boot",
		Date:        "Jun 1 2020",
		Info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/setup/boot/")),
	},
	"create post": {
		Date: "Feb 1 2020",
	},
	"create spring": {
		Replacement: "jx create project",
		Date:        "Mar 1 2020",
		Info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/commands/jx_create_project/")),
	},
	"upgrade platform": {
		Replacement: "jx upgrade boot",
		Date:        "Jun 1 2020",
		Info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/setup/boot/")),
	},
	"upgrade ingress": {
		Replacement: "jx boot",
		Date:        "Jun 1 2020",
		Info: fmt.Sprintf("Please check %s for more details.",
			util.ColorStatus("https://jenkins-x.io/docs/getting-started/setup/boot/")),
	},
	"get post": {
		Date: "Feb 1 2020",
	},
	"delete extension": {
		Date: "Feb 1 2020",
		Info: "This is now replaced by apps",
	},
	"edit extensionsrepository": {
		Date: "Feb 1 2020",
		Info: "This is now replaced by apps",
	},
	"step post": {
		Date: "Feb 1 2020",
	},
	"step nexus drop": {
		Date: "Feb 1 2020",
	},
	"step nexus release": {
		Date: "Feb 1 2020",
	},
	"create vault": {
		Date: "Sep 1 2020",
		Info: "This commands will have no replacement.",
	},
	"delete vault": {
		Date: "Sep 1 2020",
		Info: "This commands will have no replacement.",
	},
}

// deprecateInfo keeps some deprecation details related to a command
type DeprecationInfo struct {
	Replacement string
	Date        string
	Info        string
}

// DeprecateCommands runs recursively over all commands and set the deprecation message
// on every command defined the deprecated commands map.
func DeprecateCommands(cmd *cobra.Command) {
	path := commandPath(cmd)
	if deprecation, ok := DeprecatedCommands[path]; ok {
		updatedLongMessage := fmt.Sprintf("command 'jx %s' is deprecated. %s\n\n", path, deprecationMessage(deprecation)) + cmd.Long
		cmd.Long = updatedLongMessage
	}
	if !cmd.HasSubCommands() {
		return
	}
	for _, c := range cmd.Commands() {
		DeprecateCommands(c)
	}
}

// GetRemovalDate returns the date when the command is planned to be removed
func GetRemovalDate(cmd *cobra.Command) string {
	path := commandPath(cmd)
	if deprecation, ok := DeprecatedCommands[path]; ok {
		return deprecation.Date
	}
	return ""
}

// GetReplacement returns the command replacement if any available
func GetReplacement(cmd *cobra.Command) string {
	path := commandPath(cmd)
	if deprecation, ok := DeprecatedCommands[path]; ok {
		return deprecation.Replacement
	}
	return ""
}

func deprecationMessage(dep DeprecationInfo) string {
	var date string
	if dep.Date != "" {
		date = fmt.Sprintf("it will be removed on %s.", util.ColorInfo(dep.Date))
	} else {
		date = "it will be soon removed."
	}
	var replacement string
	if dep.Replacement != "" {
		replacement = fmt.Sprintf("We now highly recommend you use %s instead.", util.ColorInfo(dep.Replacement))
	}
	msg := fmt.Sprintf("%s %s", date, replacement)
	if dep.Info != "" {
		return fmt.Sprintf("%s %s", msg, dep.Info)
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
