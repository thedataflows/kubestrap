/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/defaults"
	"github.com/thedataflows/kubestrap/pkg/kubernetes"
)

type Secrets struct {
	cmd    *cobra.Command
	parent *Root
}

var (
	// secretsCmd represents the secrets command
	secretsCmd = &cobra.Command{
		Use:     "secrets",
		Short:   "Manages local encrypted secrets. Generates age and ssh keys.",
		Long:    ``,
		Aliases: []string{"s"},
	}

	secrets = NewSecrets(root)
)

func init() {
	rootCmd.AddCommand(secretsCmd)
	secretsCmd.SilenceErrors = secretsCmd.Parent().SilenceErrors

	secretsCmd.PersistentFlags().StringP(
		secrets.KeySecretsContext(),
		"c",
		secrets.DefaultSecretsContext(),
		fmt.Sprintf("[Required] Kubernetes context as defined in '%s'", kubernetes.GetKubeconfigPath()),
	)

	secretsCmd.PersistentFlags().StringP(
		secrets.KeySecretsDir(),
		"d",
		secrets.DefaultSecretsDir(),
		"Encrypted secrets directory",
	)

	secretsCmd.PersistentFlags().StringP(
		secrets.KeyClusterBootstrapPath(),
		"p",
		secrets.DefaultClusterBootstrapPath(),
		"Cluster definition path in the current repo",
	)

	// Bind flags
	config.ViperBindPFlagSet(secretsCmd, secretsCmd.PersistentFlags())

	secrets.SetCmd(secretsCmd)
}

func NewSecrets(parent *Root) *Secrets {
	return &Secrets{
		parent: parent,
	}
}

func (s *Secrets) SetCmd(cmd *cobra.Command) {
	s.cmd = cmd
}

func (s *Secrets) CheckRequiredFlags() error {
	return config.CheckRequiredFlags(s.cmd, []string{s.KeySecretsContext()})
}

// Flags keys, defaults and value getters
// DefaultSecretsContext returns default Kubernetes context
func (s *Secrets) DefaultSecretsContext() string {
	return defaults.Undefined
}

// KeySecretsContext returns key for SecretsContext
func (s *Secrets) KeySecretsContext() string {
	return "context"
}

// GetSecretsContext returns SecretsContext
func (s *Secrets) GetSecretsContext() string {
	return config.ViperGetString(s.cmd, s.KeySecretsContext())
}

// KeySecretsDir returns key for SecretsDir
func (s *Secrets) KeySecretsDir() string {
	return "directory"
}

// DefaultSecretsDir returns default SecretsDir
func (s *Secrets) DefaultSecretsDir() string {
	return "secrets"
}

// GetSecretsDir returns SecretsDir
func (s *Secrets) GetSecretsDir() string {
	return config.ViperGetString(s.cmd, s.KeySecretsDir())
}

func (s *Secrets) KeyClusterBootstrapPath() string {
	return "cluster-path"
}

func (s *Secrets) DefaultClusterBootstrapPath() string {
	return fmt.Sprintf("bootstrap/cluster-%s", defaults.Undefined)
}

func (s *Secrets) GetClusterBootstrapPath() string {
	clusterPath := config.ViperGetString(s.cmd, s.KeyClusterBootstrapPath())
	if clusterPath == s.DefaultClusterBootstrapPath() {
		clusterPath = fmt.Sprintf("bootstrap/cluster-%s", s.GetSecretsContext())
	}
	return clusterPath
}
