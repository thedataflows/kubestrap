package kubernetes

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
)

// GetKubeconfigPath returns path to kubernetes config file
func GetKubeconfigPath() string {
	kubeConfig := os.Getenv("KUBECONFIG")
	if kubeConfig == "" {
		env := "HOME"
		if runtime.GOOS == "windows" {
			env = "USERPROFILE"
		}
		kubeConfig = filepath.Join(os.Getenv(env), "/.kube/config")
	}
	if !file.IsFile(kubeConfig) {
		log.Warnf("Kubernetes config '%s' is not a valid file\n", kubeConfig)
	}
	return kubeConfig
}
