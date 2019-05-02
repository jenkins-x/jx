package apps

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"

	corev1 "k8s.io/api/core/v1"

	"github.com/jenkins-x/jx/pkg/kube"

	"github.com/pborman/uuid"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gopkg.in/AlecAivazis/survey.v1/terminal"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/environments"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/vault"
	"k8s.io/client-go/kubernetes"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/surveyutils"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

// InstallOptions are shared options for installing, removing or upgrading apps for either GitOps or HelmOps
type InstallOptions struct {
	Helmer          helm.Helmer
	KubeClient      kubernetes.Interface
	InstallTimeout  string
	JxClient        versioned.Interface
	Namespace       string
	EnvironmentsDir string
	GitProvider     gits.GitProvider
	ConfigureGitFn  gits.ConfigureGitFn
	Gitter          gits.Gitter
	Verbose         bool
	DevEnv          *jenkinsv1.Environment
	BatchMode       bool
	In              terminal.FileReader
	Out             terminal.FileWriter
	Err             io.Writer
	GitOps          bool
	TeamName        string
	VaultClient     vault.Client

	valuesFiles *environments.ValuesFiles // internal variable used to track, most be passed in
}

var defaultPrefixes = []string{
	"jx-app-",
}

// AddApp adds the app at a particular version (
// or latest if not specified) from the repository with username and password. A releaseName can be specified.
// Values can be passed with in files or as a slice of name=value pairs. An alias can be specified.
// GitOps or HelmOps will be automatically chosen based on the o.GitOps flag
func (o *InstallOptions) AddApp(app string, version string, repository string, username string, password string,
	releaseName string, valuesFiles []string, setValues []string, alias string, helmUpdate bool) error {

	o.valuesFiles = &environments.ValuesFiles{
		Items: valuesFiles,
	}

	username, password, err := helm.DecorateWithCredentials(repository, username, password, o.VaultClient, o.In,
		o.Out, o.Err)
	if err != nil {
		return errors.Wrapf(err, "locating credentials for %s", repository)
	}

	_, err = helm.AddHelmRepoIfMissing(repository, "", username, password, o.Helmer, o.VaultClient, o.In, o.Out, o.Err)
	if err != nil {
		return errors.Wrapf(err, "adding helm repo")
	}

	chartName, err := o.resolvePrefixesAgainstRepos(repository, app)
	if err != nil {
		return errors.WithStack(err)
	}

	if chartName == "" {
		return errors.Errorf("unable to find %s in %s", app, repository)
	}

	// The chart inspector allows us to operate on the unpacked chart.
	// We need to ask questions then as we have access to the schema, and can add secrets.
	interrogateChartFn := o.createInterrogateChartFn(version, chartName, repository, username, password, alias, true)

	// Called whilst the chart is unpacked and modifiable
	installAppFunc := func(dir string) error {
		//Ask the questions, this is an install, so no existing values
		chartDetails, err := interrogateChartFn(dir, make(map[string]interface{}))
		defer chartDetails.Cleanup()
		if err != nil {
			return err
		}
		if o.GitOps {
			opts := GitOpsOptions{
				InstallOptions: o,
			}
			err := opts.AddApp(chartDetails.Name, dir, chartDetails.Version, repository, alias)
			if err != nil {
				return errors.Wrapf(err, "adding app %s version %s with alias %s using gitops", chartName, version, alias)
			}
		} else {
			opts := HelmOpsOptions{
				InstallOptions: o,
			}
			if releaseName == "" {
				releaseName = fmt.Sprintf("%s-%s", o.Namespace, chartDetails.Name)
			}
			err = opts.AddApp(chartName, dir, chartDetails.Name, chartDetails.Version, chartDetails.Values, repository,
				username, password,
				releaseName,
				setValues,
				helmUpdate)
			if err != nil {
				errStr := fmt.Sprintf("adding app %s version %s using helm", chartName, version)
				if alias != "" {
					errStr = fmt.Sprintf("%s with alias %s", errStr, alias)
				}
				errStr = fmt.Sprintf("%s with helm", errStr)
				return errors.Wrap(err, errStr)
			}
		}
		return nil
	}

	// Do the actual work
	return helm.InspectChart(chartName, version, repository, username, password, o.Helmer, installAppFunc)
}

//GetApps gets a list of installed apps
func (o *InstallOptions) GetApps(appNames []string) (apps *jenkinsv1.AppList, err error) {
	if err != nil {
		return nil, errors.Wrap(err, "getting jx client")
	}
	listOptions := metav1.ListOptions{}
	if len(appNames) > 0 {
		in := appNames
		if !o.GitOps {
			prefixes := o.getPrefixes()
			in := make([]string, 0)
			for _, prefix := range prefixes {
				for _, appName := range appNames {
					in = append(in, fmt.Sprintf("%s%s", prefix, appName))
				}
			}
		}
		selector := fmt.Sprintf(helm.LabelAppName+" in (%s)", strings.Join(in, ", "))
		listOptions.LabelSelector = selector
	}
	apps, err = o.JxClient.JenkinsV1().Apps(o.Namespace).List(listOptions)
	if err != nil {
		return nil, errors.Wrap(err, "listing apps")
	}
	return apps, nil
}

//DeleteApp deletes the app. An alias and releaseName can be specified. GitOps or HelmOps will be automatically chosen based on the o.GitOps flag
func (o *InstallOptions) DeleteApp(app string, alias string, releaseName string, purge bool) error {
	o.valuesFiles = &environments.ValuesFiles{
		Items: make([]string, 0),
	}

	chartName := app
	if o.GitOps {
		opts := GitOpsOptions{
			InstallOptions: o,
		}
		err := opts.DeleteApp(chartName, alias)
		if err != nil {
			return err
		}
		// TODO support prefixed name (requires get apps to support gitops see
	} else {
		apps, err := o.GetApps([]string{app})
		if err != nil {
			return errors.WithStack(err)
		}
		if len(apps.Items) == 0 {
			return errors.Errorf("No app found for %s", app)
		}
		if len(apps.Items) > 1 {
			appNames := make([]string, 0)
			for _, app := range apps.Items {
				appNames = append(appNames, app.Labels[helm.LabelAppName])
			}
			return errors.Errorf("Found more than one app for %s (%v)", app, appNames)
		}
		chartName := apps.Items[0].Labels[helm.LabelAppName]

		opts := HelmOpsOptions{
			InstallOptions: o,
		}
		err = opts.DeleteApp(chartName, releaseName, true)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpgradeApp upgrades the app (or all apps if empty) to a particular version (
// or the latest if not specified) from the repository with username and password. An alias can be specified.
// GitOps or HelmOps will be automatically chosen based on the o.GitOps flag
func (o *InstallOptions) UpgradeApp(app string, version string, repository string, username string, password string,
	releaseName string, alias string, update bool, askExisting bool) error {
	o.valuesFiles = &environments.ValuesFiles{
		Items: make([]string, 0),
	}

	if releaseName == "" {
		releaseName = fmt.Sprintf("%s-%s", o.Namespace, app)
	}

	username, password, err := helm.DecorateWithCredentials(repository, username, password, o.VaultClient, o.In,
		o.Out, o.Err)
	if err != nil {
		return errors.Wrapf(err, "locating credentials for %s", repository)
	}
	_, err = helm.AddHelmRepoIfMissing(repository, "", username, password, o.Helmer, o.VaultClient, o.In, o.Out, o.Err)

	if err != nil {
		return errors.Wrapf(err, "adding helm repo")
	}

	chartName := ""
	// empty app means upgrade all
	if app != "" {
		chartName, err = o.resolvePrefixesAgainstRepos(repository, app)
		if err != nil {
			return errors.WithStack(err)
		}

		if chartName == "" {
			return errors.Errorf("unable to find %s in %s", chartName, repository)
		}
	}

	interrogateChartFunc := o.createInterrogateChartFn(version, app, repository, username, password, alias, askExisting)

	// The chart inspector allows us to operate on the unpacked chart.
	// We need to ask questions then as we have access to the schema, and can add secrets.

	if o.GitOps {
		opts := GitOpsOptions{
			InstallOptions: o,
		}
		// Asking questions is a bit more complex in this case as the existing values file is in the environment
		// repo, so we need to ask questions once we have that repo available
		err := opts.UpgradeApp(chartName, version, repository, username, password, alias, interrogateChartFunc)
		if err != nil {
			return err
		}
	} else {
		upgradeAppFunc := func(dir string) error {
			// Try to load existing answers from the apps CRD
			appCrdName := fmt.Sprintf("%s-%s", releaseName, chartName)
			appResource, err := o.JxClient.JenkinsV1().Apps(o.Namespace).Get(appCrdName, metav1.GetOptions{})
			if err != nil {
				return errors.Wrapf(err, "getting App CRD %s", appResource.Name)
			}
			var existingValues map[string]interface{}
			if appResource.Annotations != nil {
				if encodedValues, ok := appResource.Annotations[ValuesAnnotation]; ok && encodedValues != "" {
					existingValuesBytes, err := base64.StdEncoding.DecodeString(encodedValues)
					if err != nil {
						log.Warnf("Error decoding base64 encoded string from %s on %s\n%s\n", ValuesAnnotation,
							appCrdName, encodedValues)
					}
					err = json.Unmarshal(existingValuesBytes, &existingValues)
					if err != nil {
						return errors.Wrapf(err, "unmarshaling %s", string(existingValuesBytes))
					}
				}
			}

			// Ask the questions
			chartDetails, err := interrogateChartFunc(dir, existingValues)
			defer chartDetails.Cleanup()
			if err != nil {
				return errors.Wrapf(err, "asking questions")
			}

			opts := HelmOpsOptions{
				InstallOptions: o,
			}
			err = opts.UpgradeApp(chartName, version, repository, username, password, releaseName, alias, update)
			if err != nil {
				return err
			}
			return nil
		}
		// Do the actual work
		err := helm.InspectChart(chartName, version, repository, username, password, o.Helmer, upgradeAppFunc)
		if err != nil {
			return err
		}
	}
	return nil

}

// ChartDetails are details about a chart returned by the chart interrogator
type ChartDetails struct {
	Values  []byte
	Version string
	Name    string
	Cleanup func()
}

func (o *InstallOptions) createInterrogateChartFn(version string, chartName string, repository string, username string,
	password string, alias string, askExisting bool) func(chartDir string,
	existing map[string]interface{}) (*ChartDetails, error) {

	return func(chartDir string, existing map[string]interface{}) (*ChartDetails, error) {
		var schema []byte
		chartDetails := ChartDetails{
			Cleanup: func() {},
		}
		chartyamlpath := filepath.Join(chartDir, "Chart.yaml")
		if _, err := os.Stat(chartyamlpath); err == nil {
			loadedName, loadedVersion, err := helm.LoadChartNameAndVersion(chartyamlpath)
			if err != nil {
				return &chartDetails, errors.Wrapf(err, "error loading chart from %s", chartDir)
			}
			chartDetails.Name = loadedName
			chartDetails.Version = loadedVersion
		} else {
			chartDetails.Name = chartName
			chartDetails.Version = version
		}

		requirements, err := helm.LoadRequirementsFile(helm.RequirementsFileName)
		if err != nil {
			return &chartDetails, errors.Wrapf(err, "loading requirements.yaml for %s", chartDir)
		}
		for _, requirement := range requirements.Dependencies {
			// repositories that start with an @ are aliases to helm repo names
			if !strings.HasPrefix(requirement.Repository, "@") {
				_, err := helm.AddHelmRepoIfMissing(requirement.Repository, "", "", "", o.Helmer, o.VaultClient, o.In,
					o.Out, o.Err)
				if err != nil {
					return &chartDetails, errors.Wrapf(err, "")
				}
			}
		}

		if version == "" {
			if o.Verbose {
				log.Infof("No version specified so using latest version which is %s\n",
					util.ColorInfo(chartDetails.Version))
			}
		}

		schemaFile := filepath.Join(chartDir, "values.schema.json")
		if _, err := os.Stat(schemaFile); !os.IsNotExist(err) {
			schema, err = ioutil.ReadFile(schemaFile)
			if err != nil {
				return &chartDetails, errors.Wrapf(err, "error reading schema file %s", schemaFile)
			}
		}
		var values []byte

		if schema != nil {
			if o.valuesFiles != nil && len(o.valuesFiles.Items) > 0 {
				log.Warnf("values.yaml specified by --valuesFiles will be used despite presence of schema in app")
			}

			appResource, _, err := environments.LocateAppResource(o.Helmer, chartDir, chartDetails.Name)
			if err != nil {
				return &chartDetails, errors.Wrapf(err, "locating app resource in %s", chartDir)
			}
			if appResource.Spec.SchemaPreprocessor != nil {
				id := uuid.New()
				cmName := toValidName(chartDetails.Name, "schema", id)
				cm := corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: cmName,
					},
					Data: map[string]string{
						"values.schema.json": string(schema),
					},
				}
				_, err := o.KubeClient.CoreV1().ConfigMaps(o.Namespace).Create(&cm)
				defer func() {
					err := o.KubeClient.CoreV1().ConfigMaps(o.Namespace).Delete(cmName, &metav1.DeleteOptions{})
					if err != nil {
						log.Errorf("error removing configmap %s: %v", cmName, err)
					}
				}()
				if err != nil {
					return &chartDetails, errors.Wrapf(err, "creating configmap %s for values.schema."+
						"json preprocessing", cmName)
				}
				// We launch this as a pod in the cluster, mounting the values.schema.json
				if appResource.Spec.SchemaPreprocessor.Env == nil {
					appResource.Spec.SchemaPreprocessor.Env = make([]corev1.EnvVar, 0)
				}
				appResource.Spec.SchemaPreprocessor.Env = append(appResource.Spec.SchemaPreprocessor.
					Env, corev1.EnvVar{
					Name:  "VALUES_SCHEMA_JSON_CONFIG_MAP_NAME",
					Value: cmName,
				})
				serviceAccountName := toValidName(chartName, "schema-sa%s", id)

				role := &rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name: toValidName(chartName, "schema-role", id),
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{
								corev1.GroupName,
							},
							Resources: []string{
								"configmaps",
							},
							Verbs: []string{
								"get",
								"update",
								"delete",
							},
							ResourceNames: []string{
								cmName,
							},
						},
					},
				}
				if appResource.Spec.SchemaPreprocessor.Name == "" {
					appResource.Spec.SchemaPreprocessor.Name = "preprocessor"
				}
				if appResource.Spec.SchemaPreprocessorRole != nil {
					role = appResource.Spec.SchemaPreprocessorRole
				}
				_, err = o.KubeClient.RbacV1().Roles(o.Namespace).Create(role)
				defer func() {
					err := o.KubeClient.RbacV1().Roles(o.Namespace).Delete(role.Name, &metav1.DeleteOptions{})
					if err != nil {
						log.Errorf("Error deleting role %s created for values.schema.json preprocessing: %v",
							role.Name, err)
					}
				}()
				if err != nil {
					return &chartDetails, errors.Wrapf(err, "creating role %s for values.schema.json preprocessing",
						role.Name)
				}
				serviceAccount := &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name: serviceAccountName,
					},
				}
				_, err = o.KubeClient.CoreV1().ServiceAccounts(o.Namespace).Create(serviceAccount)
				defer func() {
					err := o.KubeClient.CoreV1().ServiceAccounts(o.Namespace).Delete(serviceAccountName, &metav1.DeleteOptions{})
					if err != nil {
						log.Errorf("Error deleting serviceaccount %s created for values.schema.json preprocessing: %v",
							serviceAccountName, err)
					}
				}()
				if err != nil {
					return &chartDetails, errors.Wrapf(err, "creating serviceaccount %s for values.schema."+
						"json preprocessing: %v", serviceAccountName, err)
				}
				roleBinding := rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: toValidName(chartName, "schema-rolebinding", id),
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Name:     role.Name,
						Kind:     "Role",
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      rbacv1.ServiceAccountKind,
							Name:      serviceAccountName,
							Namespace: o.Namespace,
							APIGroup:  corev1.GroupName,
						},
					},
				}
				_, err = o.KubeClient.RbacV1().RoleBindings(o.Namespace).Create(&roleBinding)
				defer func() {
					err := o.KubeClient.RbacV1().RoleBindings(o.Namespace).Delete(roleBinding.Name,
						&metav1.DeleteOptions{})
					if err != nil {
						log.Errorf("Error deleting rolebinding %s for values.schema.json preprocessing: %v",
							roleBinding.Name, err)
					}
				}()
				pod := corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: toValidName(chartName, "values-preprocessor", id),
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							*appResource.Spec.SchemaPreprocessor,
						},
						ServiceAccountName: serviceAccountName,
						RestartPolicy:      corev1.RestartPolicyNever,
					},
				}
				log.Infof("Preparing questions to configure %s.\n"+
					"If this is the first time you have installed the app, this may take a couple of minutes.",
					chartDetails.Name)
				_, err = o.KubeClient.CoreV1().Pods(o.Namespace).Create(&pod)
				if err != nil {
					return &chartDetails, errors.Wrapf(err, "creating pod %s for values.schema.json proprocessing",
						pod.Name)
				}
				timeout, err := time.ParseDuration(fmt.Sprintf("%ss", o.InstallTimeout))
				if err != nil {
					return &chartDetails, errors.Wrapf(err, "invalid timeout %s", o.InstallTimeout)
				}
				err = kube.WaitForPodNameToBeComplete(o.KubeClient, o.Namespace, pod.Name, timeout)
				if err != nil {
					return &chartDetails, errors.Wrapf(err, "waiting for %s to complete for values.schema."+
						"json preprocessing",
						pod.Name)
				}
				completePod, err := o.KubeClient.Core().Pods(o.Namespace).Get(pod.Name, metav1.GetOptions{})
				if err != nil {
					return &chartDetails, errors.Wrapf(err, "getting pod %s", pod.Name)
				}
				if kube.PodStatus(completePod) == string(corev1.PodFailed) {
					log.Errorf("Pod Log")
					log.Errorf("-----------")
					err := kube.TailLogs(o.Namespace, pod.Name, appResource.Spec.SchemaPreprocessor.Name, o.Err, o.Out)
					log.Errorf("-----------")
					if err != nil {
						return &chartDetails, errors.Wrapf(err, "getting pod logs for %s container %s", pod.Name,
							appResource.Spec.SchemaPreprocessor.Name)
					}
					return &chartDetails, errors.Errorf("failed to prepare questions")
				}
				log.Infof("Questions prepared.")
				newCm, err := o.KubeClient.CoreV1().ConfigMaps(o.Namespace).Get(cmName, metav1.GetOptions{})
				if err != nil {
					return &chartDetails, errors.Wrapf(err, "getting configmap %s for values.schema."+
						"json preprocessing", cmName)
				}
				if v, ok := newCm.Data["values.schema.json"]; !ok {
					return &chartDetails, errors.Errorf("no key values.schema.json in configmap %s for values.schema."+
						"json preprocessing", cmName)
				} else {
					schema = []byte(v)
				}
			}

			if err != nil {
				return &chartDetails, errors.Wrapf(err, "locating app resource for %s", chartName)
			}
			var secrets []*surveyutils.GeneratedSecret
			var basepath string
			if o.GitOps {
				gitInfo, err := gits.ParseGitURL(o.DevEnv.Spec.Source.URL)
				if err != nil {
					return nil, err
				}
				basepath = strings.Join([]string{"gitOps", gitInfo.Organisation, gitInfo.Name}, "/")
			} else {
				basepath = strings.Join([]string{"teams", o.TeamName}, "/")
			}
			values, secrets, err = GenerateQuestions(schema, o.BatchMode, askExisting, basepath, o.VaultClient != nil, existing, o.In, o.Out, o.Err)
			if err != nil {
				return &chartDetails, errors.Wrapf(err, "asking questions for schema %s", schemaFile)
			}
			cleanupValues, err := o.handleValues(chartDir, chartName, values)
			chartDetails.Cleanup = func() {
				cleanupValues()
			}
			if err != nil {
				return &chartDetails, err
			}
			cleanupSecrets, err := o.handleSecrets(chartDir, chartName, secrets)
			chartDetails.Cleanup = func() {
				cleanupSecrets()
				cleanupValues()
			}
			if err != nil {
				return &chartDetails, err
			}
			chartDetails.Cleanup = func() {
				cleanupSecrets()
				cleanupValues()
			}
		}
		chartDetails.Values = values
		return &chartDetails, nil
	}
}

func toValidName(appName string, name string, id string) string {
	base := fmt.Sprintf("%s-%s", name, appName)
	l := len(base)
	if l > 20 {
		l = 20
	}
	return kube.ToValidName(fmt.Sprintf("%s-%s", base[0:l], id))
}

func (o *InstallOptions) handleValues(dir string, chartName string, values []byte) (func(), error) {
	valuesFile, cleanup, err := AddValuesToChart(chartName, values, o.Verbose)
	if err != nil {
		return cleanup, err
	}
	if valuesFile != "" {
		o.valuesFiles.Items = append(o.valuesFiles.Items, valuesFile)
	}
	return cleanup, nil
}

func (o *InstallOptions) handleSecrets(dir string, chartName string, generatedSecrets []*surveyutils.GeneratedSecret) (func(),
	error) {
	if o.VaultClient != nil {
		f, err := AddSecretsToVault(generatedSecrets, o.VaultClient)
		if err != nil {
			return func() {}, errors.Wrapf(err, "adding secrets to vault for %s", chartName)
		}
		return f, nil
	}
	secretsFile, f, err := AddSecretsToTemplate(dir, chartName, generatedSecrets)
	if err != nil {
		return func() {}, errors.Wrapf(err, "adding secrets to template for %s", chartName)
	}
	if secretsFile != "" {
		o.valuesFiles.Items = append(o.valuesFiles.Items, secretsFile)
	}
	return f, nil
}

func (o *InstallOptions) getPrefixes() []string {
	// Set the default prefixes
	prefixes := o.DevEnv.Spec.TeamSettings.AppsPrefixes
	if prefixes == nil {
		prefixes = []string{
			"jx-app-",
		}
	}
	prefixes = append(prefixes, "")
	return prefixes
}

func (o *InstallOptions) resolvePrefixesAgainstRepos(repository string, chartName string) (string, error) {
	prefixes := o.getPrefixes()

	// Create the short chart name
	repos, err := o.Helmer.ListRepos()
	if err != nil {
		return "", errors.Wrapf(err, "listing helm repos")
	}
	possiblesRepoNames := make([]string, 0)
	for repo, url := range repos {
		if url == repository {
			possiblesRepoNames = append(possiblesRepoNames, repo)
		}
	}
	charts, err := o.Helmer.SearchCharts("")
	if err != nil {
		return "", errors.Wrapf(err, "searching charts")
	}

	for _, prefix := range prefixes {
		for _, possibleRepoName := range possiblesRepoNames {
			fullName := fmt.Sprintf("%s/%s%s", possibleRepoName, prefix, chartName)
			for _, chart := range charts {
				if chart.Name == fullName {
					// Chart found!
					return fmt.Sprintf("%s%s", prefix, chartName), nil
				}
			}
		}
	}
	return chartName, nil
}
