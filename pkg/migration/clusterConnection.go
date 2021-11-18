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

package migration

import (
	"fmt"
	"strings"

	"persistent-volume-migrator/pkg/ceph/rbd"
	"persistent-volume-migrator/pkg/k8sutil"

	logger "persistent-volume-migrator/pkg/log"

	v1 "k8s.io/api/core/v1"
	k8s "k8s.io/client-go/kubernetes"
)

// createClusterConnection creates a connection to the ceph cluster.
func createClusterConnection(client *k8s.Clientset, csiPV *v1.PersistentVolume,
	rookNamespace, cephClusterNamespace string) (*rbd.Connection, error) {
	poolName := k8sutil.GetCSIPoolName(csiPV)
	if poolName == "" {
		return nil, fmt.Errorf("poolName cannot be empty in PV object")
	}
	logger.DefaultLog("csi poolname: %v ", poolName)
	clusterID := k8sutil.GetClusterID(csiPV)
	if clusterID == "" {
		return nil, fmt.Errorf("clusterID cannot be empty in PV object")
	}
	csiConfig, err := k8sutil.GetCSIConfiguration(client, rookNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get configmap %v", err)
	}

	var monitor string
	for _, c := range csiConfig {
		if c.ClusterID == clusterID {
			monitor = strings.Join(c.Monitors, ",")
		}
	}
	if monitor == "" {
		return nil, fmt.Errorf("failed to get monitor information")
	}
	logger.DefaultLog("clusterID: %v, monitors: %v, poolname: %v", clusterID, monitor, poolName)
	user, key, err := k8sutil.GetRBDUserAndKeyFromSecret(client, cephClusterNamespace)
	if err != nil {
		return nil, fmt.Errorf("err in GetRBDUserAndKeyFromSecret %v", err)
	}
	conn, err := rbd.NewConnection(monitor, user, key, poolName, "")
	if err != nil {
		return nil, fmt.Errorf("err in GetRBDUserAndKeyFromSecret %v", err)
	}
	return conn, err
}
