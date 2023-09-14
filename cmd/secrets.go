/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/defaults"
	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/go-commons/pkg/reflectutil"
	"github.com/thedataflows/go-commons/pkg/search"
	"github.com/thedataflows/go-commons/pkg/stringutil"
	"github.com/thedataflows/kubestrap/pkg/kubernetes"
	"github.com/thedataflows/kubestrap/pkg/kubestrap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

var (
	typeSecrets                    = &kubestrap.Secrets{}
	keySecretsContext              = reflectutil.GetStructFieldTag(typeSecrets, "Context", "")
	keySecretsNamespace            = reflectutil.GetStructFieldTag(typeSecrets, "Namespace", "")
	keySecretsDir                  = reflectutil.GetStructFieldTag(typeSecrets, "Directory", "")
	keySecretsPrivateKeyPath       = reflectutil.GetStructFieldTag(typeSecrets, "PrivateKey", "")
	keySecretssecretsPublicKeyPath = reflectutil.GetStructFieldTag(typeSecrets, "PublicKey", "")
	keySecretsForce                = reflectutil.GetStructFieldTag(typeSecrets, "Force", "")
	keySecretsSshKeySize           = reflectutil.GetStructFieldTag(typeSecrets, "SshKeySize", "")
	requiredSecretsFlags           = []string{keySecretsContext}
	secretContext                  string
	secretsNamespace               string
	secretsPrivateKeyPath          string
	secretsPublicKeyPath           string
	secretsForce                   bool
	secretsSshKeySize              int
)

// secretsCmd represents the secrets command
var secretsCmd = &cobra.Command{
	Use:     "secrets",
	Short:   "Manages local encrypted secrets. Generates age and ssh keys.",
	Long:    ``,
	Aliases: []string{"s"},
	RunE:    RunSecretsCommand,
}

func initSecretsCmd() {
	secretsCmd.SilenceErrors = rootCmd.SilenceErrors
	rootCmd.AddCommand(secretsCmd)

	secretContext = config.ViperGetString(secretsCmd, keySecretsContext)
	secretsCmd.Flags().StringVarP(
		&secretContext,
		keySecretsContext,
		"c",
		secretContext,
		fmt.Sprintf("[Required] Kubernetes context as defined in '%s'", kubernetes.GetKubeconfigPath()),
	)
	if len(secretContext) == 0 {
		secretContext = defaults.Undefined
	}

	secretsNamespace = config.ViperGetString(secretsCmd, keySecretsNamespace)
	if len(secretsNamespace) == 0 {
		secretsNamespace = "flux-system"
	}
	secretsCmd.Flags().StringVarP(
		&secretsNamespace,
		keySecretsNamespace,
		"n",
		secretsNamespace,
		"Kubernetes namespace for FluxCD Secrets",
	)

	var secretsDir string
	secretsCmd.Flags().StringVarP(
		&secretsDir,
		keySecretsDir,
		"d",
		"secrets",
		"Encrypted secrets directory",
	)

	secretsCmd.Flags().StringVar(
		&secretsPrivateKeyPath,
		keySecretsPrivateKeyPath,
		stringutil.ConcatStrings(secretsDir, "/", secretContext, ".priv.age"),
		"Private key path",
	)
	secretsCmd.Flags().StringVar(
		&secretsPublicKeyPath,
		keySecretssecretsPublicKeyPath,
		stringutil.ConcatStrings(secretsDir, "/", secretContext, ".pub.age"),
		"Public key path",
	)

	secretsForce = config.ViperGetBool(secretsCmd, keySecretsForce)
	secretsCmd.Flags().BoolVar(
		&secretsForce,
		keySecretsForce,
		secretsForce,
		"Force overwrites",
	)

	clusterBootstrapPath = config.ViperGetString(secretsCmd, keyClusterBootstrapPath)
	if len(clusterBootstrapPath) == 0 {
		clusterBootstrapPath = fmt.Sprintf(
			"bootstrap/cluster-%s",
			secretContext,
		)
	}
	secretsCmd.Flags().StringVarP(
		&clusterBootstrapPath,
		keyClusterBootstrapPath,
		"p",
		clusterBootstrapPath,
		"Cluster definition path in the current repo",
	)

	secretsSshKeySize = config.ViperGetInt(secretsCmd, keySecretsSshKeySize)
	secretsCmd.Flags().IntVar(
		&secretsSshKeySize,
		keySecretsSshKeySize,
		256,
		"SSH Private Key Size. Valid values are 224, 256, 384, 521",
	)

	config.ViperBindPFlagSet(secretsCmd, nil)
}

func RunSecretsCommand(cmd *cobra.Command, args []string) error {
	if err := config.CheckRequiredFlags(cmd, requiredSecretsFlags); err != nil {
		return err
	}

	if err := GenerateAgeKeys(cmd); err != nil {
		return err
	}

	// Try to generate ssh private key if not exists, but continue on failure
	if err := GenerateSshKeys("cluster.ssh"); err != nil {
		log.Warnf("Error generating SSH keys: %s", err)
	}
	return nil
}

func GenerateAgeKeys(cmd *cobra.Command) error {
	err := os.MkdirAll(config.ViperGetString(cmd, keySecretsDir), 0700)
	if err != nil {
		return err
	}

	encrypt := false
	if !file.IsAccessible(secretsPrivateKeyPath) {
		// Create the private key
		if err = RunRawCommand(
			rawCmd,
			[]string{
				"age-keygen",
				"--output",
				secretsPrivateKeyPath,
			},
		); err != nil {
			return err
		}
		encrypt = true
	} else {
		finder := search.TextFinder{
			Text: []byte("AGE-SECRET-KEY"),
		}
		found := finder.Grep(secretsPrivateKeyPath)
		if found == nil {
			log.Warnf("Error determining '%s' is encrypted", secretsPrivateKeyPath)
		} else {
			encrypt = len(found.Results) > 0
		}
	}
	if encrypt {
		if file.IsAccessible(secretsPrivateKeyPath+".enc") && !secretsForce {
			return fmt.Errorf("'%s' exists. Use --force flag to override", secretsPrivateKeyPath+".enc")
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
				secretsPrivateKeyPath + ".enc", secretsPrivateKeyPath,
			},
		); err != nil {
			return err
		}
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
				secretsPrivateKeyPath,
			},
		); err != nil {
			return err
		}
	}

	if err = os.Remove(secretsPrivateKeyPath); err != nil {
		log.Warnf("Failed to remove unencrypted '%s': %s", secretsPrivateKeyPath, err)
	}

	return nil
}

func GenerateSshKeys(keyFileBase string) error {
	if len(keyFileBase) == 0 {
		return fmt.Errorf("key base filename is required")
	}
	privateKeyFile := filepath.Join(clusterBootstrapPath, keyFileBase+".enc")
	if file.IsFile(privateKeyFile) && !secretsForce {
		return fmt.Errorf("'%s' exists. Use --force flag to override", privateKeyFile)
	}
	sshPubKey, sshPrivKey, err := GenerateECDSAKeys(secretsSshKeySize)
	if err != nil {
		return fmt.Errorf("failed: %s. Perhaps try with ssh-keygen?", err)
	}

	if err := os.WriteFile(
		privateKeyFile,
		[]byte(sshPrivKey),
		0600,
	); err != nil {
		return fmt.Errorf("error writing private key: %s", err)
	}
	log.Infof("Wrote: %s", privateKeyFile)

	log.Infof("SSH Public key: %s", sshPubKey[:len(sshPubKey)-1])
	publicKeyFile := filepath.Join(clusterBootstrapPath, keyFileBase+".pub")
	if err := os.WriteFile(
		publicKeyFile,
		[]byte(sshPubKey),
		0600,
	); err != nil {
		return fmt.Errorf("error writing public key: %s", err)
	}
	log.Infof("Wrote: %s", publicKeyFile)

	return nil
}

// GenerateECDSAKeys generates ECDSA public and private key pair with given size for SSH.
func GenerateECDSAKeys(bitSize int) (pubKey string, privKey string, err error) {
	// generate private key
	var privateKey *ecdsa.PrivateKey
	if privateKey, err = ecdsa.GenerateKey(
		func(l int) elliptic.Curve {
			switch l {
			case 224:
				return elliptic.P224()
			case 256:
				return elliptic.P256()
			case 521:
				return elliptic.P521()
			}
			return elliptic.P384()
		}(bitSize),
		rand.Reader,
	); err != nil {
		return "", "", err
	}

	// encode public key
	var publicKey ssh.PublicKey
	if publicKey, err = ssh.NewPublicKey(privateKey.Public()); err != nil {
		return "", "", err
	}
	pubBytes := ssh.MarshalAuthorizedKey(publicKey)

	// encrypt private key with password from stdin
	fmt.Print("Enter password: ")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", "", err
	}
	fmt.Println()
	var privBlock *pem.Block
	if privBlock, err = ssh.MarshalPrivateKeyWithPassphrase(privateKey, "", password); err != nil {
		return "", "", err
	}
	privBytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  privBlock.Type,
			Bytes: privBlock.Bytes,
		},
	)
	return string(pubBytes), string(privBytes), nil
}
