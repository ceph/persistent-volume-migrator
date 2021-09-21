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

	"persistent-volume-migrator/internal/ceph/rbd"
	"persistent-volume-migrator/internal/kubernetes"

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
	// 2. List all the PVC from the source storageclass
	pvcs, err := kubernetes.ListAllPVCWithStorageclass(client, sourceStorageClass)
	if err != nil {
		fail(fmt.Sprintf("failed to list PVCs from the storageclass %v\n", err))
	}
	fmt.Printf("listing all pvc from sourceStorageClass %s \n %v ", sourceStorageClass, pvcs)
	if pvcs == nil || len(*pvcs) == 0 {
		fail(fmt.Sprintf("no PVC found with storageclass %v\n", sourceStorageClass))
	}

	//  3. Change Reclaim policy from Delete to Reclaim
	for _, p := range *pvcs {
		pv, err := kubernetes.GetPV(client, p.Spec.VolumeName)
		if err != nil {
			fmt.Printf("failed to get PV object with name %s \n", p.Spec.VolumeName)
			continue
		}
		fmt.Printf("PV found %v ", pv)
		err = kubernetes.UpdateReclaimPolicy(client, pv)
		if err != nil {
			fmt.Printf("failed to update ReclaimPolicy for PV object %s \n", p.Spec.VolumeName)
			continue
		}
		// 3. Retrive old ceph volume name from PV
		rbdImageName := kubernetes.GetFlexVolumeName(pv)
		if rbdImageName == "" {
			fail("rbdImageName cannot be nil in Flex PV object")
		}
		fmt.Printf("flex rbdImageName name %v ", rbdImageName)

		// 4. Delete the PVC object
		err = kubernetes.DeletePVC(client, &p)
		if err != nil {
			// TODO have an option to revert back if some Delete fails?
			fmt.Printf("failed to Delete PVC object %s \n", p.Name)
			continue
		}
		//5. Create new PVC with same name in destination storageclass
		csiPVC := kubernetes.GenerateCSIPVC(destinationStorageClass, &p)
		fmt.Printf("structure of the generated PVC in destination Storageclass \n %v ", csiPVC)

		// 6. Create new CSI PVC
		csiPV, err := kubernetes.CreatePVC(client, csiPVC, 5)
		if err != nil {
			fmt.Println("failed to Create CSI PVC object", p.Name, err)
			continue
		}
		fmt.Printf("new PVC with same name created in CSI. Persistent Volume: %v ", csiPV)
		//6. Extract new volume name from CSI PV
		csiRBDImageName := kubernetes.GetCSIVolumeName(csiPV)
		if csiRBDImageName == "" {
			fail("csiRBDImageName cannot be nil in PV object")
		}
		fmt.Printf("csi new volume name: %v ", csiRBDImageName)
		clusterID := kubernetes.GetClusterID(csiPV)
		if clusterID == "" {
			fail("clusterID cannot be nil in PV object")
		}
		poolName := kubernetes.GetCSIPoolName(csiPV)
		if poolName == "" {
			fail("poolName cannot be nil in PV object")
		}

		csiConfig, err := kubernetes.GetCSIConfiguration(client, rookNamespace)
		if err != nil {
			fmt.Printf("fail to get configmap %v\n", err)
			continue
		}

		var monitor string
		for _, c := range csiConfig {
			if c.ClusterID == clusterID {
				monitor = strings.Join(c.Monitors, ",")
			}
		}
		if monitor == "" {
			fail(fmt.Sprintf("failed to get monitor information %v \n", csiConfig))
		}
		user, key, err := kubernetes.GetRBDUserAndKeyFromSecret(client, cephClusterNamespace)
		if err != nil {
			fmt.Println("err in GetRBDUserAndKeyFromSecret", err)
			fail(err.Error())
		}
		conn, err := rbd.NewConnection(monitor, user, key, poolName, "")
		if err != nil {
			fmt.Println("err in NewConnection: ", err)
			fail(err.Error())
		}

		defer conn.Destroy()
		// 7. Delete the CSI volume in ceph cluster
		err = conn.RemoveVolumeAdmin(poolName, csiRBDImageName)
		if err != nil {
			fmt.Println("err in RemoveVolume", err)
			fail(err.Error())
		}

		//  8. Rename old ceph volume to new CSI volume
		err = conn.RenameVolume(csiRBDImageName, rbdImageName)
		if err != nil {
			fmt.Println("err in RenameVolume", err)
			fail(err.Error())
		}

		//  9. Delete old PV object
		err = kubernetes.DeletePV(client, pv)
		if err != nil {
			fmt.Println("err in DeletePV: ", err)
			fail(err.Error())
		}
	}
	fmt.Println()
	fmt.Printf("successfully migrated all the PVC from FlexVolume to CSI")
}
