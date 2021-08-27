package kubernetes

import (
	"fmt"
	"os"

	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClient create kubernetes client.
func NewClient(configPath string) (*k8s.Clientset, error) {
	var cfg *rest.Config
	var err error
	if configPath == "" {
		configPath = os.Getenv("KUBERNETES_CONFIG_PATH")
	}
	if configPath != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", configPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to get cluster config with error: %v\n", err)
		}
	} else {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("Failed to get cluster config with error: %v\n", err)
		}
	}
	client, err := k8s.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("Failed to create client with error: %v\n", err)
	}
	return client, nil
}
