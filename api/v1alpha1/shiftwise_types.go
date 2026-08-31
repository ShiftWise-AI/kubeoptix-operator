package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DefaultTargetNamespace is the OpenShift project used when spec.targetNamespace is empty.
const DefaultTargetNamespace = "shiftwise-ai"

// ShiftWiseSpec defines the desired state of a ShiftWise platform instance.
type ShiftWiseSpec struct {
	// TargetNamespace is the OpenShift project where KubeOptix components are created.
	// +kubebuilder:default=shiftwise-ai
	TargetNamespace string `json:"targetNamespace,omitempty"`

	// Images overrides the container images used for each component.
	// Empty fields default to quay.io/parraes/kubeoptix-<component>:0.2.1.
	Images ImagesSpec `json:"images,omitempty"`

	// Storage configures the shared PVC used by Harvester, Analyzer, Core AI and Reporter.
	Storage StorageSpec `json:"storage,omitempty"`

	// Credentials references or inlines PostgreSQL and LLM secrets.
	Credentials CredentialsSpec `json:"credentials,omitempty"`

	// GitSource is unused. Operand images are pulled from Quay, not built in-cluster.
	GitSource GitSourceSpec `json:"gitSource,omitempty"`

	// Components toggles individual KubeOptix workloads. All are enabled by default.
	Components ComponentsSpec `json:"components,omitempty"`
}

// GitSourceSpec is retained for compatibility and is ignored by the operator.
type GitSourceSpec struct {
	// ExistingSecret is ignored. Operand images are pulled from Quay.
	ExistingSecret string `json:"existingSecret,omitempty"`
}

// ImagesSpec lists operand container images.
type ImagesSpec struct {
	Harvester      string `json:"harvester,omitempty"`
	Analyzer       string `json:"analyzer,omitempty"`
	CoreAI         string `json:"coreAi,omitempty"`
	Configurations string `json:"configurations,omitempty"`
	Reporter       string `json:"reporter,omitempty"`
	Dashboard      string `json:"dashboard,omitempty"`
	Postgres       string `json:"postgres,omitempty"`
}

// StorageSpec configures the shared data volume created by the Harvester chart.
type StorageSpec struct {
	ExistingClaim    string   `json:"existingClaim,omitempty"`
	Name             string   `json:"name,omitempty"`
	Size             string   `json:"size,omitempty"`
	StorageClassName string   `json:"storageClassName,omitempty"`
	AccessModes      []string `json:"accessModes,omitempty"`
}

// CredentialsSpec points at existing Secrets or supplies values for the operator to create them.
type CredentialsSpec struct {
	ExistingSecret    string `json:"existingSecret,omitempty"`
	PostgresUser      string `json:"postgresUser,omitempty"`
	PostgresPassword  string `json:"postgresPassword,omitempty"`
	PostgresDatabase  string `json:"postgresDatabase,omitempty"`
	LLMExistingSecret string `json:"llmExistingSecret,omitempty"`
	LLMAPIKey         string `json:"llmApiKey,omitempty"`
	LLMBaseURL        string `json:"llmBaseUrl,omitempty"`
	LLMModel          string `json:"llmModel,omitempty"`
	CursorAPIKey      string `json:"cursorApiKey,omitempty"`
	CursorModel       string `json:"cursorModel,omitempty"`
}

// ComponentsSpec enables or disables each platform component.
type ComponentsSpec struct {
	Harvester      ComponentSpec `json:"harvester,omitempty"`
	Analyzer       ComponentSpec `json:"analyzer,omitempty"`
	CoreAI         ComponentSpec `json:"coreAi,omitempty"`
	Configurations ComponentSpec `json:"configurations,omitempty"`
	Reporter       ComponentSpec `json:"reporter,omitempty"`
	Dashboard      ComponentSpec `json:"dashboard,omitempty"`
}

// ComponentSpec toggles a single component. Omitted or null enabled means true.
type ComponentSpec struct {
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`
}

// IsEnabled reports whether the component should be reconciled. Default is true.
func (c ComponentSpec) IsEnabled() bool {
	return c.Enabled == nil || *c.Enabled
}

// Namespace returns the operand project, defaulting to shiftwise-ai.
func (s ShiftWiseSpec) Namespace() string {
	if s.TargetNamespace != "" {
		return s.TargetNamespace
	}
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
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=".spec.targetNamespace"
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
