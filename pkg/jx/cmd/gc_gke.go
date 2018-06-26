package cmd

import (
	"io"
	"strings"

	"github.com/spf13/cobra"

	"encoding/json"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/log"
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

	err := o.cleanUpFirewalls()
	if err != nil {
		return err
	}

	err = o.cleanUpPersistentDisks()
	if err != nil {
		return err
	}
	return nil
}

func (p *GCGKEOptions) cleanUpFirewalls() error {
	o := CommonOptions{}
	data, err := o.getCommandOutput("", "gcloud", "compute", "firewall-rules", "list", "--format", "json")
	if err != nil {
		return err
	}

	var rules []Rule
	err = json.Unmarshal([]byte(data), &rules)
	if err != nil {
		return err
	}

	out, err := o.getCommandOutput("", "gcloud", "container", "clusters", "list")
	if err != nil {
		return err
	}

	lines := strings.Split(string(out), "\n")
	var existingClusters []string
	for _, l := range lines {
		if strings.Contains(l, "NAME") {
			continue
		}
		log.Infof("Line: %s \n", l)
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

					log.Infof("Matched: %s %s\n", rule.Name, tagFull)
					nameToDelete = append(nameToDelete, rule.Name)
				}
			}
		}
	}
	message := `

###################################################################################################
#
#  WARNING: this command is experimental and should be executed at the users own risk.  We use this
#  generated command on the Jenkins X project itself but it has not been tested on other clusters.
#
###################################################################################################


`
	log.Warn(message)
	args := "gcloud compute firewall-rules delete "
	for _, name := range nameToDelete {
		args = args + " " + name
	}
	log.Info(args)
	return nil
}

func (o *GCGKEOptions) cleanUpPersistentDisks() error {

	zones := o.getCommandOutput()

	echo "Checking whether there are disks to delete..."
	for zone in $(gcloud compute zones list | grep -v NAME | awk '{printf $1 " "}')
do
echo "Checking zone $zone"
for disk in $(gcloud compute disks list --filter="NOT users:* AND zone:($zone)" | grep -v NAME | awk '{printf $1 " "}')
do
echo "deleting ${disk}..."
#gcloud compute disks delete --zone=$zone --quiet ${disk}
done
done

	return nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if strings.HasPrefix(e, a) {
			return true
		}
	}
	return false
}
