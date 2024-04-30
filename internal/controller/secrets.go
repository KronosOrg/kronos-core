package kronosapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gitlab.infra.wecraft.tn/wecraft/automation/ifra/kronos/api/v1alpha1"
	object "gitlab.infra.wecraft.tn/wecraft/automation/ifra/kronos/internal/controller/included-objects"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
			err := errors.New(fmt.Sprintf("WARNING: %s was not found but recorded as created. Possible tamper or missing data.", name))
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

func addEntryInExistingData(secret *corev1.Secret, kind string, resourceList []object.ResourceInt) error {
	existingData, err := getSecretDatas(secret, kind)
	if err != nil {
		return err
	}
	fmt.Println("existingData", existingData)
	var combinedData = append(resourceList, existingData...)
	fmt.Println(combinedData)
	return err
}

/*func compareSavedAndIncomingData(secret *corev1.Secret, kind string, incomingData []object.ResourceInt) (bool, error) {
	savedData := getSecretDatas(secret, kind)

}*/

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
		err := errors.New(fmt.Sprintf("secret %s does not contain any data", secret.Name))
		return err
	}
	return nil
}

func CheckIfSecretContainsDataOfKind(secret *corev1.Secret, kind string) bool {
	if secret.Data[kind] != nil {
		return true
	}
	return false
}

/*func getSecretDataOfKind(secret *corev1.Secret, kind string) (object.ResourceInt, error) {
	var jsonData object.ResourceInt
	var err error
	switch kind {
	case "Deployment", "StatefulSet":
		{
			jsonData = object.ReplicaResource{}
			err = json.Unmarshal(secret.Data[kind], &jsonData)
		}
	case "CronJob":
		{
			jsonData = object.StatusResource{}
			err = json.Unmarshal(secret.Data[kind], &jsonData)
		}
	}

	if err != nil {
		return nil, err
	}
	return jsonData, nil
}*/

func getSecretDatas(secret *corev1.Secret, kind string) ([]object.ResourceInt, error) {
	var resourceList []object.ResourceInt
	var err error
	switch kind {
	case "Deployment", "StatefulSet":
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
