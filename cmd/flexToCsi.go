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
	// 2. List all the PVC from the source storageclass
	pvcs, err := kubernetes.ListAllPVCWithStorageclass(client, sourceStorageClass)
	if err != nil {
		fail(fmt.Sprintf("failed to create kubernetes client %v\n", err))
	}

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
		// 4. Delete the PVC object
		err = kubernetes.DeletePVC(client, &p)
		if err != nil {
			// TODO have an option to revert back if some Delete fails?
			fmt.Printf("failed to Delete PVC object %s \n", p.Name)
			continue
		}
		//5. Create new PVC with same name in destination storageclass
		csiPVC := kubernetes.GenerateCSIPVC(destinationStorageClass, &p)
		// 6. Create new CSI PVC
		csiPV, err := kubernetes.CreatePVC(client, csiPVC, 5)
		//6. Extract new volume name from CSI PV
		csiRBDImageName := kubernetes.GetCSIVolumeName(csiPV)
		if csiRBDImageName == "" {
			fail("csiRBDImageName cannot be nil in PV object")
		}
		clusterID := kubernetes.GetClusterID(csiPV)
		if clusterID == "" {
			fail("clusterID cannot be nil in PV object")
		}
		poolName := kubernetes.GetCSIPoolName(csiPV)
		if poolName == "" {
			fail("poolName cannot be nil in PV object")
		}
		// TODO add support for datapool

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
			fail(err.Error())
		}

		conn, err := rbd.NewConnection(monitor, user, key, poolName, "")
		if err != nil {
			fail(err.Error())
		}
		defer conn.Destroy()
		//7. Delete the CSI volume in ceph cluster
		err = conn.RemoveVolume(csiRBDImageName)
		if err != nil {
			fail(err.Error())
		}
		//  8. Rename old ceph volume to new CSI volume
		err = conn.RenameVolume(csiRBDImageName, rbdImageName)
		if err != nil {
			fail(err.Error())
		}
		//  9. Delete old PV object
		err = kubernetes.DeletePV(client, pv)
		if err != nil {
			fail(err.Error())
		}
	}
	fmt.Printf("successfully migrated all the PVC from FlexVolume to CSI")
}
