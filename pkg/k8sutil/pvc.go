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
	"k8s.io/client-go/kubernetes"
	k8s "k8s.io/client-go/kubernetes"
)

const (
	storageClassBetaAnnotationKey = "volume.beta.kubernetes.io/storage-class"
)

func ListAllPVCWithStorageclass(client *k8s.Clientset, scName string) (*[]corev1.PersistentVolumeClaim, error) {
	pl := &[]corev1.PersistentVolumeClaim{}
	listOpt := v1.ListOptions{}
	ctx := context.TODO()
	ns, err := client.CoreV1().Namespaces().List(ctx, listOpt)
	if err != nil {
		return nil, err
	}
	for _, n := range ns.Items {
		pvc, err := client.CoreV1().PersistentVolumeClaims(n.Name).List(ctx, listOpt)
		if err != nil {
			continue
		}
		for _, p := range pvc.Items {
			sc := p.Spec.StorageClassName
			if sc == nil {
				if val, ok := p.Annotations[storageClassBetaAnnotationKey]; !ok {
					continue
				} else {
					sc = &val
				}
			}
			if *sc == scName {
				*pl = append(*pl, p)
			}
		}
	}
	return pl, nil
}

func DeletePVC(client *k8s.Clientset, pvc *corev1.PersistentVolumeClaim) error {
	err := client.CoreV1().PersistentVolumeClaims("default").Delete(context.TODO(), pvc.Name, v1.DeleteOptions{})
	if err != nil {
		return err
	}

	timeout := time.Duration(1) * time.Minute
	start := time.Now()

	pvcToDelete := pvc
	return wait.PollImmediate(poll, timeout, func() (bool, error) {
		// Check that the PVC is deleted.
		logger.DefaultLog("waiting for PVC %s in state %s to be deleted (%d seconds elapsed) \n", pvcToDelete.Name, pvcToDelete.Status.String(), int(time.Since(start).Seconds()))
		pvcToDelete, err = client.CoreV1().PersistentVolumeClaims(pvcToDelete.Namespace).Get(context.TODO(), pvcToDelete.Name, v1.GetOptions{})
		if err == nil {
			if pvcToDelete.Status.Phase == "" {
				// this is unexpected, an empty Phase is not defined
				fmt.Printf("PVC %s is in a weird state: %s", pvcToDelete.Name, pvcToDelete.String())
			}
			return false, nil
		}
		if !apierrs.IsNotFound(err) {
			return false, fmt.Errorf("get on deleted PVC %v failed with error other than \"not found\": %w", pvc.Name, err)
		}

		return true, nil
	})
}

func GenerateCSIPVC(storageclass string, pvc *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
	// csiPVC := &corev1.PersistentVolumeClaim{}
	csiPVC := pvc.DeepCopy()
	csiPVC.ResourceVersion = ""
	csiPVC.Spec.VolumeName = ""
	csiPVC.ObjectMeta.Annotations = make(map[string]string)
	csiPVC.Status = corev1.PersistentVolumeClaimStatus{}
	csiPVC.Spec.StorageClassName = &storageclass

	return csiPVC
}

func CreatePVC(c *k8s.Clientset, pvc *corev1.PersistentVolumeClaim, t int) (*corev1.PersistentVolume, error) {
	timeout := time.Duration(t) * time.Minute
	pv := &corev1.PersistentVolume{}
	var err error
	_, err = c.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(context.TODO(), pvc, v1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	name := pvc.Name
	start := time.Now()
	fmt.Printf("Waiting up to %v to be in Bound state\n", pvc)
	err = wait.PollImmediate(poll, timeout, func() (bool, error) {
		logger.DefaultLog("waiting for PVC %s (%d seconds elapsed) \n", pvc.Name, int(time.Since(start).Seconds()))
		pvc, err = c.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(context.TODO(), name, v1.GetOptions{})
		if err != nil {
			fmt.Printf("Error getting pvc in namespace: '%s': %v\n", pvc.Namespace, err)
			// TODO check need to check retry error
			if apierrs.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		if pvc.Spec.VolumeName == "" {
			return false, nil
		}
		pv, err = c.CoreV1().PersistentVolumes().Get(context.TODO(), pvc.Spec.VolumeName, v1.GetOptions{})
		if err != nil {
			return false, err
		}
		if apierrs.IsNotFound(err) {
			return false, nil
		}

		err = WaitOnPVandPVC(c, pvc.Namespace, pv, pvc)
		if err != nil {
			return false, nil
		}
		return true, nil
	})

	if err == nil {
		pvc, err = c.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(context.TODO(), name, v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		pv, err = c.CoreV1().PersistentVolumes().Get(context.TODO(), pvc.Spec.VolumeName, v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return pv, err
	}

	return nil, err
}

// WaitOnPVandPVC waits for the pv and pvc to bind to each other.
func WaitOnPVandPVC(c *kubernetes.Clientset, ns string, pv *corev1.PersistentVolume, pvc *corev1.PersistentVolumeClaim) error {
	// Wait for newly created PVC to bind to the PV
	logger.DefaultLog("Waiting for PV %q to bind to PVC %q", pv.Name, pvc.Name)
	err := WaitForPersistentVolumeClaimPhase(corev1.ClaimBound, c, ns, pvc.Name, poll, 30000000)
	if err != nil {
		return fmt.Errorf("PVC %q did not become Bound: %v", pvc.Name, err)
	}

	// Wait for PersistentVolume.Status.Phase to be Bound, which it should be
	// since the PVC is already bound.
	err = WaitForPersistentVolumePhase(c, corev1.VolumeBound, pv.Name, poll, 3000000000)
	if err != nil {
		return fmt.Errorf("PV %q did not become Bound: %v", pv.Name, err)
	}

	// Re-get the pv and pvc objects
	pv, err = c.CoreV1().PersistentVolumes().Get(context.TODO(), pv.Name, v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("PV Get API error: %v", err)
	}
	pvc, err = c.CoreV1().PersistentVolumeClaims(ns).Get(context.TODO(), pvc.Name, v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("PVC Get API error: %v", err)
	}

	// The pv and pvc are both bound, but to each other?
	// Check that the PersistentVolume.ClaimRef matches the PVC
	if pv.Spec.ClaimRef == nil {
		return fmt.Errorf("PV %q ClaimRef is nil", pv.Name)
	}
	if pv.Spec.ClaimRef.Name != pvc.Name {
		return fmt.Errorf("PV %q ClaimRef's name (%q) should be %q", pv.Name, pv.Spec.ClaimRef.Name, pvc.Name)
	}
	if pvc.Spec.VolumeName != pv.Name {
		return fmt.Errorf("PVC %q VolumeName (%q) should be %q", pvc.Name, pvc.Spec.VolumeName, pv.Name)
	}
	if pv.Spec.ClaimRef.UID != pvc.UID {
		return fmt.Errorf("PV %q ClaimRef's UID (%q) should be %q", pv.Name, pv.Spec.ClaimRef.UID, pvc.UID)
	}
	return nil
}

// WaitForPersistentVolumeClaimPhase waits for a PersistentVolumeClaim to be in a specific phase or until timeout occurs, whichever comes first.
func WaitForPersistentVolumeClaimPhase(phase corev1.PersistentVolumeClaimPhase, c *kubernetes.Clientset, ns string, pvcName string, poll, timeout time.Duration) error {
	return WaitForPersistentVolumeClaimsPhase(phase, c, ns, []string{pvcName}, poll, timeout, true)
}

// WaitForPersistentVolumeClaimsPhase waits for any (if matchAny is true) or all (if matchAny is false) PersistentVolumeClaims
// to be in a specific phase or until timeout occurs, whichever comes first.
func WaitForPersistentVolumeClaimsPhase(phase corev1.PersistentVolumeClaimPhase, c *kubernetes.Clientset, ns string, pvcNames []string, poll, timeout time.Duration, matchAny bool) error {

	if len(pvcNames) == 0 {
		return fmt.Errorf("Incorrect parameter: Need at least one PVC to track. Found 0")
	}
	fmt.Printf("Waiting up to %v for PersistentVolumeClaims %v to have phase %s\n", timeout, pvcNames, phase)
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		phaseFoundInAllClaims := true
		for _, pvcName := range pvcNames {
			pvc, err := c.CoreV1().PersistentVolumeClaims(ns).Get(context.TODO(), pvcName, v1.GetOptions{})
			if err != nil {
				fmt.Printf("Failed to get claim %q, retrying in %v. Error: %v\n", pvcName, poll, err)
				phaseFoundInAllClaims = false
				break
			}
			if pvc.Status.Phase == phase {
				fmt.Printf("PersistentVolumeClaim %s found and phase=%s (%v) \n", pvcName, phase, time.Since(start))
				if matchAny {
					return nil
				}
			} else {
				fmt.Printf("PersistentVolumeClaim %s found but phase is %s instead of %s.\n", pvcName, pvc.Status.Phase, phase)
				phaseFoundInAllClaims = false
			}
		}
		if phaseFoundInAllClaims {
			return nil
		}
	}
	return fmt.Errorf("PersistentVolumeClaims %v not all in phase %s within %v", pvcNames, phase, timeout)
}
