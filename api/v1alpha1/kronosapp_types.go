/*
Copyright 2024 IsmailAbdelkefi.

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

package v1alpha1

import (
	"context"
	
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Holiday struct {
	Name string `json:"name"`
	Date string `json:"date"`
}

type IncludedObject struct {
	ApiVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace"`
	IncludeRef string `json:"includeRef"`
	ExcludeRef string `json:"excludeRef"`
}

// KronosAppSpec defines the desired state of KronosApp
type KronosAppSpec struct {
	StartSleep      string           `json:"startSleep"`
	EndSleep        string           `json:"endSleep"`
	WeekDays        string           `json:"weekdays"`
	TimeZone        string           `json:"timezone,omitempty"`
	Holidays        []Holiday        `json:"holidays,omitempty"`
	IncludedObjects []IncludedObject `json:"includedObjects"`
	ForceWake       bool             `json:"forceWake,omitempty"`
	ForceSleep      bool             `json:"forceSleep,omitempty"`
}

// KronosAppStatus defines the observed state of KronosApp
type KronosAppStatus struct {
	Status           string   `json:"status"`
	Reason           string   `json:"reason"`
	HandledResources int      `json:"handledResources"`
	NextOperation    string   `json:"nextOperation"`
	CreatedSecrets   []string `json:"secretCreated,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.reason"
// +kubebuilder:printcolumn:name="Handled Resources",type="string",JSONPath=".status.handledResources"
// +kubebuilder:printcolumn:name="Next Operation",type="string",JSONPath=".status.nextOperation"

// KronosApp is the Schema for the kronosapps API
type KronosApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KronosAppSpec   `json:"spec,omitempty"`
	Status KronosAppStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KronosAppList contains a list of KronosApp
type KronosAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KronosApp `json:"items"`
}

func (k KronosApp) SetKronosAppStatus(ctx context.Context, Client client.Client, status, reason bool, nextOperation string, handledResources int) error {
	kdc := k.DeepCopy()
	newStatus := KronosAppStatus{}
	if status {
		newStatus.Status = "Asleep"
		if kdc.Spec.ForceSleep {
			newStatus.Reason = "ForceSleep"
		} else if reason {
			newStatus.Reason = "Holiday"
		} else {
			newStatus.Reason = "Scheduled"
		}
	} else {
		newStatus.Status = "Awake"
		if kdc.Spec.ForceWake {
			newStatus.Reason = "ForceWake"
		} else {
			newStatus.Reason = "Scheduled"
		}
	}
	newStatus.HandledResources = handledResources
	newStatus.NextOperation = nextOperation
	kdc.Status = newStatus
	err := Client.Status().Update(ctx, kdc)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	SchemeBuilder.Register(&KronosApp{}, &KronosAppList{})
}
