/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"
	"os"

	rigLog "github.com/k0sproject/rig/log"
	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
)

type ClusterBootstrap struct {
	cmd    *cobra.Command
	parent *Cluster
}

// clusterBootstrapCmd represents the clusterBootstrap command
var (
	clusterBootstrapCmd = &cobra.Command{
		Use:     "bootstrap",
		Short:   "Bootstrap new cluster config and secrets",
		Long:    ``,
		RunE:    RunClusterBootstrapCommand,
		Aliases: []string{"b"},
	}

	clusterBootstrap = NewClusterBootstrap(mycluster)
)

func init() {
	clusterCmd.AddCommand(clusterBootstrapCmd)
	clusterBootstrapCmd.SilenceErrors = clusterBootstrapCmd.Parent().SilenceErrors

	// Bind flags
	config.ViperBindPFlagSet(clusterBootstrapCmd, nil)

	clusterBootstrap.SetCmd(clusterBootstrapCmd)

	rigLog.Log = &log.Log
}

// RunClusterBootstrapCommand runs a command on the cluster
func RunClusterBootstrapCommand(cmd *cobra.Command, args []string) error {
	if err := clusterBootstrap.CheckRequiredFlags(); err != nil {
		return err
	}

	clusterBootstrapPath := clusterBootstrap.parent.GetClusterBootstrapPath()
	err := os.MkdirAll(clusterBootstrapPath, 0700)
	if err != nil {
		return err
	}

	clusterFile := clusterBootstrapPath + "/cluster.yaml"
	if !file.IsAccessible(clusterFile) {
		out, err := RunRawCommandCaptureStdout(
			rawCmd,
			[]string{
				"k0sctl",
				"init",
				"--k0s",
				"--cluster-name",
				clusterBootstrap.parent.GetClusterContext(),
				"--key-path",
				"cluster.sshkey.enc.pub",
				"root@10.0.0.1:@22",
			},
		)
		if err != nil {
			return err
		}
		if len(out) == 0 {
			return fmt.Errorf("empty output from k0sctl init")
		}
		if err = os.WriteFile(clusterFile, []byte(out), 0600); err != nil {
			return err
		}
	} else {
		log.Warnf("Cluster file already exists: %s", clusterFile)
	}

	sshKey := clusterBootstrapPath + "/cluster.sshkey.enc"
	if !file.IsAccessible(sshKey) {
		config.ViperSet(secretsCmd, secrets.KeySecretsContext(), clusterBootstrap.parent.GetClusterContext())
		if err = RunBootstrapSecretsCommand(
			secretsBootstrapCmd,
			args,
		); err != nil {
			return err
		}
	} else {
		log.Warnf("SSH key already exists: %s", sshKey)
	}

	return nil
}

func NewClusterBootstrap(parent *Cluster) *ClusterBootstrap {
	return &ClusterBootstrap{
		parent: parent,
	}
}

func (c *ClusterBootstrap) SetCmd(cmd *cobra.Command) {
	c.cmd = cmd
}

func (c *ClusterBootstrap) CheckRequiredFlags() error {
	return c.parent.CheckRequiredFlags()
}

// Flags keys, defaults and value getters
