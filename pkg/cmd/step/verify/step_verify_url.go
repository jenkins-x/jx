package verify

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	optionEndpoint = "endpoint"
	optionCode     = "code"
	optionTimeout  = "timeout"
)

var (
	stepVerifyURLLong = templates.LongDesc(`
		This step checks the status of a URL
	`)

	stepVerifyURLExample = templates.Examples(`
        jx step verify url --endpoint https://jenkins-x.io
	`)
)

type StepVerifyURLOptions struct {
	step.StepOptions

	Endpoint string
	Code     int
	Timeout  time.Duration
}

// NewCmdStepVerifyURL creates a new verify url command
func NewCmdStepVerifyURL(commonOpts *opts.CommonOptions) *cobra.Command {
	options := StepVerifyURLOptions{
		StepOptions: step.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "url",
		Short:   "Verifies a URL returns an expected HTTP code",
		Long:    stepVerifyURLLong,
		Example: stepVerifyURLExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Endpoint, optionEndpoint, "e", "", "The endpoint on which to wait for expected HTTP code")
	cmd.Flags().IntVarP(&options.Code, optionCode, "c", http.StatusOK, "The HTTP code which should be returned by the endpoint")
	cmd.Flags().DurationVarP(&options.Timeout, optionTimeout, "t", 10*time.Minute, "The default timeout for the endpoint to return the expected HTTP code")

	return cmd
}

func (o *StepVerifyURLOptions) checkFlags() error {
	if o.Endpoint == "" {
		return util.MissingOption(optionEndpoint)
	}

	return nil
}

func (o *StepVerifyURLOptions) logError(err error) error {
	log.Logger().Infof("Retrying due to: %s", err)
	return err
}

// Run waits with exponential backoff for an endpoint to return an expected HTTP status code
func (o *StepVerifyURLOptions) Run() error {
	if err := o.checkFlags(); err != nil {
		return errors.Wrap(err, "checking flags")
	}

	log.Logger().Infof("Waiting for %q endpoint to return %d HTTP code", o.Endpoint, o.Code)
	err := util.Retry(o.Timeout, func() error {
		resp, err := http.Get(o.Endpoint)
		if err != nil {
			return o.logError(err)
		}
		if resp.StatusCode != o.Code {
			err := fmt.Errorf("invalid status code %d expecting %d", resp.StatusCode, o.Code)
			return o.logError(err)
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "waiting for %q", o.Endpoint)
	}
	log.Logger().Infof("Endpoint %q returns expected status code %d", o.Endpoint, o.Code)
	return nil
}
