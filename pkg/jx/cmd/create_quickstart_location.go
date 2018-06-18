package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
)

const (
	optionOwner   = "owner"
	optionGitUrl  = "url"
	optionGitKind = "kind"
)

var (
	createQuickstartLocationLong = templates.LongDesc(`
		Create a location of quickstarts for your team

		For more documentation see: [https://jenkins-x.io/developing/create-quickstart/#customising-your-teams-quickstarts](https://jenkins-x.io/developing/create-quickstart/#customising-your-teams-quickstarts)

`)

	createQuickstartLocationExample = templates.Examples(`
		# Create a quickstart location using a github repository organisation 
		jx create quickstartlocation --owner my-quickstarts

		# Create a quickstart location using a github repository organisation via an abbreviation
		jx create qsloc --owner my-quickstarts

		# Create a quickstart location for your git repo and organisation 
		jx create quickstartlocation --url https://mygit.server.com --owner my-quickstarts

	`)
)

// CreateQuickstartLocationOptions the options for the create spring command
type CreateQuickstartLocationOptions struct {
	CreateOptions

	GitUrl   string
	GitKind  string
	Owner    string
	Includes []string
	Excludes []string
}

// NewCmdCreateQuickstartLocation creates a command object for the "create" command
func NewCmdCreateQuickstartLocation(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateQuickstartLocationOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     quickstartLocation,
		Short:   "Create a location of quickstarts for your team",
		Aliases: quickstartLocationsAliases,
		Long:    createQuickstartLocationLong,
		Example: createQuickstartLocationExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.GitUrl, optionGitUrl, "u", gits.GitHubURL, "The URL of the git service")
	cmd.Flags().StringVarP(&options.GitKind, optionGitKind, "k", "", "The kind of git service at the URL")
	cmd.Flags().StringVarP(&options.Owner, optionOwner, "o", "", "The owner is the user or organisation of the git provider used to find repositories")
	cmd.Flags().StringArrayVarP(&options.Includes, "includes", "i", []string{"*"}, "The patterns to include repositories")
	cmd.Flags().StringArrayVarP(&options.Excludes, "excludes", "x", []string{"WIP-*"}, "The patterns to exclude repositories")

	options.addCommonFlags(cmd)
	return cmd
}

// Run implements the command
func (o *CreateQuickstartLocationOptions) Run() error {
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	err = o.registerEnvironmentCRD()
	if err != nil {
		return err
	}

	if o.GitUrl == "" {
		return util.MissingOption(optionGitUrl)
	}
	if o.Owner == "" {
		return util.MissingOption(optionOwner)
	}

	if o.GitKind == "" {
		authConfigSvc, err := o.CreateGitAuthConfigService()
		if err != nil {
			return err
		}
		server := authConfigSvc.Config().GetServer(o.GitUrl)
		if server != nil {
			o.GitKind = server.Kind
		}
	}
	if o.GitKind == "" {
		return util.MissingOption(optionGitKind)
	}
	locations, err := kube.GetQuickstartLocations(jxClient, ns)
	if err != nil {
		return err
	}

	var location *v1.QuickStartLocation
	for i, l := range locations {
		if l.GitURL == o.GitUrl && l.Owner == o.Owner {
			location = &locations[i]
		}
	}
	if location == nil {
		locations = append(locations, v1.QuickStartLocation{
			GitURL:  o.GitUrl,
			GitKind: o.GitKind,
			Owner:   o.Owner,
		})
	}
	location = &locations[len(locations)-1]
	location.Includes = o.Includes
	location.Excludes = o.Excludes

	callback := func(env *v1.Environment) error {
		env.Spec.TeamSettings.QuickstartLocations = locations
		o.Printf("Adding the quickstart git owner %s\n", util.ColorInfo(util.UrlJoin(o.GitUrl, o.Owner)))
		return nil
	}
	return o.modifyDevEnvironment(jxClient, ns, callback)
}
