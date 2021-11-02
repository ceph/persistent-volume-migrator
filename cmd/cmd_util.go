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

package cmd

import (
	"fmt"
	"strings"

	"persistent-volume-migrator/internal/ceph/rbd"
	"persistent-volume-migrator/internal/kubernetes"
	logger "persistent-volume-migrator/internal/log"

	v1 "k8s.io/api/core/v1"
	k8s "k8s.io/client-go/kubernetes"
)

// migratePVC migrates a list of PVCs to CSI.
func migratePVC(client *k8s.Clientset, pvcs *[]v1.PersistentVolumeClaim) error {
	for _, pvc := range *pvcs {
		logger.DefaultLog("Fetch PV information from PVC %s", pvc.Name)
		pv, err := kubernetes.GetPV(client, pvc.Spec.VolumeName)
		if err != nil {
			return fmt.Errorf("failed to get PV object with name %s: %v", pvc.Spec.VolumeName, err)
		}
		logger.DefaultLog("PV found %v ", pv)

		logger.DefaultLog("Update Reclaim policy from Delete to Reclaim for PV: %s", pv.Name)
		err = kubernetes.UpdateReclaimPolicy(client, pv)
		if err != nil {
			return fmt.Errorf("failed to update ReclaimPolicy for PV object %s: %v", pvc.Spec.VolumeName, err)
		}

		logger.DefaultLog("Retrieving old ceph volume name from PV object: %s", pv.Name)
		rbdImageName := kubernetes.GetFlexVolumeName(pv)
		if rbdImageName == "" {
			return fmt.Errorf("rbdImageName cannot be empty in Flex PV object: %v", pv)
		}
		logger.DefaultLog("Flex rbdImageName name %v ", rbdImageName)

		logger.DefaultLog("Deleting pvc object: %s", pvc.Name)
		err = kubernetes.DeletePVC(client, &pvc) // nolint:gosec // skip gosec as pvc is accessed via it's reference.
		if err != nil {
			// TODO have an option to revert back if some Delete fails?
			return fmt.Errorf("failed to Delete PVC object %s: %v", pvc.Name, err)
		}

		logger.DefaultLog("Generate new PVC with same name in destination storageclass")
		csiPVC := kubernetes.GenerateCSIPVC(destinationStorageClass, &pvc) // nolint:gosec // skip gosec as pvc is accessed via it's reference.
		logger.DefaultLog("Structure of the generated PVC in destination Storageclass %v ", csiPVC)

		logger.DefaultLog("Create new csi pvc")
		csiPV, err := kubernetes.CreatePVC(client, csiPVC, 5)
		if err != nil {
			return fmt.Errorf("failed to Create CSI PVC object %s: %v", pvc.Name, err)
		}
		logger.DefaultLog("New PVC with same name created in CSI. Persistent Volume: %v ", csiPV)

		logger.DefaultLog("Extracting new volume name from CSI PV")
		csiRBDImageName := kubernetes.GetCSIVolumeName(csiPV)
		if csiRBDImageName == "" {
			return fmt.Errorf("csiRBDImageName cannot be empty in PV object %v", csiPV)
		}
		logger.DefaultLog("CSI new volume name: %v ", csiRBDImageName)

		logger.DefaultLog("Fetching csi pool name")
		poolName := kubernetes.GetCSIPoolName(csiPV)
		if poolName == "" {
			return fmt.Errorf("poolName cannot be empty in PV object")
		}
		logger.DefaultLog("csi poolname: %v ", poolName)

		logger.DefaultLog("Create new connection")
		conn, err := createClusterConnection(client, csiPV)
		if err != nil {
			return fmt.Errorf("failed to get cluster config %v \n", err)
		}
		defer func() {
			err = conn.Destroy()
			if err != nil {
				logger.ErrorLog("failed to destroy the connection: %v", err)
			}
		}()
		logger.DefaultLog("Cluster connection created")

		logger.DefaultLog("Delete the CSI volume in ceph cluster")
		err = conn.RemoveVolumeAdmin(poolName, csiRBDImageName)
		if err != nil {
			return fmt.Errorf("failed to delete the CSI volume in ceph cluster: %v", err)
		}
		logger.DefaultLog("Successfully removed volume %s", csiRBDImageName)

		logger.DefaultLog("Rename old ceph volume to new CSI volume")
		err = conn.RenameVolume(csiRBDImageName, rbdImageName)
		if err != nil {
			return fmt.Errorf("failed to rename old ceph volume %s to new CSI volume %s: %v", rbdImageName, csiRBDImageName, err)
		}
		logger.DefaultLog("successfully renamed volume %s -> %s", csiRBDImageName, rbdImageName)

		logger.DefaultLog("Delete old PV object: %s", pv.Name)
		err = kubernetes.DeletePV(client, pv)
		if err != nil {
			return fmt.Errorf("failed to delete persistent volume %s: %v", pv.Name, err)
		}
		logger.DefaultLog("deleted persistent volume %s", pv)
	}

	return nil
}

// creareClusterConnection creates a connection to the ceph cluster.
func createClusterConnection(client *k8s.Clientset, csiPV *v1.PersistentVolume) (*rbd.Connection, error) {
	poolName := kubernetes.GetCSIPoolName(csiPV)
	if poolName == "" {
		return nil, fmt.Errorf("poolName cannot be empty in PV object")
	}
	logger.DefaultLog("csi poolname: %v ", poolName)
	clusterID := kubernetes.GetClusterID(csiPV)
	if clusterID == "" {
		return nil, fmt.Errorf("clusterID cannot be empty in PV object")
	}
	csiConfig, err := kubernetes.GetCSIConfiguration(client, rookNamespace)
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
	logger.DefaultLog("clusterID: %v ,monitors: %v , poolname: %v", clusterID, monitor, poolName)
	user, key, err := kubernetes.GetRBDUserAndKeyFromSecret(client, cephClusterNamespace)
	if err != nil {
		return nil, fmt.Errorf("err in GetRBDUserAndKeyFromSecret %v", err)
	}
	conn, err := rbd.NewConnection(monitor, user, key, poolName, "")
	if err != nil {
		return nil, fmt.Errorf("err in GetRBDUserAndKeyFromSecret %v", err)
	}
	return conn, err
}
