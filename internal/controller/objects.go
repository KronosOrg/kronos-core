package kronosapp

import (
	"context"
	"errors"
	"fmt"
	"gitlab.infra.wecraft.tn/wecraft/automation/ifra/kronos/api/v1alpha1"
	object "gitlab.infra.wecraft.tn/wecraft/automation/ifra/kronos/internal/controller/included-objects"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets;replicasets,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;update

type ObjectList struct {
	Deployments  *appsv1.DeploymentList
	StatefulSets *appsv1.StatefulSetList
	ReplicaSets  *appsv1.ReplicaSetList
	CronJobs     *batchv1.CronJobList
}

func (objectList *ObjectList) GetObjectsNames() map[string][]string {
	objectNames := make(map[string][]string)
	if objectList.Deployments != nil {
		objectNames["Deployments"] = object.GetDeploymentListNames(objectList.Deployments)
	}
	if objectList.StatefulSets != nil {
		objectNames["StatefulSets"] = object.GetStatefulsetListNames(objectList.StatefulSets)
	}
	if objectList.CronJobs != nil {
		objectNames["CronJobs"] = object.GetCronjobListNames(objectList.CronJobs)
	}
	if objectList.ReplicaSets != nil {
		objectNames["ReplicaSets"] = object.GetReplicaSetListNames(objectList.ReplicaSets)
	}
	return objectNames
}

func (objectList *ObjectList) GetObjectsCount() map[string]int {
	objectCount := make(map[string]int)
	if objectList.Deployments != nil {
		objectCount["Deployments"] = len(objectList.Deployments.Items)
	}
	if objectList.StatefulSets != nil {
		objectCount["StatefulSets"] = len(objectList.StatefulSets.Items)
	}
	if objectList.CronJobs != nil {
		objectCount["CronJobs"] = len(objectList.CronJobs.Items)
	}
	if objectList.ReplicaSets != nil {
		objectCount["ReplicaSets"] = len(objectList.ReplicaSets.Items)
	}
	return objectCount
}

func (objectList *ObjectList) GetObjectsTotalCount() int {
	var count int
	if objectList.Deployments != nil {
		count += len(objectList.Deployments.Items)
	}
	if objectList.StatefulSets != nil {
		count += len(objectList.StatefulSets.Items)
	}
	if objectList.CronJobs != nil {
		count += len(objectList.CronJobs.Items)
	}
	if objectList.ReplicaSets != nil {
		count += len(objectList.ReplicaSets.Items)
	}
	return count
}

type APIVersionKindMap struct {
	Mapping map[string][]string
}

func NewEmptyAPIVersionKindMap() *APIVersionKindMap {
	return &APIVersionKindMap{
		Mapping: make(map[string][]string),
	}
}

func NewAPIVersionKindMap(input map[string][]string) *APIVersionKindMap {
	return &APIVersionKindMap{
		Mapping: input,
	}
}

func (m *APIVersionKindMap) Add(apiVersion, kind string) {
	if _, ok := m.Mapping[apiVersion]; !ok {
		m.Mapping[apiVersion] = []string{}
	}
	m.Mapping[apiVersion] = append(m.Mapping[apiVersion], kind)
}

func (m *APIVersionKindMap) GetKind(apiVersion string) []string {
	return m.Mapping[apiVersion]
}
func (m *APIVersionKindMap) GetAPIVersion(kind string) string {
	for apiVersion, kinds := range m.Mapping {
		for _, k := range kinds {
			if k == kind {
				return apiVersion
			}
		}
	}
	return ""
}

func (m *APIVersionKindMap) KindExists(kind string) bool {
	for _, kinds := range m.Mapping {
		for _, k := range kinds {
			if k == kind {
				return true
			}
		}
	}
	return false
}

func getSupportedObjectsApiVersionAndKind() *APIVersionKindMap {
	kindToAPIVersion := NewAPIVersionKindMap(map[string][]string{
		"apps/v1":  {"Deployment", "StatefulSet", "ReplicaSet"},
		"batch/v1": {"CronJob"},
	})
	return kindToAPIVersion
}

func getAllKinds() []string {
	var allKinds []string
	var kindToAPIVersion = getSupportedObjectsApiVersionAndKind()
	for _, apiKinds := range kindToAPIVersion.Mapping {
		allKinds = append(allKinds, apiKinds...)
	}
	return allKinds
}

func validateIncludedObject(includedObject v1alpha1.IncludedObject, supportedObjectsApiVersion *APIVersionKindMap) (bool, bool, error) {
	var extractedKind []string
	isApiVersionInclusive := true
	isKindInclusive := true
	if includedObject.ApiVersion != "*" {
		isApiVersionInclusive = false
		extractedKind = supportedObjectsApiVersion.GetKind(includedObject.ApiVersion)
		if len(extractedKind) == 0 {
			err := errors.New(fmt.Sprintf("Specified ApiVersion: %s is not supported!", includedObject.ApiVersion))
			return false, false, err
		}
	}
	if includedObject.Kind != "*" {
		isKindInclusive = false
		if isApiVersionInclusive {
			found := supportedObjectsApiVersion.KindExists(includedObject.Kind)
			if !found {
				err := errors.New(fmt.Sprintf("Specified Kind: %s is not supported!", includedObject.Kind))
				return false, false, err
			}
		} else {
			if !IsInArray(extractedKind, includedObject.Kind) {
				err := errors.New(fmt.Sprintf("Specified Kind: %s is not supported or do not correspond with ApiVersion: %s !", includedObject.Kind, includedObject.ApiVersion))
				return false, false, err
			}
		}
	}
	return isApiVersionInclusive, isKindInclusive, nil
}

func ValidateIncludedObjects(includedObjects []v1alpha1.IncludedObject) (map[int][]bool, error) {
	supportedObjectsApiVersionAndKind := getSupportedObjectsApiVersionAndKind()
	var inclusive map[int][]bool
	inclusive = make(map[int][]bool)
	for index, includedObject := range includedObjects {
		isApiVersionInclusive, isKindInclusive, err := validateIncludedObject(includedObject, supportedObjectsApiVersionAndKind)
		if err != nil {
			return map[int][]bool{}, err
		}
		inclusive[index+1] = []bool{isApiVersionInclusive, isKindInclusive}
	}
	return inclusive, nil
}

func FetchAndFilter(ctx context.Context, Client client.Client, objectList *ObjectList, apiVersion, kind, includeRef, excludeRef, namespace string) error {
	var err error
	switch apiVersion {
	case "*":
		{
			deployments, err := object.FetchDeployments(ctx, Client, includeRef, excludeRef, namespace)
			if err != nil {
				return err
			}
			if len(deployments.Items) > 0 {
				objectList.Deployments = deployments
			}
			statefulsets, err := object.FetchStatefulsets(ctx, Client, includeRef, excludeRef, namespace)
			if err != nil {
				return err
			}
			if len(statefulsets.Items) > 0 {
				objectList.StatefulSets = statefulsets
			}
			cronjobs, err := object.FetchCronjobs(ctx, Client, includeRef, excludeRef, namespace)
			if err != nil {
				return err
			}
			if len(cronjobs.Items) > 0 {
				objectList.CronJobs = cronjobs
			}
			replicasets, err := object.FetchReplicaSets(ctx, Client, includeRef, excludeRef, namespace)
			if err != nil {
				return err
			}
			if len(replicasets.Items) > 0 {
				objectList.ReplicaSets = replicasets
			}
		}
	case "apps/v1":
		switch kind {
		case "Deployment":
			{
				deployments, err := object.FetchDeployments(ctx, Client, includeRef, excludeRef, namespace)
				if err != nil {
					return err
				}
				if len(deployments.Items) > 0 {
					objectList.Deployments = deployments
				}
			}
		case "StatefulSet":
			{
				statefulsets, err := object.FetchStatefulsets(ctx, Client, includeRef, excludeRef, namespace)
				if err != nil {
					return err
				}
				if len(statefulsets.Items) > 0 {
					objectList.StatefulSets = statefulsets
				}
			}
		case "ReplicaSet":
			{
				replicasets, err := object.FetchReplicaSets(ctx, Client, includeRef, excludeRef, namespace)
				if err != nil {
					return err
				}
				if len(replicasets.Items) > 0 {
					objectList.ReplicaSets = replicasets
				}
			}
		case "*":
			{
				deployments, err := object.FetchDeployments(ctx, Client, includeRef, excludeRef, namespace)
				if err != nil {
					return err
				}
				if len(deployments.Items) > 0 {
					objectList.Deployments = deployments
				}
				statefulsets, err := object.FetchStatefulsets(ctx, Client, includeRef, excludeRef, namespace)
				if err != nil {
					return err
				}
				if len(statefulsets.Items) > 0 {
					objectList.StatefulSets = statefulsets
				}
				replicasets, err := object.FetchReplicaSets(ctx, Client, includeRef, excludeRef, namespace)
				if err != nil {
					return err
				}
				if len(replicasets.Items) > 0 {
					objectList.ReplicaSets = replicasets
				}
			}
		}

	case "batch/v1":
		cronjobs, err := object.FetchCronjobs(ctx, Client, includeRef, excludeRef, namespace)
		if err != nil {
			return err
		}
		if len(cronjobs.Items) > 0 {
			objectList.CronJobs = cronjobs
		}
	}
	return err
}

func FetchIncludedObjects(ctx context.Context, Client client.Client, includedObjects []v1alpha1.IncludedObject, inclusive map[int][]bool) (ObjectList, error) {
	var objectList ObjectList
	objectList = ObjectList{}
	var err error

	for index, includedObject := range includedObjects {
		if inclusive[index+1][1] {
			if inclusive[index+1][0] {
				err = FetchAndFilter(ctx, Client, &objectList, "*", "*", includedObject.IncludeRef, includedObject.ExcludeRef, includedObject.Namespace)
				if err != nil {
					return ObjectList{}, err
				}
			} else {
				err = FetchAndFilter(ctx, Client, &objectList, includedObject.ApiVersion, includedObject.Kind, includedObject.IncludeRef, includedObject.ExcludeRef, includedObject.Namespace)
				if err != nil {
					return ObjectList{}, err
				}
			}
		} else {
			err = FetchAndFilter(ctx, Client, &objectList, includedObject.ApiVersion, includedObject.Kind, includedObject.IncludeRef, includedObject.ExcludeRef, includedObject.Namespace)
			if err != nil {
				return ObjectList{}, err
			}
		}
	}
	return objectList, err
}

func WriteChanges(ctx context.Context, Client client.Client, secret *corev1.Secret, list []object.ResourceInt, resourceKind string, failedObjects []string, failedObjectsSleepActions map[string][]string) error {
	if len(list) != 0 {
		err := SaveObjectsData(ctx, Client, secret, resourceKind, list)
		if err != nil {
			return err
		}
	}
	if len(failedObjects) > 0 {
		failedObjectsSleepActions[resourceKind] = failedObjects
	}
	return nil
}

func checkOccurenceInSavedData(savedData []object.ResourceInt, resourceName, resourceNamespace string) (int, bool) {
	for index, item := range savedData {
		if item.GetName() == resourceName && item.GetNamespace() == resourceNamespace {
			return index, true
		}
	}
	return -1, false
}

func removeElementFromArray(arr []object.ResourceInt, index int) []object.ResourceInt {
	// Check if the index is valid
	if index < 0 || index >= len(arr) {
		return arr
	}

	// Create a new array with a length one less than the original array
	newArr := make([]object.ResourceInt, len(arr)-1)

	// Copy the elements before the index
	copy(newArr, arr[:index])

	// Copy the elements after the index
	copy(newArr[index:], arr[index+1:])

	return newArr
}

func putIncludedObjectsToSleep(ctx context.Context, Client client.Client, secret *corev1.Secret, includedObjects ObjectList) (map[string][]string, error) {
	failedObjectsSleepActions := make(map[string][]string)
	failedObjects := []string{}
	replicaResourceMap := object.NewReplicaResourceMap()
	statusResourceMap := object.NewStatusResourceMap()

	var err error
	if includedObjects.Deployments != nil {
		var savedResources []object.ResourceInt
		var sleptResourcesToSave []object.ResourceInt
		dataExists := CheckIfSecretContainsDataOfKind(secret, "Deployment")
		if dataExists {
			savedResources, err = getSecretDatas(secret, "Deployment")
			if err != nil {
				return nil, err
			}
		}
		for _, item := range includedObjects.Deployments.Items {
			objectExists := false
			index := 0
			if len(savedResources) != 0 {
				index, objectExists = checkOccurenceInSavedData(savedResources, item.Name, item.Namespace)
			}

			if !objectExists {
				deployment := object.NewReplicaResource(item.Kind, item.Name, item.Namespace, *item.Spec.Replicas)
				deployment.AddToList(replicaResourceMap)
				deployment.PutToSleep(ctx, Client)
			} else {
				sleptResourcesToSave = append(sleptResourcesToSave, savedResources[index])
				savedResources = removeElementFromArray(savedResources, index)
			}
		}

		if len(savedResources) != 0 {
			for _, resource := range savedResources {
				resource.Wake(ctx, Client)
			}
		}

		deploymentList := object.CastReplicaToGeneral(replicaResourceMap.Items["Deployment"])
		deploymentList = append(deploymentList, sleptResourcesToSave...)

		err = WriteChanges(ctx, Client, secret, deploymentList, "Deployment", failedObjects, failedObjectsSleepActions)
		if err != nil {
			return nil, err
		}
	}

	if includedObjects.StatefulSets != nil {
		savedResources := []object.ResourceInt{}
		sleptResourcesToSave := []object.ResourceInt{}

		dataExists := CheckIfSecretContainsDataOfKind(secret, "StatefulSet")
		if dataExists {
			savedResources, err = getSecretDatas(secret, "StatefulSet")
			if err != nil {
				return nil, err
			}
		}

		for _, item := range includedObjects.StatefulSets.Items {
			objectExists := false
			index := 0
			if len(savedResources) != 0 {
				index, objectExists = checkOccurenceInSavedData(savedResources, item.Name, item.Namespace)
			}

			if !objectExists {
				statefulset := object.NewReplicaResource(item.Kind, item.Name, item.Namespace, *item.Spec.Replicas)
				statefulset.AddToList(replicaResourceMap)
				statefulset.PutToSleep(ctx, Client)
			} else {
				sleptResourcesToSave = append(sleptResourcesToSave, savedResources[index])
				savedResources = removeElementFromArray(savedResources, index)
			}
		}
		if len(savedResources) != 0 {
			for _, resource := range savedResources {
				resource.Wake(ctx, Client)
			}
		}

		statefulsetList := object.CastReplicaToGeneral(replicaResourceMap.Items["StatefulSet"])
		statefulsetList = append(statefulsetList, sleptResourcesToSave...)

		err = WriteChanges(ctx, Client, secret, statefulsetList, "StatefulSet", failedObjects, failedObjectsSleepActions)
		if err != nil {
			return nil, err
		}
	}

	if includedObjects.CronJobs != nil {
		savedResources := []object.ResourceInt{}
		sleptResourcesToSave := []object.ResourceInt{}

		dataExists := CheckIfSecretContainsDataOfKind(secret, "CronJob")
		if dataExists {
			savedResources, err = getSecretDatas(secret, "CronJob")
			if err != nil {
				return nil, err
			}
		}
		for _, item := range includedObjects.CronJobs.Items {
			objectExists := false
			index := 0
			if len(savedResources) != 0 {
				index, objectExists = checkOccurenceInSavedData(savedResources, item.Name, item.Namespace)
			}

			if !objectExists {
				cronjob := object.NewStatusResource(item.Kind, item.Name, item.Namespace, *item.Spec.Suspend)
				cronjob.AddToList(statusResourceMap)
				cronjob.PutToSleep(ctx, Client)
			} else {
				sleptResourcesToSave = append(sleptResourcesToSave, savedResources[index])
				savedResources = removeElementFromArray(savedResources, index)
			}
		}
		if len(savedResources) != 0 {
			for _, resource := range savedResources {
				resource.Wake(ctx, Client)
			}
		}

		cronjobList := object.CastStatusToGeneral(statusResourceMap.Items["CronJob"])
		cronjobList = append(cronjobList, sleptResourcesToSave...)

		err = WriteChanges(ctx, Client, secret, cronjobList, "CronJob", failedObjects, failedObjectsSleepActions)
		if err != nil {
			return nil, err
		}
	}
	if includedObjects.ReplicaSets != nil {
		var savedResources []object.ResourceInt
		var sleptResourcesToSave []object.ResourceInt
		dataExists := CheckIfSecretContainsDataOfKind(secret, "ReplicaSet")
		if dataExists {
			savedResources, err = getSecretDatas(secret, "ReplicaSet")
			if err != nil {
				return nil, err
			}
		}
		for _, item := range includedObjects.ReplicaSets.Items {
			objectExists := false
			index := 0
			if len(savedResources) != 0 {
				index, objectExists = checkOccurenceInSavedData(savedResources, item.Name, item.Namespace)
			}

			if !objectExists {
				replicaset := object.NewReplicaResource(item.Kind, item.Name, item.Namespace, *item.Spec.Replicas)
				replicaset.AddToList(replicaResourceMap)
				replicaset.PutToSleep(ctx, Client)
			} else {
				sleptResourcesToSave = append(sleptResourcesToSave, savedResources[index])
				savedResources = removeElementFromArray(savedResources, index)
			}
		}

		if len(savedResources) != 0 {
			for _, resource := range savedResources {
				resource.Wake(ctx, Client)
			}
		}

		replicasetList := object.CastReplicaToGeneral(replicaResourceMap.Items["ReplicaSet"])
		replicasetList = append(replicasetList, sleptResourcesToSave...)

		err = WriteChanges(ctx, Client, secret, replicasetList, "ReplicaSet", failedObjects, failedObjectsSleepActions)
		if err != nil {
			return nil, err
		}
	}
	return failedObjectsSleepActions, nil
}

func getDataFromSecret(secret *corev1.Secret) ([]object.ResourceInt, error) {
	var resourceList []object.ResourceInt
	var allKinds = getAllKinds()

	for _, kind := range allKinds {
		var newResourceList, err = getSecretDatas(secret, kind)
		if err != nil {
			return nil, err
		}
		resourceList = append(resourceList, newResourceList...)
	}
	return resourceList, nil
}

func WakeUpResources(ctx context.Context, Client client.Client, secret *corev1.Secret) error {
	resourceList, err := getDataFromSecret(secret)
	if err != nil {
		return err
	}

	for _, resource := range resourceList {
		err = resource.Wake(ctx, Client)
	}
	return nil
}
