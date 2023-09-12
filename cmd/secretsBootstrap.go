/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/go-commons/pkg/search"
)

const (
	keySecretsBootstrapForce = "force"
)

var (
	secretsBootstrapForce bool
)

// secretsBootstrapCmd represents the secretsBootstrap command
var secretsBootstrapCmd = &cobra.Command{
	Use:     "bootstrap",
	Short:   "Bootstrap secrets",
	Long:    ``,
	Aliases: []string{"b"},
	RunE:    RunSecretsBootstrapCommand,
}

func init() {
	secretsCmd.AddCommand(secretsBootstrapCmd)

	secretsBootstrapForce = config.ViperGetBool(secretsBootstrapCmd, keySecretsBootstrapForce)
	secretsBootstrapCmd.Flags().BoolVar(
		&secretsBootstrapForce,
		keySecretsBootstrapForce,
		secretsBootstrapForce,
		"Force overwrites",
	)

	config.ViperBindPFlagSet(secretsBootstrapCmd, nil)
}

func RunSecretsBootstrapCommand(cmd *cobra.Command, args []string) error {
	config.CheckRequiredFlags(cmd.Parent(), requiredSecretsFlags, 2)

	err := os.MkdirAll(config.ViperGetString(cmd.Parent(), keySecretsDir), 0700)
	if err != nil {
		return err
	}

	privateKeyPath := config.ViperGetString(cmd.Parent(), keySecretsPrivateKeyPath)

	encrypt := false
	if !file.IsAccessible(privateKeyPath) {
		// Create the private key
		if err = RunRawCommand(
			rawCmd,
			[]string{
				"age-keygen",
				"--output",
				privateKeyPath,
			},
		); err != nil {
			return err
		}
		encrypt = true
	} else {
		finder := search.TextFinder{
			Text: []byte("AGE-SECRET-KEY"),
		}
		found := finder.Grep(privateKeyPath)
		if found == nil {
			log.Warnf("Error determining '%s' is encrypted", privateKeyPath)
		} else {
			encrypt = len(found.Results) > 0
		}
	}
	if encrypt {
		if file.IsAccessible(privateKeyPath+".enc") && !secretsBootstrapForce {
			return fmt.Errorf("'%s' exists. Use --force flag to override", privateKeyPath+".enc")
		}
		// Encrypt the private key in place
		if err = RunRawCommand(
			rawCmd,
			[]string{
				"age",
				"--encrypt",
				"--armor",
				"--passphrase",
				"--output",
				privateKeyPath + ".enc", privateKeyPath,
			},
		); err != nil {
			return err
		}
	}

	publicKeyPath := config.ViperGetString(cmd.Parent(), keySecretsPublicKeyPath)
	if !file.IsAccessible(publicKeyPath) {
		// try to create the private key
		if err = RunRawCommand(
			rawCmd,
			[]string{
				"age-keygen",
				"-y",
				"--output",
				publicKeyPath,
				privateKeyPath,
			},
		); err != nil {
			return err
		}
	}

	if err = os.Remove(privateKeyPath); err != nil {
		log.Warnf("Error removing '%s': %s", privateKeyPath, err)
	}

	return nil
}
