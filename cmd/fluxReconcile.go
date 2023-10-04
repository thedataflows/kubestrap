/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thedataflows/go-commons/pkg/config"
)

type FluxReconcile struct {
	cmd    *cobra.Command
	parent *Flux
}

// fluxReconcileCmd represents the fluxReconcile command
var (
	fluxReconcileCmd = &cobra.Command{
		Use:     "reconcile",
		Short:   "Reconcile FluxCD",
		Long:    ``,
		Aliases: []string{"r"},
		RunE:    RunFluxReconcileCommand,
	}

	fluxReconcile = NewFluxReconcile(flux)
)

func init() {
	fluxCmd.AddCommand(fluxReconcileCmd)
	fluxReconcileCmd.SilenceErrors = fluxReconcileCmd.Parent().SilenceErrors
	fluxReconcileCmd.SilenceUsage = fluxReconcileCmd.Parent().SilenceUsage

	// Bind flags
	config.ViperBindPFlagSet(fluxReconcileCmd, nil)

	fluxReconcile.SetCmd(fluxReconcileCmd)
}

// RunFluxReconcileCommand runs flux bootstrap subcommand
func RunFluxReconcileCommand(cmd *cobra.Command, args []string) error {
	if err := fluxReconcile.CheckRequiredFlags(); err != nil {
		return err
	}

	// Run the main command
	newArgs := []string{
		cmd.Parent().Use,
		cmd.Use,
		fmt.Sprintf("--%s=%s", fluxReconcile.parent.KeyFluxContext(), fluxReconcile.parent.FluxContext()),
	}
	if len(args) > 0 {
		newArgs = append(newArgs, args...)
	} else {
		newArgs = append(newArgs, regexp.MustCompile(`\s+`).Split(viper.GetString(cmd.Parent().Use+"."+cmd.Use), -1)...)
	}
	if err := RunRawCommand(rawCmd, newArgs); err != nil {
		return err
	}
	return nil
}

func NewFluxReconcile(parent *Flux) *FluxReconcile {
	return &FluxReconcile{
		parent: parent,
	}
}

func (f *FluxReconcile) SetCmd(cmd *cobra.Command) {
	f.cmd = cmd
}

func (f *FluxReconcile) CheckRequiredFlags() error {
	return f.parent.CheckRequiredFlags()
}

// Flags keys, defaults and value getters
