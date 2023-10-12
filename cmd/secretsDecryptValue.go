/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/defaults"
	"github.com/thedataflows/go-commons/pkg/log"

	"github.com/dlclark/regexp2"
)

type SecretsDecryptValue struct {
	cmd    *cobra.Command
	parent *Secrets
}

var (
	_              = NewSecretsDecryptValue(secrets)
	inputFileTypes = []string{"yaml", "json", "any"}
)

func init() {

}

func NewSecretsDecryptValue(parent *Secrets) *SecretsDecryptValue {
	sd := &SecretsDecryptValue{
		parent: parent,
	}

	sd.cmd = &cobra.Command{
		Use:           "decrypt-value",
		Short:         "Decrypt a secret that are relative to the current project root directory and extract a value",
		Example:       parent.parent.cmd.Use + " " + parent.cmd.Use + " --context mycontext decrypt-value --private-key secrets/age.key myfile.yaml",
		Long:          ``,
		Aliases:       []string{"dv"},
		RunE:          sd.RunSecretsDecryptValueCommand,
		SilenceErrors: parent.Cmd().SilenceErrors,
		SilenceUsage:  parent.Cmd().SilenceUsage,
	}

	parent.Cmd().AddCommand(sd.cmd)

	sd.cmd.Flags().StringP(
		sd.KeyInputType(),
		"i",
		inputFileTypes[0],
		fmt.Sprintf("Input file type. One of %s. If empty, grep will be performed", inputFileTypes),
	)

	sd.cmd.Flags().String(
		sd.KeyPrivateKeyPath(),
		sd.DefaultPrivateKeyPath(),
		"Private key path",
	)

	sd.cmd.Flags().StringP(
		sd.KeyYqExpression(),
		"y",
		".",
		"yq expression to extract value. Valid only with yaml or json input type",
	)

	sd.cmd.Flags().StringP(
		sd.KeyRegex(),
		"r",
		"",
		"Regular expression to extract value. Regex library: https://github.com/dlclark/regexp2. Valid only with input type 'any' or empty",
	)

	// Bind flags to config
	config.ViperBindPFlagSet(sd.cmd, nil)

	return sd
}

func (s *SecretsDecryptValue) RunSecretsDecryptValueCommand(cmd *cobra.Command, args []string) error {
	if err := s.CheckRequiredFlags(); err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("no files to decrypt")
	}

	if err := loadAgePrivateKey(s.PrivateKeyPath()); err != nil {
		return err
	}

	for _, arg := range args {
		if filepath.IsAbs(arg) || arg[0] != '/' {
			arg = s.parent.ProjectRoot() + "/" + arg
		}
		log.Infof("decrypting: %s", arg)
		out, err := raw.RunRawCommandCaptureStdout(
			raw.Cmd(),
			[]string{
				"sops",
				"--decrypt",
				"--in-place=false",
				arg,
			},
		)
		if err != nil {
			if len(out) == 0 {
				return err
			}
			return fmt.Errorf("%v\n%s", err, out)
		}

		switch s.InputType() {
		case "yaml", "json":
			reader, writer := io.Pipe()
			raw.SetStdin(reader)
			defer raw.SetStdin(nil)
			go func() {
				_, _ = io.Copy(writer, bytes.NewBufferString(out))
				_ = writer.Close()
			}()
			out, err = raw.RunRawCommandCaptureStdout(
				raw.Cmd(),
				[]string{
					"yq",
					"e",
					s.YqExpression(),
					"-",
				},
			)
			if err != nil {
				if len(out) == 0 {
					return err
				}
				return fmt.Errorf("%v\n%s", err, out)
			}

			fmt.Println(out)
		default:
			if s.Regex() == "" {
				fmt.Println(out)
				break
			}
			pattern, err := regexp2.Compile(s.Regex(), 0)
			if err != nil {
				return err
			}
			found, err := pattern.FindStringMatch(out)
			if err != nil {
				return err
			}
			for found != nil {
				fmt.Printf("%s\n", found.String())
				found, _ = pattern.FindNextMatch(found)
			}
		}
	}

	return nil
}

func (s *SecretsDecryptValue) CheckRequiredFlags() error {
	return s.parent.CheckRequiredFlags()
}

func (s *SecretsDecryptValue) KeyInputType() string {
	return "input-type"
}

func (s *SecretsDecryptValue) InputType() string {
	return config.ViperGetString(s.cmd, s.KeyInputType())
}

func (s *SecretsDecryptValue) KeyPrivateKeyPath() string {
	return "private-key"
}

func (s *SecretsDecryptValue) DefaultPrivateKeyPath() string {
	return "secrets/" + defaults.Undefined + ".age"
}

func (s *SecretsDecryptValue) PrivateKeyPath() string {
	privateKeyPath := config.ViperGetString(s.cmd, s.KeyPrivateKeyPath())
	if privateKeyPath == s.DefaultPrivateKeyPath() {
		privateKeyPath = s.parent.SecretsDir() + "/" + s.parent.SecretsContext() + ".age"
	}
	return privateKeyPath
}

func (s *SecretsDecryptValue) KeyYqExpression() string {
	return "yq-expression"
}

func (s *SecretsDecryptValue) YqExpression() string {
	return config.ViperGetString(s.cmd, s.KeyYqExpression())
}

func (s *SecretsDecryptValue) KeyRegex() string {
	return "regex"
}

func (s *SecretsDecryptValue) Regex() string {
	return config.ViperGetString(s.cmd, s.KeyRegex())
}
