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
	"github.com/thedataflows/go-commons/pkg/stringutil"
	"github.com/thedataflows/kubestrap/pkg/kubernetes"
	"github.com/thedataflows/kubestrap/pkg/kubestrap"
)

var (
	typeSecrets              = &kubestrap.Secrets{}
	keySecretsContext        = reflectutil.GetStructFieldTag(typeSecrets, "Context", "")
	keySecretsNamespace      = reflectutil.GetStructFieldTag(typeSecrets, "Namespace", "")
	keySecretsDir            = reflectutil.GetStructFieldTag(typeSecrets, "Directory", "")
	keySecretsPrivateKeyPath = reflectutil.GetStructFieldTag(typeSecrets, "PrivateKey", "")
	keySecretsPublicKeyPath  = reflectutil.GetStructFieldTag(typeSecrets, "PublicKey", "")
	requiredSecretsFlags     = []string{keySecretsContext}
	secretContext            string
	secretsNamespace         string
)

// secretsCmd represents the secrets command
var secretsCmd = &cobra.Command{
	Use:     "secrets",
	Short:   "Manages local encrypted secrets",
	Long:    ``,
	Aliases: []string{"s"},
	// Run: func(cmd *cobra.Command, args []string) {},
}

func initSecretsCmd() {
	rootCmd.AddCommand(secretsCmd)

	secretContext = config.ViperGetString(secretsCmd, keySecretsContext)
	secretsCmd.PersistentFlags().StringVarP(
		&secretContext,
		keySecretsContext,
		"c",
		secretContext,
		fmt.Sprintf("[Required] Kubernetes context as defined in '%s'", kubernetes.GetKubeconfigPath()),
	)
	if len(secretContext) == 0 {
		secretContext = defaults.Undefined
	}

	secretsNamespace = config.ViperGetString(secretsCmd, keySecretsNamespace)
	if len(secretsNamespace) == 0 {
		secretsNamespace = "flux-system"
	}
	secretsCmd.PersistentFlags().StringVarP(
		&secretsNamespace,
		keySecretsNamespace,
		"n",
		secretsNamespace,
		"Kubernetes namespace for FluxCD Secrets",
	)

	var secretsDir string
	secretsCmd.PersistentFlags().StringVarP(
		&secretsDir,
		keySecretsDir,
		"d",
		"secrets",
		"Encrypted secrets directory",
	)

	secretsCmd.PersistentFlags().String(
		keySecretsPrivateKeyPath,
		stringutil.ConcatStrings(secretsDir, "/", secretContext, ".private.age"),
		"Private key path",
	)
	secretsCmd.PersistentFlags().String(
		keySecretsPublicKeyPath,
		stringutil.ConcatStrings(secretsDir, "/", secretContext, ".public.age"),
		"Public key path",
	)

	config.ViperBindPFlagSet(secretsCmd, secretsCmd.PersistentFlags())

	// Init subcommands
	initSecretsBootstrapCmd()
}
