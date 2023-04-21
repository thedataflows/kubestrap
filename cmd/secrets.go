/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/reflectutil"
	"github.com/thedataflows/kubestrap/pkg/kubernetes"
	"github.com/thedataflows/kubestrap/pkg/kubestrap"
)

const defaultContextForUsage = "<context>"

var (
	typeSecrets          = &kubestrap.Secrets{}
	keySecretsContext    = reflectutil.GetStructFieldTag(typeSecrets, "Context", "")
	keySecretsNamespace  = reflectutil.GetStructFieldTag(typeSecrets, "Namespace", "")
	keySecretsDir        = reflectutil.GetStructFieldTag(typeSecrets, "Directory", "")
	keySecretsPrivateKey = reflectutil.GetStructFieldTag(typeSecrets, "PrivateKey", "")
	keySecretsPublicKey  = reflectutil.GetStructFieldTag(typeSecrets, "PublicKey", "")
	requiredSecretsFlags = []string{keySecretsContext}
)

// secretsCmd represents the secrets command
var secretsCmd = &cobra.Command{
	Use:     "secrets",
	Short:   "Manages local encrypted secrets",
	Long:    ``,
	Aliases: []string{"s"},
	// Run: func(cmd *cobra.Command, args []string) {
	// },
}

func init() {
	rootCmd.AddCommand(secretsCmd)

	secretsCmd.PersistentFlags().StringP(keySecretsContext, "c", "", fmt.Sprintf("[Required] Kubernetes context as defined in '%s'", kubernetes.GetKubeconfigPath()))
	secretsCmd.PersistentFlags().StringP(keySecretsNamespace, "n", "flux-system", "Kubernetes namespace for Secrets")
	var secretsDir string
	secretsCmd.PersistentFlags().StringVarP(&secretsDir, keySecretsDir, "d", "secrets", "Encrypted secrets directory")
	context := config.ViperGetString(secretsCmd, keySecretsContext)
	if context == "" {
		context = defaultContextForUsage
	}
	secretsCmd.PersistentFlags().String(keySecretsPrivateKey, kubestrap.ConcatStrings(secretsDir, "/", context, ".private.age"), "Private key")
	secretsCmd.PersistentFlags().String(keySecretsPublicKey, kubestrap.ConcatStrings(secretsDir, "/", context, ".public.age"), "Public key")

	config.ViperBindPFlagSet(secretsCmd, nil)
}
