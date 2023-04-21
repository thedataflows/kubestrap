package kubestrap

type Secrets struct {
	// Kubernetes context name
	Context string `yaml:"context"`
	// Kubernetes namespace
	Namespace string `yaml:"namespace"`
	// Secrets directory
	Directory string `yaml:"directory"`
	// Private key path
	PrivateKey string `yaml:"private-key"`
	// Public key path
	PublicKey string `yaml:"public-key"`
}
