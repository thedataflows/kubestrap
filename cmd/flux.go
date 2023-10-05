/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/defaults"
	"github.com/thedataflows/kubestrap/pkg/kubernetes"
)

type Flux struct {
	cmd    *cobra.Command
	parent *Root
}

var (
	flux = NewFlux(root)
)

func init() {

}

func NewFlux(parent *Root) *Flux {
	f := &Flux{
		parent: parent,
	}

	f.cmd = &cobra.Command{
		Use:           "flux",
		Short:         "FluxCD wrapper",
		Long:          ``,
		Aliases:       []string{"f"},
		RunE:          f.RunFluxCommand,
		SilenceErrors: parent.Cmd().SilenceErrors,
		SilenceUsage:  parent.Cmd().SilenceUsage,
	}

	parent.Cmd().AddCommand(f.cmd)

	f.cmd.PersistentFlags().StringP(
		f.KeyFluxContext(),
		"c",
		f.DefaultFluxContext(),
		fmt.Sprintf("[Required] Kubernetes context as defined in '%s'", kubernetes.GetKubeconfigPath()),
	)

	f.cmd.PersistentFlags().StringP(
		f.KeyFluxNamespace(),
		"n",
		f.DefaultFluxNamespace(),
		"Kubernetes namespace for FluxCD",
	)

	defaultTimeout, _ := time.ParseDuration("5m0s")
	f.cmd.PersistentFlags().DurationP(
		f.KeyTimeout(),
		"t",
		defaultTimeout,
		"Timeout for executing flux commands. After time elapses, the command will be terminated",
	)

	// Bind flags to config
	config.ViperBindPFlagSet(f.cmd, f.cmd.PersistentFlags())

	return f
}

func (f *Flux) RunFluxCommand(cmd *cobra.Command, args []string) error {
	if err := f.CheckRequiredFlags(); err != nil {
		return err
	}

	newArgs := []string{cmd.Use}
	newArgs = config.AppendStringArgsf("--%s=%s", cmd, newArgs, f.KeyFluxContext())
	if len(args) > 0 {
		newArgs = append(newArgs, args...)
	} else {
		newArgs = append(newArgs, fmt.Sprintf("--%s=%s", f.KeyFluxNamespace(), f.FluxNamespace()))
		newArgs = config.AppendStringSplitArgs(cmd, newArgs, "", "")
	}
	return raw.RunRawCommand(raw.Cmd(), newArgs)
}

func (f *Flux) Cmd() *cobra.Command {
	return f.cmd
}

func (f *Flux) CheckRequiredFlags() error {
	return config.CheckRequiredFlags(f.cmd, []string{f.KeyFluxContext()})
}

func (f *Flux) KeyFluxContext() string {
	return "context"
}

func (f *Flux) DefaultFluxContext() string {
	return defaults.Undefined
}

func (f *Flux) FluxContext() string {
	return config.ViperGetString(f.cmd, f.KeyFluxContext())
}

func (f *Flux) KeyFluxNamespace() string {
	return "namespace"
}

func (f *Flux) DefaultFluxNamespace() string {
	return "flux-system"
}

func (f *Flux) FluxNamespace() string {
	return config.ViperGetString(f.cmd, f.KeyFluxNamespace())
}

func (f *Flux) ProjectRoot() string {
	return f.parent.ProjectRoot()
}

func (f *Flux) KeyTimeout() string {
	return "timeout"
}

func (f *Flux) Timeout() string {
	return config.ViperGetDuration(f.cmd, f.KeyTimeout()).String()
}
