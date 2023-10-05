/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

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

var (
	_ = NewFluxBootstrap(flux)
)

func init() {

}

func NewFluxBootstrap(parent *Flux) *FluxBootstrap {
	fb := &FluxBootstrap{
		parent: parent,
	}

	fb.cmd = &cobra.Command{
		Use:           "bootstrap",
		Short:         "Bootstrap or upgrade FluxCD",
		Long:          ``,
		Aliases:       []string{"b"},
		RunE:          fb.RunFluxBoostrapCommand,
		SilenceErrors: parent.Cmd().SilenceErrors,
		SilenceUsage:  parent.Cmd().SilenceUsage,
	}

	parent.Cmd().AddCommand(fb.cmd)

	fb.cmd.Flags().StringP(
		fb.KeyFluxBootstrapPath(),
		"p",
		fb.DefaultFluxBootstrapPath(),
		"FluxCD system path in the current repo",
	)

	fb.cmd.Flags().String(
		fb.KeyFluxBootstrapCommand(),
		"",
		"FluxCD bootstrap command",
	)

	fb.cmd.Flags().String(
		fb.KeyFluxBootstrapPatchesFile(),
		"flux-patches.yaml",
		"FluxCD patches file",
	)

	// Bind flags to config
	config.ViperBindPFlagSet(fb.cmd, nil)

	return fb
}

func (f *FluxBootstrap) RunFluxBoostrapCommand(cmd *cobra.Command, args []string) error {
	if err := f.CheckRequiredFlags(); err != nil {
		return err
	}

	// Run the main command
	newArgs := []string{cmd.Parent().Use, cmd.Use}
	newArgs = config.AppendStringArgsf("--%s=%s", cmd.Parent(), newArgs, f.parent.KeyFluxContext())
	if len(args) > 0 {
		newArgs = append(newArgs, args...)
	} else {
		newArgs = append(newArgs, fmt.Sprintf("--%s=%s", f.parent.KeyFluxNamespace(), f.parent.FluxNamespace()))
		newArgs = config.AppendStringSplitArgs(cmd, newArgs, f.KeyFluxBootstrapCommand(), "")
	}

	config.ViperSet(raw.Cmd(), f.parent.KeyTimeout(), f.parent.Timeout())
	if err := raw.RunRawCommand(raw.Cmd(), newArgs); err != nil {
		return err
	}

	kustomizationFilePath, err := filepath.Abs(f.FluxBootstrapPath() + "/kustomization.yaml")
	if err != nil {
		return err
	}

	// Patch flux kustomization
	kData, err := yaml.ReadFile(kustomizationFilePath)
	if err != nil {
		return err
	}
	pData, err := yaml.ReadFile(f.FluxBootstrapPatchesFile())
	if err != nil {
		return err
	}
	log.Infof("patching %s", kustomizationFilePath)
	k, err := merge2.Merge(
		pData,
		kData,
		yaml.MergeOptions{
			ListIncreaseDirection: yaml.MergeOptionsListAppend,
		})
	if err != nil {
		return err
	}
	if err := writeYaml(k, kustomizationFilePath); err != nil {
		return err
	}

	// TODO repair this, it corrupts the git repo
	// Git commit and push the patched kustomization
	// r, err := git.PlainOpen(f.parent.ProjectRoot())
	// if err != nil {
	// 	return fmt.Errorf("error opening git repository: %v", err)
	// }
	// w, err := r.Worktree()
	// if err != nil {
	// 	return fmt.Errorf("error getting work tree: %v", err)
	// }
	// status, err := w.Status()
	// if err != nil {
	// 	return fmt.Errorf("error getting git status: %v", err)
	// }
	// if status.File(kustomizationFilePath).Staging == git.Unmodified {
	// 	return nil
	// }
	// // file relative to the git root
	// rootDirAbs, err := filepath.Abs(f.parent.ProjectRoot())
	// if err != nil {
	// 	return err
	// }
	// kustomizationFilePath = strings.Replace(kustomizationFilePath, rootDirAbs, "", -1)
	// hash, err := w.Add(kustomizationFilePath)
	// if err != nil {
	// 	return fmt.Errorf("error adding %s to the git repo: %v", kustomizationFilePath, err)
	// }

	// hash, err = w.Commit("Patch kustomization",
	// 	&git.CommitOptions{
	// 		Parents: []plumbing.Hash{hash},
	// 	})
	// if err != nil {
	// 	return fmt.Errorf("error committing: %v", err)
	// }
	// log.Infof("committed %v", hash)
	// if err := r.Push(&git.PushOptions{}); err != nil {
	// 	return fmt.Errorf("error pushing: %v", err)
	// }

	return nil
}

// writeYaml writes yaml node to a file. Inspired from yaml.String() with WideSequenceStyle formatting options and space trimming
func writeYaml(node *yaml.RNode, filePath string) error {
	b := &bytes.Buffer{}
	node.Document().Style = yaml.FlowStyle
	e := yaml.NewEncoderWithOptions(b, &yaml.EncoderOptions{
		SeqIndent: yaml.WideSequenceStyle,
	})
	if err := e.Encode(node.Document()); err != nil {
		return err
	}
	if err := e.Close(); err != nil {
		return err
	}

	return os.WriteFile(filePath, b.Bytes(), 0600)
}

func (f *FluxBootstrap) CheckRequiredFlags() error {
	if err := config.CheckRequiredFlags(f.cmd, []string{f.KeyFluxBootstrapCommand()}); err != nil {
		return err
	}
	return f.parent.CheckRequiredFlags()
}

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

func (f *FluxBootstrap) FluxBootstrapPath() string {
	fluxBootstrapPath := config.ViperGetString(f.cmd, f.KeyFluxBootstrapPath())
	if fluxBootstrapPath == f.DefaultFluxBootstrapPath() {
		fluxBootstrapPath = fmt.Sprintf(
			"%s/kubernetes/cluster-%s/%s",
			f.parent.ProjectRoot(),
			f.parent.FluxContext(),
			f.parent.FluxNamespace(),
		)
	}
	return fluxBootstrapPath
}

func (f *FluxBootstrap) KeyFluxBootstrapCommand() string {
	return "command"
}

func (f *FluxBootstrap) KeyFluxBootstrapPatchesFile() string {
	return "patches-file"
}

func (f *FluxBootstrap) FluxBootstrapPatchesFile() string {
	return config.ViperGetString(f.cmd, f.KeyFluxBootstrapPatchesFile())
}
