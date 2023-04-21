/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/log"

	"github.com/spf13/cobra"
)

const (
	keySampleCommandFlag1 = "flag1"
	keySampleCommandFlag2 = "flag2"
	keySampleCommandFlag3 = "flag3"
)

var (
	requiredSetFlags = []string{keySampleCommandFlag1, keySampleCommandFlag3}

	setSampleCmmand = &cobra.Command{
		Use:     "sample-command",
		Short:   "This is a sample command",
		Long:    ``,
		Aliases: []string{"s"},
		Run:     RunSampleCommand,
	}
)

func init() {
	rootCmd.AddCommand(setSampleCmmand)

	setSampleCmmand.Flags().Bool(keySampleCommandFlag1, false, "Boolean flag")
	setSampleCmmand.Flags().StringP(keySampleCommandFlag2, "s", "", "[Local Mandatory] StringP flag")
	setSampleCmmand.Flags().Duration(keySampleCommandFlag3, 10, "Duration flag")

	config.ViperBindPFlagSet(setSampleCmmand, nil)
}

// RunSampleCommand does some things when you run "sample-command"
func RunSampleCommand(cmd *cobra.Command, args []string) {
	// Validations
	config.CheckRequiredFlags(cmd, requiredSetFlags)

	flag2Value := config.ViperGetString(cmd, keySampleCommandFlag2)
	if len(flag2Value) == 0 {
		log.Fatalf("Please set --%v", keySampleCommandFlag2)
	}

	log.Infof(
		"Flags:\n--%s=%s\n--%s=%s\n--%s=%v",
		keySampleCommandFlag1, keySampleCommandFlag2, keySampleCommandFlag3,
		config.ViperGetString(cmd, keySampleCommandFlag1),
		flag2Value,
		config.ViperGetDuration(cmd, keySampleCommandFlag3),
	)

	// Some more logic
}
