package object

import (
	"context"
	"regexp"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getCronjobsByPattern(object Object, allObjects *batchv1.CronJobList) *batchv1.CronJobList {
	filteredCronjobs := &batchv1.CronJobList{}
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
			filteredCronjobs.Items = append(filteredCronjobs.Items, deployment)
		}
	}
	return filteredCronjobs
}

func getAllCronjobs(ctx context.Context, object Object) (*batchv1.CronJobList, error) {
	cronjobList := &batchv1.CronJobList{}
	err := object.Client.List(ctx, cronjobList, client.InNamespace(object.Namespace))
	if err != nil {
		return nil, err
	}
	return cronjobList, err
}

func GetCronjobListNames(objects *batchv1.CronJobList) []string {
	var listNames []string
	for _, objectItem := range objects.Items {
		listNames = append(listNames, objectItem.Name)
	}
	return listNames
}

func FetchCronjobs(ctx context.Context, Client client.Client, includeRef, excludeRef, namespace string) (*batchv1.CronJobList, error) {
	resource := NewObject(Client, includeRef, excludeRef, namespace)
	allObjects, err := getAllCronjobs(ctx, resource)
	if err != nil {
		return nil, err
	}
	filteredObjects := getCronjobsByPattern(resource, allObjects)
	return filteredObjects, nil
}

func GetCronjob(ctx context.Context, Client client.Client, resource StatusResource) (*batchv1.CronJob, error) {
	cronjob := batchv1.CronJob{}
	err := Client.Get(ctx, types.NamespacedName{Name: resource.ResourceName, Namespace: resource.ResourceNamespace}, &cronjob)
	if err != nil {
		return nil, err
	}
	cronjob.Spec.Suspend = resource.ResourceStatus
	return &cronjob, nil
}
