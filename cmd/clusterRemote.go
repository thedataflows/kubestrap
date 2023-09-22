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
	"github.com/thedataflows/go-commons/pkg/reflectutil"
	"github.com/thedataflows/kubestrap/pkg/kubestrap"
)

var (
	typeClusterRemote     = &kubestrap.Remote{}
	keyClusterRemoteHosts = reflectutil.GetStructFieldTag(typeClusterRemote, "Hosts", "")
	clusterRemoteHosts    []string
)

// clusterRemoteCmd represents the clusterRemote command
var clusterRemoteCmd = &cobra.Command{
	Use:     "remote",
	Short:   "Execute command clusterRemotely on the cluster",
	Long:    ``,
	RunE:    RunClusterRemoteCommand,
	Aliases: []string{"r"},
}

func init() {
	clusterCmd.AddCommand(clusterRemoteCmd)
	clusterRemoteCmd.SilenceErrors = clusterRemoteCmd.Parent().SilenceErrors

	clusterRemoteCmd.Flags().StringSlice(
		keyClusterRemoteHosts,
		clusterRemoteHosts,
		"List of hosts defined in the cluster to run the command on. If not specified, will execute on all hosts",
	)

	// Bind flags
	config.ViperBindPFlagSet(clusterRemoteCmd, nil)

	rigLog.Log = &log.Log
}

// RunClusterRemoteCommand runs a command on the cluster
func RunClusterRemoteCommand(cmd *cobra.Command, args []string) error {
	if err := config.CheckRequiredFlags(cmd.Parent(), requiredClusterFlags); err != nil {
		return err
	}

	clusterContext := config.ViperGetString(cmd.Parent(), keyClusterContext)
	clusterBootstrapPath := config.ViperGetString(cmd.Parent(), keyClusterBootstrapPath)
	clusterRemoteHosts = config.ViperGetStringSlice(cmd, keyClusterRemoteHosts)

	// Load cluster spec
	cl, err := kubestrap.NewK0sCluster(clusterContext, clusterBootstrapPath)
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
			log.Warnf("Failed to connect to %s: %v", hosts[i].Address(), err)
			continue
		}
		remoteCommand := strings.Join(args, " ")
		o, err := hosts[i].ExecOutput(remoteCommand)
		if err != nil {
			log.Warnf("Failed to execute '%s' on '%s': %v", remoteCommand, hosts[i].Address(), err)
			continue
		}
		log.Infof("Executed '%s' on '%s': %v", remoteCommand, hosts[i].Address(), o)
	}
	return nil
}
