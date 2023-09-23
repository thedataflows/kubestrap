/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"

	rigLog "github.com/k0sproject/rig/log"
	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/defaults"
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

	// Bind flags
	config.ViperBindPFlagSet(clusterCmd, clusterCmd.PersistentFlags())

	mycluster.SetCmd(clusterCmd)

	rigLog.Log = &log.Log
}

func RunClusterCommand(cmd *cobra.Command, args []string) error {
	if err := mycluster.CheckRequiredFlags(); err != nil {
		return err
	}

	clusterContext := mycluster.GetClusterContext()
	clusterBootstrapPath := mycluster.GetClusterBootstrapPath()
	log.Infof("clusterContext=%s; clusterBootstrapPath=%s", clusterContext, clusterBootstrapPath)

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
		clusterBootstrapPath = fmt.Sprintf("bootstrap/cluster-%s", c.GetClusterContext())
	}
	return clusterBootstrapPath
}