/*
Copyright 2021.

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
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
)

type PodTemplateApplyConfiguration corev1apply.PodTemplateApplyConfiguration

func (ac *PodTemplateApplyConfiguration) DeepCopy() *PodTemplateApplyConfiguration {
	out := new(PodTemplateApplyConfiguration)
	deepCopy(out, ac)
	return out
}

func deepCopy(dst interface{}, src interface{}) {
	if dst == nil {
		panic("dst cannot be nil")
	}
	if src == nil {
		panic("src cannot be nil")
	}
	bytes, err := json.Marshal(src)
	if err != nil {
		panic("Unable to marshal src")
	}
	err = json.Unmarshal(bytes, dst)
	if err != nil {
		panic("Unable to unmarshal into dst")
	}
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MyAppSpec defines the desired state of MyApp
type MyAppSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	PodTemplate *PodTemplateApplyConfiguration `json:"podTemplate"`
}

// MyAppStatus defines the observed state of MyApp
type MyAppStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MyApp is the Schema for the myapps API
type MyApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MyAppSpec   `json:"spec,omitempty"`
	Status MyAppStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MyAppList contains a list of MyApp
type MyAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MyApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MyApp{}, &MyAppList{})
}
