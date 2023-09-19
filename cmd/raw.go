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
)

const (
	keyRawUtilities = "utilities"
	keyRawTimeout   = "timeout"
	keyRawRawOutput = "raw-output"
)

// rawCmd represents the raw command
var rawCmd = &cobra.Command{
	Use:     "raw",
	Short:   "Directly run one of the predefined utilities. To pass flags for the raw command, use --",
	Long:    ``,
	RunE:    RunRawCommand,
	Aliases: []string{"r"},
}

func init() {
	rootCmd.AddCommand(rawCmd)
	rawCmd.SilenceErrors = rawCmd.Parent().SilenceErrors

	d, _ := time.ParseDuration("1m0s")
	rawCmd.Flags().DurationP(
		keyRawTimeout,
		"t",
		d,
		"Timeout for executing raw command. After time elapses, the command will be terminated",
	)
	rawCmd.Flags().BoolP(
		keyRawRawOutput,
		"r",
		true,
		"Display raw output, outside of the logger",
	)

	// Bind flags
	config.ViperBindPFlagSet(rawCmd, nil)
}

// RunRawCommand unmarshal commands and executes with provided arguments
func RunRawCommand(cmd *cobra.Command, args []string) error {
	var commands []kubestrap.RawCommand
	if err := viper.UnmarshalKey(
		config.PrefixKey(cmd, keyRawUtilities),
		&commands,
		func(config *mapstructure.DecoderConfig) {
			config.TagName = "yaml"
			config.ErrorUnused = true
			// config.ErrorUnset = true
		},
	); err != nil {
		return err
	}
	if len(args) == 0 {
		_ = cmd.Help()
		fmt.Printf("\nAvailable utilities:\n")
		for _, c := range commands {
			fmt.Printf("  - %s %s\n", c.Name, c.Release)
		}
		return nil
	}
	for _, c := range commands {
		if c.Name == args[0] || slices.Contains(c.Additional, args[0]) {
			timeout := config.ViperGetDuration(cmd, keyRawTimeout)
			log.Debugf("execution timeout: %s", timeout)
			c.Command = args
			retCode, err := c.ExecuteCommand(timeout, config.ViperGetBool(cmd, keyRawRawOutput), false)
			if retCode != 0 {
				log.Errorf("command '%s' failed with code %d", args[0], retCode)
			}
			// Config allows for duplicates, but here we stop at the first match
			return err
		}
	}
	// If we get here, the command is not in the config, do not allow that
	return fmt.Errorf("command '%s' is not supported, perhaps add it to the config?", args[0])
}
