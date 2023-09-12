/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/go-commons/pkg/process"
	"github.com/thedataflows/kubestrap/pkg/constants"

	"github.com/spf13/cobra"
)

const keyRootProjectRoot = "project-root"

var (
	projectRootDir string
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

	configOpts = config.NewOptions(
		config.WithEnvPrefix(constants.ViperEnvPrefix),
		config.WithUserConfigPaths(
			[]string{
				filepath.Join(process.CurrentProcessDirectory() + constants.DefaultConfig),
				filepath.Join(file.WorkingDirectory() + constants.DefaultConfig),
			},
		),
	)
)

func init() {
	// cobra.OnInitialize(configOpts.InitConfig)
	// configOpts.InitConfig()

	configOpts.Flags.StringVar(
		&projectRootDir,
		keyRootProjectRoot,
		file.WorkingDirectory(),
		"Project root directory",
	)

	rootCmd.PersistentFlags().AddFlagSet(configOpts.Flags)

	config.ViperBindPFlagSet(rootCmd, configOpts.Flags)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_ = log.ErrWithTrace(err)
		os.Exit(1)
	}
}
