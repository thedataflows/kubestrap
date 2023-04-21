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
	Run:     RunRawCommand,
	Aliases: []string{"r"},
}

func init() {
	rootCmd.AddCommand(rawCmd)

	d, _ := time.ParseDuration("1m0s")
	rawCmd.Flags().DurationP(
		keyRawTimeout, "t", d, "Timeout for executing raw command. After time elapses, the command will be terminated",
	)
	config.ViperBindPFlag(rawCmd, keyRawTimeout)

	rawCmd.Flags().BoolP(
		keyRawRawOutput, "r", true, "Display raw output, outside of the logger",
	)
	config.ViperBindPFlag(rawCmd, keyRawRawOutput)
}

// RunRawCommand unmarshal commands and executes with provided arguments
func RunRawCommand(cmd *cobra.Command, args []string) {
	var commands []kubestrap.RawCommand
	err := viper.UnmarshalKey(config.PrefixKey(cmd, keyRawUtilities), &commands, func(config *mapstructure.DecoderConfig) {
		config.TagName = "yaml"
		config.ErrorUnused = true
		//config.ErrorUnset = true
	})
	log.Fatal(err)
	if len(args) == 0 {
		cmd.Help()
		fmt.Printf("\nAvailable utilities:\n")
		for _, c := range commands {
			fmt.Printf("  - %s %s\n", c.Name, c.Release)
		}
		return
	}
	for _, c := range commands {
		if c.Name == args[0] || slices.Contains(c.Additional, args[0]) {
			timeout := config.ViperGetDuration(cmd, keyRawTimeout)
			log.Debugf("execution timeout: %s", timeout)
			c.Command = args
			if _, errExecute := c.ExecuteCommand(timeout, config.ViperGetBool(cmd, keyRawRawOutput), false); errExecute != nil {
				log.Fatal(errExecute)
			}
			// Config allows for duplicates, but here we stop at the first match
			return
		}
	}
	// If we get here, the command is not in the config, do not allow that
	log.Errorf("command '%s' is not supported, perhaps add it to the config?\n", args[0])
}
