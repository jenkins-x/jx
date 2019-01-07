package cmd

import (
	"io"
	"sort"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/kube"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// GetStorageOptions containers the CLI options
type GetStorageOptions struct {
	GetOptions
}

var (
	getStorageLong = templates.LongDesc(`
		Display the storage configuration for different classifications
`)

	getStorageExample = templates.Examples(`
		# List the storage configurations for different classifications for the current team
		jx get storage
	`)
)

// NewCmdGetStorage creates the new command for: jx get env
func NewCmdGetStorage(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetStorageOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "storage",
		Short:   "Display the storage configuration for different classifications",
		Aliases: []string{"helm"},
		Long:    getStorageLong,
		Example: getStorageExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetStorageOptions) Run() error {
	settings, err := o.TeamSettings()
	if err != nil {
		return err
	}
	m := map[string]*v1.StorageLocation{}
	for i, ls := range settings.StorageLocations {
		m[ls.Classifier] = &settings.StorageLocations[i]
	}
	for _, name := range kube.Classifications {
		if m[name] == nil {
			m[name] = settings.StorageLocation(name)
		}
	}
	names := []string{}
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	table := o.createTable()
	table.AddRow("CLASSIFICATION", "LOCATION")
	for _, n := range names {
		ls := m[n]
		if ls != nil {
			table.AddRow(n, ls.Description())
		}
	}
	table.Render()
	return nil
}
