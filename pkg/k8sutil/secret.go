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

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetRBDUserAndKeyFromSecret(client *kubernetes.Clientset, namespace string) (string, string, error) {
	name := "rook-csi-rbd-provisioner"
	secret, err := client.CoreV1().Secrets(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	if secret == nil {
		return "", "", fmt.Errorf("secret data is nil for %v in %v namespace", name, namespace)
	}
	if _, ok := secret.Data["userID"]; !ok {
		return "", "", fmt.Errorf("userID is empty for %v in %v namespace", name, namespace)
	}
	if _, ok := secret.Data["userKey"]; !ok {
		return "", "", fmt.Errorf("userKey is empty for %v in %v namespace", name, namespace)
	}
	user := string(secret.Data["userID"])
	key := string(secret.Data["userKey"])
	return user, key, nil
}
