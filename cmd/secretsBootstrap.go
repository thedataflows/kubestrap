/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/defaults"
	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/go-commons/pkg/search"
	"github.com/thedataflows/kubestrap/pkg/constants"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type SecretsBootstrap struct {
	cmd    *cobra.Command
	parent *Secrets
}

// secretsBootstrapCmd represents the secrets command
var (
	sshKeyTypes      = []string{"ecdsa-256", "ecdsa-384", "ecdsa-521", "ed25519"}
	secretsBootstrap = NewSecretsBootstrap(secrets)
)

func init() {

}

func NewSecretsBootstrap(parent *Secrets) *SecretsBootstrap {
	sb := &SecretsBootstrap{
		parent: parent,
	}

	sb.cmd = &cobra.Command{
		Use:     "bootstrap",
		Short:   "Generates age and ssh keys.",
		Long:    ``,
		Aliases: []string{"b"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			keyType := sb.SshKeyType()
			if !slices.Contains[[]string, string](sshKeyTypes, keyType) {
				return fmt.Errorf("invalid SSH key size: %s. Valid: %v", keyType, sshKeyTypes)
			}
			return nil
		},
		RunE:          sb.RunBootstrapSecretsCommand,
		SilenceErrors: parent.Cmd().SilenceErrors,
		SilenceUsage:  parent.Cmd().SilenceUsage,
	}

	parent.Cmd().AddCommand(sb.cmd)

	sb.cmd.Flags().StringP(
		sb.KeyNamespace(),
		"n",
		"flux-system",
		"Kubernetes namespace for FluxCD Secrets",
	)

	sb.cmd.Flags().String(
		sb.KeyPrivateKeyPath(),
		sb.DefaultPrivateKeyPath(),
		"Private key path",
	)

	sb.cmd.Flags().String(
		sb.KeyPublicKeyPath(),
		sb.DefaultPublicKeyPath(),
		"Public key path. Can have multiple keys separated by new lines",
	)

	sb.cmd.Flags().Bool(
		sb.KeyForce(),
		false,
		"Force overwrites",
	)

	sb.cmd.Flags().String(
		sb.KeySshKeyType(),
		sshKeyTypes[0],
		fmt.Sprintf("SSH Private Key Type. Valid values: %v", sshKeyTypes),
	)

	// Bind flags to config
	config.ViperBindPFlagSet(sb.cmd, nil)

	return sb
}

func (s *SecretsBootstrap) RunBootstrapSecretsCommand(cmd *cobra.Command, args []string) error {
	if err := s.CheckRequiredFlags(); err != nil {
		return err
	}

	log.Info("generating source files encryption keys")
	if err := s.GenerateAgeKeys(); err != nil {
		log.Errorf("error generating source files encryption keys: %s", err)
	}

	if err := s.PatchSopsConfig(); err != nil {
		log.Errorf("error patching sops config: %s", err)
	}

	// Try to generate ssh private key if not exists, but continue on failure
	log.Info("generating SSH keys")
	if err := s.GenerateSshKeys(constants.DefaultClusterSshKeyFileName); err != nil {
		log.Errorf("error generating SSH keys: %s", err)
	}

	return nil
}

// GenerateAgeKeys generates age public and private key pair and writes them to files
func (s *SecretsBootstrap) GenerateAgeKeys() error {
	if err := os.MkdirAll(s.parent.SecretsDir(), 0700); err != nil {
		return err
	}

	privateKeyPath := s.PrivateKeyPath()
	plainKeyFile := privateKeyPath + ".plain"
	encrypt := false
	if !file.IsAccessible(privateKeyPath) || s.Force() {
		// Create the private key
		if err := raw.RunRawCommand(
			raw.Cmd(),
			[]string{
				"age-keygen",
				"--output",
				plainKeyFile,
			},
		); err != nil {
			return err
		}
		encrypt = true
	} else {
		finder := search.TextFinder{
			Text: []byte("-----BEGIN AGE ENCRYPTED FILE-----"),
		}
		found := finder.Grep(privateKeyPath)
		if found == nil {
			log.Errorf("error determining if '%s' is encrypted", privateKeyPath)
		} else {
			if len(found.Results) > 0 {
				return fmt.Errorf("'%s' exists. Use --force flag to override", privateKeyPath)
			}
			if err := os.Rename(privateKeyPath, plainKeyFile); err != nil {
				return err
			}
			encrypt = true
		}
	}
	if encrypt {
		// Encrypt the private key in place
		if err := raw.RunRawCommand(
			raw.Cmd(),
			[]string{
				"age",
				"--encrypt",
				"--armor",
				"--passphrase",
				"--output",
				privateKeyPath,
				plainKeyFile,
			},
		); err != nil {
			return err
		}
	}

	if !file.IsAccessible(s.PublicKeyPath()) || s.Force() {
		// try to create the public key
		if err := raw.RunRawCommand(
			raw.Cmd(),
			[]string{
				"age-keygen",
				"-y",
				"--output",
				s.PublicKeyPath(),
				plainKeyFile,
			},
		); err != nil {
			return err
		}
	}

	if file.IsAccessible(plainKeyFile) {
		if err := os.Remove(plainKeyFile); err != nil {
			log.Errorf("failed to remove unencrypted '%s': %s", plainKeyFile, err)
		}
	}

	return nil
}

// PatchSopsConfig patches sops config file with age public key
func (s *SecretsBootstrap) PatchSopsConfig() error {
	sopsConfigPath := s.parent.SopsConfig()
	log.Infof("patching sops config: %s", sopsConfigPath)
	if !file.IsAccessible(sopsConfigPath) {
		return fmt.Errorf("'%s' is not accessible", sopsConfigPath)
	}
	pubKeysData, err := os.ReadFile(s.PublicKeyPath())
	if err != nil {
		return err
	}
	pubKeys := strings.Split(string(pubKeysData), "\n")
	filteredPubKeys := make([]string, 0, len(pubKeys))
	for _, pk := range pubKeys {
		pk = strings.TrimSpace(pk)
		if len(pk) > 0 {
			filteredPubKeys = append(filteredPubKeys, "\""+pk+"\"")
		}
	}
	if len(filteredPubKeys) == 0 {
		return fmt.Errorf("'%s' contains empty lines", s.PublicKeyPath())
	}
	const yqExpr = `.creation_rules[].key_groups[].age`
	if err := raw.RunRawCommand(
		raw.Cmd(),
		[]string{
			"yq",
			"--inplace",
			"--prettyPrint",
			fmt.Sprintf("%s += [%s] | %s  = (%s | unique)", yqExpr, strings.Join(filteredPubKeys, ","), yqExpr, yqExpr),
			sopsConfigPath,
		},
	); err != nil {
		return err
	}
	return nil
}

// GenerateSshKeys generates SSH public and private key pair with given size and writes them to files
func (s *SecretsBootstrap) GenerateSshKeys(keyBaseFileName string) error {
	if len(keyBaseFileName) == 0 {
		return fmt.Errorf("key base filename is required")
	}

	clusterBootstrapPath := s.parent.ClusterBootstrapPath()
	if err := os.MkdirAll(clusterBootstrapPath, 0700); err != nil {
		return err
	}

	privateKeyFile := clusterBootstrapPath + "/" + keyBaseFileName
	if file.IsFile(privateKeyFile) && !s.Force() {
		return fmt.Errorf("'%s' exists. Use --force flag to override", privateKeyFile)
	}
	sshPubKey, sshPrivKey, err := GenerateEncodedKeyPair(s.SshKeyType())
	if err != nil {
		return fmt.Errorf("failed: %s. Perhaps try with ssh-keygen?", err)
	}

	if err := os.WriteFile(
		privateKeyFile,
		sshPrivKey,
		0600,
	); err != nil {
		return fmt.Errorf("error writing private key: %s", err)
	}
	log.Infof("wrote: %s", privateKeyFile)

	log.Infof("SSH Public key: %s", sshPubKey[:len(sshPubKey)-1])
	publicKeyFile := privateKeyFile + ".pub"
	if err := os.WriteFile(
		publicKeyFile,
		sshPubKey,
		0600,
	); err != nil {
		return fmt.Errorf("error writing public key: %s", err)
	}
	log.Infof("wrote: %s", publicKeyFile)

	return nil
}

func GenerateEncodedKeyPair(keyType string) (pubKeyBytes, privKeyBytes []byte, err error) {
	var (
		privateKeyRaw crypto.PrivateKey
		publicKeySsh  ssh.PublicKey
	)
	switch keyType {
	// "ecdsa-P256":
	case sshKeyTypes[0]:
		publicKeySsh, privateKeyRaw, err = generateEcdsaKey(elliptic.P256())
	// "ecdsa-P384"
	case sshKeyTypes[1]:
		publicKeySsh, privateKeyRaw, err = generateEcdsaKey(elliptic.P384())
	// "ecdsa-P521"
	case sshKeyTypes[2]:
		publicKeySsh, privateKeyRaw, err = generateEcdsaKey(elliptic.P521())
	// "ed25519"
	case sshKeyTypes[3]:
		var publicKeyRaw ed25519.PublicKey
		publicKeyRaw, privateKeyRaw, err = ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		publicKeySsh, err = ssh.NewPublicKey(publicKeyRaw)
	default:
		return nil, nil, fmt.Errorf("invalid curve: %s. Supported: %v", keyType, sshKeyTypes)
	}
	if err != nil {
		return nil, nil, err
	}

	// encode public key
	pubKeyBytes = ssh.MarshalAuthorizedKey(publicKeySsh)

	// encrypt private key with passphrase
	passphrase, err := readOrGeneratePassphrase("SSH key", 32)
	if err != nil {
		return nil, nil, err
	}
	var privBlock *pem.Block
	if privBlock, err = ssh.MarshalPrivateKeyWithPassphrase(privateKeyRaw, "", passphrase); err != nil {
		return nil, nil, err
	}
	privKeyBytes = pem.EncodeToMemory(
		&pem.Block{
			Type:  privBlock.Type,
			Bytes: privBlock.Bytes,
		},
	)
	return pubKeyBytes, privKeyBytes, nil
}

func generateEcdsaKey(curve elliptic.Curve) (ssh.PublicKey, *ecdsa.PrivateKey, error) {
	privateKeyRaw, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	publicKeySsh, err := ssh.NewPublicKey(privateKeyRaw.Public())
	if err != nil {
		return nil, nil, err
	}

	return publicKeySsh, privateKeyRaw, nil
}

func readOrGeneratePassphrase(subject string, length int) ([]byte, error) {
	fmt.Fprintf(os.Stderr, "Enter passphrase for %s or leave blank to generate: ", subject)
	password1, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, err
	}
	if len(password1) == 0 {
		password1 = []byte(randomBytes(length))
		fmt.Printf("%s generated passphrase: %s\n", subject, string(password1))
	} else {
		fmt.Fprintf(os.Stderr, "Confirm %s passphrase: ", subject)
		password2, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(password1, password2) {
			return nil, fmt.Errorf("%s passphrases do not match", subject)
		}
	}
	return password1, nil
}

// randomBytes generates random bytes of given length from an existing charset
func randomBytes(length int) []byte {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return nil
	}
	for i := 0; i < length; i++ {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return b
}

func (s *SecretsBootstrap) Cmd() *cobra.Command {
	return s.cmd
}

func (s *SecretsBootstrap) CheckRequiredFlags() error {
	return s.parent.CheckRequiredFlags()
}

// Flags keys, defaults and value getters
func (s *SecretsBootstrap) KeyNamespace() string {
	return "namespace"
}

func (s *SecretsBootstrap) Namespace() string {
	return config.ViperGetString(s.cmd, s.KeyNamespace())
}

func (s *SecretsBootstrap) KeyPrivateKeyPath() string {
	const p = "private-key"
	return p
}

func (s *SecretsBootstrap) DefaultPrivateKeyPath() string {
	return "secrets/" + defaults.Undefined + ".age"
}

func (s *SecretsBootstrap) PrivateKeyPath() string {
	privateKeyPath := config.ViperGetString(s.cmd, s.KeyPrivateKeyPath())
	if privateKeyPath == s.DefaultPrivateKeyPath() {
		privateKeyPath = s.parent.SecretsDir() + "/" + s.parent.SecretsContext() + ".age"
	}
	return privateKeyPath
}

func (s *SecretsBootstrap) KeyPublicKeyPath() string {
	return "public-key"
}

func (s *SecretsBootstrap) DefaultPublicKeyPath() string {
	return s.DefaultPrivateKeyPath() + ".pub"
}

func (s *SecretsBootstrap) PublicKeyPath() string {
	publicKeyPath := config.ViperGetString(s.cmd, s.KeyPublicKeyPath())
	if publicKeyPath == s.DefaultPublicKeyPath() {
		publicKeyPath = s.PrivateKeyPath() + ".pub"
	}
	return publicKeyPath
}

func (s *SecretsBootstrap) KeyForce() string {
	return "force"
}

func (s *SecretsBootstrap) Force() bool {
	return config.ViperGetBool(s.cmd, s.KeyForce())
}

func (s *SecretsBootstrap) KeySshKeyType() string {
	return "ssh-key-type"
}

func (s *SecretsBootstrap) SshKeyType() string {
	return config.ViperGetString(s.cmd, s.KeySshKeyType())
}
