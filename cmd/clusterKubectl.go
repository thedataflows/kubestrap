/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
)

type ClusterKubectl struct {
	cmd    *cobra.Command
	parent *Cluster
}

var (
	_ = NewClusterKubectl(mycluster)
)

func init() {

}

func NewClusterKubectl(parent *Cluster) *ClusterKubectl {
	ck := &ClusterKubectl{
		parent: parent,
	}

	ck.cmd = &cobra.Command{
		Use:           "kubectl",
		Short:         "Execute kubectl with a specified context",
		Long:          ``,
		RunE:          ck.RunClusterKubectlCommand,
		Aliases:       []string{"k"},
		SilenceErrors: parent.Cmd().SilenceErrors,
		SilenceUsage:  parent.Cmd().SilenceUsage,
	}

	parent.Cmd().AddCommand(ck.cmd)

	// Bind flags to config
	config.ViperBindPFlagSet(ck.cmd, nil)

	return ck
}

func (c *ClusterKubectl) RunClusterKubectlCommand(cmd *cobra.Command, args []string) error {
	if err := c.CheckRequiredFlags(); err != nil {
		return err
	}

	config.ViperSet(raw.Cmd(), c.parent.KeyTimeout(), c.parent.Timeout())
	out, err := raw.RunRawCommandCaptureStdout(
		raw.Cmd(),
		append(
			[]string{
				"kubectl",
				"--context",
				c.parent.ClusterContext(),
			},
			args...),
	)
	if err != nil {
		if len(out) == 0 {
			return err
		}
		return fmt.Errorf("%v\n%s", err, out)
	}

	fmt.Println(out)

	return nil
}

func (c *ClusterKubectl) CheckRequiredFlags() error {
	return c.parent.CheckRequiredFlags()
}
