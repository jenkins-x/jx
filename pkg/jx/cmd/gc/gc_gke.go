package gc

import (
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"

	"github.com/spf13/cobra"

	"encoding/json"

	"fmt"

	"io/ioutil"

	"os"

	gojenkins "github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
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
)

type Rules struct {
	Rules []Rule
}

type Rule struct {
	Name       string   `json:"name"`
	TargetTags []string `json:"targetTags"`
}

type address struct {
	Name   string `json:"name"`
	Region string `json:"region"`
}

// NewCmd s a command object for the "step" command
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

	message := `

###################################################################################################
#
#  WARNING: this command is experimental and the generated script should be executed at the users own risk.  We use this
#  generated command on the Jenkins X project itself but it has not been tested on other clusters.
#
###################################################################################################

%s

%v

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

	data := fmt.Sprintf(message, fw, disks, addr)
	data = strings.Replace(data, "[", "", -1)
	data = strings.Replace(data, "]", "", -1)

	err = ioutil.WriteFile("gc_gke.sh", []byte(data), util.DefaultWritePermissions)

	log.Logger().Info("Script 'gc_gke.sh' created!\n")
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

	cmd := "gcloud compute zones list | grep -v NAME | awk '{printf $1 \" \"}'"

	rs, err := o.GetCommandOutput("", "bash", "-c", cmd)
	if err != nil {
		return nil, err
	}
	zones := strings.Split(rs, " ")
	var line []string

	for _, z := range zones {
		diskCmd := fmt.Sprintf("gcloud compute disks list --filter=\"NOT users:* AND zone:(%s)\" | grep -v NAME | awk '{printf $1 \" \"}'", z)
		diskRs, err := o.GetCommandOutput("", "bash", "-c", diskCmd)
		if err != nil {
			return nil, err
		}

		disks := strings.Split(diskRs, " ")
		for _, d := range disks {
			if strings.HasPrefix(d, "gke-") {
				line = append(line, fmt.Sprintf("gcloud compute disks delete --zone=%s --quiet %s\n", z, d))
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
			line = append(line, fmt.Sprintf("gcloud compute addresses delete %s %s\n", address.Name, scope))
		}
		return line, nil
	}

	if len(line) == 0 {
		line = append(line, "# No addresses found for deletion\n")
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
