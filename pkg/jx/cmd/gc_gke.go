package cmd

import (
	"io"
	"strings"

	"github.com/spf13/cobra"

	"encoding/json"

	"fmt"

	"io/ioutil"

	"os"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type GCGKEOptions struct {
	CommonOptions

	RevisionHistoryLimit int
	jclient              *gojenkins.Jenkins
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

// NewCmd s a command object for the "step" command
func NewCmdGCGKE(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GCGKEOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
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
			cmdutil.CheckErr(err)
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

`
	log.Warn("This command is experimental and the generated script should be executed at the users own risk\n")
	log.Warn("We will generate a script for you to review and execute, this command will not delete any resources by itself\n")
	log.Info("It may take a few minutes to create the script\n")

	fw, err := o.cleanUpFirewalls()
	if err != nil {
		return err
	}

	disks, err := o.cleanUpPersistentDisks()
	if err != nil {
		return err
	}

	data := fmt.Sprintf(message, fw, disks)
	data = strings.Replace(data, "[", "", -1)
	data = strings.Replace(data, "]", "", -1)

	err = ioutil.WriteFile("gc_gke.sh", []byte(data), util.DefaultWritePermissions)
	return nil
}

func (p *GCGKEOptions) cleanUpFirewalls() (string, error) {
	o := CommonOptions{}
	data, err := o.getCommandOutput("", "gcloud", "compute", "firewall-rules", "list", "--format", "json")
	if err != nil {
		return "", err
	}

	var rules []Rule
	err = json.Unmarshal([]byte(data), &rules)
	if err != nil {
		return "", err
	}

	out, err := o.getCommandOutput("", "gcloud", "container", "clusters", "list")
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(out), "\n")
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

	args := "gcloud compute firewall-rules delete "
	for _, name := range nameToDelete {
		args = args + " " + name
	}

	return args, nil
}

func (o *GCGKEOptions) cleanUpPersistentDisks() ([]string, error) {

	cmd := "gcloud compute zones list | grep -v NAME | awk '{printf $1 \" \"}'"

	rs, err := o.getCommandOutput("", "bash", "-c", cmd)
	if err != nil {
		return nil, err
	}
	zones := strings.Split(rs, " ")
	var line []string

	for _, z := range zones {
		diskCmd := fmt.Sprintf("gcloud compute disks list --filter=\"NOT users:* AND zone:(%s)\" | grep -v NAME | awk '{printf $1 \" \"}'", z)
		diskRs, err := o.getCommandOutput("", "bash", "-c", diskCmd)
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
