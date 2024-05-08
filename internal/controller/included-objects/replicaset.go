package object

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getReplicaSetsByPattern(object Object, allObjects *appsv1.ReplicaSetList) *appsv1.ReplicaSetList {
	filteredReplicaSets := &appsv1.ReplicaSetList{}
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

	includeRe := regexp.MustCompile(object.IncludeRef)
	excludeRe := regexp.MustCompile(object.ExcludeRef)
	for _, replicaset := range allObjects.Items {
		if includeRe.MatchString(replicaset.Name) && !excludeRe.MatchString(replicaset.Name) {
			filteredReplicaSets.Items = append(filteredReplicaSets.Items, replicaset)
		}
	}
	return filteredReplicaSets
}

func getAllReplicaSets(ctx context.Context, object Object) (*appsv1.ReplicaSetList, error) {
	replicasetList := &appsv1.ReplicaSetList{}
	err := object.Client.List(ctx, replicasetList, client.InNamespace(object.Namespace))
	if err != nil {
		return nil, err
	}
	return replicasetList, err
}

func GetReplicaSetListNames(objects *appsv1.ReplicaSetList) []string {
	var listNames []string
	for _, objectItem := range objects.Items {
		listNames = append(listNames, objectItem.Name)
	}
	return listNames
}

func FetchReplicaSets(ctx context.Context, Client client.Client, includeRef, excludeRef, namespace string) (*appsv1.ReplicaSetList, error) {
	resource := NewObject(Client, includeRef, excludeRef, namespace)
	allObjects, err := getAllReplicaSets(ctx, resource)
	if err != nil {
		return nil, err
	}
	filteredObjects := getReplicaSetsByPattern(resource, allObjects)
	return filteredObjects, nil
}

func GetReplicaSet(ctx context.Context, Client client.Client, resource ReplicaResource) (*appsv1.ReplicaSet, error) {
	replicaset := appsv1.ReplicaSet{}
	err := Client.Get(ctx, types.NamespacedName{Name: resource.ResourceName, Namespace: resource.ResourceNamespace}, &replicaset)
	if err != nil {
		return nil, err
	}
	replicaset.Spec.Replicas = &resource.ResourceReplicas
	return &replicaset, nil
}
