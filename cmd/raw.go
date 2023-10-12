/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/kubestrap/pkg/kubestrap"
	"golang.org/x/exp/slices"
)

type Raw struct {
	cmd    *cobra.Command
	parent *Root
	stdin  io.Reader
}

var (
	raw = NewRaw(root)
)

func init() {

}

func NewRaw(parent *Root) *Raw {
	r := &Raw{
		parent: parent,
	}

	r.cmd = &cobra.Command{
		Use:           "raw",
		Short:         "Directly run one of the predefined utilities. To pass flags for the raw command, use --",
		Long:          ``,
		Aliases:       []string{"r"},
		RunE:          r.RunRawCommand,
		SilenceErrors: parent.Cmd().SilenceErrors,
		SilenceUsage:  parent.Cmd().SilenceUsage,
	}

	parent.Cmd().AddCommand(r.cmd)

	defaultTimeout, _ := time.ParseDuration("1m0s")
	r.cmd.Flags().DurationP(
		r.KeyTimeout(),
		"t",
		defaultTimeout,
		"Timeout for executing raw command. After time elapses, the command will be terminated",
	)

	r.cmd.Flags().BoolP(
		r.KeyBufferedOutput(),
		"b",
		false,
		"Commands output should be buffered or streamed",
	)

	// Bind flags to config
	config.ViperBindPFlagSet(r.cmd, nil)

	return r
}

func (r *Raw) RunRawCommand(cmd *cobra.Command, args []string) error {
	var commands []kubestrap.RawCommand
	if err := viper.UnmarshalKey(
		config.PrefixKey(cmd, r.KeyRawUtilities()),
		&commands,
		func(config *mapstructure.DecoderConfig) {
			config.TagName = "yaml"
			config.ErrorUnused = true
			// config.ErrorUnset = true
		},
	); err != nil {
		return err
	}
	if len(args) == 0 {
		_ = cmd.Help()
		fmt.Printf("\nAvailable utilities:\n")
		for _, c := range commands {
			fmt.Printf("  - %s %s\n", c.Name, c.Release)
		}
		return nil
	}
	for _, c := range commands {
		if c.Name == args[0] || slices.Contains(c.Additional, args[0]) {
			timeout := r.Timeout()
			log.Debugf("execution timeout: %s", timeout)
			c.Command = args
			status, err := c.ExecuteCommand(timeout, r.BufferedOutput(), r.stdin)
			if err != nil {
				return fmt.Errorf("error running '%s': %v", c.Command, err)
			}
			if len(status.Stdout) > 0 {
				fmt.Println(strings.Join(status.Stdout, "\n"))
			}
			if status.Exit != 0 {
				if len(status.Stderr) > 0 {
					return fmt.Errorf("command '%s' failed with exit code %d:\n%s", strings.Join(c.Command, " "), status.Exit, strings.Join(status.Stderr, "\n"))
				}
				return fmt.Errorf("command '%s' failed with exit code %d", strings.Join(c.Command, " "), status.Exit)
			}
			// Config allows for duplicates, but here we stop at the first match
			return nil
		}
	}
	// If we get here, the command is not in the config, do not allow that
	return fmt.Errorf("command '%s' is not supported, perhaps add it to the config?", args[0])
}

func (r *Raw) RunRawCommandCaptureStdout(cmd *cobra.Command, args []string) (string, error) {
	// Capture stdout
	p, err := file.NewPipeStdout()
	if err != nil {
		return "", err
	}

	// Run the command
	currentBufferedOutput := r.BufferedOutput()
	r.SetBufferedOutput(true)
	rawErr := r.RunRawCommand(cmd, args)
	r.SetBufferedOutput(currentBufferedOutput)

	// back to normal state
	out, err := p.CloseStdout()
	if err != nil {
		return out, err
	}

	return out, rawErr
}

func (r *Raw) Cmd() *cobra.Command {
	return r.cmd
}

func (r *Raw) KeyTimeout() string {
	return "timeout"
}

func (r *Raw) Timeout() time.Duration {
	return config.ViperGetDuration(r.cmd, r.KeyTimeout())
}

func (r *Raw) KeyRawUtilities() string {
	return "utilities"
}

func (r *Raw) KeyBufferedOutput() string {
	return "buffered-output"
}

func (r *Raw) BufferedOutput() bool {
	return config.ViperGetBool(r.cmd, r.KeyBufferedOutput())
}

func (r *Raw) SetBufferedOutput(v bool) {
	config.ViperSet(r.cmd, r.KeyBufferedOutput(), fmt.Sprint(v))
}

func (r *Raw) SetStdin(stdin io.Reader) {
	r.stdin = stdin
}
