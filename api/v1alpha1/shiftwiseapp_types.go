package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ShiftWiseAppSpec defines the desired state of ShiftWiseApp.
type ShiftWiseAppSpec struct {
	// Image is the container image to deploy.
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// Replicas is the desired number of pod replicas.
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Port is the container port to expose.
	// +kubebuilder:default=8080
	// +optional
	Port int32 `json:"port,omitempty"`

	// Env is a list of environment variables to set in the container.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Resources specifies the compute resource requirements.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// ServiceType specifies the type of Service to create.
	// +kubebuilder:default=ClusterIP
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
	// +optional
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`
}

// ShiftWiseAppStatus defines the observed state of ShiftWiseApp.
type ShiftWiseAppStatus struct {
	// ReadyReplicas is the number of ready pod replicas.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Conditions represent the latest available observations of the ShiftWiseApp's state.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.readyReplicas
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ShiftWiseApp is the Schema for the shiftWiseApps API.
type ShiftWiseApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ShiftWiseAppSpec   `json:"spec,omitempty"`
	Status ShiftWiseAppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ShiftWiseAppList contains a list of ShiftWiseApp.
type ShiftWiseAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ShiftWiseApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ShiftWiseApp{}, &ShiftWiseAppList{})
}
