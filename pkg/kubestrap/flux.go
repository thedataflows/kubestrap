package kubestrap

type Flux struct {
	// Kubernetes context name
	Context string `yaml:"context"`
	// Kubernetes namespace for FluxCD
	Namespace string `yaml:"namespace"`
}
