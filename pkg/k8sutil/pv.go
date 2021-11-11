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

package k8sutil

import (
	"context"
	"fmt"
	"time"

	logger "persistent-volume-migrator/pkg/log"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	k8s "k8s.io/client-go/kubernetes"
)

var poll = 2 * time.Second

func GetPV(client *k8s.Clientset, pvName string) (*corev1.PersistentVolume, error) {
	getOpt := v1.GetOptions{}
	ctx := context.TODO()

	pv, err := client.CoreV1().PersistentVolumes().Get(ctx, pvName, getOpt)
	if err != nil {
		return nil, err
	}
	return pv, nil
}

func DeletePV(client *k8s.Clientset, pv *corev1.PersistentVolume) error {
	err := client.CoreV1().PersistentVolumes().Delete(context.TODO(), pv.Name, v1.DeleteOptions{})
	if err != nil {
		return err
	}

	timeout := time.Duration(1) * time.Minute
	start := time.Now()
	pvToDelete := pv
	return wait.PollImmediate(poll, timeout, func() (bool, error) {
		// Check that the PV is deleted.
		logger.DefaultLog("waiting for PV %s in state %s to be deleted (%d seconds elapsed) \n", pvToDelete.Name, pvToDelete.Status.String(), int(time.Since(start).Seconds()))

		pvToDelete, err = client.CoreV1().PersistentVolumes().Get(context.TODO(), pvToDelete.Name, v1.GetOptions{})
		if err == nil {
			return false, nil
		}

		if !apierrs.IsNotFound(err) {
			return false, fmt.Errorf("delete PV %v failed with error other than \"not found\": %w", pv.Name, err)
		}

		return true, nil
	})
}

func UpdateReclaimPolicy(client *k8s.Clientset, pv *corev1.PersistentVolume) error {
	pv.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimRetain
	updateOpt := v1.UpdateOptions{}
	ctx := context.TODO()
	_, err := client.CoreV1().PersistentVolumes().Update(ctx, pv, updateOpt)
	return err
}

func GetVolumeName(pv *corev1.PersistentVolume) string {
	// Rook creates rbd image with PV name
	return pv.Name
}

func WaitForRBDImage(pv *corev1.PersistentVolume) string {
	retry := 0
	maxRetry := 15
	for retry < maxRetry {
		imageName := pv.Spec.CSI.VolumeAttributes["imageName"]
		if imageName != "" {
			// CSI created rbd image name
			return imageName
		}
		logger.DefaultLog("Waiting for PersistentVolume %q to be Created, attempt: %d", pv.Name, retry)
		time.Sleep(time.Second * 2)
		retry++
	}

	return ""
}

func GetCSIPoolName(pv *corev1.PersistentVolume) string {
	// Pool in which RBD image is created
	return pv.Spec.CSI.VolumeAttributes["pool"]
}

func GetClusterID(pv *corev1.PersistentVolume) string {
	// clusterID which denotes the cluster namespace where image is created
	return pv.Spec.CSI.VolumeAttributes["clusterID"]
}

// WaitForPersistentVolumePhase waits for a PersistentVolume to be in a specific phase or until timeout occurs, whichever comes first.
func WaitForPersistentVolumePhase(c *k8s.Clientset, phase corev1.PersistentVolumePhase, pvName string, poll, timeout time.Duration) error {
	logger.DefaultLog("Waiting up to %v for PersistentVolume %s to have phase %s \n", timeout, pvName, phase)
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		pv, err := c.CoreV1().PersistentVolumes().Get(context.TODO(), pvName, v1.GetOptions{})
		if err != nil {
			logger.DefaultLog("Get persistent volume %s in failed, ignoring for %v: %v \n", pvName, poll, err)
			continue
		}
		if pv.Status.Phase == phase {
			logger.DefaultLog("PersistentVolume %s found and phase=%s (%v)\n", pvName, phase, time.Since(start))
			return nil
		}
		logger.DefaultLog("PersistentVolume %s found but phase is %s instead of %s.\n", pvName, pv.Status.Phase, phase)
	}
	return fmt.Errorf("PersistentVolume %s not in phase %s within %v", pvName, phase, timeout)
}
