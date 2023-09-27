package kubestrap

import (
	"os"

	"github.com/goccy/go-yaml"
	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1"
	"github.com/k0sproject/rig"
	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
)

type KubestrapCluster struct {
	// Kubernetes context name
	Context string `yaml:"context"`
	// Kubernetes cluster bootstrap path
	BootstrapPath string `yaml:"bootstrap-path"`
	// SSH Connections
	Connection []rig.Connection `yaml:"-"`
}

type K0sCluster struct {
	spec    *v1beta1.Cluster
	cluster *KubestrapCluster
}

func NewK0sCluster(context, bootstrapPath string) (*K0sCluster, error) {
	rig.SetLogger(&log.Log)
	newCluster := &K0sCluster{
		spec: &v1beta1.Cluster{},
		cluster: &KubestrapCluster{
			Context:       context,
			BootstrapPath: bootstrapPath,
		},
	}
	currentDir := file.WorkingDirectory()
	if err := os.Chdir(bootstrapPath); err != nil {
		return nil, err
	}
	defer func() { _ = os.Chdir(currentDir) }()
	clusterData, err := os.ReadFile("cluster.yaml")
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(clusterData, newCluster.spec); err != nil {
		return nil, err
	}
	return newCluster, nil
}

func (c *K0sCluster) GetCluster() *KubestrapCluster {
	return c.cluster
}

func (c *K0sCluster) GetClusterSpec() *v1beta1.Cluster {
	return c.spec
}

type Remote struct {
	// List of hosts to run the command on
	Hosts []string `yaml:"hosts"`
}
