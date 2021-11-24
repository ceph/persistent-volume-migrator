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
	"context"
	"fmt"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
)

// validateResources checks if required areguments exists
func validateResources(client *k8s.Clientset, sourceSC, destinationSC, rookNS, cephClusterNS string) error {
	getOpt := v1.GetOptions{}
	ctx := context.TODO()

	_, err := client.StorageV1().StorageClasses().Get(ctx, destinationSC, getOpt)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return fmt.Errorf("destination storageClass %s doesn't exist. %w", destinationSC, err)
		}
		return fmt.Errorf("failed to get destination StorageClass %s. %w", destinationSC, err)
	}

	if sourceSC != "" {
		_, err = client.StorageV1().StorageClasses().Get(ctx, sourceSC, getOpt)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return fmt.Errorf("source storageClass %s doesn't exist. %w", sourceSC, err)
			}
			return fmt.Errorf("failed to get Source StorageClass name %s. %w", sourceSC, err)
		}
	}

	if rookNS != "" {
		_, err = client.CoreV1().Namespaces().Get(ctx, rookNS, getOpt)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return fmt.Errorf("rook namespace %s doesn't exist. %w", rookNS, err)
			}
			return fmt.Errorf("failed to get Rook namespace %s. %w", rookNS, err)
		}
	}

	if cephClusterNS != "" {
		_, err = client.CoreV1().Namespaces().Get(ctx, cephClusterNS, getOpt)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return fmt.Errorf("ceph cluster namespace %s doesn't exist. %w", cephClusterNS, err)
			}
			return fmt.Errorf("failed to get Ceph cluster namespace %s. %w", cephClusterNS, err)
		}
	}

	return nil
}
