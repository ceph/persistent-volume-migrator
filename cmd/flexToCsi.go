/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

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
	"os"
	"strings"

	"github.com/Madhu-1/ceph-pvc-migration-to-csi/internal/ceph/rbd"
	"github.com/Madhu-1/ceph-pvc-migration-to-csi/internal/kubernetes"
	"github.com/spf13/cobra"
)

// flexToCSICmd represents the flexToCSI command
var flexToCSICmd = &cobra.Command{
	Use:   "flexToCSI",
	Short: "command to migrate flex volume to CSI volume",
	Long: `This command does series of operation mentioned below to migrate flex volume to CSI:

1. List all the PVC from the source storageclass
2. Change Reclaim policy from Delete to Reclaim
3. Retrive old ceph volume name from PV
4. Delete the PVC object
5. Create new PVC with same name in destination storageclass
6. Extract new volume name from CSI PV
7. Delete the CSI volume in ceph cluster
8. Rename old ceph volume to new CSI volume
9. Delete old PV object
`,
	Run: func(cmd *cobra.Command, args []string) {
		migrateFlexToCSI()
	},
}

func init() {
	rootCmd.AddCommand(flexToCSICmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// flexToCSICmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// flexToCSICmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func fail(msg string) {
	fmt.Printf(msg)
	os.Exit(1)
}
func migrateFlexToCSI() {
	// TODO
	/*
		add validation to check source,destination storageclass and Rook namespace
	*/

	/*

	 */
	// 1. Create Kubernetes Client
	client, err := kubernetes.NewClient(kubeConfig)
	if err != nil {
		fail(fmt.Sprintf("failed to create kubernetes client %v\n", err))
	}
	fmt.Println("Yug Created Kubernetes Client")
	fmt.Println()
	// 2. List all the PVC from the source storageclass
	pvcs, err := kubernetes.ListAllPVCWithStorageclass(client, sourceStorageClass)
	if err != nil {
		fail(fmt.Sprintf("failed to list PVCs from the storageclass %v\n", err))
	}
	fmt.Println("Yug PVCs listed", pvcs)
	fmt.Println()
	fmt.Println()
	if pvcs == nil || len(*pvcs) == 0 {
		fail(fmt.Sprintf("no PVC found with storageclass %v\n", sourceStorageClass))
	}
	fmt.Println()
	fmt.Println()
	//  3. Change Reclaim policy from Delete to Reclaim
	for _, p := range *pvcs {
		fmt.Println("New PVC", p)
		fmt.Println()
		fmt.Println()
		pv, err := kubernetes.GetPV(client, p.Spec.VolumeName)
		if err != nil {
			fmt.Printf("failed to get PV object with name %s \n", p.Spec.VolumeName)
			continue
		}
		fmt.Println("Yug PV found", pv)
		fmt.Println()
		err = kubernetes.UpdateReclaimPolicy(client, pv)
		if err != nil {
			fmt.Printf("failed to update ReclaimPolicy for PV object %s \n", p.Spec.VolumeName)
			continue
		}
		fmt.Println()
		// 3. Retrive old ceph volume name from PV
		rbdImageName := kubernetes.GetFlexVolumeName(pv)
		if rbdImageName == "" {
			fail("rbdImageName cannot be nil in Flex PV object")
		}
		fmt.Println("Yug rbdImageName found", rbdImageName)
		fmt.Println()
		// 4. Delete the PVC object
		err = kubernetes.DeletePVC(client, &p)
		if err != nil {
			fmt.Println()
			// TODO have an option to revert back if some Delete fails?
			fmt.Printf("failed to Delete PVC object %s \n", p.Name)
			continue
		}
		fmt.Println("Yug rbdImageName found", rbdImageName)
		fmt.Println()
		//5. Create new PVC with same name in destination storageclass
		csiPVC := kubernetes.GenerateCSIPVC(destinationStorageClass, &p)
		fmt.Println("Yug structure of new PVC with same name created in CSI", csiPVC)
		fmt.Println()
		// 6. Create new CSI PVC
		csiPV, err := kubernetes.CreatePVC(client, csiPVC, 5)
		if err != nil {
			fmt.Println("failed to Create CSI PVC object", p.Name, err)
			continue
		}
		fmt.Println("Yug new PVC with same name created in CSI. Persistent Volume: ", csiPV)
		fmt.Println()
		//6. Extract new volume name from CSI PV
		csiRBDImageName := kubernetes.GetCSIVolumeName(csiPV)
		if csiRBDImageName == "" {
			fail("csiRBDImageName cannot be nil in PV object")
		}
		fmt.Println("Yug new volume name from CSI PV: ", csiRBDImageName)
		clusterID := kubernetes.GetClusterID(csiPV)
		if clusterID == "" {
			fail("clusterID cannot be nil in PV object")
		}
		fmt.Println("Yug clusterID of csiPV ", clusterID)
		poolName := kubernetes.GetCSIPoolName(csiPV)
		if poolName == "" {
			fail("poolName cannot be nil in PV object")
		}
		fmt.Println("Yug Poolname of csiPV ", poolName)
		// TODO add support for datapool

		csiConfig, err := kubernetes.GetCSIConfiguration(client, rookNamespace)
		if err != nil {
			fmt.Printf("fail to get configmap %v\n", err)
			continue
		}
		fmt.Println("Yug csiConfig ", csiConfig)
		fmt.Println()
		var monitor string
		for _, c := range csiConfig {
			if c.ClusterID == clusterID {
				monitor = strings.Join(c.Monitors, ",")
			}
		}
		fmt.Println("mon")
		if monitor == "" {
			fail(fmt.Sprintf("failed to get monitor information %v \n", csiConfig))
		}
		fmt.Println("YUG mon found", monitor)
		user, key, err := kubernetes.GetRBDUserAndKeyFromSecret(client, cephClusterNamespace)
		if err != nil {
			fmt.Println("YUG err in GetRBDUserAndKeyFromSecret", err)
			fmt.Println()
			fail(err.Error())
		}
		fmt.Println()
		fmt.Println("YUG NO err in GetRBDUserAndKeyFromSecret", err)
		fmt.Println()
		conn, err := rbd.NewConnection(monitor, user, key, poolName, "")
		if err != nil {
			fmt.Println("YUG err in NewConnection", err)
			fmt.Println()
			fail(err.Error())
		}
		fmt.Println("YUG NO err in NewConnection", err)
		defer conn.Destroy()
		//7. Delete the CSI volume in ceph cluster
		err = conn.RemoveVolume(csiRBDImageName)
		if err != nil {
			fmt.Println("YUG err in RemoveVolume", err)
			fmt.Println()
			fail(err.Error())
		}
		fmt.Println()
		fmt.Println("YUG NO err in RemoveVolume", err)
		//  8. Rename old ceph volume to new CSI volume
		err = conn.RenameVolume(csiRBDImageName, rbdImageName)
		if err != nil {
			fmt.Println("YUG err in RenameVolume", err)
			fmt.Println()
			fail(err.Error())
		}
		fmt.Println()
		fmt.Println("YUG NO err in RenameVolume", err)
		//  9. Delete old PV object
		err = kubernetes.DeletePV(client, pv)
		if err != nil {
			fmt.Println("YUG err in DeletePV", err)
			fmt.Println()
			fail(err.Error())
		}
		fmt.Println()
		fmt.Println("YUG NO err in DeletePV", err)
		fmt.Println()
	}
	fmt.Println()
	fmt.Println("YUG Done with all PVC")
	fmt.Println()
	fmt.Printf("successfully migrated all the PVC from FlexVolume to CSI")
}
