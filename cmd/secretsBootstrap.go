/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"bytes"
	"crypto/ecdsa"
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
	sshKeySizes         = []int{256, 384, 521}
	secretsBootstrapCmd = &cobra.Command{
		Use:     "bootstrap",
		Short:   "Generates age and ssh keys.",
		Long:    ``,
		Aliases: []string{"b"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			secretsBootstrap.SetCmd(cmd)
			keySize := secretsBootstrap.GetSshKeySize()
			if !slices.Contains[[]int, int](sshKeySizes, keySize) {
				return fmt.Errorf("invalid SSH key size: %d. Valid: %v", keySize, sshKeySizes)
			}
			return nil
		},
		RunE: RunBootstrapSecretsCommand,
	}
	secretsBootstrap = NewSecretsBootstrap(secrets)
)

func init() {
	secretsCmd.AddCommand(secretsBootstrapCmd)
	secretsBootstrapCmd.SilenceErrors = secretsBootstrapCmd.Parent().SilenceErrors

	secretsBootstrapCmd.Flags().StringP(
		secretsBootstrap.KeyNamespace(),
		"n",
		secretsBootstrap.DefaultNamespace(),
		"Kubernetes namespace for FluxCD Secrets",
	)

	secretsBootstrapCmd.Flags().String(
		secretsBootstrap.KeyPrivateKeyPath(),
		secretsBootstrap.DefaultPrivateKeyPath(),
		"Private key path",
	)

	secretsBootstrapCmd.Flags().String(
		secretsBootstrap.KeyPublicKeyPath(),
		secretsBootstrap.DefaultPublicKeyPath(),
		"Public key path",
	)

	secretsBootstrapCmd.Flags().Bool(
		secretsBootstrap.KeyForce(),
		secretsBootstrap.DefaultForce(),
		"Force overwrites",
	)

	secretsBootstrapCmd.Flags().Int(
		secretsBootstrap.KeySshKeySize(),
		secretsBootstrap.DefaultSshKeySize(),
		fmt.Sprintf("SSH Private Key Size. Valid values: %v", sshKeySizes),
	)

	// Bind flags
	config.ViperBindPFlagSet(secretsBootstrapCmd, nil)

	secretsBootstrap.SetCmd(secretsBootstrapCmd)
}

func RunBootstrapSecretsCommand(cmd *cobra.Command, args []string) error {
	if err := secretsBootstrap.CheckRequiredFlags(); err != nil {
		return err
	}

	log.Info("Generating source files encryption keys")
	if err := secretsBootstrap.GenerateAgeKeys(); err != nil {
		log.Errorf("error generating source files encryption keys: %s", err)
	}

	log.Info("Patching sops config")
	if err := secretsBootstrap.PatchSopsConfig(); err != nil {
		log.Errorf("error patching sops config: %s", err)
	}

	// Try to generate ssh private key if not exists, but continue on failure
	log.Info("Generating SSH keys")
	if err := secretsBootstrap.GenerateSshKeys(constants.DefaultClusterSshKeyFileName); err != nil {
		log.Errorf("error generating SSH keys: %s", err)
	}

	return nil
}

// GenerateAgeKeys generates age public and private key pair and writes them to files
func (s *SecretsBootstrap) GenerateAgeKeys() error {
	err := os.MkdirAll(s.parent.GetSecretsDir(), 0700)
	if err != nil {
		return err
	}

	privateKeyPath := s.GetPrivateKeyPath()
	plainKeyFile := privateKeyPath + ".plain"
	encrypt := false
	if !file.IsAccessible(privateKeyPath) || s.GetForce() {
		// Create the private key
		if err = RunRawCommand(
			rawCmd,
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
			log.Warnf("Error determining '%s' is encrypted", privateKeyPath)
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
		if err = RunRawCommand(
			rawCmd,
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

	if !file.IsAccessible(s.GetPublicKeyPath()) || s.GetForce() {
		// try to create the public key
		if err = RunRawCommand(
			rawCmd,
			[]string{
				"age-keygen",
				"-y",
				"--output",
				s.GetPublicKeyPath(),
				plainKeyFile,
			},
		); err != nil {
			return err
		}
	}

	if file.IsAccessible(plainKeyFile) {
		if err = os.Remove(plainKeyFile); err != nil {
			log.Warnf("Failed to remove unencrypted '%s': %s", plainKeyFile, err)
		}
	}

	return nil
}

// PatchSopsConfig patches sops config file with age public key
func (s *SecretsBootstrap) PatchSopsConfig() error {
	sopsConfigPath := s.parent.GetSopsConfig()
	if !file.IsAccessible(sopsConfigPath) {
		return fmt.Errorf("'%s' is not accessible", sopsConfigPath)
	}
	pubKeysData, err := os.ReadFile(s.GetPublicKeyPath())
	if err != nil {
		return err
	}
	pubKeys := strings.Split(string(pubKeysData), "\n")
	filteredPubKeys := make([]string, 0, len(pubKeys))
	for _, pk := range pubKeys {
		pk = strings.Trim(pk, " \t")
		if len(pk) > 0 {
			filteredPubKeys = append(filteredPubKeys, "\""+pk+"\"")
		}
	}
	if len(filteredPubKeys) == 0 {
		return fmt.Errorf("'%s' contains empty lines", s.GetPublicKeyPath())
	}
	const yqExpr = `.creation_rules[].key_groups[].age`
	if err := RunRawCommand(
		rawCmd,
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

	clusterBootstrapPath := s.parent.GetClusterBootstrapPath()
	err := os.MkdirAll(clusterBootstrapPath, 0700)
	if err != nil {
		return err
	}

	privateKeyFile := clusterBootstrapPath + "/" + keyBaseFileName
	if file.IsFile(privateKeyFile) && !s.GetForce() {
		return fmt.Errorf("'%s' exists. Use --force flag to override", privateKeyFile)
	}
	sshPubKey, sshPrivKey, err := GenerateECDSAKeys(s.GetSshKeySize())
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
	log.Infof("Wrote: %s", privateKeyFile)

	log.Infof("SSH Public key: %s", sshPubKey[:len(sshPubKey)-1])
	publicKeyFile := privateKeyFile + ".pub"
	if err := os.WriteFile(
		publicKeyFile,
		sshPubKey,
		0600,
	); err != nil {
		return fmt.Errorf("error writing public key: %s", err)
	}
	log.Infof("Wrote: %s", publicKeyFile)

	return nil
}

func ellipticCurve(bitSize int) (elliptic.Curve, error) {
	switch bitSize {
	case 256:
		return elliptic.P256(), nil
	case 384:
		return elliptic.P384(), nil
	case 521:
		return elliptic.P521(), nil
	}

	return nil, fmt.Errorf("invalid bit size: %d. Supported: %v", bitSize, sshKeySizes)
}

// GenerateECDSAKeys generates ECDSA public and private key pair with given size for SSH.
func GenerateECDSAKeys(bitSize int) (pubKey, privKey []byte, err error) {
	curve, err := ellipticCurve(bitSize)
	if err != nil {
		return nil, nil, err
	}
	// generate private key
	var privateKey *ecdsa.PrivateKey
	if privateKey, err = ecdsa.GenerateKey(curve, rand.Reader); err != nil {
		return nil, nil, err
	}

	// encode public key
	var publicKey ssh.PublicKey
	if publicKey, err = ssh.NewPublicKey(privateKey.Public()); err != nil {
		return nil, nil, err
	}
	pubBytes := ssh.MarshalAuthorizedKey(publicKey)

	passphrase, err := readOrGeneratePassphrase("SSH key", 32)
	if err != nil {
		return nil, nil, err
	}
	// encrypt private key with passphrase
	var privBlock *pem.Block
	if privBlock, err = ssh.MarshalPrivateKeyWithPassphrase(privateKey, "", passphrase); err != nil {
		return nil, nil, err
	}
	privBytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  privBlock.Type,
			Bytes: privBlock.Bytes,
		},
	)
	return pubBytes, privBytes, nil
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

func NewSecretsBootstrap(parent *Secrets) *SecretsBootstrap {
	return &SecretsBootstrap{
		parent: parent,
	}
}

func (s *SecretsBootstrap) SetCmd(cmd *cobra.Command) {
	s.cmd = cmd
}

func (s *SecretsBootstrap) CheckRequiredFlags() error {
	return s.parent.CheckRequiredFlags()
}

// Flags keys, defaults and value getters
func (s *SecretsBootstrap) KeyNamespace() string {
	return "namespace"
}

func (s *SecretsBootstrap) DefaultNamespace() string {
	return "flux-system"
}

func (s *SecretsBootstrap) GetNamespace() string {
	return config.ViperGetString(s.cmd, s.KeyNamespace())
}

func (s *SecretsBootstrap) KeyPrivateKeyPath() string {
	return "private-key"
}

func (s *SecretsBootstrap) DefaultPrivateKeyPath() string {
	return "secrets/" + defaults.Undefined + ".age.enc"
}

func (s *SecretsBootstrap) GetPrivateKeyPath() string {
	privateKeyPath := config.ViperGetString(s.cmd, s.KeyPrivateKeyPath())
	if privateKeyPath == s.DefaultPrivateKeyPath() {
		privateKeyPath = s.parent.GetSecretsDir() + "/" + s.parent.GetSecretsContext() + ".age"
	}
	return privateKeyPath
}

func (s *SecretsBootstrap) KeyPublicKeyPath() string {
	return "public-key"
}

func (s *SecretsBootstrap) DefaultPublicKeyPath() string {
	return s.DefaultPrivateKeyPath() + ".pub"
}

func (s *SecretsBootstrap) GetPublicKeyPath() string {
	publicKeyPath := config.ViperGetString(s.cmd, s.KeyPublicKeyPath())
	if publicKeyPath == s.DefaultPublicKeyPath() {
		publicKeyPath = s.GetPrivateKeyPath() + ".pub"
	}
	return publicKeyPath
}

func (s *SecretsBootstrap) KeyForce() string {
	return "force"
}

func (s *SecretsBootstrap) DefaultForce() bool {
	return false
}

func (s *SecretsBootstrap) GetForce() bool {
	return config.ViperGetBool(s.cmd, s.KeyForce())
}

func (s *SecretsBootstrap) KeySshKeySize() string {
	return "ssh-key-size"
}

func (s *SecretsBootstrap) DefaultSshKeySize() int {
	return 256
}

func (s *SecretsBootstrap) GetSshKeySize() int {
	return config.ViperGetInt(s.cmd, s.KeySshKeySize())
}
