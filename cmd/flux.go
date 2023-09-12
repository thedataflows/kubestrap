/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/defaults"
	"github.com/thedataflows/go-commons/pkg/reflectutil"
	"github.com/thedataflows/kubestrap/pkg/kubernetes"
	"github.com/thedataflows/kubestrap/pkg/kubestrap"
)

var (
	typeFlux          = &kubestrap.Flux{}
	keyFluxContext    = reflectutil.GetStructFieldTag(typeFlux, "Context", "")
	keyFluxNamespace  = reflectutil.GetStructFieldTag(typeFlux, "Namespace", "")
	requiredFluxFlags = []string{keyFluxContext}
	fluxNamespace     string
	fluxContext       string
)

// fluxCmd represents the flux command
var fluxCmd = &cobra.Command{
	Use:   "flux",
	Short: "FluxCD wrapper",
	Long:  ``,
	// Run: func(cmd *cobra.Command, args []string) {},
	Aliases: []string{"f"},
}

func init() {
	if err := configOpts.InitConfig(); err != nil {
		panic(err)
	}

	rootCmd.AddCommand(fluxCmd)

	fluxContext = config.ViperGetString(fluxCmd, keyFluxContext)
	fluxCmd.PersistentFlags().StringVarP(
		&fluxContext,
		keyFluxContext,
		"c",
		fluxContext,
		fmt.Sprintf("[Required] Kubernetes context as defined in '%s'", kubernetes.GetKubeconfigPath()),
	)
	if len(fluxContext) == 0 {
		fluxContext = defaults.Undefined
	}

	fluxNamespace = config.ViperGetString(fluxCmd, keyFluxNamespace)
	if len(fluxNamespace) == 0 {
		fluxNamespace = "flux-system"
	}
	fluxCmd.PersistentFlags().StringVarP(
		&fluxNamespace,
		keyFluxNamespace,
		"n",
		fluxNamespace,
		"Kubernetes namespace for FluxCD",
	)

	config.ViperBindPFlagSet(fluxCmd, fluxCmd.PersistentFlags())
}

// RunFluxCommand runs flux subcommands with appropriate context
func RunFluxCommand(cmd *cobra.Command, args []string) error {
	if err := config.CheckRequiredFlags(cmd.Parent(), requiredFluxFlags); err != nil {
		return err
	}

	newArgs := []string{cmd.Parent().Use, cmd.Use}
	newArgs = config.AppendStringArgsf("--%s=%s", cmd.Parent(), newArgs, keyFluxContext)
	if len(args) > 0 {
		newArgs = append(newArgs, args...)
	} else {
		newArgs = append(newArgs, fmt.Sprintf("--%s=%s", keyFluxNamespace, fluxNamespace))
		newArgs = config.AppendStringSplitArgs(cmd, newArgs, "", "")
	}
	return RunRawCommand(rawCmd, newArgs)
}
