package migration

import (
	"fmt"

	"persistent-volume-migrator/pkg/ceph/rbd"
	"persistent-volume-migrator/pkg/k8sutil"
	logger "persistent-volume-migrator/pkg/log"

	v1 "k8s.io/api/core/v1"
	k8s "k8s.io/client-go/kubernetes"
)

const (
	pvcCreateTimeout = 5
)

func MigrateToCSI(kubeConfig, sourceStorageClass, destinationStorageClass, rookNamespace, cephClusterNamespace, pvcName, pvcNamespace string) error {
	// TODO
	/*
		add validation to check source,destination storageclass and Rook namespace
	*/

	// Create Kubernetes Client
	logger.DefaultLog("Create Kubernetes Client")
	client, err := k8sutil.NewClient(kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %v", err)
	}

	logger.DefaultLog("List all the PVC from the source storageclass")
	var pvcs *[]v1.PersistentVolumeClaim
	if pvcNamespace != "" && pvcName != "" {
		pvcs, err = k8sutil.ListSinglePVCWithStorageclass(client, pvcName, pvcNamespace)
		if err != nil {
			return fmt.Errorf("failed to list PVCs from the pvc name %s and pvc namespace %s : %v", pvcName, pvcNamespace, err)
		}
		if pvcs == nil || len(*pvcs) == 0 {
			logger.DefaultLog("no PVCs found with the pvc name %s and pvc namespace %s : %v", pvcName, pvcNamespace, err)
			return nil
		}
	} else {
		pvcs, err = k8sutil.ListAllPVCWithStorageclass(client, sourceStorageClass)
		if err != nil {
			return fmt.Errorf("failed to list PVCs from the storageclass: %v", err)
		}
		if pvcs == nil || len(*pvcs) == 0 {
			logger.DefaultLog("no PVCs found with storageclass: %v", sourceStorageClass)
			return nil
		}
	}

	logger.DefaultLog("%d PVCs found with source StorageClass %s ", len(*pvcs), sourceStorageClass)

	logger.DefaultLog("Start Migration of PVCs to CSI")
	for _, pvc := range *pvcs {
		err = migratePVC(client, pvc, destinationStorageClass, rookNamespace, cephClusterNamespace)
		if err != nil {
			return fmt.Errorf("failed to migrate PVC %s : %v", pvc.Name, err)
		}
	}
	logger.DefaultLog("Successfully migrated all the PVCs to CSI")

	return err
}

// migratePVC migrates a PVC to CSI.
func migratePVC(client *k8s.Clientset, pvc v1.PersistentVolumeClaim, destinationStorageClass,
	rookNamespace, cephClusterNamespace string) error {

	logger.DefaultLog("migrating PVC %q from namespace %q", pvc.Name, pvc.Namespace)

	logger.DefaultLog("Fetch PV information from PVC %s", pvc.Name)
	pv, err := k8sutil.GetPV(client, pvc.Spec.VolumeName)
	if err != nil {
		return fmt.Errorf("failed to get PV object with name %s: %v", pvc.Spec.VolumeName, err)
	}
	logger.DefaultLog("PV found %q ", pv.Name)

	logger.DefaultLog("Update Reclaim policy from Delete to Reclaim for PV: %s", pv.Name)
	err = k8sutil.UpdateReclaimPolicy(client, pv)
	if err != nil {
		return fmt.Errorf("failed to update ReclaimPolicy for PV object %s: %v", pvc.Spec.VolumeName, err)
	}

	logger.DefaultLog("Retrieving old ceph volume name from PV object: %s", pv.Name)
	rbdImageName := k8sutil.GetVolumeName(pv)
	if rbdImageName == "" {
		return fmt.Errorf("rbdImageName cannot be empty in Flex PV object: %v", pv)
	}
	logger.DefaultLog("rbd image name is %q ", rbdImageName)

	logger.DefaultLog("Deleting pvc object: %s", pvc.Name)
	err = k8sutil.DeletePVC(client, &pvc) // nolint:gosec // skip gosec as pvc is accessed via it's reference.
	if err != nil {
		return fmt.Errorf("failed to Delete PVC object %s: %v", pvc.Name, err)
	}

	logger.DefaultLog("Generate new PVC with same name in destination storageclass")
	csiPVC := k8sutil.GenerateCSIPVC(destinationStorageClass, &pvc) // nolint:gosec // skip gosec as pvc is accessed via it's reference.

	logger.DefaultLog("Create new csi pvc")
	csiPV, err := k8sutil.CreatePVC(client, csiPVC, pvcCreateTimeout)
	if err != nil {
		return fmt.Errorf("failed to Create CSI PVC object %s: %v", pvc.Name, err)
	}
	logger.DefaultLog("New PVC with same name %q created via CSI", csiPVC.Name)

	logger.DefaultLog("Extracting new volume name from CSI PV")
	csiRBDImageName := k8sutil.WaitForRBDImage(csiPV)
	if csiRBDImageName == "" {
		return fmt.Errorf("csiRBDImageName cannot be empty in PV object %v", csiPV)
	}
	logger.DefaultLog("CSI new volume name: %v ", csiRBDImageName)

	logger.DefaultLog("Fetching csi pool name")
	poolName := k8sutil.GetCSIPoolName(csiPV)
	if poolName == "" {
		return fmt.Errorf("poolName cannot be empty in PV object")
	}
	logger.DefaultLog("csi poolname: %v ", poolName)

	logger.DefaultLog("Create new Ceph connection")
	conn, err := createClusterConnection(client, csiPV, rookNamespace, cephClusterNamespace)
	if err != nil {
		return fmt.Errorf("failed to get cluster config %v", err)
	}
	defer func() {
		err = rbd.RemoveKeyDir()
		if err != nil {
			logger.ErrorLog("failed to destroy the connection: %v", err)
		}
	}()
	logger.DefaultLog("Cluster connection created")

	logger.DefaultLog("Delete the placeholder CSI volume in ceph cluster")
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
	err = k8sutil.DeletePV(client, pv)
	if err != nil {
		return fmt.Errorf("failed to delete persistent volume %s: %v", pv.Name, err)
	}
	logger.DefaultLog("deleted persistent volume %s", pv.Name)
	logger.DefaultLog("successfully migrated pvc %s", pvc.Name)
	return nil
}
