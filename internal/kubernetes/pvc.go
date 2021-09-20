package kubernetes

import (
	"context"
	"fmt"
	"time"

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
	fmt.Println(1)
	pl := &[]corev1.PersistentVolumeClaim{}
	fmt.Println(2)
	listOpt := v1.ListOptions{}
	fmt.Println(3)
	ctx := context.TODO()
	fmt.Println(4)
	ns, err := client.CoreV1().Namespaces().List(ctx, listOpt)
	fmt.Println("YUG", ns)
	if err != nil {
		return nil, err
	}
	fmt.Println(5)
	for _, n := range ns.Items {
		pvc, err := client.CoreV1().PersistentVolumeClaims(n.Name).List(ctx, listOpt)

		fmt.Println("YUG& subham", pvc)
		if err != nil {
			continue
		}
		for _, p := range pvc.Items {
			sc := p.Spec.StorageClassName
			fmt.Println("iterating sc", *sc)
			if sc == nil {
				if val, ok := p.Annotations[storageClassBetaAnnotationKey]; !ok {
					continue
				} else {
					sc = &val
				}
			}
			fmt.Println("sc name", *p.Spec.StorageClassName)
			fmt.Println("var sc name", scName)
			if *p.Spec.StorageClassName == scName {
				*pl = append(*pl, p)
			}
		}
	}
	fmt.Println(7)
	fmt.Println(pl)
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
		fmt.Printf("waiting for PVC %s in state %s to be deleted (%d seconds elapsed) \n", pvcToDelete.Name, pvcToDelete.Status.String(), int(time.Since(start).Seconds()))
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
	fmt.Println()
	fmt.Println("YUG inside GenerateCSIPVC , csiPVC: ", csiPVC)
	fmt.Println()

	return csiPVC
}

func CreatePVC(c *k8s.Clientset, pvc *corev1.PersistentVolumeClaim, t int) (*corev1.PersistentVolume, error) {
	fmt.Println(10)
	timeout := time.Duration(t) * time.Minute
	fmt.Println(11)
	pv := &corev1.PersistentVolume{}
	fmt.Println(12)
	var err error
	_, err = c.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(context.TODO(), pvc, v1.CreateOptions{})
	if err != nil {
		fmt.Println(13)
		return nil, err
	}
	fmt.Println(14)
	name := pvc.Name
	start := time.Now()
	fmt.Printf("Waiting up to %v to be in Bound state\n", pvc)
	fmt.Println(15)
	err = wait.PollImmediate(poll, timeout, func() (bool, error) {
		fmt.Println(16)
		fmt.Printf("waiting for PVC %s (%d seconds elapsed) \n", pvc.Name, int(time.Since(start).Seconds()))
		pvc, err = c.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(context.TODO(), name, v1.GetOptions{})
		if err != nil {
			fmt.Println(17)
			fmt.Printf("Error getting pvc in namespace: '%s': %v\n", pvc.Namespace, err)
			// TODO check need to check retry error

			if apierrs.IsNotFound(err) {
				fmt.Println(18)
				return false, nil
			}
			fmt.Println(19)
			return false, err
		}
		fmt.Println(20)
		if pvc.Spec.VolumeName == "" {
			fmt.Println(21)
			return false, nil
		}
		fmt.Println(22)

		pv, err = c.CoreV1().PersistentVolumes().Get(context.TODO(), pvc.Spec.VolumeName, v1.GetOptions{})
		fmt.Println(23)
		if err != nil {
			fmt.Println(24)
			return false, err
		}
		fmt.Println(25)
		if apierrs.IsNotFound(err) {
			fmt.Println(26)
			return false, nil
		}
		fmt.Println(27)
		err = WaitOnPVandPVC(c, pvc.Namespace, pv, pvc)
		fmt.Println(28)
		if err != nil {
			fmt.Println(29, err)
			return false, nil
		}
		fmt.Println(30)
		return true, nil
	})
	fmt.Println(31)
	if err == nil {
		fmt.Println(31)
		pvc, err = c.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(context.TODO(), name, v1.GetOptions{})
		if err != nil {
			fmt.Println(32)
			return nil, err
		}
		fmt.Println(33)
		pv, err = c.CoreV1().PersistentVolumes().Get(context.TODO(), pvc.Spec.VolumeName, v1.GetOptions{})
		fmt.Println(34)
		if err != nil {
			fmt.Println(35)
			return nil, err
		}
		fmt.Println(36)
		return pv, err
	}
	fmt.Println(37)
	return nil, err
}

// WaitOnPVandPVC waits for the pv and pvc to bind to each other.
func WaitOnPVandPVC(c *kubernetes.Clientset, ns string, pv *corev1.PersistentVolume, pvc *corev1.PersistentVolumeClaim) error {
	// Wait for newly created PVC to bind to the PV
	fmt.Printf("Waiting for PV %v to bind to PVC %v", pv.Name, pvc.Name)
	fmt.Println("TEST parameters in WaitOnPVandPVC", corev1.ClaimBound, c, ns, pvc.Name, poll, 30000000)
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
			fmt.Println("TEST NEW PVC", pvcName)
			pvc, err := c.CoreV1().PersistentVolumeClaims(ns).Get(context.TODO(), pvcName, v1.GetOptions{})
			if err != nil {
				fmt.Printf("Failed to get claim %q, retrying in %v. Error: %v\n", pvcName, poll, err)
				phaseFoundInAllClaims = false
				break
			}
			fmt.Println("TEST NEW PVC PHASE ", pvc.Status.Phase)
			fmt.Println("TEST PASSED PVC PHASE ", phase)
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
