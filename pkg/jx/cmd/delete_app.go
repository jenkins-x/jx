package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
)

var (
	deleteAppLong = templates.LongDesc(`
		Deletes one or more Applications from Jenkins

		Note that this command does not remove the underlying Git Repositories. 

		For that see the [jx delete repo](http://jenkins-x.io/commands/jx_delete_repo/) command.

`)

	deleteAppExample = templates.Examples(`
		# prompt for the available apps to delete
		jx delete app 

		# delete a specific app 
		jx delete app cheese
	`)
)

// DeleteAppOptions are the flags for this delete commands
type DeleteAppOptions struct {
	CommonOptions
}

// NewCmdDeleteApp creates a command object for this command
func NewCmdDeleteApp(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteAppOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "application",
		Short:   "Deletes one or many applications from Jenkins",
		Long:    deleteAppLong,
		Example: deleteAppExample,
		Aliases: []string{"applications", "app", "apps"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	return cmd
}

// Run implements this command
func (o *DeleteAppOptions) Run() error {
	args := o.Args

	jenk, err := o.JenkinsClient()
	if err != nil {
		return err
	}

	jobs, err := jenkins.LoadAllJenkinsJobs(jenk)
	if err != nil {
		return err
	}

	names := []string{}
	m := map[string]*gojenkins.Job{}

	for _, j := range jobs {
		if jenkins.IsMultiBranchProject(j) {
			name := j.FullName
			names = append(names, name)
			m[name] = j
		}
	}

	if len(names) == 0 {
		return fmt.Errorf("There are no Apps in Jenkins")
	}

	if len(args) == 0 {
		args, err = util.PickNames(names, "Pick Applications to remove from Jenkins:")
		if err != nil {
			return err
		}
	}
	if len(args) == 0 {
		return fmt.Errorf("There are no Apps in Jenkins")
	}
	deleteMessage := strings.Join(args, ", ")

	if !util.Confirm("You are about to delete these Applications from Jenkins: "+deleteMessage, false, "The list of Applications names to be deleted from Jenkins") {
		return nil
	}
	for _, name := range args {
		job := m[name]
		if job != nil {
			err = jenk.DeleteJob(*job)
			if err != nil {
				return err
			}
		}
	}
	o.Printf("Deleted Applications %s\n", util.ColorInfo(deleteMessage))
	return nil
}
