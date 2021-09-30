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
	"context"
	"encoding/json"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type csiClusterConfigEntry struct {
	ClusterID string   `json:"clusterID"`
	Monitors  []string `json:"monitors"`
}

type csiClusterConfig []csiClusterConfigEntry

func parseCsiClusterConfig(c string) (csiClusterConfig, error) {
	var cc csiClusterConfig
	err := json.Unmarshal([]byte(c), &cc)
	if err != nil {
		return cc, fmt.Errorf("failed to parse csi cluster config %w", err)
	}
	return cc, nil
}

func GetCSIConfiguration(client *kubernetes.Clientset, namespace string) (csiClusterConfig, error) {
	var cc csiClusterConfig
	getOpt := v1.GetOptions{}
	ctx := context.TODO()
	cm, err := client.CoreV1().ConfigMaps(namespace).Get(ctx, "rook-ceph-csi-config", getOpt)
	if err != nil {
		return nil, err
	}
	c := cm.Data["csi-cluster-config-json"]
	err = json.Unmarshal([]byte(c), &cc)
	if err != nil {
		return cc, fmt.Errorf("failed to parse csi cluster config %w", err)
	}
	return cc, nil
}
