package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DefaultTargetNamespace is the OpenShift project used for KubeOptix components.
const DefaultTargetNamespace = "shiftwise-ai"

// ShiftWiseSpec defines the desired state of a ShiftWise platform instance.
type ShiftWiseSpec struct {
	// Storage configures the shared PVC used by Harvester, Analyzer, Core AI and Reporter.
	Storage StorageSpec `json:"storage,omitempty"`
}

// StorageSpec configures the shared data volume.
type StorageSpec struct {
	ExistingClaim    string   `json:"existingClaim,omitempty"`
	Name             string   `json:"name,omitempty"`
	Size             string   `json:"size,omitempty"`
	StorageClassName string   `json:"storageClassName,omitempty"`
	AccessModes      []string `json:"accessModes,omitempty"`
}

// Namespace returns the operand project (always shiftwise-ai).
func (s ShiftWiseSpec) Namespace() string {
	return DefaultTargetNamespace
}

// ShiftWiseStatus defines the observed state of ShiftWise.
type ShiftWiseStatus struct {
	// Phase is a high-level summary: Pending, Initializing, Progressing, Ready or Error.
	// +kubebuilder:validation:Enum=Pending;Initializing;Progressing;Ready;Error
	Phase string `json:"phase,omitempty"`

	// ReadyComponents is a ready/desired count such as "4/6".
	ReadyComponents string `json:"readyComponents,omitempty"`

	// ObservedGeneration is the last reconciled metadata.generation.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Message is a human-readable reconciliation note.
	Message string `json:"message,omitempty"`

	// Conditions follow Kubernetes conventional status conditions.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=sw,scope=Namespaced,categories=shiftwise
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.readyComponents"
// +kubebuilder:printcolumn:name="Storage",type=string,JSONPath=".spec.storage.size"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// ShiftWise is the Schema for the shiftwises API.
type ShiftWise struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ShiftWiseSpec   `json:"spec,omitempty"`
	Status ShiftWiseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ShiftWiseList contains a list of ShiftWise.
type ShiftWiseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ShiftWise `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ShiftWise{}, &ShiftWiseList{})
}
