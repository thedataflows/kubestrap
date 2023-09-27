/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"strings"

	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"
	rigLog "github.com/k0sproject/rig/log"
	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/kubestrap/pkg/kubestrap"
)

type ClusterRemote struct {
	cmd    *cobra.Command
	parent *Cluster
}

// clusterRemoteCmd represents the clusterRemote command
var (
	clusterRemoteCmd = &cobra.Command{
		Use:     "remote",
		Short:   "Execute command clusterRemotely on the cluster",
		Long:    ``,
		RunE:    RunClusterRemoteCommand,
		Aliases: []string{"r"},
	}

	clusterRemote = NewClusterRemote(mycluster)
)

func init() {
	clusterCmd.AddCommand(clusterRemoteCmd)
	clusterRemoteCmd.SilenceErrors = clusterRemoteCmd.Parent().SilenceErrors

	clusterRemoteCmd.Flags().StringSlice(
		clusterRemote.KeyClusterRemoteHosts(),
		clusterRemote.DefaultClusterRemoteHosts(),
		"List of hosts defined in the cluster to run the command on. If not specified, will execute on all hosts",
	)

	// Bind flags
	config.ViperBindPFlagSet(clusterRemoteCmd, nil)

	clusterRemote.SetCmd(clusterRemoteCmd)

	rigLog.Log = &log.Log
}

// RunClusterRemoteCommand runs a command on the cluster
func RunClusterRemoteCommand(cmd *cobra.Command, args []string) error {
	if err := clusterRemote.CheckRequiredFlags(); err != nil {
		return err
	}

	clusterRemoteHosts := clusterRemote.GetClusterRemoteHosts()

	// Load cluster spec
	cl, err := kubestrap.NewK0sCluster(clusterRemote.parent.GetClusterContext(), clusterRemote.parent.GetClusterBootstrapPath())
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
		log.Infof("[%s] Executed '%s':\n%v", hosts[i].Address(), remoteCommand, o)
	}
	return nil
}

func NewClusterRemote(parent *Cluster) *ClusterRemote {
	return &ClusterRemote{
		parent: parent,
	}
}

func (c *ClusterRemote) SetCmd(cmd *cobra.Command) {
	c.cmd = cmd
}

func (c *ClusterRemote) CheckRequiredFlags() error {
	return c.parent.CheckRequiredFlags()
}

// Flags keys, defaults and value getters
func (c *ClusterRemote) KeyClusterRemoteHosts() string {
	return "hosts"
}

func (c *ClusterRemote) DefaultClusterRemoteHosts() []string {
	return []string{}
}

func (c *ClusterRemote) GetClusterRemoteHosts() []string {
	return config.ViperGetStringSlice(c.cmd, c.KeyClusterRemoteHosts())
}
