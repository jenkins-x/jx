package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
		"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"github.com/ghodss/yaml"
	"io"
	"io/ioutil"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"os"
	"path"
	"time"
)

// ControllerBackupOptions are the flags for the commands
type ControllerBackupOptions struct {
	ControllerOptions

	GitRepositoryOptions gits.GitRepositoryOptions

	Namespace    string
	Organisation string
}

// NewCmdControllerBackup creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdControllerBackup(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ControllerBackupOptions{
		ControllerOptions: ControllerOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Runs the backup controller",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
		Aliases: []string{"backups"},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "The namespace to watch or defaults to the current namespace")
	cmd.Flags().StringVarP(&options.Organisation, "organisation", "o", "", "The organisation to backup")

	options.addCommonFlags(cmd)

	return cmd
}

// Run implements this command
func (o *ControllerBackupOptions) Run() error {
	// ensure Environment / Team / User CRDs are registered before we start
	err := o.registerEnvironmentCRD()
	if err != nil {
		return err
	}

	err = o.registerTeamCRD()
	if err != nil {
		return err
	}

	err = o.registerUserCRD()
	if err != nil {
		return err
	}

	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	ns := o.Namespace
	if ns == "" {
		ns = devNs
	}

	backupDir, err := util.BackupDir()
	if err != nil {
		return err
	}

	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return err
	}

	defaultRepoName := fmt.Sprintf("organisation-%s-backup", o.Organisation)

	details, err := gits.PickNewOrExistingGitRepository(o.Stdout(), o.BatchMode, authConfigSvc,
		defaultRepoName, &o.GitRepositoryOptions, nil, nil, o.Git(), true)
	if err != nil {
		return err
	}
	org := details.Organisation
	repoName := details.RepoName
	owner := org
	if owner == "" {
		owner = details.User.Username
	}
	provider := details.GitProvider
	repo, err := provider.GetRepository(owner, repoName)
	remoteRepoExists := err == nil
	var dir string

	if !remoteRepoExists {
		fmt.Fprintf(o.Stdout(), "Creating git repository %s/%s\n", util.ColorInfo(owner), util.ColorInfo(repoName))

		repo, err = details.CreateRepository()
		if err != nil {
			return err
		}

		dir, err = util.CreateUniqueDirectory(backupDir, details.RepoName, util.MaximumNewDirectoryAttempts)
		if err != nil {
			return err
		}

		err = o.Git().Init(dir)
		if err != nil {
			return err
		}

		pushGitURL, err := o.Git().CreatePushURL(repo.CloneURL, details.User)
		if err != nil {
			return err
		}

		err = o.Git().SetRemoteURL(dir, "origin", pushGitURL)
		if err != nil {
			return err
		}
	} else {
		fmt.Fprintf(o.Stdout(), "git repository %s/%s already exists\n", util.ColorInfo(owner), util.ColorInfo(repoName))

		dir = path.Join(backupDir, details.RepoName)
		localDirExists, err := util.FileExists(dir)
		if err != nil {
			return err
		}

		if localDirExists {
			// if remote repo does exist & local does exist, git pull the local repo
			fmt.Fprintf(o.Stdout(), "local directory already exists\n")

			err = o.Git().Pull(dir)
			if err != nil {
				return err
			}
		} else {
			fmt.Fprintf(o.Stdout(), "cloning repository locally\n")
			err = os.MkdirAll(dir, os.FileMode(0755))
			if err != nil {
				return err
			}

			// if remote repo does exist & local directory does not exist, clone locally
			pushGitURL, err := o.Git().CreatePushURL(repo.CloneURL, details.User)
			if err != nil {
				return err
			}

			err = o.Git().Clone(pushGitURL, dir)
			if err != nil {
				return err
			}
		}
	}

	log.Infof("Watching for users/teams/environments in namespace %s\n", util.ColorInfo(ns))

	_, environmentController := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(lo meta_v1.ListOptions) (runtime.Object, error) {
				return jxClient.JenkinsV1().Environments(ns).List(lo)
			},
			WatchFunc: func(lo meta_v1.ListOptions) (watch.Interface, error) {
				return jxClient.JenkinsV1().Environments(ns).Watch(lo)
			},
		},
		&v1.Environment{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				o.onEnvironmentChange(obj, jxClient, ns, dir)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				o.onEnvironmentChange(newObj, jxClient, ns, dir)
			},
			DeleteFunc: func(obj interface{}) {
				//o.onEnvironmentChange(obj, jxClient, ns)
			},
		},
	)

	stop := make(chan struct{})
	go environmentController.Run(stop)

	// Wait forever
	select {}
}

func (o *ControllerBackupOptions) onEnvironmentChange(obj interface{}, jxClient versioned.Interface, ns string, dir string) {
	env, ok := obj.(*v1.Environment)
	if !ok {
		log.Infof("Object is not a Environment %#v\n", obj)
		return
	}

	out, err := yaml.Marshal(env)
	if err != nil {
		log.Infof("Unable to marshall environment %s\n", err)
		return
	}

	o.Debugf("Dumping %s...\n", util.ColorInfo(env.ObjectMeta.Name))
	o.Debugf("%s\n", string(out))

	nsDir := path.Join(dir, "environments", ns)
	err = os.MkdirAll(nsDir, os.FileMode(0755))
	if err != nil {
		log.Infof("Unable to create directory %s\n", err)
		return
	}

	envFile := path.Join(nsDir, fmt.Sprintf("%s.yaml", env.ObjectMeta.Name))
	err = ioutil.WriteFile(envFile, out, 0644)
	if err != nil {
		log.Infof("Unable to write file %s\n", err)
		return
	}

	o.commitDirIfChanges(dir, fmt.Sprintf("Updating environment %s", env.ObjectMeta.Name))
}

func (o *ControllerBackupOptions) commitDirIfChanges(dir string, message string) {
	changes, err := o.Git().HasChanges(dir)
	if err != nil {
		log.Infof("Unable to determine changes %s\n", err)
		return
	}

	if changes {
		err = o.Git().Add(dir, "*")
		if err != nil {
			log.Infof("Unable to add files %s\n", err)
			return
		}

		err = o.Git().CommitDir(dir, message)
		if err != nil {
			log.Infof("Unable to commit dir %s\n", err)
			return
		}

		err = o.Git().PushMaster(dir)
		if err != nil {
			log.Infof("Unable to push master %s\n", err)
			return
		}

		fmt.Fprintf(o.Stdout(), "Pushed update '%s' git repository %s\n", util.ColorInfo(message), util.ColorInfo(dir))
	}
}
