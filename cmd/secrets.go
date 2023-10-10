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
	secrets = NewSecrets(root)
)

func init() {

}

func NewSecrets(parent *Root) *Secrets {
	s := &Secrets{
		parent: parent,
	}

	s.cmd = &cobra.Command{
		Use:           "secrets",
		Short:         "Manages local encrypted secrets. Generates age and ssh keys.",
		Long:          ``,
		Aliases:       []string{"s"},
		SilenceErrors: parent.Cmd().SilenceErrors,
		SilenceUsage:  parent.Cmd().SilenceUsage,
	}

	parent.Cmd().AddCommand(s.cmd)

	s.cmd.PersistentFlags().StringP(
		s.KeySecretsContext(),
		"c",
		defaults.Undefined,
		fmt.Sprintf("[Required] Kubernetes context as defined in '%s'", kubernetes.GetKubeconfigPath()),
	)

	s.cmd.PersistentFlags().StringP(
		s.KeySecretsDir(),
		"d",
		"secrets",
		"Encrypted secrets directory",
	)

	s.cmd.PersistentFlags().StringP(
		s.KeyClusterBootstrapPath(),
		"p",
		s.DefaultClusterBootstrapPath(),
		"Cluster definition path in the current repo",
	)

	s.cmd.PersistentFlags().StringP(
		s.KeySopsConfig(),
		"s",
		s.DefaultSopsConfig(),
		"SOPS configuration file",
	)

	s.cmd.PersistentFlags().String(
		s.KeyKubeClusterDir(),
		s.DefaultKubeClusterDir(),
		"Kubernetes cluster directory",
	)

	// Bind flags to config
	config.ViperBindPFlagSet(s.cmd, s.cmd.PersistentFlags())

	return s
}

func (s *Secrets) Cmd() *cobra.Command {
	return s.cmd
}

func (s *Secrets) CheckRequiredFlags() error {
	return config.CheckRequiredFlags(s.cmd, []string{s.KeySecretsContext()})
}

// KeySecretsContext returns key for SecretsContext
func (s *Secrets) KeySecretsContext() string {
	return "context"
}

// SecretsContext returns SecretsContext
func (s *Secrets) SecretsContext() string {
	return config.ViperGetString(s.cmd, s.KeySecretsContext())
}

// KeySecretsDir returns key for SecretsDir
func (s *Secrets) KeySecretsDir() string {
	return "directory"
}

// SecretsDir returns SecretsDir
func (s *Secrets) SecretsDir() string {
	return fmt.Sprintf(
		"%s/%s",
		s.ProjectRoot(),
		config.ViperGetString(s.cmd, s.KeySecretsDir()),
	)
}

func (s *Secrets) KeyClusterBootstrapPath() string {
	return "cluster-path"
}

func (s *Secrets) DefaultClusterBootstrapPath() string {
	return fmt.Sprintf("bootstrap/cluster-%s", defaults.Undefined)
}

func (s *Secrets) ClusterBootstrapPath() string {
	clusterPath := config.ViperGetString(s.cmd, s.KeyClusterBootstrapPath())
	if clusterPath == s.DefaultClusterBootstrapPath() {
		clusterPath = fmt.Sprintf(
			"%s/bootstrap/cluster-%s",
			s.ProjectRoot(),
			s.SecretsContext(),
		)
	}
	return clusterPath
}

func (s *Secrets) ProjectRoot() string {
	return s.parent.ProjectRoot()
}

func (s *Secrets) KeySopsConfig() string {
	return "sops-config"
}

func (s *Secrets) DefaultSopsConfig() string {
	return s.DefaultKubeClusterDir() + "/.sops.yaml"
}

func (s *Secrets) SopsConfig() string {
	secretsEncryptSopsConfig := config.ViperGetString(s.cmd, s.KeySopsConfig())
	if secretsEncryptSopsConfig == s.DefaultSopsConfig() {
		secretsEncryptSopsConfig = s.KubeClusterDir() + "/.sops.yaml"
	}
	return secretsEncryptSopsConfig
}

func (s *Secrets) KeyKubeClusterDir() string {
	return "kube-cluster-dir"
}

func (s *Secrets) DefaultKubeClusterDir() string {
	return fmt.Sprintf("kubernetes/cluster-%s", defaults.Undefined)
}

func (s *Secrets) KubeClusterDir() string {
	secretsEncryptKubeClusterDir := config.ViperGetString(s.cmd, s.KeyKubeClusterDir())
	if secretsEncryptKubeClusterDir == s.DefaultKubeClusterDir() {
		secretsEncryptKubeClusterDir = fmt.Sprintf(
			"%s/kubernetes/cluster-%s",
			s.ProjectRoot(),
			s.SecretsContext(),
		)
	}
	return secretsEncryptKubeClusterDir
}
