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
	"persistent-volume-migrator/pkg/migration"

	"github.com/spf13/cobra"
)

var (
	kubeConfig              string
	sourceStorageClass      string
	destinationStorageClass string
	rookNamespace           string
	cephClusterNamespace    string
	pvcName                 string
	pvcNamespace            string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Tool to migrate kubernetes ceph in-tree and flex volume to CSI",
	Long:  `Tool to migrate kubernetes ceph in-tree and flex volume to CSI`,

	// 1. List all the PVC from the source storageclass
	// 2. Change Reclaim policy from Delete to Reclaim
	// 3. Retrive old ceph volume name from PV
	// 4. Delete the PVC object
	// 5. Create new PVC with same name in destination storageclass
	// 6. Extract new volume name from CSI PV
	// 7. Delete the CSI volume in ceph cluster
	// 8. Rename old ceph volume to new CSI volume
	// 9. Delete old PV object
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := migration.MigrateToCSI(kubeConfig, sourceStorageClass,
			destinationStorageClass, rookNamespace, cephClusterNamespace, pvcName, pvcNamespace); err != nil {
			return err
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", "", "kubernetes config path")
	rootCmd.PersistentFlags().StringVar(&sourceStorageClass, "sourcestorageclass", "", "source storageclass from which all PVC need to be migrated")
	rootCmd.PersistentFlags().StringVar(&destinationStorageClass, "destinationstorageclass", "", "destination storageclass (CSI storageclass) to which all PVC need to be migrated")
	rootCmd.PersistentFlags().StringVar(&rookNamespace, "rook-namespace", "rook-ceph", "Kubernetes namespace where rook operator is running")
	rootCmd.PersistentFlags().StringVar(&cephClusterNamespace, "ceph-cluster-namespace", "rook-ceph", "Kubernetes namespace where ceph cluster is created")
	rootCmd.PersistentFlags().StringVar(&pvcName, "pvc", "", "Name of the specific pvc you want to migrate")
	rootCmd.PersistentFlags().StringVar(&pvcNamespace, "pvc-namespace", "", "Namespace of the specific pvc you want to migrate")
}
