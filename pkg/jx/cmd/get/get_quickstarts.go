package get

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	"github.com/jenkins-x/jx/pkg/log"
	"strings"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/quickstarts"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
)

//GetQuickstartsOptions -  the command line options
type GetQuickstartsOptions struct {
	GetOptions
	GitHubOrganisations []string
	Filter              quickstarts.QuickstartFilter
	ShortFormat         bool
}

var (
	getQuickstartsLong = templates.LongDesc(`
		Display the available quickstarts

`)

	getQuickstartsExample = templates.Examples(`
		# List all the available quickstarts
		jx get quickstarts
	`)
)

//NewCmdGetQuickstarts creates the command
func NewCmdGetQuickstarts(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &GetQuickstartsOptions{
		GetOptions: GetOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "quickstarts [flags]",
		Short:   "Lists the available quickstarts",
		Long:    getQuickstartsLong,
		Example: getQuickstartsExample,
		Aliases: []string{"quickstart", "qs"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringArrayVarP(&options.GitHubOrganisations, "organisations", "g", []string{}, "The GitHub organisations to query for quickstarts")
	cmd.Flags().StringArrayVarP(&options.Filter.Tags, "tag", "t", []string{}, "The tags on the quickstarts to filter")
	cmd.Flags().StringVarP(&options.Filter.Text, "filter", "f", "", "The text filter")
	cmd.Flags().StringVarP(&options.Filter.Owner, "owner", "", "", "The owner to filter on")
	cmd.Flags().StringVarP(&options.Filter.Language, "language", "l", "", "The language to filter on")
	cmd.Flags().StringVarP(&options.Filter.Framework, "framework", "", "", "The framework to filter on")
	cmd.Flags().BoolVarP(&options.ShortFormat, "short", "s", false, "return minimal details")

	return cmd
}

// Run implements this command
func (o *GetQuickstartsOptions) Run() error {
	var locations []v1.QuickStartLocation
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	locations, err = kube.GetQuickstartLocations(jxClient, ns)
	if err != nil {
		return err
	}

	// lets add any extra github organisations if they are not already configured
	for _, org := range o.GitHubOrganisations {
		found := false
		for _, loc := range locations {
			if loc.GitURL == gits.GitHubURL && loc.Owner == org {
				found = true
				break
			}
		}
		if !found {
			locations = append(locations, v1.QuickStartLocation{
				GitURL:   gits.GitHubURL,
				GitKind:  gits.KindGitHub,
				Owner:    org,
				Includes: []string{"*"},
				Excludes: []string{"WIP-*"},
			})
		}
	}

	gitMap := map[string]map[string]v1.QuickStartLocation{}
	for _, loc := range locations {
		m := gitMap[loc.GitURL]
		if m == nil {
			m = map[string]v1.QuickStartLocation{}
			gitMap[loc.GitURL] = m
		}
		m[loc.Owner] = loc
	}

	model := quickstarts.NewQuickstartModel()

	for gitURL, m := range gitMap {
		for _, location := range m {
			kind := location.GitKind
			if kind == "" {
				kind = gits.KindGitHub
			}
			gitProvider, err := o.GitProviderForGitServerURL(gitURL, kind)
			if err != nil {
				return err
			}
			log.Logger().Debugf("Searching for repositories in Git server %s owner %s includes %s excludes %s as user %s ", gitProvider.ServerURL(), location.Owner, strings.Join(location.Includes, ", "), strings.Join(location.Excludes, ", "), gitProvider.CurrentUsername())
			err = model.LoadGithubQuickstarts(gitProvider, location.Owner, location.Includes, location.Excludes)
			if err != nil {
				log.Logger().Debugf("Quickstart load error: %s", err.Error())
			}
		}
	}

	//output list of available quickstarts and exit
	filteredQuickstarts := model.Filter(&o.Filter)
	for _, qs := range filteredQuickstarts {
		if o.ShortFormat {
			fmt.Fprintf(o.Out, "%s\n", qs.Name)
		} else {
			fmt.Fprintf(o.Out, "%s/%s/%s\n", qs.GitProvider.ServerURL(), qs.Owner, qs.Name)
		}
	}
	return nil
}
