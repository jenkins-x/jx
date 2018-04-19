package cmd

import (
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/tests"
)

func configureOptions(o *CommonOptions) {
	o.Out = tests.Output()
	o.BatchMode = true
	o.Factory = cmdutil.NewFactory()
}
