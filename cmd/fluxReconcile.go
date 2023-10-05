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

var (
	_ = NewFluxReconcile(flux)
)

func init() {
}

func NewFluxReconcile(parent *Flux) *FluxReconcile {
	fr := &FluxReconcile{
		parent: parent,
	}

	fr.cmd = &cobra.Command{
		Use:           "reconcile",
		Short:         "Reconcile FluxCD",
		Long:          ``,
		Aliases:       []string{"r"},
		RunE:          fr.RunFluxReconcileCommand,
		SilenceErrors: parent.Cmd().SilenceErrors,
		SilenceUsage:  parent.Cmd().SilenceUsage,
	}

	parent.Cmd().AddCommand(fr.cmd)

	// Bind flags to config
	config.ViperBindPFlagSet(fr.cmd, nil)

	return fr
}

func (f *FluxReconcile) RunFluxReconcileCommand(cmd *cobra.Command, args []string) error {
	if err := f.CheckRequiredFlags(); err != nil {
		return err
	}

	// Run the main command
	newArgs := []string{
		cmd.Parent().Use,
		cmd.Use,
		fmt.Sprintf("--%s=%s", f.parent.KeyFluxContext(), f.parent.FluxContext()),
	}
	if len(args) > 0 {
		newArgs = append(newArgs, args...)
	} else {
		newArgs = append(newArgs, regexp.MustCompile(`\s+`).Split(viper.GetString(cmd.Parent().Use+"."+cmd.Use), -1)...)
	}
	if err := raw.RunRawCommand(raw.Cmd(), newArgs); err != nil {
		return err
	}
	return nil
}

func (f *FluxReconcile) CheckRequiredFlags() error {
	return f.parent.CheckRequiredFlags()
}
