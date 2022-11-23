/*
Copyright © 2022 Dataflows
*/
package cmd

import (
	"fmt"
	"time"

	"dataflows.com/kubestrap/internal/pkg/kubestrap"
	"dataflows.com/kubestrap/internal/pkg/logging"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	Run:     RunRawCommand,
	Aliases: []string{"r"},
}

func init() {
	rootCmd.AddCommand(rawCmd)

	d, _ := time.ParseDuration("1m0s")
	rawCmd.Flags().DurationP(
		keyRawTimeout, "t", d, "Timeout for executing raw command. After time elapses, the command will be terminated",
	)
	viperBindPFlag(rawCmd, keyRawTimeout)

	rawCmd.Flags().BoolP(
		keyRawRawOutput, "r", true, "Display raw output, outside of the logger",
	)
	viperBindPFlag(rawCmd, keyRawRawOutput)
}

// RunRawCommand unmarshal commands and executes with provided arguments
func RunRawCommand(cmd *cobra.Command, args []string) {
	var commands []kubestrap.RawCommand
	err := viper.UnmarshalKey(prefixKey(cmd, keyRawUtilities), &commands, func(config *mapstructure.DecoderConfig) {
		config.TagName = "yaml"
		config.ErrorUnused = true
		//config.ErrorUnset = true
	})
	logging.ExitOnError(err, 1)
	if len(args) == 0 {
		cmd.Help()
		fmt.Printf("\nAvailable utilities:\n")
		for _, c := range commands {
			fmt.Printf("  - %s %s\n", c.Name, c.Release)
		}
		return
	}
	for _, c := range commands {
		if c.Name == args[0] {
			timeout := viper.GetViper().GetDuration(prefixKey(cmd, keyRawTimeout))
			logging.Logger.Debugf("execution timeout: %s", timeout)
			remainingArgs := args[1:]
			if len(remainingArgs) > 0 {
				c.Arguments = remainingArgs
			}
			rawOutput := viper.GetViper().GetBool(prefixKey(cmd, keyRawRawOutput))
			if retCode, errExecute := c.ExecuteCommand(timeout, rawOutput, false); retCode != 0 {
				logging.ExitOnError(errExecute, retCode)
			}
			// Config allows for duplicates, but here we stop at the first match
			return
		}
	}
	// If we get here, the command is not in the config, do not allow that
	logging.Logger.Errorf("command '%s' is not supported, perhaps add it to the config?\n", args[0])
}
