/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"
	rigLog "github.com/k0sproject/rig/log"
	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/kubestrap/pkg/kubestrap"
)

type ClusterRemote struct {
	cmd    *cobra.Command
	parent *Cluster
}

var (
	_ = NewClusterRemote(mycluster)
)

func init() {

}

func NewClusterRemote(parent *Cluster) *ClusterRemote {
	cr := &ClusterRemote{
		parent: parent,
	}

	cr.cmd = &cobra.Command{
		Use:           "remote",
		Short:         "Execute command remotely on the cluster",
		Long:          ``,
		RunE:          cr.RunClusterRemoteCommand,
		Aliases:       []string{"r"},
		SilenceErrors: parent.Cmd().SilenceErrors,
		SilenceUsage:  parent.Cmd().SilenceUsage,
	}

	parent.Cmd().AddCommand(cr.cmd)

	cr.cmd.Flags().StringSlice(
		cr.KeyClusterRemoteHosts(),
		[]string{},
		"List of hosts defined in the cluster to run the command on. If not specified, will execute on all hosts",
	)

	// Bind flags to config
	config.ViperBindPFlagSet(cr.cmd, nil)

	rigLog.Log = &log.Log

	return cr
}

func (c *ClusterRemote) RunClusterRemoteCommand(cmd *cobra.Command, args []string) error {
	if err := c.CheckRequiredFlags(); err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("command to execute is not specified")
	}

	clusterRemoteHosts := c.ClusterRemoteHosts()

	// Load cluster spec
	cl, err := kubestrap.NewK0sCluster(c.parent.ClusterContext(), c.parent.ClusterBootstrapPath())
	if err != nil {
		return err
	}

	hosts := cl.GetClusterSpec().Spec.Hosts.Filter(
		func(h *cluster.Host) bool {
			for _, filterHost := range clusterRemoteHosts {
				if h.Address() == filterHost || h.Metadata.Hostname == filterHost || h.HostnameOverride == filterHost {
					return true
				}
			}
			return len(clusterRemoteHosts) == 0
		},
	)

	currentDir := file.WorkingDirectory()
	if err := os.Chdir(c.parent.ClusterBootstrapPath()); err != nil {
		return err
	}
	defer func() { _ = os.Chdir(currentDir) }()

	for i := 0; i < len(hosts); i += 1 {
		err := hosts[i].Connect()
		defer hosts[i].Disconnect()
		if err != nil {
			log.Errorf("[%s] Failed to connect: %v", hosts[i].Address(), err)
			continue
		}
		remoteCommand := strings.Join(args, " ")
		o, err := hosts[i].ExecOutput(remoteCommand)
		if err != nil {
			log.Errorf("[%s] Failed to execute '%s': %v", hosts[i].Address(), remoteCommand, err)
			continue
		}
		if len(o) == 0 {
			log.Infof("[%s] Executed '%s'", hosts[i].Address(), remoteCommand)
			continue
		}
		log.Infof("[%s] Executed '%s':\n%v", hosts[i].Address(), remoteCommand, o)
	}
	return nil
}

func (c *ClusterRemote) CheckRequiredFlags() error {
	return c.parent.CheckRequiredFlags()
}

func (c *ClusterRemote) KeyClusterRemoteHosts() string {
	return "hosts"
}

func (c *ClusterRemote) ClusterRemoteHosts() []string {
	return config.ViperGetStringSlice(c.cmd, c.KeyClusterRemoteHosts())
}
