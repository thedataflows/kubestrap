/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// fluxBootstrapCmd represents the fluxBootstrap command
var fluxBootstrapCmd = &cobra.Command{
	Use:     "bootstrap",
	Short:   "Bootstrap or upgrade FluxCD",
	Long:    ``,
	Run:     RunFluxCommand,
	Aliases: []string{"bs"},
}

func init() {
	fluxCmd.AddCommand(fluxBootstrapCmd)
}
