package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jenkins-x/jx/pkg/extensions"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
)

func (o *CommonOptions) GetCommands() ([]templates.CommandGroup, error) {
	result := make([]templates.CommandGroup, 0)
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return result, err
	}
	apisClient, err := o.CreateApiExtensionsClient()
	if err != nil {
		return result, err
	}
	err = kube.RegisterPluginCRD(apisClient)
	if err != nil {
		return result, err
	}

	plugins, err := jxClient.JenkinsV1().Plugins(ns).List(metav1.ListOptions{})
	if err != nil {
		return result, err
	}
	groups := make(map[string][]jenkinsv1.Plugin, 0)
	for _, p := range plugins.Items {
		if _, ok := groups[p.Spec.Group]; !ok {
			groups[p.Spec.Group] = make([]jenkinsv1.Plugin, 0)
		}
		groups[p.Spec.Group] = append(groups[p.Spec.Group], p)
	}
	return result, nil
}

func (o *CommonOptions) getPluginCommandGroups(verifier extensions.PathVerifier) (templates.PluginCommandGroups, error) {
	apisClient, err := o.CreateApiExtensionsClient()
	if err != nil {
		return nil, err
	}

	err = kube.RegisterPluginCRD(apisClient)
	if err != nil {
		return nil, err
	}

	jxClient, ns, err := o.Factory.CreateJXClient()
	if err != nil {
		return nil, err
	}
	plugs := jxClient.JenkinsV1().Plugins(ns)

	plugins, err := plugs.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	groups := make(map[string]templates.PluginCommandGroup, 0)
	otherCommands := templates.PluginCommandGroup{
		Message: "Other Commands",
	}
	pathCommands := templates.PluginCommandGroup{
		Message: "Locally Available Commands:",
	}
	for _, plugin := range plugins.Items {
		pluginCommand := &templates.PluginCommand{
			PluginSpec: plugin.Spec,
		}
		if plugin.Spec.Group == "" {
			otherCommands.Commands = append(otherCommands.Commands, pluginCommand)
		} else {
			if g, ok := groups[plugin.Spec.Group]; !ok {
				groups[plugin.Spec.Group] = templates.PluginCommandGroup{
					Message: fmt.Sprintf("%s:", plugin.Spec.Group),
					Commands: []*templates.PluginCommand{
						pluginCommand,
					},
				}
			} else {
				g.Commands = append(g.Commands, pluginCommand)
			}
		}
	}

	path := "PATH"
	if runtime.GOOS == "windows" {
		path = "path"
	}

	paths := sets.NewString(filepath.SplitList(os.Getenv(path))...)
	for _, dir := range paths.List() {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, f := range files {
			if f.IsDir() {
				continue
			}
			if !strings.HasPrefix(f.Name(), "jx-") {
				continue
			}

			pluginPath := filepath.Join(dir, f.Name())
			subCommand := strings.TrimPrefix(strings.Replace(filepath.Base(pluginPath), "-", " ", -1), "jx ")
			pc := &templates.PluginCommand{
				PluginSpec: jenkinsv1.PluginSpec{
					SubCommand:  subCommand,
					Description: pluginPath,
				},
				Errors: make([]error, 0),
			}
			pathCommands.Commands = append(pathCommands.Commands, pc)
			if errs := verifier.Verify(filepath.Join(dir, f.Name())); len(errs) != 0 {
				for _, err := range errs {
					pc.Errors = append(pc.Errors, err)
				}
			}
		}
	}

	pcgs := templates.PluginCommandGroups{}
	for _, g := range groups {
		pcgs = append(pcgs, g)
	}
	if len(otherCommands.Commands) > 0 {
		pcgs = append(pcgs, otherCommands)
	}
	if len(pathCommands.Commands) > 0 {
		pcgs = append(pcgs, pathCommands)
	}
	return pcgs, nil
}
