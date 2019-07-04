package gc

import (
	"strings"

	"github.com/jenkins-x/jx/pkg/cmd/helper"

	"github.com/spf13/cobra"

	"encoding/json"

	"fmt"

	"io/ioutil"

	"os"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type GCGKEOptions struct {
	*opts.CommonOptions

	RevisionHistoryLimit int
	jclient              gojenkins.JenkinsClient
}

var (
	GCGKELong = templates.LongDesc(`
		Garbage collect Google Container Engine resources that are not deleted when a delete cluster is performed

		This command will generate the gcloud command to run and delete external loadbalancers and persistent disks
		that are no longer in use.

`)

	GCGKEExample = templates.Examples(`
		jx garbage collect gke
		jx gc gke
`)

	ServiceAccountSuffixes = []string{"-vt", "-ko", "-tf"}
)

type Rules struct {
	Rules []Rule
}

type Rule struct {
	Name       string   `json:"name"`
	TargetTags []string `json:"targetTags"`
}

type cluster struct {
	Name string `json:"name"`
}

type address struct {
	Name   string `json:"name"`
	Region string `json:"region"`
}

type zone struct {
	Name string `json:"name"`
}

type disk struct {
	Name string `json:"name"`
}

type serviceAccount struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

type iamPolicy struct {
	Bindings []iamBinding `json:"bindings"`
}

type iamBinding struct {
	Members []string `json:"members"`
	Role    string   `json:"role"`
}

// NewCmdGCGKE is a command object for the "step" command
func NewCmdGCGKE(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &GCGKEOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:     "gke",
		Short:   "garbage collection for gke",
		Long:    GCGKELong,
		Example: GCGKEExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	return cmd
}

// Run implements this command
func (o *GCGKEOptions) Run() error {

	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	path := util.UrlJoin(dir, "gc_gke.sh")
	util.FileExists(path)

	os.Remove(path)

	message := `#!/bin/bash

set -euo pipefail

###################################################################################################
#
#  WARNING: this command is experimental and the generated script should be executed at the users own risk.  We use this
#  generated command on the Jenkins X project itself but it has not been tested on other clusters.
#
###################################################################################################

%s

%s

%s

%s

`
	log.Logger().Warn("This command is experimental and the generated script should be executed at the users own risk\n")
	log.Logger().Warn("We will generate a script for you to review and execute, this command will not delete any resources by itself\n")
	log.Logger().Info("It may take a few minutes to create the script\n")

	fw, err := o.cleanUpFirewalls()
	if err != nil {
		return err
	}

	disks, err := o.cleanUpPersistentDisks()
	if err != nil {
		return err
	}

	addr, err := o.cleanUpAddresses()
	if err != nil {
		return err
	}

	serviceAccounts, err := o.cleanUpServiceAccounts()
	if err != nil {
		return err
	}

	data := fmt.Sprintf(message, fw, strings.Join(disks, "\n"), strings.Join(addr, "\n"), strings.Join(serviceAccounts, "\n"))
	data = strings.Replace(data, "[", "", -1)
	data = strings.Replace(data, "]", "", -1)

	err = ioutil.WriteFile("gc_gke.sh", []byte(data), util.DefaultWritePermissions)

	log.Logger().Info("Script 'gc_gke.sh' created!")
	return nil
}

func (p *GCGKEOptions) cleanUpFirewalls() (string, error) {
	o := &opts.CommonOptions{}
	data, err := o.GetCommandOutput("", "gcloud", "compute", "firewall-rules", "list", "--format", "json")
	if err != nil {
		return "", err
	}

	var rules []Rule
	err = json.Unmarshal([]byte(data), &rules)
	if err != nil {
		return "", err
	}

	out, err := o.GetCommandOutput("", "gcloud", "container", "clusters", "list")
	if err != nil {
		return "", err
	}

	lines := strings.Split(out, "\n")
	var existingClusters []string
	for _, l := range lines {
		if strings.Contains(l, "NAME") {
			continue
		}
		if strings.TrimSpace(l) == "" {
			break
		}
		fields := strings.Fields(l)
		existingClusters = append(existingClusters, fields[0])
	}

	var nameToDelete []string
	for _, rule := range rules {
		name := strings.TrimPrefix(rule.Name, "gke-")

		if !contains(existingClusters, name) {
			for _, tagFull := range rule.TargetTags {
				tag := strings.TrimPrefix(tagFull, "gke-")

				if !contains(existingClusters, tag) {
					nameToDelete = append(nameToDelete, rule.Name)
				}
			}
		}
	}

	if nameToDelete != nil {
		args := "gcloud compute firewall-rules delete "
		for _, name := range nameToDelete {
			args = args + " " + name
		}
		return args, nil
	}

	return "# No firewalls found for deletion", nil
}

func (o *GCGKEOptions) cleanUpPersistentDisks() ([]string, error) {
	zones, err := o.getZones()
	if err != nil {
		return nil, err
	}
	var line []string

	for _, z := range zones {
		disks, err := o.getUnusedDisksForZone(z)
		if err != nil {
			return nil, err
		}

		for _, d := range disks {
			if strings.HasPrefix(d.Name, "gke-") {
				line = append(line, fmt.Sprintf("gcloud compute disks delete --zone=%s --quiet %s", z.Name, d.Name))
			}
		}
	}

	if len(line) == 0 {
		line = append(line, "# No disks found for deletion\n")
	}

	return line, nil
}

func (o *GCGKEOptions) cleanUpAddresses() ([]string, error) {

	cmd := "gcloud compute addresses list --filter=\"status:RESERVED\" --format=json"
	data, err := o.GetCommandOutput("", "bash", "-c", cmd)
	if err != nil {
		return nil, err
	}

	var addresses []address
	err = json.Unmarshal([]byte(data), &addresses)
	if err != nil {
		return nil, err
	}

	var line []string
	if len(addresses) > 0 {
		for _, address := range addresses {
			var scope string
			if address.Region != "" {
				region := getLastString(strings.Split(address.Region, "/"))
				scope = fmt.Sprintf("--region %s", region)
			} else {
				scope = "--global"
			}
			line = append(line, fmt.Sprintf("gcloud compute addresses delete %s %s", address.Name, scope))
		}
		return line, nil
	}

	if len(line) == 0 {
		line = append(line, "# No addresses found for deletion\n")
	}

	return line, nil
}

func (o *GCGKEOptions) cleanUpServiceAccounts() ([]string, error) {
	serviceAccounts, err := o.getFilteredServiceAccounts()
	if err != nil {
		return nil, err
	}
	var line []string

	if len(serviceAccounts) > 0 {
		for _, sa := range serviceAccounts {
			line = append(line, fmt.Sprintf("gcloud iam service-accounts delete %s --quiet", sa.Email))
		}
	}

	if len(line) == 0 {
		line = append(line, "# No service accounts found for deletion\n")
	}

	policy, err := o.getIamPolicy()
	if err != nil {
		return nil, err
	}

	iam, err := o.determineUnusedIamBindings(policy)
	if err != nil {
		return nil, err
	}

	line = append(line, iam...)

	if len(line) == 0 {
		line = append(line, "# No service accounts found for deletion\n")
	}

	return line, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if strings.HasPrefix(e, a) {
			return true
		}
	}
	return false
}

func getLastString(s []string) string {
	return s[len(s)-1]
}

func (o *GCGKEOptions) getZones() ([]zone, error) {
	cmd := "gcloud compute zones list --format=json"
	data, err := o.GetCommandOutput("", "bash", "-c", cmd)
	if err != nil {
		return nil, err
	}

	var zones []zone
	err = json.Unmarshal([]byte(data), &zones)
	if err != nil {
		return nil, err
	}

	return zones, nil
}

func (o *GCGKEOptions) getClusters() ([]cluster, error) {
	cmd := "gcloud container clusters list --format=json"
	data, err := o.GetCommandOutput("", "bash", "-c", cmd)
	if err != nil {
		return nil, err
	}

	var clusters []cluster
	err = json.Unmarshal([]byte(data), &clusters)
	if err != nil {
		return nil, err
	}

	return clusters, nil
}

func (o *GCGKEOptions) getUnusedDisksForZone(z zone) ([]disk, error) {
	diskCmd := fmt.Sprintf("gcloud compute disks list --filter=\"NOT users:* AND zone:(%s)\" --format=json", z.Name)
	data, err := o.GetCommandOutput("", "bash", "-c", diskCmd)
	if err != nil {
		return nil, err
	}

	var disks []disk
	err = json.Unmarshal([]byte(data), &disks)
	if err != nil {
		return nil, err
	}

	return disks, nil
}

func (o *GCGKEOptions) getFilteredServiceAccounts() ([]serviceAccount, error) {
	serviceAccounts, err := o.getServiceAccounts()
	if err != nil {
		return nil, err
	}

	clusters, err := o.getClusters()
	if err != nil {
		return nil, err
	}

	filteredServiceAccounts := []serviceAccount{}
	for _, sa := range serviceAccounts {
		if isServiceAccount(sa.DisplayName) {
			sz := len(sa.DisplayName)
			clusterName := sa.DisplayName[:sz-3]

			if !o.clusterExists(clusters, clusterName) {
				log.Logger().Debugf("cluster %s does not exist", clusterName)
				filteredServiceAccounts = append(filteredServiceAccounts, sa)
			}
		}
	}

	return filteredServiceAccounts, nil
}

func (o *GCGKEOptions) getServiceAccounts() ([]serviceAccount, error) {
	cmd := "gcloud iam service-accounts list --format=json"
	data, err := o.GetCommandOutput("", "bash", "-c", cmd)
	if err != nil {
		return nil, err
	}

	var serviceAccounts []serviceAccount
	err = json.Unmarshal([]byte(data), &serviceAccounts)
	if err != nil {
		return nil, err
	}

	for _, sa := range serviceAccounts {
		if sa.DisplayName == "" {
			sa.DisplayName = sa.Email[:strings.IndexByte(sa.Email, '@')]
		}
	}
	return serviceAccounts, nil
}

func (o *GCGKEOptions) clusterExists(clusters []cluster, clusterName string) bool {
	for _, cluster := range clusters {
		if cluster.Name == clusterName {
			return true
		}
	}
	return false
}

func (o *GCGKEOptions) getCurrentGoogleProjectId() (string, error) {
	cmd := "gcloud config get-value core/project"
	data, err := o.GetCommandOutput("", "bash", "-c", cmd)
	if err != nil {
		return "", err
	}
	return data, nil
}
func (o *GCGKEOptions) getIamPolicy() (iamPolicy, error) {
	googleProjectID, err := o.getCurrentGoogleProjectId()
	if err != nil {
		return iamPolicy{}, err
	}
	cmd := fmt.Sprintf("gcloud projects get-iam-policy %s --format=json", googleProjectID)
	data, err := o.GetCommandOutput("", "bash", "-c", cmd)
	if err != nil {
		return iamPolicy{}, err
	}

	var policy iamPolicy
	err = json.Unmarshal([]byte(data), &policy)
	if err != nil {
		return iamPolicy{}, err
	}

	return policy, nil
}

func (o *GCGKEOptions) determineUnusedIamBindings(policy iamPolicy) ([]string, error) {
	googleProjectID, err := o.getCurrentGoogleProjectId()
	if err != nil {
		return nil, err
	}

	var line []string

	clusters, err := o.getClusters()
	if err != nil {
		return line, err
	}
	for _, b := range policy.Bindings {
		for _, m := range b.Members {
			if strings.HasPrefix(m, "serviceAccount:") {
				saName := strings.TrimPrefix(m, "serviceAccount:")
				displayName := saName[:strings.IndexByte(saName, '@')]

				if isServiceAccount(displayName) {
					sz := len(displayName)
					clusterName := displayName[:sz-3]

					if !o.clusterExists(clusters, clusterName) {
						cmd := fmt.Sprintf("gcloud projects remove-iam-policy-binding %s --member=%s --role=%s --quiet", googleProjectID, m, b.Role)
						line = append(line, cmd)
					}
				}
			}
		}
	}
	return line, nil
}

func isServiceAccount(sa string) bool {
	for _, suffix := range ServiceAccountSuffixes {
		if strings.HasSuffix(sa, suffix) {
			return true
		}
	}
	return false
}
