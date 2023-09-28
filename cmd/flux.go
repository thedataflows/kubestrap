/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/defaults"
	"github.com/thedataflows/kubestrap/pkg/kubernetes"
)

type Flux struct {
	cmd    *cobra.Command
	parent *Root
}

// fluxCmd represents the flux command
var (
	fluxCmd = &cobra.Command{
		Use:     "flux",
		Short:   "FluxCD wrapper",
		Long:    ``,
		Aliases: []string{"f"},
		RunE:    RunFluxCommand,
	}

	flux = NewFlux(root)
)

func init() {
	rootCmd.AddCommand(fluxCmd)
	fluxCmd.SilenceErrors = fluxCmd.Parent().SilenceErrors
	fluxCmd.SilenceUsage = fluxCmd.Parent().SilenceUsage

	fluxCmd.PersistentFlags().StringP(
		flux.KeyFluxContext(),
		"c",
		flux.DefaultFluxContext(),
		fmt.Sprintf("[Required] Kubernetes context as defined in '%s'", kubernetes.GetKubeconfigPath()),
	)

	fluxCmd.PersistentFlags().StringP(
		flux.KeyFluxNamespace(),
		"n",
		flux.DefaultFluxNamespace(),
		"Kubernetes namespace for FluxCD",
	)

	// Bind flags
	config.ViperBindPFlagSet(fluxCmd, fluxCmd.PersistentFlags())

	flux.SetCmd(fluxCmd)
}

// RunFluxCommand runs flux subcommands with appropriate context
func RunFluxCommand(cmd *cobra.Command, args []string) error {
	if err := flux.CheckRequiredFlags(); err != nil {
		return err
	}

	newArgs := []string{cmd.Use}
	newArgs = config.AppendStringArgsf("--%s=%s", cmd, newArgs, flux.KeyFluxContext())
	if len(args) > 0 {
		newArgs = append(newArgs, args...)
	} else {
		newArgs = append(newArgs, fmt.Sprintf("--%s=%s", flux.KeyFluxNamespace(), flux.GetFluxNamespace()))
		newArgs = config.AppendStringSplitArgs(cmd, newArgs, "", "")
	}
	return RunRawCommand(rawCmd, newArgs)
}

func NewFlux(parent *Root) *Flux {
	return &Flux{
		parent: parent,
	}
}

func (f *Flux) SetCmd(cmd *cobra.Command) {
	f.cmd = cmd
}

func (f *Flux) CheckRequiredFlags() error {
	return config.CheckRequiredFlags(f.cmd, []string{f.KeyFluxContext()})
}

// Flags keys, defaults and value getters
func (f *Flux) KeyFluxContext() string {
	return "context"
}

func (f *Flux) DefaultFluxContext() string {
	return defaults.Undefined
}

func (f *Flux) GetFluxContext() string {
	return config.ViperGetString(f.cmd, f.KeyFluxContext())
}

func (f *Flux) KeyFluxNamespace() string {
	return "namespace"
}

func (f *Flux) DefaultFluxNamespace() string {
	return "flux-system"
}

func (f *Flux) GetFluxNamespace() string {
	return config.ViperGetString(f.cmd, f.KeyFluxNamespace())
}

func (f *Flux) GetProjectRoot() string {
	return f.parent.GetProjectRoot()
}
