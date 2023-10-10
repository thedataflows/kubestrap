/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/go-commons/pkg/process"
	"github.com/thedataflows/kubestrap/pkg/constants"

	"github.com/spf13/cobra"
)

type Root struct {
	cmd *cobra.Command
}

var (
	root = NewRoot()

	stdInBytes []byte
)

func init() {
	stat, err := os.Stdin.Stat()
	mode := stat.Mode() & os.ModeNamedPipe
	if err == nil && mode == os.ModeNamedPipe {
		stdInBytes, _ = io.ReadAll(os.Stdin)
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// errors.MaxStackDepth = 20
	if err := root.Cmd().Execute(); err != nil {
		log.Fatal(log.ErrWithTrace(err))
	}
}

func NewRoot() *Root {
	configOpts, err := config.NewOptions(
		config.WithEnvPrefix(constants.EnvPrefix),
		config.WithConfigName(constants.DefaultConfigName),
		config.WithUserConfigPaths(
			[]string{
				process.CurrentProcessDirectory(),
				file.WorkingDirectory(),
			},
		),
	)
	if err != nil {
		panic(err)
	}

	r := &Root{}

	r.cmd = &cobra.Command{
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
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	configOpts.Flags.String(
		r.KeyProjectRoot(),
		r.DefaultProjectRoot(),
		"Project root directory",
	)

	r.cmd.PersistentFlags().AddFlagSet(configOpts.Flags)
	config.ViperBindPFlagSet(r.cmd, configOpts.Flags)
	_ = r.cmd.ParseFlags(os.Args[1:])

	if err := configOpts.InitConfig(); err != nil {
		panic(err)
	}

	return r
}

func (r *Root) Cmd() *cobra.Command {
	return r.cmd
}

// Flags keys, defaults and value getters
func (r *Root) KeyProjectRoot() string {
	return "project-root"
}

func (r *Root) DefaultProjectRoot() string {
	return strings.ReplaceAll(file.WorkingDirectory(), "\\", "/")
}

func (r *Root) ProjectRoot() string {
	return config.ViperGetString(r.cmd, r.KeyProjectRoot())
}
