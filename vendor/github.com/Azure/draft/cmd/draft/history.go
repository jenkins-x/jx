package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Azure/draft/pkg/draft/local"
	"github.com/Azure/draft/pkg/storage"
	"github.com/Azure/draft/pkg/storage/kube/configmap"
	"github.com/ghodss/yaml"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"io"
	"k8s.io/helm/pkg/timeconv"
)

const historyDesc = `Display the build history of a Draft application.`

type historyCmd struct {
	out      io.Writer
	fmt      string
	env      string
	max      int64
	pretty   bool
	colWidth uint
}

func newHistoryCmd(out io.Writer) *cobra.Command {
	hc := &historyCmd{out: out}
	cmd := &cobra.Command{
		Use:   "history",
		Short: historyDesc,
		Long:  historyDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return hc.run()
		},
	}

	f := cmd.Flags()
	f.Int64Var(&hc.max, "max", 256, "maximum number of results to include in history")
	f.UintVar(&hc.colWidth, "col-width", 60, "specifies the max column width of output")
	f.BoolVar(&hc.pretty, "pretty", false, "pretty print output")
	f.StringVarP(&hc.fmt, "output", "o", "table", "prints the output in the specified format (json|table|yaml)")
	f.StringVarP(&hc.env, environmentFlagName, environmentFlagShorthand, defaultDraftEnvironment(), environmentFlagUsage)
	return cmd
}

func (cmd *historyCmd) run() error {
	app, err := local.DeployedApplication(draftToml, cmd.env)
	if err != nil {
		return err
	}
	client, _, err := getKubeClient(kubeContext)
	if err != nil {
		return fmt.Errorf("Could not get a kube client: %v", err)
	}
	store := configmap.NewConfigMaps(client.CoreV1().ConfigMaps(tillerNamespace))

	// get history from store
	h, err := getHistory(context.Background(), store, app.Name, cmd.max)
	if err != nil {
		return err
	}
	if len(h) > 0 {
		var output []byte
		switch bh := toBuildHistory(h); cmd.fmt {
		case "yaml":
			if output, err = yaml.Marshal(&bh); err != nil {
				return err
			}
		case "json":
			if output, err = json.Marshal(&bh); err != nil {
				return err
			}
			if cmd.pretty {
				var b bytes.Buffer
				json.Indent(&b, output, "", "  ")
				output = b.Bytes()
			}
		case "table":
			output = formatTable(bh, cmd.colWidth)
		default:
			return fmt.Errorf("unknown output format %q", cmd.fmt)
		}
		fmt.Fprintln(cmd.out, string(output))
	}
	return nil
}

func getHistory(ctx context.Context, store storage.Store, app string, max int64) (h []*storage.Object, err error) {
	if h, err = store.GetBuilds(ctx, app); err != nil {
		return nil, fmt.Errorf("failed to retrieve application (%q) build history from storage: %v", app, err)
	}
	// For deterministic return of history results we sort by the storage
	// object's created at timestamp.
	storage.SortByCreatedAt(h)

	min := func(x, y int) int {
		if x < y {
			return x
		}
		return y
	}

	return h[:min(len(h), int(max))], nil
}

type buildHistory []buildInfo

type buildInfo struct {
	BuildID string `json:"buildID"`
	Release string `json:"release"`
	Context string `json:"context"`
	Created string `json:"createdAt"`
}

func toBuildHistory(ls []*storage.Object) (h buildHistory) {
	orElse := func(str, def string) string {
		if str != "" {
			return str
		}
		return def
	}
	for i := len(ls) - 1; i >= 0; i-- {
		rls := orElse(ls[i].GetRelease(), "-")
		ctx := ls[i].GetContextID()
		h = append(h, buildInfo{
			BuildID: ls[i].GetBuildID(),
			Release: rls,
			Context: fmt.Sprintf("%X", ctx[len(ctx)-5:]),
			Created: timeconv.String(ls[i].GetCreatedAt()),
		})
	}
	return h
}

func formatTable(h buildHistory, w uint) []byte {
	tbl := uitable.New()
	tbl.MaxColWidth = w
	tbl.AddRow("BUILD_ID", "CONTEXT_ID", "CREATED_AT", "RELEASE")
	for i := 0; i < len(h); i++ {
		b := h[i]
		tbl.AddRow(b.BuildID, b.Context, b.Created, b.Release)
	}
	return tbl.Bytes()
}
