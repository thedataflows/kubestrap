/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/defaults"
	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/go-commons/pkg/search"
)

type SecretsEncryptDecrypt struct {
	cmd    *cobra.Command
	parent *Secrets
}

var (
	// secretsEncryptCmd represents the secrets command
	secretsEncryptCmd = &cobra.Command{
		Use:     "encrypt",
		Short:   "Encrypt secrets that are relative to the current project root directory",
		Long:    ``,
		Aliases: []string{"e"},
		RunE:    RunSecretsEncryptDecryptCommand,
	}

	// secretsEncryptCmd represents the secrets command
	secretsDecryptCmd = &cobra.Command{
		Use:     "decrypt",
		Short:   "Decrypt secrets that are relative to the current project root directory",
		Long:    ``,
		Aliases: []string{"d"},
		RunE:    RunSecretsEncryptDecryptCommand,
	}

	secretsEncryptDecrypt = NewSecretsEncryptDecrypt(secrets)
)

func init() {
	secretsCmd.AddCommand(secretsEncryptCmd)
	secretsEncryptCmd.SilenceErrors = secretsEncryptCmd.Parent().SilenceErrors

	secretsCmd.AddCommand(secretsDecryptCmd)
	secretsEncryptCmd.SilenceErrors = secretsDecryptCmd.Parent().SilenceErrors

	flags := pflag.FlagSet{}

	flags.BoolP(
		secretsEncryptDecrypt.KeyInplace(),
		"i",
		secretsEncryptDecrypt.DefaultInplace(),
		"Write files in-place instead of outputting to stdout",
	)

	flags.StringP(
		secretsEncryptDecrypt.KeySopsConfig(),
		"s",
		secretsEncryptDecrypt.DefaultSopsConfig(),
		"SOPS configuration file",
	)

	flags.StringP(
		secretsEncryptDecrypt.KeyKubeClusterDir(),
		"k",
		secretsEncryptDecrypt.DefaultKubeClusterDir(),
		"Kubernetes cluster directory",
	)

	secretsEncryptCmd.Flags().AddFlagSet(&flags)
	secretsDecryptCmd.Flags().AddFlagSet(&flags)

	// specific to decrypt
	secretsDecryptCmd.Flags().String(
		secretsEncryptDecrypt.KeyPrivateKeyPath(),
		secretsEncryptDecrypt.DefaultPrivateKeyPath(),
		"Private key path",
	)

	// Bind flags
	config.ViperBindPFlagSet(secretsEncryptCmd, nil)
	config.ViperBindPFlagSet(secretsDecryptCmd, nil)
}

func RunSecretsEncryptDecryptCommand(cmd *cobra.Command, args []string) error {
	secretsEncryptDecrypt.SetCmd(cmd)
	if err := secretsEncryptDecrypt.CheckRequiredFlags(); err != nil {
		return err
	}

	if len(args) == 0 {
		args = []string{secretsEncryptDecrypt.DefaultSecretsEncryptDecryptFilePattern()}
	}

	if cmd.Use == "decrypt" && os.Getenv("SOPS_AGE_KEY") == "" {
		log.Infof("Loading private key: %s", secretsEncryptDecrypt.GetPrivateKeyPath())
		// set SOPS_AGE_KEY environment variable
		status, err := LoadRawCommandsAndRunOne(
			rawCmd,
			[]string{
				"age",
				"--decrypt",
				secretsEncryptDecrypt.GetPrivateKeyPath(),
			},
			true,
		)
		if err != nil {
			return err
		}
		if status.Exit != 0 {
			cmd.SilenceUsage = true
			return fmt.Errorf("failed to decrypt private key:\n%s", strings.Join(status.Stderr, "\n"))
		}
		err = os.Setenv("SOPS_AGE_KEY", strings.Join(status.Stdout, "\n"))
		if err != nil {
			return err
		}
	}

	for _, arg := range args {
		for _, result := range findFiles(arg).Results {
			if result.Err != nil {
				log.Errorf("error finding files: %s", result.Err)
				continue
			}
			if file.IsDirectory(result.FilePath) {
				continue
			}
			log.Infof("%s%sing: %s", strings.ToUpper(cmd.Use[:1]), cmd.Use[1:], result.FilePath)
			newArgs := []string{"sops", "--" + cmd.Use}
			if cmd.Use == "encrypt" {
				newArgs = append(newArgs, "--config", secretsEncryptDecrypt.GetSopsConfig())
			}
			newArgs = append(newArgs, secretsEncryptDecrypt.GetInplace(), result.FilePath)
			status, err := LoadRawCommandsAndRunOne(rawCmd, newArgs, true)
			if err != nil {
				log.Errorf("error running '%s': %v", strings.Join(newArgs, " "), err)
				continue
			}
			if status.Exit != 0 {
				log.Errorf("command '%s' failed with exit code %d:\n%s", strings.Join(newArgs, " "), status.Exit, strings.Join(status.Stderr, "\n"))
				continue
			}
			if len(status.Stdout) > 0 {
				fmt.Println(strings.Join(status.Stdout, "\n"))
			}
		}
	}

	return nil
}

func findFiles(pattern string) *search.Results {
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

	return search.FindFile(ctx, secretsEncryptDecrypt.GetProjectRoot(), fileFilter, finder, runtime.NumCPU())
}

func NewSecretsEncryptDecrypt(parent *Secrets) *SecretsEncryptDecrypt {
	return &SecretsEncryptDecrypt{
		parent: parent,
	}
}

func (s *SecretsEncryptDecrypt) SetCmd(cmd *cobra.Command) {
	s.cmd = cmd
}

func (s *SecretsEncryptDecrypt) CheckRequiredFlags() error {
	return s.parent.CheckRequiredFlags()
}

// Flags keys, defaults and value getters
func (s *SecretsEncryptDecrypt) DefaultSecretsEncryptDecryptFilePattern() string {
	return `secret.*\.yaml`
}
func (s *SecretsEncryptDecrypt) KeyInplace() string {
	return "in-place"
}

func (s *SecretsEncryptDecrypt) DefaultInplace() bool {
	return false
}

func (s *SecretsEncryptDecrypt) GetInplace() string {
	if config.ViperGetBool(s.cmd, s.KeyInplace()) {
		return "--in-place"
	}
	return ""
}

func (s *SecretsEncryptDecrypt) KeySopsConfig() string {
	return "sops-config"
}

func (s *SecretsEncryptDecrypt) DefaultSopsConfig() string {
	return s.DefaultKubeClusterDir() + "/.sops.yaml"
}

func (s *SecretsEncryptDecrypt) GetSopsConfig() string {
	secretsEncryptSopsConfig := config.ViperGetString(s.cmd, s.KeySopsConfig())
	if secretsEncryptSopsConfig == s.DefaultSopsConfig() {
		secretsEncryptSopsConfig = s.GetKubeClusterDir() + "/.sops.yaml"
	}
	return secretsEncryptSopsConfig
}

func (s *SecretsEncryptDecrypt) KeyKubeClusterDir() string {
	return "kube-cluster-dir"
}

func (s *SecretsEncryptDecrypt) DefaultKubeClusterDir() string {
	return fmt.Sprintf("kubernetes/cluster-%s", defaults.Undefined)
}

func (s *SecretsEncryptDecrypt) GetKubeClusterDir() string {
	secretsEncryptKubeClusterDir := config.ViperGetString(s.cmd, s.KeyKubeClusterDir())
	if secretsEncryptKubeClusterDir == s.DefaultKubeClusterDir() {
		secretsEncryptKubeClusterDir = fmt.Sprintf(
			"%s/kubernetes/cluster-%s",
			s.GetProjectRoot(),
			s.parent.GetSecretsContext(),
		)
	}
	return secretsEncryptKubeClusterDir
}

func (s *SecretsEncryptDecrypt) KeyPrivateKeyPath() string {
	return "private-key"
}

func (s *SecretsEncryptDecrypt) DefaultPrivateKeyPath() string {
	return "secrets/" + defaults.Undefined + ".age.enc"
}

func (s *SecretsEncryptDecrypt) GetPrivateKeyPath() string {
	privateKeyPath := config.ViperGetString(s.cmd, s.KeyPrivateKeyPath())
	if privateKeyPath == s.DefaultPrivateKeyPath() {
		privateKeyPath = s.parent.GetSecretsDir() + "/" + s.parent.GetSecretsContext() + ".age"
	}
	return privateKeyPath
}

func (r *SecretsEncryptDecrypt) GetProjectRoot() string {
	return r.parent.GetProjectRoot()
}
