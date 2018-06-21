package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	delete_context_long = templates.LongDesc(`
		Deletes one or more kubernetes contexts.
`)

	delete_context_example = templates.Examples(`
		# Deletes a context for a cluster that no longer exists
		jx delete context something

		# Deletes all contexts containing the word cheese
		# selecting them all by default
		jx delete ctx -a cheese
	`)
)

// DeleteContextOptions the options for the create spring command
type DeleteContextOptions struct {
	CreateOptions

	SelectAll    bool
	SelectFilter string
}

// NewCmdDeleteContext creates a command object for the "delete repo" command
func NewCmdDeleteContext(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteContextOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "contexts",
		Short:   "Deletes one or more kubernetes contexts",
		Aliases: []string{"context", "ctx"},
		Long:    delete_context_long,
		Example: delete_context_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	//addDeleteFlags(cmd, &options.CreateOptions)

	cmd.Flags().BoolVarP(&options.SelectAll, "all", "a", false, "Selects all the matched contexts")
	cmd.Flags().StringVarP(&options.SelectFilter, "filter", "f", "", "Filter the list of contexts to those containing this text")
	return cmd
}

// Run implements the command
func (o *DeleteContextOptions) Run() error {
	config, po, err := kube.LoadConfig()
	if err != nil {
		return err
	}

	if config == nil || config.Contexts == nil || len(config.Contexts) == 0 {
		return fmt.Errorf("No kubernetes contexts available! Try create or connect to cluster?")
	}

	names := []string{}
	allNames := []string{}

	args := o.Args
	for k, _ := range config.Contexts {
		allNames = append(allNames, k)
		if matchesFilter(k, args) {
			names = append(names, k)
		}
	}
	sort.Strings(allNames)

	if len(names) == 0 {
		if len(args) == 0 {
			return fmt.Errorf("Failed to find a context!")
		}
		return util.InvalidArg(args[1], allNames)
	}

	selected, err := util.SelectNamesWithFilter(names, "Select the Kubernetes Contexts to delete: ", o.SelectAll, o.SelectFilter)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		return nil
	}

	flag := false
	prompt := &survey.Confirm{
		Message: "Are you sure you want to delete these these Kubernetes Contexts?",
		Default: false,
	}
	err = survey.AskOne(prompt, &flag, nil)
	if err != nil {
		return err
	}
	if !flag {
		return nil
	}

	newConfig := *config
	for _, name := range selected {
		delete(newConfig.Contexts, name)
	}
	err = clientcmd.ModifyConfig(po, newConfig, false)
	if err != nil {
		return fmt.Errorf("Failed to update the kube config %s", err)
	}

	log.Infof("Deleted kubernetes contexts: %s\n", util.ColorInfo(strings.Join(selected, ", ")))
	return nil
}

// matchesFilter returns true if there are no filters or the text contains one of the filters
func matchesFilter(text string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, f := range filters {
		if strings.Contains(text, f) {
			return true
		}
	}
	return false
}
