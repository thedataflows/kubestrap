/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// fluxReconcileCmd represents the fluxBootstrap command
var fluxReconcileCmd = &cobra.Command{
	Use:     "reconcile",
	Short:   "Reconcile FluxCD",
	Long:    ``,
	Run:     RunFluxCommand,
	Aliases: []string{"r"},
}

func init() {
	fluxCmd.AddCommand(fluxReconcileCmd)
}
