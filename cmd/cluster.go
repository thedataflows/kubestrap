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

// clusterCmd represents the cluster command
var (
	clusterCmd = &cobra.Command{
		Use:     "cluster",
		Short:   "Manages a kubernetes cluster",
		Long:    ``,
		Aliases: []string{"c"},
		RunE:    RunClusterCommand,
	}

	mycluster = NewCluster(root)
)

func init() {
	rootCmd.AddCommand(clusterCmd)
	clusterCmd.SilenceErrors = clusterCmd.Parent().SilenceErrors
	clusterCmd.SilenceUsage = clusterCmd.Parent().SilenceUsage

	clusterCmd.PersistentFlags().StringP(
		mycluster.KeyClusterContext(),
		"c",
		mycluster.DefaultClusterContext(),
		fmt.Sprintf("[Required] Kubernetes context as defined in '%s'", kubernetes.GetKubeconfigPath()),
	)

	clusterCmd.PersistentFlags().StringP(
		mycluster.KeyClusterBootstrapPath(),
		"p",
		mycluster.DefaultClusterBootstrapPath(),
		"Cluster definition path in the current repository",
	)

	clusterCmd.PersistentFlags().DurationP(
		mycluster.KeyTimeout(),
		"t",
		mycluster.DefaultTimeout(),
		"Timeout for executing cluster commands. After time elapses, the command will be terminated",
	)

	// Bind flags
	config.ViperBindPFlagSet(clusterCmd, clusterCmd.PersistentFlags())

	mycluster.SetCmd(clusterCmd)

	rigLog.Log = &log.Log
}

func RunClusterCommand(cmd *cobra.Command, args []string) error {
	if err := mycluster.CheckRequiredFlags(); err != nil {
		return err
	}

	clusterBootstrapPath := mycluster.GetClusterBootstrapPath()
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
	out, err := RunRawCommandCaptureStdout(
		rawCmd,
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
	config.ViperSet(rawCmd, mycluster.KeyTimeout(), mycluster.GetTimeout().String())
	if err := RunRawCommand(
		rawCmd,
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
	// log.Infof("loading private key: %s", secretsEncryptDecrypt.GetPrivateKeyPath())
	// out, err = RunRawCommandCaptureStdout(
	// 	rawCmd,
	// 	[]string{
	// 		"age",
	// 		"--decrypt",
	// 		secretsEncryptDecrypt.GetPrivateKeyPath(),
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

func NewCluster(parent *Root) *Cluster {
	return &Cluster{
		parent: parent,
	}
}

func (c *Cluster) SetCmd(cmd *cobra.Command) {
	c.cmd = cmd
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

func (c *Cluster) GetClusterContext() string {
	return config.ViperGetString(c.cmd, c.KeyClusterContext())
}

func (c *Cluster) KeyClusterBootstrapPath() string {
	return "cluster-path"
}

func (c *Cluster) DefaultClusterBootstrapPath() string {
	return fmt.Sprintf("bootstrap/cluster-%s", c.DefaultClusterContext())
}

func (c *Cluster) GetClusterBootstrapPath() string {
	clusterBootstrapPath := config.ViperGetString(c.cmd, c.KeyClusterBootstrapPath())
	if clusterBootstrapPath == c.DefaultClusterBootstrapPath() {
		clusterBootstrapPath = fmt.Sprintf(
			"%s/bootstrap/cluster-%s",
			c.parent.GetProjectRoot(),
			c.GetClusterContext(),
		)
	}
	return clusterBootstrapPath
}

func (c *Cluster) KeyTimeout() string {
	return "timeout"
}

func (c *Cluster) DefaultTimeout() time.Duration {
	d, _ := time.ParseDuration("10m0s")
	return d
}

func (c *Cluster) GetTimeout() time.Duration {
	return config.ViperGetDuration(c.cmd, c.KeyTimeout())
}
