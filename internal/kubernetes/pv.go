package kubernetes

import (
	"context"
	"fmt"
	"time"

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
		fmt.Printf("waiting for PV %s in state %s to be deleted (%d seconds elapsed) \n", pvToDelete.Name, pvToDelete.Status.String(), int(time.Since(start).Seconds()))

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

func GetFlexVolumeName(pv *corev1.PersistentVolume) string {
	// Rook creates rbd image with PV name
	return pv.Name
}

func GetCSIVolumeName(pv *corev1.PersistentVolume) string {
	// CSI created rbd image name
	return pv.Spec.CSI.VolumeAttributes["imageName"]
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
	fmt.Printf("Waiting up to %v for PersistentVolume %s to have phase %s \n", timeout, pvName, phase)
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		pv, err := c.CoreV1().PersistentVolumes().Get(context.TODO(), pvName, v1.GetOptions{})
		if err != nil {
			fmt.Printf("Get persistent volume %s in failed, ignoring for %v: %v \n", pvName, poll, err)
			continue
		}
		if pv.Status.Phase == phase {
			fmt.Printf("PersistentVolume %s found and phase=%s (%v)\n", pvName, phase, time.Since(start))
			return nil
		}
		fmt.Printf("PersistentVolume %s found but phase is %s instead of %s.\n", pvName, pv.Status.Phase, phase)
	}
	return fmt.Errorf("PersistentVolume %s not in phase %s within %v", pvName, phase, timeout)
}
