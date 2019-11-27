package restore

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube/velero"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// RestoreFromScheduleOptions contains the command line options
type RestoreFromBackupOptions struct {
	*StepRestoreOptions

	Namespace string
}

var (
	restoreFromBackupLong = templates.LongDesc(`
		Restores the cluster custom data from the a backup.

`)

	restoreFromBackupExample = templates.Examples(`
		# executes the step which restores data from a backup 
		jx step restore from-latest-backup
	`)
)

// NewCmdStepRestoreFromBackup creates the command
func NewCmdStepRestoreFromBackup(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &RestoreFromBackupOptions{
		StepRestoreOptions: &StepRestoreOptions {
			StepOptions: step.StepOptions {
				CommonOptions: commonOpts,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "from-latest-backup [flags]",
		Short:   "stuff",
		Long:    restoreFromBackupLong,
		Example: restoreFromBackupExample,
		Aliases: []string{"from-latest-backups"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "", "velero", "The namespace where velero has been installed")
	return cmd
}

// Run implements this command
func (o *RestoreFromBackupOptions) Run() error {

	// create the api extensions client
	apiClient, err := o.ApiExtensionsClient()
	if err != nil {
		return errors.Wrap(err, "while creating api extensions client")
	}

	// create the kubernetes client
	kubeClient, err := o.KubeClient()
	if err != nil {
		return errors.Wrap(err, "while creating kube client")
	}

	// check if a velero schedule exists
	scheduleExists, err := velero.DoesVeleroBackupScheduleExist(apiClient, o.Namespace)
	if err != nil {
		errors.Wrap(err, "when trying to check for velero schedules")
	}

	// if a Velero Schedule exists then we should be confident that is an existing operational cluster
	// and abort the restore. However if
	if scheduleExists {
		fmt.Println("A schedule exists for this cluster - aborting restore as it would be dangerous to apply the latest backup")
		fmt.Println("If you expected this command to execute automatically - perhaps the backup schdule apply step comes before this step?")
	} else {
		latestBackupName, err := velero.GetLatestBackupFromBackupResource(apiClient, o.Namespace)
		if err != nil {
			errors.Wrap(err, "when trying to get the latest backup")
		}
		log.Infof("Using backup '%s' as the latest backup to restore", util.ColorInfo(latestBackupName))

		if o.BatchMode {
			err := velero.RestoreFromBackup(apiClient, kubeClient,"velero", latestBackupName)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("when attempting to automatically restore from '%s' backup", latestBackupName))
			}
		} else {
			backupNames, err := velero.GetBackupsFromBackupResource(apiClient, o.Namespace)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("when attempting to retrieve the latest backups", latestBackupName))
			}

			selectedBackup, err := util.PickNameWithDefault(backupNames, "Which backup do you want to restore from?: ", latestBackupName, "", o.GetIOFileHandles())
			if err != nil {
				return err
			}

			args := []string{selectedBackup}
			if len(args) == 0 {
				return fmt.Errorf("No backup chosen")
			}
			selectedBackupName := args[0]
			err = velero.RestoreFromBackup(apiClient, kubeClient,"velero", selectedBackupName)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("when attempting to restore from '%s' backup", selectedBackupName))
			}
		}
	}
	return nil
}

