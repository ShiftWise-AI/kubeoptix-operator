package constants

const (
	DefaultNamespace = "shiftwise-ai"
	OperatorName     = "shiftwise-operator"
	AppName          = "kubeoptix"
	ManagedBy        = "shiftwise-operator"
	Finalizer        = "shiftwise.ai/cleanup"

	QuayRegistry     = "quay.io/parraes"
	ImageTag         = "0.2.1"
	PullAlways       = "Always"
	PullIfNotPresent = "IfNotPresent"

	HarvesterName      = "kubeoptix-harvester"
	AnalyzerName       = "kubeoptix-analyzer"
	CoreAIName         = "kubeoptix-core-ai"
	ConfigurationsName = "kubeoptix-configurations"
	ReporterName       = "kubeoptix-reporter"
	DashboardName      = "kubeoptix-dashboard"
	PostgresName       = "kubeoptix-db"

	HarvesterAPIService      = "harvester-api"
	AnalyzerAPIService       = "analyzer-api"
	CoreAIAPIService         = "core-ai-api"
	ConfigurationsAPIService = "configurations-api"
	ReporterAPIService       = "reporter-api"
	DashboardService         = "kubeoptix-dashboard"

	HarvesterRoute = "harvester"
	AnalyzerRoute  = "analyzer"
	DashboardRoute = "kubeoptix-dashboard"

	SharedServiceAccount = "shiftwisea-ai-user"
	ClusterReaderRole    = "cluster-reader"

	DataClaimName       = "harvester-app-data"
	PostgresSecretName  = "kubeoptix-db"
	DashboardConfigMap  = "kubeoptix-dashboard-env"
	DashboardContainer  = "dashboard"

	DefaultStorageSize  = "10Gi"
	PostgresStorageSize = "20Gi"

	DefaultPostgresImage = "registry.redhat.io/rhel9/postgresql-18:9.8-1787043471"
	DefaultPostgresUser  = "kubeoptix"
	DefaultPostgresDB    = "kubeoptix"

	ReconcileInterval = 30
)
