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
)

type ClusterKubeconfig struct {
	cmd    *cobra.Command
	parent *Cluster
}

// clusterKubeconfigCmd represents the clusterKubeconfig command
var (
	clusterKubeconfigCmd = &cobra.Command{
		Use:     "kubeconfig",
		Short:   "Fetch cluster kubeconfig",
		Long:    ``,
		RunE:    RunClusterKubeconfigCommand,
		Aliases: []string{"k"},
	}

	clusterKubeconfig = NewClusterKubeconfig(mycluster)
)

func init() {
	clusterCmd.AddCommand(clusterKubeconfigCmd)
	clusterKubeconfigCmd.SilenceErrors = clusterKubeconfigCmd.Parent().SilenceErrors

	// Bind flags
	config.ViperBindPFlagSet(clusterKubeconfigCmd, nil)

	clusterKubeconfig.SetCmd(clusterKubeconfigCmd)
}

// RunClusterKubeconfigCommand runs a command on the cluster
func RunClusterKubeconfigCommand(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	if err := clusterKubeconfig.CheckRequiredFlags(); err != nil {
		return err
	}

	clusterBootstrapPath := clusterBootstrap.parent.GetClusterBootstrapPath()

	currentDir := file.WorkingDirectory()
	if err := os.Chdir(clusterBootstrapPath); err != nil {
		return err
	}
	defer func() { _ = os.Chdir(currentDir) }()

	config.ViperSet(rawCmd, clusterBootstrap.parent.KeyTimeout(), clusterBootstrap.parent.GetTimeout().String())
	out, err := RunRawCommandCaptureStdout(
		rawCmd,
		[]string{
			"k0sctl",
			"kubeconfig",
			"--config",
			clusterBootstrapPath + "/cluster.yaml",
		},
	)
	if err != nil {
		if len(out) == 0 {
			return fmt.Errorf("%v", err)
		}
		return fmt.Errorf("%v\n%s", err, out)
	}
	if len(out) == 0 {
		return fmt.Errorf("empty output from k0sctl init")
	}

	fmt.Println(out)

	return nil
}

func NewClusterKubeconfig(parent *Cluster) *ClusterKubeconfig {
	return &ClusterKubeconfig{
		parent: parent,
	}
}

func (c *ClusterKubeconfig) SetCmd(cmd *cobra.Command) {
	c.cmd = cmd
}

func (c *ClusterKubeconfig) CheckRequiredFlags() error {
	return c.parent.CheckRequiredFlags()
}

// Flags keys, defaults and value getters
