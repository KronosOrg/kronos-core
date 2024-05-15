package object

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Object struct {
	Client     client.Client
	IncludeRef string
	ExcludeRef string
	Namespace  string
}

func NewObject(Client client.Client, includeRef, excludeRef, namespace string) Object {
	return Object{
		Client,
		includeRef,
		excludeRef,
		namespace,
	}
}

type ResourceInt interface {
	PutToSleep(ctx context.Context, Client client.Client) []string
	UpdateClient(ctx context.Context, Client client.Client) error
	Wake(ctx context.Context, Client client.Client) error
	GetName() string
	GetNamespace() string
}

type Resource struct {
	ResourceName      string `json:"name"`
	ResourceKind      string `json:"kind"`
	ResourceNamespace string `json:"namespace"`
}

type ResourceMap interface {
	GetLength(itemName string) int
	CastItems(kind string) []ResourceInt
}

type ReplicaResource struct {
	Resource
	ResourceReplicas int32 `json:"replicas"`
}
type ReplicaResourceMap struct {
	Items map[string][]ReplicaResource
}

func NewReplicaResourceMap() ReplicaResourceMap {
	return ReplicaResourceMap{
		Items: make(map[string][]ReplicaResource),
	}
}

func (rm ReplicaResourceMap) GetLength(itemName string) int {
	return len(rm.Items[itemName])
}

func (rm ReplicaResourceMap) CastItems(kind string) []ResourceInt {
	var general []ResourceInt
	for _, item := range rm.Items[kind] {
		general = append(general, item)
	}
	return general
}

func NewReplicaResource(resourceKind, resourceName, resourceNamespace string, resourceReplica int32) ReplicaResource {
	return ReplicaResource{
		Resource: Resource{
			ResourceName:      resourceName,
			ResourceKind:      resourceKind,
			ResourceNamespace: resourceNamespace,
		},
		ResourceReplicas: resourceReplica,
	}
}

func (o ReplicaResource) GetName() string {
	return o.ResourceName
}

func (o ReplicaResource) GetNamespace() string {
	return o.ResourceNamespace
}

func (o ReplicaResource) UpdateClient(ctx context.Context, Client client.Client) error {
	var resource client.Object
	var err error
	switch o.ResourceKind {
	case "Deployment":
		resource, err = GetDeployment(ctx, Client, o)
		if err != nil {
			return err
		}
	case "StatefulSet":
		resource, err = GetStatefulSet(ctx, Client, o)
		if err != nil {
			return err
		}
	case "ReplicaSet":
		resource, err = GetReplicaSet(ctx, Client, o)
		if err != nil {
			return err
		}
	}
	err = Client.Update(ctx, resource)
	if err != nil {
		return err
	}
	return nil
}

func (o ReplicaResource) Sleep(ctx context.Context, Client client.Client) (int32, error) {
	zeroPtr := int32(0)
	replicasToStore := int32(0)
	if o.ResourceReplicas != zeroPtr {
		replicasToStore = o.ResourceReplicas
		o.ResourceReplicas = zeroPtr
		err := o.UpdateClient(ctx, Client)
		if err != nil {
			return zeroPtr, err
		}
	}
	return replicasToStore, nil
}

func (o ReplicaResource) AddToList(replicasMap ReplicaResourceMap) {
	replicasMap.Items[o.ResourceKind] = append(replicasMap.Items[o.ResourceKind], o)
}

func (o ReplicaResource) PutToSleep(ctx context.Context, Client client.Client) []string {
	var failedObjects []string
	if o.ResourceReplicas != int32(0) {
		replicas, err := o.Sleep(ctx, Client)
		if err != nil {
			failedObjects = append(failedObjects, o.ResourceName)
		} else {
			o.ResourceReplicas = replicas
		}
	}
	return failedObjects
}

func (o ReplicaResource) Wake(ctx context.Context, Client client.Client) error {
	err := o.UpdateClient(ctx, Client)
	if err != nil {
		return err
	}
	return nil
}

type StatusResource struct {
	Resource
	ResourceStatus *bool `json:"suspended"`
}

type StatusResourceMap struct {
	Items map[string][]StatusResource
}

func NewStatusResourceMap() StatusResourceMap {
	return StatusResourceMap{
		Items: make(map[string][]StatusResource),
	}
}

func (rm StatusResourceMap) GetLength(itemName string) int {
	return len(rm.Items[itemName])
}

func (rm StatusResourceMap) CastItems(kind string) []ResourceInt {
	var general []ResourceInt
	for _, item := range rm.Items[kind] {
		general = append(general, item)
	}
	return general
}

func NewStatusResource(resourceKind, resourceName, resourceNamespace string, resourceStatus bool) StatusResource {
	return StatusResource{
		Resource: Resource{
			ResourceName:      resourceName,
			ResourceKind:      resourceKind,
			ResourceNamespace: resourceNamespace,
		},
		ResourceStatus: &resourceStatus,
	}
}

func (o StatusResource) GetName() string {
	return o.ResourceName
}

func (o StatusResource) GetNamespace() string {
	return o.ResourceNamespace
}

func (o StatusResource) UpdateClient(ctx context.Context, Client client.Client) error {
	var resource client.Object
	var err error
	switch o.ResourceKind {
	case "CronJob":
		resource, err = GetCronjob(ctx, Client, o)
		if err != nil {
			return err
		}
		err = Client.Update(ctx, resource)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o StatusResource) Sleep(ctx context.Context, Client client.Client) (*bool, error) {
	suspendStatus := true
	statusToStore := false
	if o.ResourceStatus != &suspendStatus {
		statusToStore = *o.ResourceStatus
		o.ResourceStatus = &suspendStatus
		err := o.UpdateClient(ctx, Client)
		if err != nil {
			return nil, err
		}
	}
	return &statusToStore, nil
}

func (o StatusResource) PutToSleep(ctx context.Context, Client client.Client) []string {
	var failedObjects []string
	if !*o.ResourceStatus {
		status, err := o.Sleep(ctx, Client)
		if err != nil {
			failedObjects = append(failedObjects, o.ResourceName)
		} else {
			o.ResourceStatus = status
		}
	}
	return failedObjects
}

func (o StatusResource) AddToList(statusMap StatusResourceMap) {
	statusMap.Items[o.ResourceKind] = append(statusMap.Items[o.ResourceKind], o)
}

func (o StatusResource) Wake(ctx context.Context, Client client.Client) error {
	err := o.UpdateClient(ctx, Client)
	if err != nil {
		return err
	}
	return nil
}

func CastReplicaToGeneral(resource []ReplicaResource) []ResourceInt {
	var general []ResourceInt
	for _, item := range resource {
		general = append(general, item)
	}
	return general
}

func CastStatusToGeneral(resource []StatusResource) []ResourceInt {
	var general []ResourceInt
	for _, item := range resource {
		general = append(general, item)
	}
	return general
}
