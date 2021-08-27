package kubernetes

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
