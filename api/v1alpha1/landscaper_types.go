/*
Copyright 2025.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LandscaperPhase string

const (
	Progressing LandscaperPhase = "Progressing"
	Terminating LandscaperPhase = "Terminating"
	Ready       LandscaperPhase = "Ready"
)

// LandscaperSpec defines the desired state of Landscaper.
type LandscaperSpec struct {
	// +optional
	ProviderConfigRef *corev1.LocalObjectReference `json:"providerConfigRef,omitempty"`
}

// LandscaperStatus defines the observed state of Landscaper.
type LandscaperStatus struct {
	// +optional
	ProviderConfigRef *corev1.LocalObjectReference `json:"providerConfigRef,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the last observed generation.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	Phase LandscaperPhase `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="openmcp.cloud/cluster=onboarding"

// Landscaper is the Schema for the landscapers API.
type Landscaper struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LandscaperSpec   `json:"spec,omitempty"`
	Status LandscaperStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LandscaperList contains a list of Landscaper.
type LandscaperList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Landscaper `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Landscaper{}, &LandscaperList{})
}
