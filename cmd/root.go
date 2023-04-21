/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/kubestrap/pkg/constants"

	"github.com/spf13/cobra"
)

var (
	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "kubestrap",
		Short: "Toolbox for easy bootstrap of self hosted kubernetes",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Long = fmt.Sprintf(
				"%s\n\nAll flags values can be provided via env vars starting with %s_*\nTo pass a command (e.g. 'command1') flag, use %s_COMMAND1_FLAGNAME=somevalue",
				cmd.Short,
				configOpts.EnvPrefix,
				configOpts.EnvPrefix,
			)
			_ = cmd.Help()
		},
	}

	configOpts = config.DefaultConfigOpts(
		&config.Opts{
			EnvPrefix: constants.ViperEnvPrefix,
		},
	)
)

func initConfig() {
	config.InitConfig(configOpts)
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().AddFlagSet(configOpts.Flags)
	config.ViperBindPFlagSet(rootCmd, configOpts.Flags)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
