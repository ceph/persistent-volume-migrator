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
	"github.com/spf13/cobra"
)

var (
	kubeConfig              string
	sourceStorageClass      string
	destinationStorageClass string
	rookNamespace           string
	cephClusterNamespace    string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Tool to migrate kubernetes ceph in-tree and flex volume to CSI",
	Long:  `Tool to migrate kubernetes ceph in-tree and flex volume to CSI`,
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
	rootCmd.PersistentFlags().StringVar(&sourceStorageClass, "sourcestoraageclass", "", "source storageclass from which all PVC need to be migrated")
	rootCmd.PersistentFlags().StringVar(&destinationStorageClass, "destinationstorageclass", "", "destination storageclass (CSI storageclass) to which all PVC need to be migrated")
	rootCmd.PersistentFlags().StringVar(&rookNamespace, "rook-namespace", "rook-ceph", "Kubernetes namespace where rook operator is running")
	rootCmd.PersistentFlags().StringVar(&cephClusterNamespace, "ceph-clsuter-namespace", "rook-ceph", "Kubernetes namespace where ceph cluster is created")
}
