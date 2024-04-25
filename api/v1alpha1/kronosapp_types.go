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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	CreatedSecrets []string `json:"secretCreated"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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

func init() {
	SchemeBuilder.Register(&KronosApp{}, &KronosAppList{})
}
