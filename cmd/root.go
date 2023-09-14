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

	configOpts, configOptsErr = config.NewOptions(
		config.WithEnvPrefix(constants.ViperEnvPrefix),
		config.WithConfigName(constants.DefaultConfigName),
		config.WithUserConfigPaths(
			[]string{
				filepath.Join(process.CurrentProcessDirectory()),
				filepath.Join(file.WorkingDirectory()),
			},
		),
	)
)

func init() {
	if configOptsErr != nil {
		panic(configOptsErr)
	}

	rootCmd.SilenceErrors = true

	configOpts.Flags.StringVar(
		&projectRootDir,
		keyRootProjectRoot,
		file.WorkingDirectory(),
		"Project root directory",
	)

	rootCmd.PersistentFlags().AddFlagSet(configOpts.Flags)
	config.ViperBindPFlagSet(rootCmd, configOpts.Flags)
	_ = rootCmd.ParseFlags(os.Args[1:])

	if err := configOpts.InitConfig(); err != nil {
		panic(err)
	}

	// Init subcommands
	initClusterCmd()
	initFluxCmd()
	initRawCmd()
	initSecretsCmd()
	initVersion()
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// errors.MaxStackDepth = 20
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(log.ErrWithTrace(err))
	}
}
