package object

import (
	"context"
	"regexp"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getDeploymentsByPattern(object Object, allObjects *appsv1.DeploymentList) *appsv1.DeploymentList {
	filteredDeployments := &appsv1.DeploymentList{}
	if object.IncludeRef == "" && object.ExcludeRef == "" {
		return allObjects
	}
	if object.IncludeRef == object.ExcludeRef {
		return nil
	}

	includeRe := regexp.MustCompile(object.IncludeRef)
	excludeRe := regexp.MustCompile(object.ExcludeRef)
	for _, deployment := range allObjects.Items {
		if includeRe.MatchString(deployment.Name) && !excludeRe.MatchString(deployment.Name) {
			filteredDeployments.Items = append(filteredDeployments.Items, deployment)
		}
	}
	return filteredDeployments
}

func getAllDeployments(ctx context.Context, object Object) (*appsv1.DeploymentList, error) {
	deploymentList := &appsv1.DeploymentList{}
	err := object.Client.List(ctx, deploymentList, client.InNamespace(object.Namespace))
	if err != nil {
		return nil, err
	}
	return deploymentList, err
}

func GetDeploymentListNames(objects *appsv1.DeploymentList) []string {
	var listNames []string
	for _, objectItem := range objects.Items {
		listNames = append(listNames, objectItem.Name)
	}
	return listNames
}

func FetchDeployments(ctx context.Context, Client client.Client, includeRef, excludeRef, namespace string) (*appsv1.DeploymentList, error) {
	resource := NewObject(Client, includeRef, excludeRef, namespace)
	allObjects, err := getAllDeployments(ctx, resource)
	if err != nil {
		return nil, err
	}
	filteredObjects := getDeploymentsByPattern(resource, allObjects)
	return filteredObjects, nil
}

func GetDeployment(ctx context.Context, Client client.Client, resource ReplicaResource) (*appsv1.Deployment, error) {
	deployment := appsv1.Deployment{}
	err := Client.Get(ctx, types.NamespacedName{Name: resource.ResourceName, Namespace: resource.ResourceNamespace}, &deployment)
	if err != nil {
		return nil, err
	}
	deployment.Spec.Replicas = &resource.ResourceReplicas
	return &deployment, nil
}
