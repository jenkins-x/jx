package opts

import (
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

// LogInstallDeprecated logs a warning message about deprecation
func LogInstallDeprecated() {
	log.Logger().Warn("this command is now deprecated.")
	log.Logger().Warnf("We now highly recommend you use %s instead:", util.ColorInfo("jx boot"))
	log.Logger().Warnf("for more information see: %s\n\n", util.ColorStatus("https://jenkins-x.io/docs/getting-started/setup/boot/"))

}
