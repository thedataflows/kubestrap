/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/reflectutil"
	"github.com/thedataflows/kubestrap/pkg/constants"
	"github.com/thedataflows/kubestrap/pkg/kubernetes"
	"github.com/thedataflows/kubestrap/pkg/kubestrap"
)

var (
	typeFlux          = &kubestrap.Flux{}
	keyFluxContext    = reflectutil.GetStructFieldTag(typeFlux, "Context", "")
	keyFluxNamespace  = reflectutil.GetStructFieldTag(typeFlux, "Namespace", "")
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
	Aliases: []string{"f"},
}

func init() {
	rootCmd.AddCommand(fluxCmd)

	fluxCmd.PersistentFlags().StringP(
		keyFluxContext,
		"c",
		os.Getenv(constants.ViperEnvPrefix+"_FLUX_"+keyFluxContext),
		fmt.Sprintf("[Required] Kubernetes context as defined in '%s'", kubernetes.GetKubeconfigPath()),
	)
	fluxCmd.PersistentFlags().StringVarP(&fluxNamespace, keyFluxNamespace, "n", "flux-system", "Kubernetes namespace for FluxCD")

	config.ViperBindPFlagSet(fluxCmd, fluxCmd.PersistentFlags())
}

// RunFluxCommand runs flux subcommands with appropriate context
func RunFluxCommand(cmd *cobra.Command, args []string) {
	config.CheckRequiredFlags(cmd.Parent(), requiredFluxFlags, 2)

	newArgs := []string{cmd.Parent().Use, cmd.Use}
	newArgs = config.AppendStringArgsf(cmd.Parent(), newArgs, keyFluxContext, "--%s=%s")
	if len(args) > 0 {
		newArgs = append(newArgs, args...)
	} else {
		newArgs = append(newArgs, fmt.Sprintf("--%s=%s", keyFluxNamespace, fluxNamespace))
		newArgs = config.AppendStringSplitArgs(cmd, newArgs, "", "")
	}
	RunRawCommand(rawCmd, newArgs)
}
