package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/builds"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

var (
	editBuildpackLong = templates.LongDesc(`
		Edits the build pack configuration for your team
`)

	editBuildpackExample = templates.Examples(`
		# Edit the build pack configuration for your team, picking the build pack you wish to use from the available
		jx edit buildpack

        # to switch to classic workloads for your team
		jx edit buildpack -n classic-workloads

        # to switch to kubernetes workloads for your team
		jx edit buildpack -n kubernetes-workloads
		
		For more documentation see: [https://jenkins-x.io/architecture/build-packs/](https://jenkins-x.io/architecture/build-packs/)
	`)
)

// EditBuildPackOptions the options for the create spring command
type EditBuildPackOptions struct {
	EditOptions

	BuildPackName string
	BuildPackURL  string
	BuildPackRef  string
}

// NewCmdEditBuildpack creates a command object for the "create" command
func NewCmdEditBuildpack(commonOpts *CommonOptions) *cobra.Command {
	options := &EditBuildPackOptions{
		EditOptions: EditOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     buildPack,
		Short:   "Edits the build pack configuration for your team",
		Aliases: buildPacksAliases,
		Long:    editBuildpackLong,
		Example: editBuildpackExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.BuildPackURL, "url", "u", "", "The URL for the build pack Git repository")
	cmd.Flags().StringVarP(&options.BuildPackRef, "ref", "r", "", "The Git reference (branch,tag,sha) in the Git repository to use")
	cmd.Flags().StringVarP(&options.BuildPackName, "name", "n", "", "The name of the BuildPack resource to use")
	options.addCommonFlags(cmd)
	return cmd
}

// Run implements the command
func (o *EditBuildPackOptions) Run() error {
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	m, labels, err := builds.GetBuildPacks(jxClient, ns)
	if err != nil {
		return err
	}

	buildPackURL := o.BuildPackURL
	BuildPackRef := o.BuildPackRef
	buildPackName := o.BuildPackName

	if buildPackName != "" {
		var buildPack *v1.BuildPack
		names := []string{}
		for _, v := range m {
			name := v.Name
			if name == buildPackName {
				buildPack = v
				break
			}
			names = append(names, name)
		}
		if buildPack == nil {
			sort.Strings(names)
			return util.InvalidArg(buildPackName, names)
		}
		buildPackURL = buildPack.Spec.GitURL
		BuildPackRef = buildPack.Spec.GitRef
	}
	if o.BatchMode {
		if buildPackURL == "" && BuildPackRef == "" {
			return nil
		}
		if buildPackURL == "" {
			return util.MissingOption("url")
		}
		if BuildPackRef == "" {
			return util.MissingOption("ref")
		}
	} else {
		if buildPackURL == "" || BuildPackRef == "" {
			teamSettings, err := o.TeamSettings()
			if err != nil {
				return err
			}

			defaultValue := buildPackName
			if defaultValue == "" {
				for k, v := range m {
					if v.Spec.GitURL == teamSettings.BuildPackURL || v.Name == teamSettings.BuildPackName {
						defaultValue = k
						break
					}
				}
			}
			if defaultValue == "" {
				for k := range m {
					if strings.Contains(k, "Kubernetes") {
						defaultValue = k
						break
					}
				}
			}
			label, err := util.PickNameWithDefault(labels, "Pick workload build pack: ", defaultValue, "Build packs are used to automate your CI/CD pipelines when you create or import projects", o.In, o.Out, o.Err)
			if err != nil {
				return err
			}
			buildPack := m[label]
			if buildPack == nil {
				return fmt.Errorf("No BuildPack found for label: %s", label)
			}
			if len(labels) == 1 {
				log.Infof("Only one build pack %s so configuring this build pack for your team\n", util.ColorInfo(label))
			}
			buildPackURL = buildPack.Spec.GitURL
			BuildPackRef = buildPack.Spec.GitRef
			buildPackName = buildPack.Name
		}
	}

	callback := func(env *v1.Environment) error {
		teamSettings := &env.Spec.TeamSettings
		if buildPackURL != "" {
			teamSettings.BuildPackURL = buildPackURL
		}
		if BuildPackRef != "" {
			teamSettings.BuildPackRef = BuildPackRef
		}
		teamSettings.BuildPackName = buildPackName

		log.Infof("Setting the team build pack to %s repo: %s ref: %s\n", util.ColorInfo(buildPackName), util.ColorInfo(buildPackURL), util.ColorInfo(BuildPackRef))
		return nil
	}
	return o.ModifyDevEnvironment(callback)
}
