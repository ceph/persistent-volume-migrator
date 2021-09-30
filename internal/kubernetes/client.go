/*
Copyright © 2021 The Persistent-Volume-Migrator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
