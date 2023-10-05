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

var (
	_ = NewClusterBootstrap(mycluster)
)

func init() {

}

func NewClusterBootstrap(parent *Cluster) *ClusterBootstrap {
	cb := &ClusterBootstrap{
		parent: parent,
	}

	cb.cmd = &cobra.Command{
		Use:           "bootstrap",
		Short:         "Bootstrap new cluster config and secrets",
		Long:          ``,
		RunE:          cb.RunClusterBootstrapCommand,
		Aliases:       []string{"b"},
		SilenceErrors: parent.Cmd().SilenceErrors,
		SilenceUsage:  parent.Cmd().SilenceUsage,
	}

	parent.Cmd().AddCommand(cb.cmd)

	// Bind flags to config
	config.ViperBindPFlagSet(cb.cmd, nil)

	rigLog.Log = &log.Log

	return cb
}

func (c *ClusterBootstrap) RunClusterBootstrapCommand(cmd *cobra.Command, args []string) error {
	if err := c.CheckRequiredFlags(); err != nil {
		return err
	}

	clusterBootstrapPath := c.parent.ClusterBootstrapPath()
	if err := os.MkdirAll(clusterBootstrapPath, 0700); err != nil {
		return err
	}

	clusterFile := clusterBootstrapPath + "/cluster.yaml"
	if !file.IsAccessible(clusterFile) {
		out, err := raw.RunRawCommandCaptureStdout(
			raw.Cmd(),
			[]string{
				"k0sctl",
				"init",
				"--k0s",
				"--cluster-name",
				c.parent.ClusterContext(),
				"--key-path",
				"cluster.sshkey.pub",
				"root@10.0.0.1:@22",
			},
		)
		if err != nil {
			if len(out) == 0 {
				return err
			}
			return fmt.Errorf("%v\n%s", err, out)
		}
		if len(out) == 0 {
			return fmt.Errorf("empty output from k0sctl init")
		}
		if err = os.WriteFile(clusterFile, []byte(out), 0600); err != nil {
			return err
		}
	} else {
		log.Warnf("cluster file already exists: %s", clusterFile)
	}

	sshKey := clusterBootstrapPath + "/cluster.sshkey"
	if !file.IsAccessible(sshKey) {
		config.ViperSet(secretsBootstrap.Cmd(), secrets.KeySecretsContext(), c.parent.ClusterContext())
		if err := secretsBootstrap.RunBootstrapSecretsCommand(secretsBootstrap.Cmd(), args); err != nil {
			return err
		}
	} else {
		log.Warnf("SSH key already exists: %s", sshKey)
	}

	return nil
}

func (c *ClusterBootstrap) CheckRequiredFlags() error {
	return c.parent.CheckRequiredFlags()
}
