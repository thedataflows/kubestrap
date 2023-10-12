/*
Copyright © 2023 Dataflows
*/
package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"
	"github.com/spf13/cobra"
	"github.com/thedataflows/go-commons/pkg/config"
	"github.com/thedataflows/go-commons/pkg/defaults"
	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/lang"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/kubestrap/pkg/constants"
	"github.com/thedataflows/kubestrap/pkg/kubestrap"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// const (
// 	keySecretsCopySshIdHosts          = "hosts"
// 	keySecretsCopySshIdPrivateKeyFile = "private-key-file"
// )

// var defaultSecretsCopySshIdPrivateKeyFile = fmt.Sprintf("bootstrap/cluster-%s/%s", defaults.Undefined, constants.DefaultClusterSshKeyFileName)

type SecretsCopySshId struct {
	cmd    *cobra.Command
	parent *Secrets
}

// SecretsCopySshIdCmd represents the SecretsCopySshId command
var (
	_ = NewSecretsCopySshId(secrets)
)

func init() {

}

func NewSecretsCopySshId(parent *Secrets) *SecretsCopySshId {
	sc := &SecretsCopySshId{
		parent: parent,
	}

	sc.cmd = &cobra.Command{
		Use:           "copy-ssh-id",
		Short:         "Copy SSH Identities to remote hosts",
		Long:          ``,
		Aliases:       []string{"c"},
		RunE:          sc.RunSecretsCopySshIdCommand,
		SilenceErrors: parent.Cmd().SilenceErrors,
		SilenceUsage:  parent.Cmd().SilenceUsage,
	}

	parent.Cmd().AddCommand(sc.cmd)

	sc.cmd.Flags().StringSlice(
		sc.KeyHosts(),
		[]string{},
		"List of hosts defined in the cluster. If not specified, will run on all hosts",
	)

	sc.cmd.Flags().StringP(
		sc.KeyPrivateKeyFile(),
		"k",
		sc.DefaultPrivateKeyFile(),
		"Private key file to use for SSH authentication",
	)

	// Bind flags to config
	config.ViperBindPFlagSet(sc.cmd, nil)

	return sc
}

func (s *SecretsCopySshId) RunSecretsCopySshIdCommand(cmd *cobra.Command, args []string) error {
	if err := s.CheckRequiredFlags(); err != nil {
		return err
	}

	// Try to copy ssh identity to the cluster
	// Load cluster spec
	secretsContext := s.parent.SecretsContext()
	clusterBootstrapPath := s.parent.ClusterBootstrapPath()
	cl, err := kubestrap.NewK0sCluster(secretsContext, clusterBootstrapPath)
	if err != nil {
		return err
	}
	filterHosts := s.Hosts()
	hosts := cl.GetClusterSpec().Spec.Hosts.Filter(
		func(h *cluster.Host) bool {
			for _, filterHost := range filterHosts {
				if h.Address() == filterHost || h.Metadata.Hostname == filterHost || h.HostnameOverride == filterHost {
					return true
				}
			}
			return len(filterHosts) == 0
		},
	)

	// read private key
	privateKeyFile := s.PrivateKeyFile()
	identity, err := os.ReadFile(privateKeyFile)
	if err != nil {
		log.Errorf("error reading private key file %s: %v", privateKeyFile, err)
	}

	for i := 0; i < len(hosts); i += 1 {
		host := lang.If(
			len(hosts[i].HostnameOverride) > 0,
			hosts[i].HostnameOverride,
			hosts[i].Address(),
		)
		log.Infof("[%s] connecting", host)
		// connect
		sshClient, err := connectToHost("root", hosts[i].Address(), 22, identity)
		if err != nil {
			log.Errorf("[%s] error connecting: %v", host, err)
			continue
		}
		defer sshClient.Close()
		log.Infof("[%s] connected: %s", host, string(sshClient.ServerVersion()))
		// get remote authorized_keys
		remoteAuthKeysFile := "~/.ssh/authorized_keys"
		stdoutPipe, stderrPipe, err := sshRunCommand(
			sshClient,
			fmt.Sprintf("[ -r %s ] && cat %s || true", remoteAuthKeysFile, remoteAuthKeysFile),
		)
		if err != nil {
			stderrBytes, _ := io.ReadAll(stderrPipe)
			log.Errorf("[%s] error running remote command: %v: %s", host, err, string(stderrBytes))
			continue
		}
		// get local pubkey
		pubKey, err := getPubKey(*hosts[i].SSH.KeyPath, clusterBootstrapPath)
		if err != nil {
			log.Errorf("[%s] error reading ssh pubkey: %v", host, err)
			continue
		}
		log.Debugf("[%s] using ssh pubkey '%s'", host, pubKey)
		pubKeyFound := false
		// check if pubkey already exists
		stdoutScanner := bufio.NewScanner(stdoutPipe)
		for stdoutScanner.Scan() {
			output := strings.Split(string(stdoutScanner.Bytes()), " ")
			if len(output) > 1 && strings.Contains(pubKey, output[1]) {
				log.Warnf("[%s] ssh identity '%s' already exists", host, pubKey)
				pubKeyFound = true
				break
			}
		}
		if !pubKeyFound {
			log.Infof("[%s] copying ssh identity", host)
			c := fmt.Sprintf(
				"P=%s; [ ! -d ${P%%/*} ] && mkdir -p ${P%%/*}; echo '%s' >> $P",
				remoteAuthKeysFile,
				pubKey,
			)
			_, stderrPipe, err = sshRunCommand(sshClient, c)
			if err != nil {
				stderrBytes, _ := io.ReadAll(stderrPipe)
				log.Errorf("[%s] error running remote command '%s': %v: %s", host, c, err, string(stderrBytes))
				// Try to run ssh-copy-id script if the above failed
				if err := runSshCopyIdScript(host, clusterBootstrapPath); err != nil {
					log.Errorf("error running ssh-copy-id script: %v", err)
					continue
				}
			}

		}
	}

	return nil
}

func sshRunCommand(sshClient *ssh.Client, command string) (io.Reader, io.Reader, error) {
	log.Debugf("running remote command: %s", command)
	session, err := sshClient.NewSession()
	if err != nil {
		return nil, nil, fmt.Errorf("error creating session: %v", err)
	}
	defer session.Close()

	// Run remote command
	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("error creating stdout pipe: %v", err)
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("error creating stderr pipe: %v", err)
	}
	err = session.Run(command)
	return stdoutPipe, stderrPipe, err
}

func getPubKey(keyPath, clusterBootstrapPath string) (string, error) {
	// set working dir relative to the cluster spec file
	currentDir := file.WorkingDirectory()
	if err := os.Chdir(clusterBootstrapPath); err != nil {
		return "", err
	}
	defer func() { _ = os.Chdir(currentDir) }()
	// open the file
	// TODO use ssh.ParseAuthorizedKey instead?
	f, err := os.Open(keyPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	// Only the first line
	_ = scanner.Scan()
	key := scanner.Text()
	if len(key) > 0 {
		return key, nil
	}
	return "", fmt.Errorf("error parsing ssh pubkey")
}

func readInPassword(prompt string) ([]byte, error) {
	fmt.Fprint(os.Stderr, prompt)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return nil, fmt.Errorf("error reading interactively: %v", err)
		}
		return passphrase, nil
	}
	fmt.Fprintln(os.Stderr)

	passphrase := strings.TrimRight(string(stdInBytes), "\x20\r\n")
	return []byte(passphrase), nil
}

func connectToHost(user, host string, port int, rawPrivateKey []byte) (*ssh.Client, error) {
	authMethods := []ssh.AuthMethod{}
	// try private key first
	if len(rawPrivateKey) > 0 {
		signer, err := signerFromPrivateKey(rawPrivateKey)
		if err != nil {
			log.Errorf("error parsing private key: %v", err)
		} else {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}
	// try password auth after
	authMethods = append(authMethods, ssh.PasswordCallback(
		func() (string, error) {
			password, err := readInPassword("Enter password: ")
			if err != nil {
				return "", fmt.Errorf("error reading password from terminal: %v", err)
			}
			return string(password), nil
		},
	))

	/* #nosec */
	clientConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), clientConfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func signerFromPrivateKey(privateKey []byte) (ssh.Signer, error) {
	if len(privateKey) == 0 {
		return nil, fmt.Errorf("private key is empty")
	}
	signer, err := ssh.ParsePrivateKey(privateKey)
	switch err.(type) {
	case nil:
		break
	case *ssh.PassphraseMissingError:
		passphrase, err := readInPassword("Enter passphrase to decrypt private key: ")
		if err != nil {
			return nil, fmt.Errorf("error reading passphrase from terminal: %v", err)
		}
		signer, err = ssh.ParsePrivateKeyWithPassphrase(privateKey, passphrase)
		if err != nil {
			return nil, fmt.Errorf("error decrypting private key: %v", err)
		}
	default:
		return nil, err
	}

	return signer, nil
}

func runSshCopyIdScript(host, clusterBootstrapPath string) error {
	exeName := "sh"
	sshCopyIdArgs := []string{
		"-c",
		"ssh-copy-id -i " + clusterBootstrapPath + "/" + constants.DefaultClusterSshKeyFileName + " root@" + host,
	}
	status, err := kubestrap.RunProcess(exeName, sshCopyIdArgs, 1*time.Minute, false, nil)
	if err != nil {
		return fmt.Errorf("error running '%s %s: %v'", exeName, strings.Join(sshCopyIdArgs, " "), err)
	}
	if status.Exit != 0 {
		return fmt.Errorf("'%s %s' terminated with code %d: %v", exeName, strings.Join(sshCopyIdArgs, " "), status.Exit, status.Error)
	}

	return nil
}

func (s *SecretsCopySshId) Cmd() *cobra.Command {
	return s.cmd
}

func (s *SecretsCopySshId) CheckRequiredFlags() error {
	return s.parent.CheckRequiredFlags()
}

// Flags keys, defaults and value getters
func (s *SecretsCopySshId) KeyHosts() string {
	return "hosts"
}

func (s *SecretsCopySshId) Hosts() []string {
	return config.ViperGetStringSlice(s.cmd, s.KeyHosts())
}

func (s *SecretsCopySshId) KeyPrivateKeyFile() string {
	return "private-key-file"
}

func (s *SecretsCopySshId) DefaultPrivateKeyFile() string {
	return fmt.Sprintf("bootstrap/cluster-%s/%s", defaults.Undefined, constants.DefaultClusterSshKeyFileName)
}

func (s *SecretsCopySshId) PrivateKeyFile() string {
	privateKeyFile := config.ViperGetString(s.cmd, s.KeyPrivateKeyFile())
	if privateKeyFile == s.DefaultPrivateKeyFile() {
		privateKeyFile = s.parent.ClusterBootstrapPath() + "/" + constants.DefaultClusterSshKeyFileName
	}
	return privateKeyFile
}
