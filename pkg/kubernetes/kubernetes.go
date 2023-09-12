package kubernetes

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"

	"github.com/goccy/go-yaml"

	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/kubestrap/pkg/constants"
)

// GetKubeconfigPath returns path to kubernetes config file
func GetKubeconfigPath() string {
	kubeConfig := os.Getenv("KUBECONFIG")
	if kubeConfig == "" {
		env := "HOME"
		if runtime.GOOS == constants.Windows {
			env = "USERPROFILE"
		}
		kubeConfig = filepath.Join(os.Getenv(env), "/.kube/config")
	}
	if !file.IsFile(kubeConfig) {
		log.Warnf("Kubernetes config '%s' is not a valid file\n", kubeConfig)
	}
	return kubeConfig
}

// GetKubernetesDefaultContext returns default kubernetes context
func GetKubernetesDefaultContext(configFile string) (string, error) {
	if len(configFile) == 0 {
		configFile = GetKubeconfigPath()
	}
	// Read the Kubernetes context file.
	data, err := os.ReadFile(configFile)
	if err != nil {
		return "", err
	}

	// Parse the YAML data.
	config := yaml.NewDecoder(bytes.NewReader(data))
	context := struct {
		CurrentContext string `yaml:"current-context"`
	}{}
	err = config.Decode(&context)
	if err != nil {
		return "", err
	}

	// Print the current context.
	return context.CurrentContext, nil
}
