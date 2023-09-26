/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/kubestrap/pkg/kubestrap"
	"golang.org/x/exp/slices"

	go_cmd "github.com/go-cmd/cmd"
)

type Raw struct {
	cmd    *cobra.Command
	parent *Root
}

// rawCmd represents the raw command
var (
	rawCmd = &cobra.Command{
		Use:     "raw",
		Short:   "Directly run one of the predefined utilities. To pass flags for the raw command, use --",
		Long:    ``,
		RunE:    RunRawCommand,
		Aliases: []string{"r"},
	}

	raw = NewRaw(root)
)

func init() {
	rootCmd.AddCommand(rawCmd)
	rawCmd.SilenceErrors = rawCmd.Parent().SilenceErrors

	rawCmd.Flags().DurationP(
		raw.KeyTimeout(),
		"t",
		raw.DefaultTimeout(),
		"Timeout for executing raw command. After time elapses, the command will be terminated",
	)

	// Bind flags
	config.ViperBindPFlagSet(rawCmd, nil)

	raw.SetCmd(rawCmd)
}

// RunRawCommand unmarshal commands and executes with provided arguments
func RunRawCommand(cmd *cobra.Command, args []string) error {
	_, err := LoadRawCommandsAndRunOne(cmd, args, false)
	return err
}

func LoadRawCommandsAndRunOne(cmd *cobra.Command, args []string, buffered bool) (*go_cmd.Status, error) {
	var commands []kubestrap.RawCommand
	if err := viper.UnmarshalKey(
		config.PrefixKey(cmd, raw.KeyRawUtilities()),
		&commands,
		func(config *mapstructure.DecoderConfig) {
			config.TagName = "yaml"
			config.ErrorUnused = true
			// config.ErrorUnset = true
		},
	); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		_ = cmd.Help()
		fmt.Printf("\nAvailable utilities:\n")
		for _, c := range commands {
			fmt.Printf("  - %s %s\n", c.Name, c.Release)
		}
		return nil, nil
	}
	for _, c := range commands {
		if c.Name == args[0] || slices.Contains(c.Additional, args[0]) {
			timeout := raw.GetTimeout()
			log.Debugf("execution timeout: %s", timeout)
			c.Command = args
			status, err := c.ExecuteCommand(timeout, buffered)
			if status.Exit != 0 {
				log.Errorf("command '%s' failed with code %d", args[0], status.Exit)
			}
			// Config allows for duplicates, but here we stop at the first match
			return status, err
		}
	}
	// If we get here, the command is not in the config, do not allow that
	return nil, fmt.Errorf("command '%s' is not supported, perhaps add it to the config?", args[0])
}

func NewRaw(parent *Root) *Raw {
	return &Raw{
		parent: parent,
	}
}

func (r *Raw) SetCmd(cmd *cobra.Command) {
	r.cmd = cmd
}

// Flags keys, defaults and value getters
func (r *Raw) KeyTimeout() string {
	return "timeout"
}

func (r *Raw) DefaultTimeout() time.Duration {
	d, _ := time.ParseDuration("1m0s")
	return d
}

func (r *Raw) GetTimeout() time.Duration {
	return config.ViperGetDuration(r.cmd, r.KeyTimeout())
}

func (r *Raw) KeyRawUtilities() string {
	return "utilities"
}
