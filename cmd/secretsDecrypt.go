/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/defaults"
	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/go-commons/pkg/search"
	"github.com/thedataflows/kubestrap/pkg/constants"
)

type SecretsDecrypt struct {
	cmd    *cobra.Command
	parent *Secrets
}

var (
	_ = NewSecretsDecrypt(secrets)
)

func init() {

}

func NewSecretsDecrypt(parent *Secrets) *SecretsDecrypt {
	sd := &SecretsDecrypt{
		parent: parent,
	}

	sd.cmd = &cobra.Command{
		Use:           "decrypt",
		Short:         "Decrypt secrets that are relative to the current project root directory",
		Long:          ``,
		Aliases:       []string{"d"},
		RunE:          sd.RunSecretsDecryptCommand,
		SilenceErrors: parent.Cmd().SilenceErrors,
		SilenceUsage:  parent.Cmd().SilenceUsage,
	}

	parent.Cmd().AddCommand(sd.cmd)

	sd.cmd.Flags().BoolP(
		sd.KeyInplace(),
		"i",
		true,
		"Write files in-place instead of outputting to stdout",
	)

	sd.cmd.Flags().String(
		sd.KeyPrivateKeyPath(),
		sd.DefaultPrivateKeyPath(),
		"Private key path",
	)

	// Bind flags to config
	config.ViperBindPFlagSet(sd.cmd, nil)

	return sd
}

func (s *SecretsDecrypt) RunSecretsDecryptCommand(cmd *cobra.Command, args []string) error {
	if err := s.CheckRequiredFlags(); err != nil {
		return err
	}

	if len(args) == 0 {
		args = []string{constants.DefaultSecretFilesPattern}
	}

	if os.Getenv("SOPS_AGE_KEY") == "" {
		log.Infof("loading private key: %s", s.PrivateKeyPath())
		out, err := raw.RunRawCommandCaptureStdout(
			raw.Cmd(),
			[]string{
				"age",
				"--decrypt",
				s.PrivateKeyPath(),
			},
		)
		if err != nil {
			if len(out) == 0 {
				return err
			}
			return fmt.Errorf("%v\n%s", err, out)
		}
		if len(out) == 0 {
			return fmt.Errorf("private key is empty")
		}

		// set SOPS_AGE_KEY environment variable
		if err := os.Setenv("SOPS_AGE_KEY", out); err != nil {
			return err
		}
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

func (s *SecretsDecrypt) findFiles(pattern string) *search.Results {
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

func (s *SecretsDecrypt) CheckRequiredFlags() error {
	return s.parent.CheckRequiredFlags()
}

func (s *SecretsDecrypt) KeyInplace() string {
	return "in-place"
}

func (s *SecretsDecrypt) Inplace() string {
	if config.ViperGetBool(s.cmd, s.KeyInplace()) {
		return "--in-place"
	}
	return ""
}

func (s *SecretsDecrypt) KeyPrivateKeyPath() string {
	return "private-key"
}

func (s *SecretsDecrypt) DefaultPrivateKeyPath() string {
	return "secrets/" + defaults.Undefined + ".age"
}

func (s *SecretsDecrypt) PrivateKeyPath() string {
	privateKeyPath := config.ViperGetString(s.cmd, s.KeyPrivateKeyPath())
	if privateKeyPath == s.DefaultPrivateKeyPath() {
		privateKeyPath = s.parent.SecretsDir() + "/" + s.parent.SecretsContext() + ".age"
	}
	return privateKeyPath
}
