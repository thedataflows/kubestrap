/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"
	"os"
	"time"

	rigLog "github.com/k0sproject/rig/log"
	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/defaults"
	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/kubestrap/pkg/kubernetes"
)

type Cluster struct {
	cmd    *cobra.Command
	parent *Root
}

var (
	mycluster = NewCluster(root)
)

func init() {

}

func NewCluster(parent *Root) *Cluster {
	c := &Cluster{
		parent: parent,
	}

	c.cmd = &cobra.Command{
		Use:           "cluster",
		Short:         "Manages a kubernetes cluster",
		Long:          ``,
		Aliases:       []string{"c"},
		RunE:          c.RunClusterCommand,
		SilenceErrors: parent.Cmd().SilenceErrors,
		SilenceUsage:  parent.Cmd().SilenceUsage,
	}

	parent.Cmd().AddCommand(c.cmd)

	c.cmd.PersistentFlags().StringP(
		c.KeyClusterContext(),
		"c",
		c.DefaultClusterContext(),
		fmt.Sprintf("[Required] Kubernetes context as defined in '%s'", kubernetes.GetKubeconfigPath()),
	)

	c.cmd.PersistentFlags().StringP(
		c.KeyClusterBootstrapPath(),
		"p",
		c.DefaultClusterBootstrapPath(),
		"Cluster definition path in the current repository",
	)

	defaultTimeout, _ := time.ParseDuration("10m0s")
	c.cmd.PersistentFlags().DurationP(
		c.KeyTimeout(),
		"t",
		defaultTimeout,
		"Timeout for executing cluster commands. After time elapses, the command will be terminated",
	)

	// Bind flags to config
	config.ViperBindPFlagSet(c.cmd, c.cmd.PersistentFlags())

	rigLog.Log = &log.Log

	return c
}

func (c *Cluster) RunClusterCommand(cmd *cobra.Command, args []string) error {
	if err := c.CheckRequiredFlags(); err != nil {
		return err
	}

	clusterBootstrapPath := c.ClusterBootstrapPath()
	clusterBootstrapOsTmpPath := clusterBootstrapPath + "/../os/tmp"

	if err := os.MkdirAll(clusterBootstrapOsTmpPath, 0700); err != nil {
		return err
	}

	currentDir := file.WorkingDirectory()
	if err := os.Chdir(clusterBootstrapPath); err != nil {
		return err
	}
	defer func() { _ = os.Chdir(currentDir) }()

	// Generate etc_hosts
	out, err := raw.RunRawCommandCaptureStdout(
		raw.Cmd(),
		[]string{
			"yq",
			"(.spec.hosts[]) | explode (.) | .privateAddress + \" \" + .hostname",
			clusterBootstrapPath + "/cluster.yaml",
		},
	)
	if err != nil {
		if len(out) == 0 {
			return err
		}
		return fmt.Errorf("%v\n%s", err, out)
	}
	if len(out) == 0 {
		return fmt.Errorf("empty output from yq")
	}
	if err = os.WriteFile(clusterBootstrapOsTmpPath+"/etc_hosts", []byte(out+"\n"), 0600); err != nil {
		return err
	}

	// Run k0sctl apply
	config.ViperSet(raw.Cmd(), c.KeyTimeout(), c.Timeout())
	if err := raw.RunRawCommand(
		raw.Cmd(),
		append(
			[]string{
				"k0sctl",
				"apply",
				"--config",
				clusterBootstrapPath + "/cluster.yaml",
				"--debug",
				"--force",
			},
			args...),
	); err != nil {
		return err
	}

	// TODO
	// Check if sops-age secret exists
	// Decrypt age private key
	// log.Infof("loading private key: %s", secretsEncryptDecrypt.PrivateKeyPath())
	// out, err = RunRawCommandCaptureStdout(
	// 	rawCmd,
	// 	[]string{
	// 		"age",
	// 		"--decrypt",
	// 		secretsEncryptDecrypt.PrivateKeyPath(),
	// 	},
	// )
	// if err != nil {
	// 	if len(out) == 0 {
	// 		return err
	// 	}
	// 	return fmt.Errorf("%v\n%s", err, out)
	// }
	// if len(out) == 0 {
	// 	return fmt.Errorf("private key is empty")
	// }
	// out = fmt.Sprintf(

	// Write sops-age secret
	// Annotate sops-age secret

	return nil
}

func (c *Cluster) Cmd() *cobra.Command {
	return c.cmd
}

func (c *Cluster) CheckRequiredFlags() error {
	return config.CheckRequiredFlags(c.cmd, []string{c.KeyClusterContext()})
}

// Flags keys, defaults and value getters
func (c *Cluster) KeyClusterContext() string {
	const ctx = "context"
	return ctx
}

func (c *Cluster) DefaultClusterContext() string {
	return defaults.Undefined
}

func (c *Cluster) ClusterContext() string {
	return config.ViperGetString(c.cmd, c.KeyClusterContext())
}

func (c *Cluster) KeyClusterBootstrapPath() string {
	return "cluster-path"
}

func (c *Cluster) DefaultClusterBootstrapPath() string {
	return fmt.Sprintf("bootstrap/cluster-%s", c.DefaultClusterContext())
}

func (c *Cluster) ClusterBootstrapPath() string {
	clusterBootstrapPath := config.ViperGetString(c.cmd, c.KeyClusterBootstrapPath())
	if clusterBootstrapPath == c.DefaultClusterBootstrapPath() {
		clusterBootstrapPath = fmt.Sprintf(
			"%s/bootstrap/cluster-%s",
			c.parent.ProjectRoot(),
			c.ClusterContext(),
		)
	}
	return clusterBootstrapPath
}

func (c *Cluster) KeyTimeout() string {
	const t = "timeout"
	return t
}

func (c *Cluster) Timeout() string {
	return config.ViperGetDuration(c.cmd, c.KeyTimeout()).String()
}
