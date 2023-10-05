/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"context"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/go-commons/pkg/search"
	"github.com/thedataflows/kubestrap/pkg/constants"
)

type SecretsEncrypt struct {
	cmd    *cobra.Command
	parent *Secrets
}

var (
	_ = NewSecretsEncrypt(secrets)
)

func init() {

}

func NewSecretsEncrypt(parent *Secrets) *SecretsEncrypt {
	se := &SecretsEncrypt{
		parent: parent,
	}

	se.cmd = &cobra.Command{
		Use:           "encrypt",
		Short:         "Encrypt secrets that are relative to the current project root directory",
		Long:          ``,
		Aliases:       []string{"e"},
		RunE:          se.RunSecretsEncryptCommand,
		SilenceErrors: parent.Cmd().SilenceErrors,
		SilenceUsage:  parent.Cmd().SilenceUsage,
	}

	parent.Cmd().AddCommand(se.cmd)

	se.cmd.Flags().BoolP(
		se.KeyInplace(),
		"i",
		true,
		"Write files in-place instead of outputting to stdout",
	)

	// Bind flags to config
	config.ViperBindPFlagSet(se.cmd, nil)

	return se
}

func (s *SecretsEncrypt) RunSecretsEncryptCommand(cmd *cobra.Command, args []string) error {
	if err := s.CheckRequiredFlags(); err != nil {
		return err
	}

	if len(args) == 0 {
		args = []string{constants.DefaultSecretFilesPattern}
	}

	for _, arg := range args {
		for _, result := range s.findFiles(arg).Results {
			if result.Err != nil {
				log.Errorf("error finding files: %s", result.Err)
				continue
			}
			if file.IsDirectory(result.FilePath) {
				continue
			}
			log.Infof("%sing: %s", cmd.Use, result.FilePath)
			newArgs := []string{
				"sops",
				"--" + cmd.Use,
				"--config", s.parent.SopsConfig(),
				s.Inplace(),
				result.FilePath,
			}
			config.ViperSet(raw.Cmd(), raw.KeyBufferedOutput(), fmt.Sprintf("%v", s.Inplace() == ""))
			if err := raw.RunRawCommand(raw.Cmd(), newArgs); err != nil {
				log.Error(err)
				continue
			}
		}
	}

	return nil
}

func (s *SecretsEncrypt) findFiles(pattern string) *search.Results {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	finder := &search.JustLister{
		OpenFile: false,
	}

	// This wil not filter anything, will return all files and all directories
	fileFilter := &search.FileFilterByPattern{
		PlainPattern: "",
		RegexPattern: pattern,
		ApplyToDirs:  false,
	}

	return search.FindFile(ctx, s.parent.KubeClusterDir(), fileFilter, finder, runtime.NumCPU())
}

func (s *SecretsEncrypt) CheckRequiredFlags() error {
	return s.parent.CheckRequiredFlags()
}

func (s *SecretsEncrypt) KeyInplace() string {
	return "in-place"
}

func (s *SecretsEncrypt) Inplace() string {
	if config.ViperGetBool(s.cmd, s.KeyInplace()) {
		return "--in-place"
	}
	return ""
}
