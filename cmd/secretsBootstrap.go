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

	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/defaults"
	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/go-commons/pkg/reflectutil"
	"github.com/thedataflows/go-commons/pkg/search"
	"github.com/thedataflows/go-commons/pkg/stringutil"
	"github.com/thedataflows/kubestrap/pkg/constants"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

var (
	keyBootstrapSecretsNamespace          = reflectutil.GetStructFieldTag(typeSecrets, "Namespace", "")
	keyBootstrapSecretsPrivateKeyPath     = reflectutil.GetStructFieldTag(typeSecrets, "PrivateKey", "")
	keyBootstrapSecretsPublicKeyPath      = reflectutil.GetStructFieldTag(typeSecrets, "PublicKey", "")
	keyBootstrapSecretsForce              = reflectutil.GetStructFieldTag(typeSecrets, "Force", "")
	keyBootstrapSecretsSshKeySize         = reflectutil.GetStructFieldTag(typeSecrets, "SshKeySize", "")
	defaultBootstrapSecretsPrivateKeyPath = "secrets/" + defaults.Undefined + ".age"
	defaultBootstrapSecretsPublicKeyPath  = defaultBootstrapSecretsPrivateKeyPath + ".pub"
)

// secretsBootstrapCmd represents the secrets command
var secretsBootstrapCmd = &cobra.Command{
	Use:     "bootstrap",
	Short:   "Generates age and ssh keys.",
	Long:    ``,
	Aliases: []string{"b"},
	RunE:    RunBootstrapSecretsCommand,
}

func init() {
	secretsCmd.AddCommand(secretsBootstrapCmd)
	secretsBootstrapCmd.SilenceErrors = secretsBootstrapCmd.Parent().SilenceErrors

	secretsBootstrapCmd.Flags().StringP(
		keyBootstrapSecretsNamespace,
		"n",
		"flux-system",
		"Kubernetes namespace for FluxCD Secrets",
	)

	secretsBootstrapCmd.Flags().String(
		keyBootstrapSecretsPrivateKeyPath,
		defaultBootstrapSecretsPrivateKeyPath,
		"Private key path",
	)

	secretsBootstrapCmd.Flags().String(
		keyBootstrapSecretsPublicKeyPath,
		defaultBootstrapSecretsPublicKeyPath,
		"Public key path",
	)

	secretsBootstrapCmd.Flags().Bool(
		keyBootstrapSecretsForce,
		false,
		"Force overwrites",
	)

	secretsBootstrapCmd.Flags().StringP(
		keyClusterBootstrapPath,
		"p",
		defaultClusterBootstrapPath,
		"Cluster definition path in the current repo",
	)

	secretsBootstrapCmd.Flags().Int(
		keyBootstrapSecretsSshKeySize,
		256,
		"SSH Private Key Size. Valid values are 256, 384, 521",
	)

	// Bind flags
	config.ViperBindPFlagSet(secretsBootstrapCmd, nil)
}

func RunBootstrapSecretsCommand(cmd *cobra.Command, args []string) error {
	if err := config.CheckRequiredFlags(cmd.Parent(), requiredSecretsFlags); err != nil {
		return err
	}

	// secretsNamespace := config.ViperGetString(cmd, keyBootstrapSecretsNamespace)

	log.Info("Generating source files encryption keys")
	if err := GenerateAgeKeys(cmd); err != nil {
		log.Warnf("Error generating source files encryption keys: %s", err)
	}

	// Try to generate ssh private key if not exists, but continue on failure
	log.Info("Generating SSH keys")
	if err := GenerateSshKeys(cmd, constants.DefaultClusterSshKeyFileName); err != nil {
		log.Warnf("Error generating SSH keys: %s", err)
	}

	return nil
}

// GenerateAgeKeys generates age public and private key pair and writes them to files
func GenerateAgeKeys(cmd *cobra.Command) error {
	err := os.MkdirAll(config.ViperGetString(cmd.Parent(), keySecretsDir), 0700)
	if err != nil {
		return err
	}

	encrypt := false
	secretsBootstrapPrivateKeyPath := config.ViperGetString(cmd, keyBootstrapSecretsPrivateKeyPath)
	if secretsBootstrapPrivateKeyPath == defaultBootstrapSecretsPrivateKeyPath {
		secretsBootstrapPrivateKeyPath = "secrets/" + config.ViperGetString(cmd.Parent(), keySecretsContext) + ".age"
	}
	secretsForce := config.ViperGetBool(cmd, keyBootstrapSecretsForce)

	if !file.IsAccessible(secretsBootstrapPrivateKeyPath+".enc") || secretsForce {
		// Create the private key
		if err = RunRawCommand(
			rawCmd,
			[]string{
				"age-keygen",
				"--output",
				secretsBootstrapPrivateKeyPath,
			},
		); err != nil {
			return err
		}
		encrypt = true
	} else {
		finder := search.TextFinder{
			Text: []byte("AGE-SECRET-KEY"),
		}
		found := finder.Grep(secretsBootstrapPrivateKeyPath)
		if found == nil {
			log.Warnf("Error determining '%s' is encrypted", secretsBootstrapPrivateKeyPath)
		} else {
			encrypt = len(found.Results) > 0
		}
	}
	if encrypt {
		if file.IsAccessible(secretsBootstrapPrivateKeyPath+".enc") && !secretsForce {
			return fmt.Errorf("'%s' exists. Use --force flag to override", secretsBootstrapPrivateKeyPath+".enc")
		}
		// Encrypt the private key in place
		if err = RunRawCommand(
			rawCmd,
			[]string{
				"age",
				"--encrypt",
				"--armor",
				"--passphrase",
				"--output",
				secretsBootstrapPrivateKeyPath + ".enc", secretsBootstrapPrivateKeyPath,
			},
		); err != nil {
			return err
		}
	}

	secretsPublicKeyPath := config.ViperGetString(cmd, keyBootstrapSecretsPublicKeyPath)
	if secretsPublicKeyPath == defaultBootstrapSecretsPublicKeyPath {
		secretsPublicKeyPath = secretsBootstrapPrivateKeyPath + ".pub"
	}

	if !file.IsAccessible(secretsPublicKeyPath) {
		// try to create the public key
		if err = RunRawCommand(
			rawCmd,
			[]string{
				"age-keygen",
				"-y",
				"--output",
				secretsPublicKeyPath,
				secretsBootstrapPrivateKeyPath,
			},
		); err != nil {
			return err
		}
	}

	if err = os.Remove(secretsBootstrapPrivateKeyPath); err != nil {
		log.Warnf("Failed to remove unencrypted '%s': %s", secretsBootstrapPrivateKeyPath, err)
	}

	return nil
}

// GenerateSshKeys generates SSH public and private key pair with given size and writes them to files
func GenerateSshKeys(cmd *cobra.Command, keyFileName string) error {
	if len(keyFileName) == 0 {
		return fmt.Errorf("key base filename is required")
	}

	clusterBootstrapPath := config.ViperGetString(cmd, keyClusterBootstrapPath)
	if clusterBootstrapPath == defaultClusterBootstrapPath {
		clusterBootstrapPath = fmt.Sprintf("bootstrap/cluster-%s", config.ViperGetString(cmd.Parent(), keySecretsContext))
	}
	err := os.MkdirAll(clusterBootstrapPath, 0700)
	if err != nil {
		return err
	}

	privateKeyFile := clusterBootstrapPath + "/" + keyFileName
	if file.IsFile(privateKeyFile) && !config.ViperGetBool(cmd, keyBootstrapSecretsForce) {
		return fmt.Errorf("'%s' exists. Use --force flag to override", privateKeyFile)
	}
	sshPubKey, sshPrivKey, err := GenerateECDSAKeys(config.ViperGetInt(cmd, keyBootstrapSecretsSshKeySize))
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
	publicKeyFile := stringutil.ConcatStrings(privateKeyFile, ".pub")
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

	return nil, fmt.Errorf("invalid bit size: %d. Supported: 256, 384, 521", bitSize)
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
