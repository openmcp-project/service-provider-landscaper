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
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	latestVersion  = "latest"
	unknownVersion = "unknown"
)

type ProviderConfigSpec struct {
	// +kubebuilder:validation:Required
	Deployment Deployment `json:"deployment"`
	// +kubebuilder:validation:MinLength=1
	WorkloadClusterDomain string `json:"workloadClusterDomain,omitempty"`
}

type ProviderConfigStatus struct{}

type Deployment struct {
	// +kubebuilder:validation:Required
	LandscaperController ImageConfiguration `json:"landscaperController"`
	// +kubebuilder:validation:Required
	LandscaperWebhooksServer ImageConfiguration `json:"landscaperWebhooksServer"`
	// +kubebuilder:validation:Required
	HelmDeployer ImageConfiguration `json:"helmDeployer"`
	// +kubebuilder:validation:Required
	ManifestDeployer ImageConfiguration `json:"manifestDeployer"`
}

type ImageConfiguration struct {
	// +kubebuilder:validation:Required
	Image string `json:"image"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=lspcfg
// +kubebuilder:metadata:labels="openmcp.cloud/cluster=platform"
type ProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProviderConfigSpec   `json:"spec,omitempty"`
	Status ProviderConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfig `json:"items"`
}

func getVersion(image string) string {
	if image == "" {
		return latestVersion
	}
	// Assuming the image follows the format "image:version"
	parts := strings.Split(image, ":")
	if len(parts) > 1 {
		return parts[1]
	}
	return unknownVersion
}

func GetHelmDeployerName() string {
	return "helm-deployer"
}

func GetManifestDeployerName() string {
	return "manifest-deployer"
}

func GetControllerName() string {
	return "landscaper-controller"
}

func GetWebhooksServerName() string {
	return "landscaper-webhooks-server"
}

func (pc *ProviderConfig) GetHelmDeployerVersion() string {
	if pc.Spec.Deployment.HelmDeployer.Image == "" {
		return latestVersion
	}
	return getVersion(pc.Spec.Deployment.HelmDeployer.Image)
}

func (pc *ProviderConfig) GetManifestDeployerVersion() string {
	if pc.Spec.Deployment.ManifestDeployer.Image == "" {
		return latestVersion
	}
	return getVersion(pc.Spec.Deployment.ManifestDeployer.Image)
}

func (pc *ProviderConfig) GetControllerVersion() string {
	if pc.Spec.Deployment.LandscaperController.Image == "" {
		return latestVersion
	}
	return getVersion(pc.Spec.Deployment.LandscaperController.Image)
}

func (pc *ProviderConfig) GetWebhooksServerVersion() string {
	if pc.Spec.Deployment.LandscaperWebhooksServer.Image == "" {
		return latestVersion
	}
	return getVersion(pc.Spec.Deployment.LandscaperWebhooksServer.Image)
}

func init() {
	SchemeBuilder.Register(&ProviderConfig{}, &ProviderConfigList{})
}
