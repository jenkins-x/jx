package cmd

import (
	"fmt"
	"reflect"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BranchPatterns struct {
	DefaultBranchPattern string
	ForkBranchPattern    string
}

const (
	defaultBuildPackRef = "2.1"
	defaultHelmBin      = "helm"
)

// TeamSettings returns the team settings
func (o *CommonOptions) TeamSettings() (*v1.TeamSettings, error) {
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return nil, err
	}
	err = o.registerEnvironmentCRD()
	if err != nil {
		return nil, fmt.Errorf("Failed to register Environment CRD: %s", err)
	}

	env, err := kube.EnsureDevEnvironmentSetup(jxClient, ns)
	if err != nil {
		return nil, fmt.Errorf("Failed to setup dev Environment in namespace %s: %s", ns, err)
	}
	if env == nil {
		return nil, fmt.Errorf("No Development environment found for namespace %s", ns)
	}

	teamSettings := &env.Spec.TeamSettings
	if teamSettings.BuildPackURL == "" {
		teamSettings.BuildPackURL = JenkinsBuildPackURL
	}
	if teamSettings.BuildPackRef == "" {
		teamSettings.BuildPackRef = defaultBuildPackRef
	}
	return teamSettings, nil
}

// TeamBranchPatterns returns the team branch patterns used to enable CI/CD on branches when creating/importing projects
func (o *CommonOptions) TeamBranchPatterns() (*BranchPatterns, error) {
	teamSettings, err := o.TeamSettings()
	if err != nil {
		return nil, err
	}

	branchPatterns := teamSettings.BranchPatterns
	if branchPatterns == "" {
		branchPatterns = defaultBranchPatterns
	}

	forkBranchPatterns := teamSettings.ForkBranchPatterns
	if forkBranchPatterns == "" {
		forkBranchPatterns = defaultForkBranchPatterns
	}

	return &BranchPatterns{
		DefaultBranchPattern: branchPatterns,
		ForkBranchPattern:    forkBranchPatterns,
	}, nil
}

// TeamHelmBin returns the helm binary used for a team and whether a remote tiller is disabled
func (o *CommonOptions) TeamHelmBin() (string, bool, error) {
	helmBin := defaultHelmBin
	teamSettings, err := o.TeamSettings()
	if err != nil {
		return helmBin, false, err
	}

	helmBin = teamSettings.HelmBinary
	if helmBin == "" {
		helmBin = defaultHelmBin
	}
	return helmBin, teamSettings.NoTiller, nil
}

// ModifyDevEnvironment modifies the development environment settings
func (o *CommonOptions) ModifyDevEnvironment(callback func(env *v1.Environment) error) error {
	apisClient, err := o.CreateApiExtensionsClient()
	if err != nil {
		return errors.Wrap(err, "failed to create the api extensions client")
	}
	kube.RegisterEnvironmentCRD(apisClient)

	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return errors.Wrap(err, "failed to create the jx client")
	}
	err = o.registerEnvironmentCRD()
	if err != nil {
		return errors.Wrap(err, "failed to register the environment CRD")
	}

	env, err := kube.EnsureDevEnvironmentSetup(jxClient, ns)
	if err != nil {
		return errors.Wrapf(err, "failed to setup the dev environment for namespace '%s'", ns)
	}
	if env == nil {
		return fmt.Errorf("No Development environment found for namespace %s", ns)
	}
	return o.modifyDevEnvironment(jxClient, ns, callback)
}

func (o *CommonOptions) registerReleaseCRD() error {
	apisClient, err := o.Factory.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterReleaseCRD(apisClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the Team CRD")
	}
	return nil
}

func (o *CommonOptions) registerTeamCRD() error {
	apisClient, err := o.Factory.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterTeamCRD(apisClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the Team CRD")
	}
	return nil
}

func (o *CommonOptions) registerUserCRD() error {
	apisClient, err := o.Factory.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterUserCRD(apisClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the User CRD")
	}
	return nil
}

func (o *CommonOptions) registerEnvironmentRoleBindingCRD() error {
	apisClient, err := o.Factory.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterEnvironmentRoleBindingCRD(apisClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the User CRD")
	}
	return nil
}

func (o *CommonOptions) registerPipelineActivityCRD() error {
	apisClient, err := o.Factory.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterPipelineActivityCRD(apisClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the PipelineActivity CRD")
	}
	return nil
}

func (o *CommonOptions) registerWorkflowCRD() error {
	apisClient, err := o.Factory.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterWorkflowCRD(apisClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the Workflow CRD")
	}
	return nil
}

// ModifyTeam lazily creates the team if it does not exist or updates it if it requires a change
func (o *CommonOptions) ModifyTeam(teamName string, callback func(env *v1.Team) error) error {
	err := o.registerTeamCRD()
	if err != nil {
		return err
	}
	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return err
	}
	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return errors.Wrap(err, "failed to create the jx client")
	}
	ns, err := kube.GetAdminNamespace(kubeClient, devNs)
	if err != nil {
		return err
	}

	if ns == "" {
		// there is no admin namespace yet so its too early to create a Team resource
		return nil
	}

	teamInterface := jxClient.JenkinsV1().Teams(ns)
	create := false
	team, err := teamInterface.Get(teamName, metav1.GetOptions{})
	if err != nil {
		team = kube.CreateTeam(ns, teamName, nil)
		create = true
	}

	original := *team
	if callback != nil {
		err = callback(team)
		if err != nil {
			return errors.Wrapf(err, "failed process Team %s", teamName)
		}
	}
	if create {
		_, err = teamInterface.Create(team)
		if err != nil {
			return errors.Wrapf(err, "failed create Team %s", teamName)
		}
	} else {
		if !reflect.DeepEqual(&original, team) {
			_, err = teamInterface.Update(team)
			if err != nil {
				return errors.Wrapf(err, "failed update Team %s", teamName)
			}
		}
	}
	return nil
}

// ModifyUser lazily creates the user if it does not exist or updates it if it requires a change
func (o *CommonOptions) ModifyUser(userName string, callback func(env *v1.User) error) error {
	err := o.registerUserCRD()
	if err != nil {
		return err
	}
	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return err
	}
	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return errors.Wrap(err, "failed to create the jx client")
	}
	ns, err := kube.GetAdminNamespace(kubeClient, devNs)
	if err != nil {
		return err
	}

	if ns == "" {
		// there is no admin namespace yet so its too early to create a User resource
		return nil
	}

	userInterface := jxClient.JenkinsV1().Users(ns)
	create := false
	user, err := userInterface.Get(userName, metav1.GetOptions{})
	if err != nil {
		user = kube.CreateUser(ns, userName, "", "")
		create = true
	}

	original := *user
	if callback != nil {
		err = callback(user)
		if err != nil {
			return errors.Wrapf(err, "failed process User %s", userName)
		}
	}
	if create {
		_, err = userInterface.Create(user)
		if err != nil {
			return errors.Wrapf(err, "failed create User %s", userName)
		}
	} else {
		if !reflect.DeepEqual(&original, user) {
			_, err = userInterface.Update(user)
			if err != nil {
				return errors.Wrapf(err, "failed update User %s", userName)
			}
		}
	}
	return nil
}
