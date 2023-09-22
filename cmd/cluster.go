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
	"github.com/thedataflows/go-commons/pkg/reflectutil"
	"github.com/thedataflows/kubestrap/pkg/kubernetes"
	"github.com/thedataflows/kubestrap/pkg/kubestrap"
)

var (
	typeCluster                 = &kubestrap.KubestrapCluster{}
	keyClusterContext           = reflectutil.GetStructFieldTag(typeCluster, "Context", "")
	keyClusterBootstrapPath     = reflectutil.GetStructFieldTag(typeCluster, "BootstrapPath", "")
	requiredClusterFlags        = []string{keyClusterContext}
	defaultClusterBootstrapPath = fmt.Sprintf("bootstrap/cluster-%s", defaults.Undefined)
)

// clusterCmd represents the cluster command
var clusterCmd = &cobra.Command{
	Use:     "cluster",
	Short:   "Manages a kubernetes cluster",
	Long:    ``,
	Aliases: []string{"c"},
	RunE:    RunClusterCommand,
}

func init() {
	rootCmd.AddCommand(clusterCmd)
	clusterCmd.SilenceErrors = clusterCmd.Parent().SilenceErrors

	clusterCmd.PersistentFlags().StringP(
		keyClusterContext,
		"c",
		defaults.Undefined,
		fmt.Sprintf("[Required] Kubernetes context as defined in '%s'", kubernetes.GetKubeconfigPath()),
	)

	clusterCmd.PersistentFlags().StringP(
		keyClusterBootstrapPath,
		"p",
		defaultClusterBootstrapPath,
		"Cluster definition path in the current repo",
	)

	// Bind flags
	config.ViperBindPFlagSet(clusterCmd, clusterCmd.PersistentFlags())

	rigLog.Log = &log.Log
}

func RunClusterCommand(cmd *cobra.Command, args []string) error {
	if err := config.CheckRequiredFlags(cmd, requiredClusterFlags); err != nil {
		return err
	}

	clusterContext := config.ViperGetString(cmd, keyClusterContext)
	clusterBootstrapPath := config.ViperGetString(cmd, keyClusterBootstrapPath)
	if clusterBootstrapPath == defaultClusterBootstrapPath {
		clusterBootstrapPath = fmt.Sprintf("bootstrap/cluster-%s", clusterContext)
	}
	log.Infof("clusterContext=%s; clusterBootstrapPath=%s", clusterContext, clusterBootstrapPath)



	return nil
}
