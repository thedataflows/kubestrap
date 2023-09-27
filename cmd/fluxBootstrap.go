/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/log"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge2"
)

type FluxBootstrap struct {
	cmd    *cobra.Command
	parent *Flux
}

// fluxBootstrapCmd represents the fluxBootstrap command
var (
	fluxBootstrapCmd = &cobra.Command{
		Use:     "bootstrap",
		Short:   "Bootstrap or upgrade FluxCD",
		Long:    ``,
		Aliases: []string{"b"},
		RunE:    RunFluxBoostrapCommand,
	}

	fluxBootstrap = NewFluxBootstrap(flux)
)

func init() {
	fluxCmd.AddCommand(fluxBootstrapCmd)
	fluxBootstrapCmd.SilenceErrors = fluxBootstrapCmd.Parent().SilenceErrors

	fluxBootstrapCmd.Flags().StringP(
		fluxBootstrap.KeyFluxBootstrapPath(),
		"p",
		fluxBootstrap.DefaultFluxBootstrapPath(),
		"FluxCD system path in the current repo",
	)

	fluxBootstrapCmd.Flags().String(
		fluxBootstrap.KeyFluxBootstrapCommand(),
		fluxBootstrap.DefaultFluxBootstrapCommand(),
		"FluxCD bootstrap command",
	)

	fluxBootstrapCmd.Flags().String(
		fluxBootstrap.KeyFluxBootstrapPatchesFile(),
		fluxBootstrap.DefaultBootstrapPatchesFile(),
		"FluxCD patches file",
	)

	// Bind flags
	config.ViperBindPFlagSet(fluxBootstrapCmd, nil)

	fluxBootstrap.SetCmd(fluxBootstrapCmd)
}

// RunFluxBoostrapCommand runs flux bootstrap subcommand
func RunFluxBoostrapCommand(cmd *cobra.Command, args []string) error {
	if err := fluxBootstrap.CheckRequiredFlags(); err != nil {
		return err
	}

	// Run the main command
	newArgs := []string{cmd.Parent().Use, cmd.Use}
	newArgs = config.AppendStringArgsf("--%s=%s", cmd.Parent(), newArgs, fluxBootstrap.parent.KeyFluxContext())
	if len(args) > 0 {
		newArgs = append(newArgs, args...)
	} else {
		newArgs = append(newArgs, fmt.Sprintf("--%s=%s", fluxBootstrap.parent.KeyFluxNamespace(), fluxBootstrap.parent.GetFluxNamespace()))
		newArgs = config.AppendStringSplitArgs(cmd, newArgs, fluxBootstrap.KeyFluxBootstrapCommand(), "")
	}
	if err := RunRawCommand(rawCmd, newArgs); err != nil {
		return err
	}

	// Patch flux kustomization
	kustomizationFilePath := fluxBootstrap.GetFluxBootstrapPath() + "/kustomization.yaml"

	kData, err := yaml.ReadFile(kustomizationFilePath)
	if err != nil {
		return err
	}
	pData, err := yaml.ReadFile(fluxBootstrap.GetFluxBootstrapPatchesFile())
	if err != nil {
		return err
	}
	log.Infof("Patching %s", kustomizationFilePath)
	k, err := merge2.Merge(
		pData,
		kData,
		yaml.MergeOptions{
			ListIncreaseDirection: yaml.MergeOptionsListAppend,
		})
	if err != nil {
		return err
	}
	if err = yaml.WriteFile(k, kustomizationFilePath); err != nil {
		return err
	}

	// Git commit and push the patched kustomization
	r, err := git.PlainOpen(root.GetProjectRoot())
	if err != nil {
		return fmt.Errorf("error opening git repository: %v", err)
	}
	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("error getting work tree: %v", err)
	}
	hash, err := w.Add(kustomizationFilePath)
	if err != nil {
		return fmt.Errorf("error adding %s to the git repo: %v", kustomizationFilePath, err)
	}
	hash, err = w.Commit("Patch kustomization",
		&git.CommitOptions{
			Parents: []plumbing.Hash{hash},
		})
	if err != nil {
		return fmt.Errorf("error committing: %v", err)
	}
	log.Infof("Committed %v", hash)
	if err = r.Push(&git.PushOptions{}); err != nil {
		return fmt.Errorf("error pushing: %v", err)
	}

	return nil
}

func NewFluxBootstrap(parent *Flux) *FluxBootstrap {
	return &FluxBootstrap{
		parent: parent,
	}
}

func (f *FluxBootstrap) SetCmd(cmd *cobra.Command) {
	f.cmd = cmd
}

func (f *FluxBootstrap) CheckRequiredFlags() error {
	if err := config.CheckRequiredFlags(f.cmd, []string{f.KeyFluxBootstrapCommand()}); err != nil {
		return err
	}
	return f.parent.CheckRequiredFlags()
}

// Flags keys, defaults and value getters
func (f *FluxBootstrap) KeyFluxBootstrapPath() string {
	return "path"
}

func (f *FluxBootstrap) DefaultFluxBootstrapPath() string {
	return fmt.Sprintf(
		"kubernetes/cluster-%s/%s",
		f.parent.DefaultFluxContext(),
		f.parent.DefaultFluxNamespace(),
	)
}

func (f *FluxBootstrap) GetFluxBootstrapPath() string {
	fluxBootstrapPath := config.ViperGetString(f.cmd, f.KeyFluxBootstrapPath())
	if fluxBootstrapPath == f.DefaultFluxBootstrapPath() {
		fluxBootstrapPath = fmt.Sprintf(
			"%s/kubernetes/cluster-%s/%s",
			f.parent.GetProjectRoot(),
			f.parent.GetFluxContext(),
			f.parent.GetFluxNamespace(),
		)
	}
	return fluxBootstrapPath
}

func (f *FluxBootstrap) KeyFluxBootstrapCommand() string {
	return "command"
}

func (f *FluxBootstrap) DefaultFluxBootstrapCommand() string {
	return ""
}

func (f *FluxBootstrap) GetFluxBootstrapCommand() string {
	return config.ViperGetString(f.cmd, f.KeyFluxBootstrapCommand())
}

func (f *FluxBootstrap) KeyFluxBootstrapPatchesFile() string {
	return "patches-file"
}

func (f *FluxBootstrap) DefaultBootstrapPatchesFile() string {
	return "flux-patches.yaml"
}

func (f *FluxBootstrap) GetFluxBootstrapPatchesFile() string {
	return config.ViperGetString(f.cmd, f.KeyFluxBootstrapPatchesFile())
}
