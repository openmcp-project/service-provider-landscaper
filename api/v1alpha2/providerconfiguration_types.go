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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	landscaperComponentPrefix         = "github.com/gardener/landscaper"
	landscaperControllerImageLocation = landscaperComponentPrefix + "/images/landscaper-controller"
	landscaperWebhooksImageLocations  = landscaperComponentPrefix + "/images/landscaper-webhooks-server"
	helmDeployerImageLocation         = landscaperComponentPrefix + "/helm-deployer/images/helm-deployer-controller"
	manifestDeployerController        = landscaperComponentPrefix + "/manifest-deployer/images/manifest-deployer-controller"
)

// ProviderConfigSpec is the specification of the Landscaper Service Provider configuration
type ProviderConfigSpec struct {
	// +kubebuilder:validation:Required
	Deployment Deployment `json:"deployment"`
	// +kubebuilder:validation:MinLength=1
	WorkloadClusterDomain string `json:"workloadClusterDomain,omitempty"`
}

// ProviderConfigStatus is the status of the Landscaper Service Provider configuration
type ProviderConfigStatus struct{}

// Deployment specifies the OCI image locations and available versions of the landscaper
type Deployment struct {
	// Repository is the OCI repository where the Landscaper images are stored
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Repository string `json:"repository,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	AvailableVersions []string `json:"availableVersions,omitempty"`

	// LandscaperController allows to override the image location of the landscaper controller manually
	// +optional
	LandscaperController *ImageConfiguration `json:"landscaperController,omitempty"`
	// LandscaperWebhooksServer allows to override the image location of the landscaper webhooks server manually
	// +optional
	LandscaperWebhooksServer *ImageConfiguration `json:"landscaperWebhooksServer,omitempty"`
	// HelmDeployer allows to override the image location of the landscaper helm deployer manually
	// +optional
	HelmDeployer *ImageConfiguration `json:"helmDeployer,omitempty"`
	// ManifestDeployer allows to override the image location of the landscaper manifest deployer manually
	// +optional
	ManifestDeployer *ImageConfiguration `json:"manifestDeployer,omitempty"`
}

type ImageConfiguration struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
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

func (d *Deployment) IsVersionAvailable(version string) bool {
	for _, v := range d.AvailableVersions {
		if v == version {
			return true
		}
	}
	return false
}

func (pc *ProviderConfig) GetLandscaperControllerImageLocation(version string) string {
	if pc.Spec.Deployment.LandscaperController != nil {
		return imageWithVersion(pc.Spec.Deployment.LandscaperController.Image, version)
	}

	return imageWithVersion(pc.Spec.Deployment.Repository+"/"+landscaperControllerImageLocation, version)
}

func (pc *ProviderConfig) GetLandscaperWebhooksServerImageLocation(version string) string {
	if pc.Spec.Deployment.LandscaperWebhooksServer != nil {
		return imageWithVersion(pc.Spec.Deployment.LandscaperWebhooksServer.Image, version)
	}

	return imageWithVersion(pc.Spec.Deployment.Repository+"/"+landscaperWebhooksImageLocations, version)
}

func (pc *ProviderConfig) GetHelmDeployerImageLocation(version string) string {
	if pc.Spec.Deployment.HelmDeployer != nil {
		return imageWithVersion(pc.Spec.Deployment.HelmDeployer.Image, version)
	}

	return imageWithVersion(pc.Spec.Deployment.Repository+"/"+helmDeployerImageLocation, version)
}

func (pc *ProviderConfig) GetManifestDeployerImageLocation(version string) string {
	if pc.Spec.Deployment.ManifestDeployer != nil {
		return imageWithVersion(pc.Spec.Deployment.ManifestDeployer.Image, version)
	}

	return imageWithVersion(pc.Spec.Deployment.Repository+"/"+manifestDeployerController, version)
}

func imageWithVersion(image, version string) string {
	return image + ":" + version
}

func init() {
	SchemeBuilder.Register(&ProviderConfig{}, &ProviderConfigList{})
}
