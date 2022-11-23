/*
Copyright © 2022 Dataflows
*/
package cmd

import (
	"fmt"

	"dataflows.com/kubestrap/internal/pkg/kubestrap"
	"dataflows.com/kubestrap/internal/pkg/reflectutil"
	"github.com/spf13/cobra"
)

var (
	typeFlux          = &kubestrap.Flux{}
	keyFluxContext    = reflectutil.GetStructFieldTag(typeFlux, "Context")
	keyFluxNamespace  = reflectutil.GetStructFieldTag(typeFlux, "Namespace")
	requiredFluxFlags = []string{keyFluxContext}
	fluxNamespace     string
)

// fluxCmd represents the flux command
var fluxCmd = &cobra.Command{
	Use:   "flux",
	Short: "FluxCD wrapper",
	Long:  ``,
	// Run: func(cmd *cobra.Command, args []string) {
	// },
}

func init() {
	rootCmd.AddCommand(fluxCmd)

	fluxCmd.PersistentFlags().StringP(
		keyFluxContext, "c", "", fmt.Sprintf("[Required] Kubernetes context as defined in '%s'", kubernetesConfig),
	)
	viperBindPersistentPFlag(fluxCmd, keyFluxContext)

	fluxCmd.PersistentFlags().StringVarP(
		&fluxNamespace,
		keyFluxNamespace, "n", "flux-system", "Kubernetes namespace for FluxCD",
	)
	viperBindPersistentPFlag(fluxCmd, keyFluxNamespace)
}

// RunFluxCommand runs flux subcommands with appropriate context
func RunFluxCommand(cmd *cobra.Command, args []string) {
	checkRequiredFlags(cmd.Parent(), requiredFluxFlags)

	newArgs := []string{cmd.Parent().Use, cmd.Use}
	newArgs = appendStringArgsf(cmd.Parent(), newArgs, keyFluxContext, "--%s=%s")
	if len(args) > 0 {
		newArgs = append(newArgs, args...)
	} else {
		newArgs = appendStringSplitArgs(cmd, newArgs, "", "")
		newArgs = appendStringArgsf(cmd.Parent(), newArgs, keyFluxNamespace, "--%s=%s")
	}
	RunRawCommand(rawCmd, newArgs)
}
