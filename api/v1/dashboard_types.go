/*
Copyright 2023.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DashboardSpec defines the desired state of Dashboard
type DashboardSpec struct {
	// +kubebuilder:validation:required
	Generator string `json:"generator,omitempty"`

	// +kubebuilder:validation:required
	Config string `json:"config,omitempty"`
}

const (
	DashboardStatusUnknown = "Unknown"
	DashboardStatusError   = "Error"
	DashboardStatusOK      = "OK"
)

// DashboardStatus defines the observed state of Dashboard
type DashboardStatus struct {
	SyncStatus string      `json:"syncStatus,omitempty"`
	Grafana    GrafanaInfo `json:"grafana,omitempty"`
	Error      string      `json:"error,omitempty"`
}

type GrafanaInfo struct {
	ID      int    `json:"id,omitempty"`
	UID     string `json:"uid,omitempty"`
	Version int    `json:"version,omitempty"`
	URL     string `json:"url,omitempty"`
	Slug    string `json:"slug,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Dashboard is the Schema for the dashboards API
type Dashboard struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DashboardSpec   `json:"spec,omitempty"`
	Status DashboardStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DashboardList contains a list of Dashboard
type DashboardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Dashboard `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Dashboard{}, &DashboardList{})
}
