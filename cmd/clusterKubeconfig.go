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

var (
	_ = NewClusterKubeconfig(mycluster)
)

func init() {

}

func NewClusterKubeconfig(parent *Cluster) *ClusterKubeconfig {
	ck := &ClusterKubeconfig{
		parent: parent,
	}

	ck.cmd = &cobra.Command{
		Use:           "kubeconfig",
		Short:         "Fetch cluster kubeconfig",
		Long:          ``,
		RunE:          ck.RunClusterKubeconfigCommand,
		Aliases:       []string{"kc"},
		SilenceErrors: parent.Cmd().SilenceErrors,
		SilenceUsage:  parent.Cmd().SilenceUsage,
	}

	parent.Cmd().AddCommand(ck.cmd)

	// Bind flags to config
	config.ViperBindPFlagSet(ck.cmd, nil)

	return ck
}

func (c *ClusterKubeconfig) RunClusterKubeconfigCommand(cmd *cobra.Command, args []string) error {
	if err := c.CheckRequiredFlags(); err != nil {
		return err
	}

	clusterBootstrapPath := c.parent.ClusterBootstrapPath()

	// Set working directory to cluster bootstrap path
	currentDir := file.WorkingDirectory()
	if err := os.Chdir(clusterBootstrapPath); err != nil {
		return err
	}
	defer func() { _ = os.Chdir(currentDir) }()

	config.ViperSet(raw.Cmd(), c.parent.KeyTimeout(), c.parent.Timeout())
	out, err := raw.RunRawCommandCaptureStdout(
		raw.Cmd(),
		append(
			[]string{
				"k0sctl",
				"kubeconfig",
				"--config",
				clusterBootstrapPath + "/cluster.yaml",
			},
			args...),
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

	fmt.Println(out)

	return nil
}

func (c *ClusterKubeconfig) CheckRequiredFlags() error {
	return c.parent.CheckRequiredFlags()
}
