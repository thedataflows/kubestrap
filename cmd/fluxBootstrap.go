/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"fmt"
	"os"

	"dario.cat/mergo"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/log"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	keyFluxBootstrapPath       = "path"
	keyFluxBootstrapSubCommand = "subcommand"
	keyFluxPatchesFile         = "patches-file"
)

var (
	fluxBootstrapPath    string
	fluxBootstrapCommand string
	fluxPatchesFile      string
)

// fluxBootstrapCmd represents the fluxBootstrap command
var fluxBootstrapCmd = &cobra.Command{
	Use:     "bootstrap",
	Short:   "Bootstrap or upgrade FluxCD",
	Long:    ``,
	RunE:    RunFluxBoostrapCommand,
	Aliases: []string{"b"},
}

func init() {
	fluxCmd.AddCommand(fluxBootstrapCmd)

	fluxBootstrapPath = config.ViperGetString(fluxBootstrapCmd, keyFluxBootstrapPath)
	if len(fluxBootstrapPath) == 0 {
		fluxBootstrapPath = fmt.Sprintf(
			"kubernetes/cluster-%s/%s",
			fluxContext,
			fluxNamespace,
		)
	}
	fluxBootstrapCmd.Flags().StringVarP(
		&fluxBootstrapPath,
		keyFluxBootstrapPath,
		"p",
		fluxBootstrapPath,
		"FluxCD system path in the current repo",
	)

	fluxBootstrapCmd.Flags().StringVar(
		&fluxBootstrapCommand,
		keyFluxBootstrapSubCommand,
		"",
		"FluxCD bootstrap command",
	)

	fluxBootstrapCmd.Flags().StringVar(
		&fluxPatchesFile,
		keyFluxPatchesFile,
		"flux-patches.yaml",
		"FluxCD patches file",
	)

	config.ViperBindPFlagSet(fluxBootstrapCmd, nil)
}

// RunFluxBoostrapCommand runs flux bootstrap subcommand
func RunFluxBoostrapCommand(cmd *cobra.Command, args []string) error {
	if err := config.CheckRequiredFlags(cmd.Parent(), requiredFluxFlags); err != nil {
		return err
	}
	if err := config.CheckRequiredFlags(cmd, []string{keyFluxBootstrapSubCommand}); err != nil {
		return err
	}

	// Run the main command
	newArgs := []string{cmd.Parent().Use, cmd.Use}
	newArgs = config.AppendStringArgsf("--%s=%s", cmd.Parent(), newArgs, keyFluxContext)
	if len(args) > 0 {
		newArgs = append(newArgs, args...)
	} else {
		newArgs = append(newArgs, fmt.Sprintf("--%s=%s", keyFluxNamespace, fluxNamespace))
		newArgs = config.AppendStringSplitArgs(cmd, newArgs, keyFluxBootstrapSubCommand, "")
	}
	if err := RunRawCommand(rawCmd, newArgs); err != nil {
		return err
	}

	// Patch flux kustomization
	kustomizationFilePath := fmt.Sprintf(
		"%s/%s/kustomization.yaml",
		projectRootDir,
		fluxBootstrapPath,
	)
	kData, err := os.ReadFile(kustomizationFilePath)
	if err != nil {
		return err
	}
	log.Infof("Patching %s", kustomizationFilePath)
	var k types.Kustomization
	if err = yaml.Unmarshal(kData, &k); err != nil {
		return err
	}
	pData, err := os.ReadFile(fluxPatchesFile)
	if err != nil {
		return err
	}
	patch := []types.Patch{}
	if err = yaml.Unmarshal(pData, &patch); err != nil {
		return err
	}
	if err = mergo.Merge(&k.Patches, patch, mergo.WithOverride); err != nil {
		return err
	}
	kOutData, err := yaml.MarshalWithOptions(k, &yaml.EncoderOptions{SeqIndent: yaml.WideSequenceStyle})
	if err != nil {
		return err
	}
	if err = os.WriteFile(kustomizationFilePath, kOutData, 0600); err != nil {
		return err
	}

	// Git commit and push the patched kustomization
	r, err := git.PlainOpen(projectRootDir)
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
