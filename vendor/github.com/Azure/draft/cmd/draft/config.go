package main

import (
	"fmt"
	"io"
	"os"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
)

const (
	configHelp = `Manage global Draft configuration stored in $DRAFT_HOME/config.toml.`
)

// DraftConfig is the configuration stored in $DRAFT_HOME/config.toml
type DraftConfig map[string]string

// ReadConfig reads in global configuration from $DRAFT_HOME/config.toml
func ReadConfig() (DraftConfig, error) {
	var data DraftConfig
	h := draftpath.Home(draftHome)
	f, err := os.Open(h.Config())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("Could not open file %s: %s", h.Config(), err)
	}
	defer f.Close()
	if _, err := toml.DecodeReader(f, &data); err != nil {
		return nil, fmt.Errorf("Could not decode config %s: %s", h.Config(), err)
	}
	return data, nil
}

// SaveConfig saves global configuration to $DRAFT_HOME/config.toml
func SaveConfig(data DraftConfig) error {
	h := draftpath.Home(draftHome)
	f, err := os.OpenFile(h.Config(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Could not open file %s: %s", h.Config(), err)
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(data)
}

func newConfigCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "manage Draft configuration",
		Long:  configHelp,
	}
	cmd.AddCommand(
		newConfigListCmd(out),
		newConfigGetCmd(out),
		newConfigSetCmd(out),
		newConfigUnsetCmd(out),
	)
	return cmd
}
