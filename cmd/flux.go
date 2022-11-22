/*
Copyright © 2022 Dataflows
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"dataflows.com/kubestrap/internal/pkg/kubestrap"
	"dataflows.com/kubestrap/internal/pkg/reflectutil"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	typeFluxContext   = &kubestrap.FluxContext{}
	keyFluxContext    = reflectutil.GetStructFieldTag(typeFluxContext, "Context")
	requiredFluxFlags = []string{keyFluxContext}
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

	kubeConfig := os.Getenv("KUBECONFIG")
	if kubeConfig == "" {
		kubeConfig = filepath.Join(os.Getenv("HOME"), "/.kube/config")
	}
	fluxCmd.PersistentFlags().StringP(
		keyFluxContext, "c", "", fmt.Sprintf("[Required] Kubernetes context as defined in '%s'", kubeConfig),
	)
	viper.BindPFlag(PrefixKey(fluxCmd, keyFluxContext), fluxCmd.PersistentFlags().Lookup(keyFluxContext))
}

// RunFluxCommand runs flux subcommands with appropriate context
func RunFluxCommand(cmd *cobra.Command, args []string) {
	CheckRequiredFlags(cmd.Parent(), requiredFluxFlags)

	newArgs := []string{cmd.Parent().Use, cmd.Use}
	context := viper.GetViper().GetString(PrefixKey(cmd.Parent(), keyFluxContext))
	if context != "" {
		newArgs = append(newArgs, fmt.Sprintf("--%s=%s", keyFluxContext, context))
	}
	if len(args) > 0 {
		newArgs = append(newArgs, args...)
	} else {
		arguments := viper.GetViper().GetString(PrefixKey(cmd, ""))
		if arguments != "" {
			newArgs = append(newArgs, regexp.MustCompile(`\s+`).Split(arguments, -1)...)
		}
	}
	RunRawCommand(rawCmd, newArgs)
}
