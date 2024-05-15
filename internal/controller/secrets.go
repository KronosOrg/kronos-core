package kronosapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/KronosOrg/kronos-core/api/v1alpha1"
	object "github.com/KronosOrg/kronos-core/internal/controller/included-objects"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;create;update;delete;watch

func (r *KronosAppReconciler) getSecret(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func checkIfSecretWasCreatedPreviously(kronosApp *v1alpha1.KronosApp, name string) error {
	if kronosApp.Status.CreatedSecrets == nil && len(kronosApp.Status.CreatedSecrets) == 0 {
		return errors.New("there is no created secrets")
	} else {
		ok := IsInArray(kronosApp.Status.CreatedSecrets, name)
		if ok {
			err := fmt.Errorf("WARNING: %s was not found but recorded as created. Possible tamper or missing data", name)
			return err
		} else {
			return nil
		}
	}
}

func getSecretName(name string) string {
	return fmt.Sprintf("kronosapp-%s", name)
}

func (r *KronosAppReconciler) createSecret(ctx context.Context, name, namespace string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	err := r.Create(ctx, secret)
	if err != nil {
		return err
	}
	return nil
}

func (r *KronosAppReconciler) registerSecret(ctx context.Context, name string, kronosApp *v1alpha1.KronosApp) error {
	kronosappCopy := kronosApp.DeepCopy()
	kronosappCopy.Status.CreatedSecrets = append(kronosappCopy.Status.CreatedSecrets, name)
	err := r.Update(ctx, kronosappCopy)
	if err != nil {
		return err
	}
	return nil
}

func SaveObjectsData(ctx context.Context, Client client.Client, secret *corev1.Secret, kind string, resourceList []object.ResourceInt) error {
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	// Create or update the JSON object for the given kind
	dataJSON, err := json.Marshal(resourceList)
	if err != nil {
		return err
	}
	secret.Data[kind] = dataJSON
	secret.ObjectMeta.SetResourceVersion("") // Ensure update
	if err := Client.Update(ctx, secret); err != nil {
		return err
	}
	return nil
}

func CheckIfSecretContainsData(secret *corev1.Secret) error {
	if secret.Data == nil {
		err := fmt.Errorf("secret %s does not contain any data", secret.Name)
		return err
	}
	return nil
}

func CheckIfSecretContainsDataOfKind(secret *corev1.Secret, kind string) bool {
	return secret.Data[kind] != nil
}

func getSecretDatas(secret *corev1.Secret, kind string) ([]object.ResourceInt, error) {
	var resourceList []object.ResourceInt
	var err error
	switch kind {
	case "Deployment", "StatefulSet", "ReplicaSet":
		{
			var jsonData = []object.ReplicaResource{}
			if secret.Data[kind] != nil {
				err = json.Unmarshal(secret.Data[kind], &jsonData)
				resourceList = object.CastReplicaToGeneral(jsonData)
			}
		}
	case "CronJob":
		{
			var jsonData = []object.StatusResource{}
			if secret.Data[kind] != nil {
				err = json.Unmarshal(secret.Data[kind], &jsonData)
				resourceList = object.CastStatusToGeneral(jsonData)
			}
		}
	}

	if err != nil {
		return nil, err
	}
	return resourceList, nil
}

func purgeSecretData(ctx context.Context, Client client.Client, secret *corev1.Secret) error {
	secret.Data = nil
	err := Client.Update(ctx, secret)
	if err != nil {
		return err
	}
	return nil
}
