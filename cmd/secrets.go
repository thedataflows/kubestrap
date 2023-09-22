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
	typeSecrets       = &kubestrap.Secrets{}
	keySecretsContext = reflectutil.GetStructFieldTag(typeSecrets, "Context", "")

	keySecretsDir = reflectutil.GetStructFieldTag(typeSecrets, "Directory", "")

	requiredSecretsFlags = []string{keySecretsContext}
)

// secretsCmd represents the secrets command
var secretsCmd = &cobra.Command{
	Use:     "secrets",
	Short:   "Manages local encrypted secrets. Generates age and ssh keys.",
	Long:    ``,
	Aliases: []string{"s"},
}

func init() {
	rootCmd.AddCommand(secretsCmd)
	secretsCmd.SilenceErrors = secretsCmd.Parent().SilenceErrors

	secretsCmd.PersistentFlags().StringP(
		keySecretsContext,
		"c",
		defaults.Undefined,
		fmt.Sprintf("[Required] Kubernetes context as defined in '%s'", kubernetes.GetKubeconfigPath()),
	)

	secretsCmd.PersistentFlags().StringP(
		keySecretsDir,
		"d",
		"secrets",
		"Encrypted secrets directory",
	)

	// Bind flags
	config.ViperBindPFlagSet(secretsCmd, secretsCmd.PersistentFlags())
}
