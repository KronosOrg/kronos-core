package object

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getStatefulsetsByPattern(object Object, allObjects *appsv1.StatefulSetList) *appsv1.StatefulSetList {
	filteredDeployments := &appsv1.StatefulSetList{}
	if object.IncludeRef == "" {
		object.IncludeRef = ".*"
	}
	if object.ExcludeRef == "" {
		object.ExcludeRef = "^$"
	}
	if object.IncludeRef == "" && object.ExcludeRef == "" {
		return allObjects
	}
	if object.IncludeRef == object.ExcludeRef {
		return nil
	}

	includeRef := regexp.MustCompile(object.IncludeRef)
	excludeRef := regexp.MustCompile(object.ExcludeRef)
	for _, deployment := range allObjects.Items {
		if includeRef.MatchString(deployment.Name) && !excludeRef.MatchString(deployment.Name) {
			filteredDeployments.Items = append(filteredDeployments.Items, deployment)
		}
	}
	return filteredDeployments
}

func getAllStatefulsets(ctx context.Context, Client client.Client, namespace string) (*appsv1.StatefulSetList, error) {
	statefulsetList := &appsv1.StatefulSetList{}
	err := Client.List(ctx, statefulsetList, client.InNamespace(namespace))
	if err != nil {
		return nil, err
	}
	return statefulsetList, err
}

func GetStatefulsetListNames(objects *appsv1.StatefulSetList) []string {
	var listNames []string
	for _, objectItem := range objects.Items {
		listNames = append(listNames, objectItem.Name)
	}
	return listNames
}

func FetchStatefulsets(ctx context.Context, Client client.Client, includeRef, excludeRef, namespace string) (*appsv1.StatefulSetList, error) {
	resource := NewObject(Client, includeRef, excludeRef, namespace)
	allObjects, err := getAllStatefulsets(ctx, Client, namespace)
	if err != nil {
		return nil, err
	}
	filteredObjects := getStatefulsetsByPattern(resource, allObjects)
	return filteredObjects, nil
}

func GetStatefulSet(ctx context.Context, Client client.Client, resource ReplicaResource) (*appsv1.StatefulSet, error) {
	statefulset := appsv1.StatefulSet{}
	err := Client.Get(ctx, types.NamespacedName{Name: resource.ResourceName, Namespace: resource.ResourceNamespace}, &statefulset)
	if err != nil {
		return nil, err
	}
	statefulset.Spec.Replicas = &resource.ResourceReplicas
	return &statefulset, nil
}
