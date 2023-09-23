/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/defaults"
	"github.com/thedataflows/go-commons/pkg/log"
)

type SecretsEncrypt struct {
	cmd    *cobra.Command
	parent *Secrets
}

var (
	// secretsEncryptCmd represents the secrets command
	secretsEncryptCmd = &cobra.Command{
		Use:     "encrypt",
		Short:   "Encrypt secrets.",
		Long:    ``,
		Aliases: []string{"e"},
		RunE:    RunSecretsEncryptCommand,
	}

	secretsEncrypt = NewSecretsEncrypt(secrets)
)

func init() {
	secretsCmd.AddCommand(secretsEncryptCmd)
	secretsEncryptCmd.SilenceErrors = secretsEncryptCmd.Parent().SilenceErrors

	secretsEncryptCmd.Flags().BoolP(
		secretsEncrypt.KeyInplace(),
		"i",
		secretsEncrypt.DefaultInplace(),
		"Write files in-place instead of outputting to stdout",
	)

	secretsEncryptCmd.Flags().StringP(
		secretsEncrypt.KeySopsConfig(),
		"s",
		secretsEncrypt.DefaultSopsConfig(),
		"SOPS configuration file",
	)

	secretsEncryptCmd.Flags().StringP(
		secretsEncrypt.KeyKubeClusterDir(),
		"k",
		secretsEncrypt.DefaultKubeClusterDir(),
		"Kubernetes cluster directory",
	)

	// Bind flags
	config.ViperBindPFlagSet(secretsEncryptCmd, nil)

	secretsEncrypt.SetCmd(secretsEncryptCmd)
}

func RunSecretsEncryptCommand(cmd *cobra.Command, args []string) error {
	if err := secretsEncrypt.CheckRequiredFlags(); err != nil {
		return err
	}

	if len(args) == 0 {
		args = []string{secretsEncrypt.DefaultSecretsEncryptFilePattern()}
	}

	for _, arg := range args {
		newArgs := []string{"sops", "--config", secretsEncrypt.GetSopsConfig(), "--encrypt", secretsEncrypt.GetInplace(), arg}
		log.Info("Generating source files encryption keys")
		if err := RunRawCommand(rawCmd, newArgs); err != nil {
			log.Warnf("error running: %s", err)
		}
	}

	return nil
}

func NewSecretsEncrypt(parent *Secrets) *SecretsEncrypt {
	return &SecretsEncrypt{
		parent: parent,
	}
}

func (s *SecretsEncrypt) SetCmd(cmd *cobra.Command) {
	s.cmd = cmd
}

func (s *SecretsEncrypt) CheckRequiredFlags() error {
	return s.parent.CheckRequiredFlags()
}

// Flags keys, defaults and value getters
func (s *SecretsEncrypt) DefaultSecretsEncryptFilePattern() string {
	return "secrets.*.yaml"
}
func (s *SecretsEncrypt) KeyInplace() string {
	return "in-place"
}

func (s *SecretsEncrypt) DefaultInplace() bool {
	return false
}

func (s *SecretsEncrypt) GetInplace() string {
	if config.ViperGetBool(s.cmd, s.KeyInplace()) {
		return "--in-place"
	}
	return ""
}

func (s *SecretsEncrypt) KeySopsConfig() string {
	return "sops-config"
}

func (s *SecretsEncrypt) DefaultSopsConfig() string {
	return s.DefaultKubeClusterDir() + "/.sops.yaml"
}

func (s *SecretsEncrypt) GetSopsConfig() string {
	secretsEncryptSopsConfig := config.ViperGetString(s.cmd, s.KeySopsConfig())
	if secretsEncryptSopsConfig == s.DefaultSopsConfig() {
		secretsEncryptSopsConfig = s.GetKubeClusterDir() + "/.sops.yaml"
	}
	return secretsEncryptSopsConfig
}

func (s *SecretsEncrypt) KeyKubeClusterDir() string {
	return "kube-cluster-dir"
}

func (s *SecretsEncrypt) DefaultKubeClusterDir() string {
	return fmt.Sprintf("kubernetes/cluster-%s", defaults.Undefined)
}

func (s *SecretsEncrypt) GetKubeClusterDir() string {
	secretsEncryptKubeClusterDir := config.ViperGetString(s.cmd, s.KeyKubeClusterDir())
	if secretsEncryptKubeClusterDir == s.DefaultKubeClusterDir() {
		secretsEncryptKubeClusterDir = fmt.Sprintf(
			"kubernetes/cluster-%s",
			s.parent.GetSecretsContext(),
		)
	}
	return secretsEncryptKubeClusterDir
}
