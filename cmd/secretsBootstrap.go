/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/go-commons/pkg/search"
)

// secretsBootstrapCmd represents the secretsBootstrap command
var secretsBootstrapCmd = &cobra.Command{
	Use:     "bootstrap",
	Short:   "Bootstrap secrets",
	Long:    ``,
	Aliases: []string{"bs"},
	Run: func(cmd *cobra.Command, args []string) {
		config.CheckRequiredFlags(cmd.Parent(), requiredSecretsFlags)

		err := os.MkdirAll(config.ViperGetString(cmd.Parent(), keySecretsDir), 0700)
		log.Fatal(err)

		context := config.ViperGetString(cmd.Parent(), keySecretsContext)

		privateKey := strings.ReplaceAll(
			config.ViperGetString(cmd.Parent(), keySecretsPrivateKey),
			defaultContextForUsage, context,
		)
		encrypt := false
		if !file.IsAccessible(privateKey) {
			// Create the private key
			RunRawCommand(rawCmd, []string{"age-keygen", "-o", privateKey})
			encrypt = true
		} else {
			finder := search.TextFinder{
				Text: []byte("AGE-SECRET-KEY"),
			}

			found := finder.Grep(privateKey)
			if found == nil {
				log.Warnf("Error determining '%s' is encrypted", privateKey)
			}
			encrypt = len(found.Results) > 0
		}
		if encrypt {
			// Encrypt the private key in place
			RunRawCommand(rawCmd, []string{"age", "-a", "-e", "-p", "-o", privateKey, privateKey})
		}

		publicKey := strings.ReplaceAll(
			config.ViperGetString(cmd.Parent(), keySecretsPublicKey),
			defaultContextForUsage, context,
		)
		if !file.IsAccessible(publicKey) {
			// try to create the private key
			RunRawCommand(rawCmd, []string{"age-keygen", "-o", publicKey, "-y", privateKey})
		}
	},
}

func init() {
	secretsCmd.AddCommand(secretsBootstrapCmd)
}
