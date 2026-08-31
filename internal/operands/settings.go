package operands

import (
	shiftwisev1alpha1 "github.com/ShiftWise-AI/kubeoptix-operator/api/v1alpha1"
	"github.com/ShiftWise-AI/kubeoptix-operator/internal/constants"
)

// Settings is the resolved inventory for one ShiftWise instance (Helm values + CR).
type Settings struct {
	Namespace string
	Instance  string

	HarvesterImage      string
	AnalyzerImage       string
	CoreAIImage         string
	ConfigurationsImage string
	ReporterImage       string
	DashboardImage      string
	PostgresImage       string

	ClaimName    string
	StorageSize  string
	StorageClass string
	AccessModes  []string

	GitSecret      string
	PostgresSecret string
	LLMSecret      string

	PostgresUser     string
	PostgresPassword string
	PostgresDatabase string
	CreatePostgres   bool

	CreateLLM    bool
	LLMAPIKey    string
	LLMBaseURL   string
	LLMModel     string
	CursorAPIKey string
	CursorModel  string

	Harvester      bool
	Configurations bool
	Analyzer       bool
	CoreAI         bool
	Reporter       bool
	Dashboard      bool
}

func FromCR(sw *shiftwisev1alpha1.ShiftWise) Settings {
	s := Settings{
		Namespace: sw.Spec.Namespace(),
		Instance:  sw.Name,

		ClaimName:    firstNonEmpty(sw.Spec.Storage.ExistingClaim, sw.Spec.Storage.Name, constants.DataClaimName),
		StorageSize:  firstNonEmpty(sw.Spec.Storage.Size, constants.DefaultStorageSize),
		StorageClass: sw.Spec.Storage.StorageClassName,
		AccessModes:  sw.Spec.Storage.AccessModes,

		GitSecret:      firstNonEmpty(sw.Spec.GitSource.ExistingSecret, constants.GitSourceSecretName),
		PostgresSecret: firstNonEmpty(sw.Spec.Credentials.ExistingSecret, constants.PostgresSecretName),
		LLMSecret:      firstNonEmpty(sw.Spec.Credentials.LLMExistingSecret, constants.LLMSecretName),

		PostgresUser:     firstNonEmpty(sw.Spec.Credentials.PostgresUser, constants.DefaultPostgresUser),
		PostgresPassword: sw.Spec.Credentials.PostgresPassword,
		PostgresDatabase: firstNonEmpty(sw.Spec.Credentials.PostgresDatabase, constants.DefaultPostgresDB),
		CreatePostgres:   sw.Spec.Credentials.ExistingSecret == "",

		CreateLLM:    sw.Spec.Credentials.LLMExistingSecret == "",
		LLMAPIKey:    sw.Spec.Credentials.LLMAPIKey,
		LLMBaseURL:   firstNonEmpty(sw.Spec.Credentials.LLMBaseURL, "https://api.openai.com/v1"),
		LLMModel:     firstNonEmpty(sw.Spec.Credentials.LLMModel, "gpt-4o-mini"),
		CursorAPIKey: sw.Spec.Credentials.CursorAPIKey,
		CursorModel:  firstNonEmpty(sw.Spec.Credentials.CursorModel, "default"),

		Harvester:      sw.Spec.Components.Harvester.IsEnabled(),
		Configurations: sw.Spec.Components.Configurations.IsEnabled(),
		Analyzer:       sw.Spec.Components.Analyzer.IsEnabled(),
		CoreAI:         sw.Spec.Components.CoreAI.IsEnabled(),
		Reporter:       sw.Spec.Components.Reporter.IsEnabled(),
		Dashboard:      sw.Spec.Components.Dashboard.IsEnabled(),
	}
	if len(s.AccessModes) == 0 {
		s.AccessModes = []string{"ReadWriteOnce"}
	}
	ns := s.Namespace
	s.HarvesterImage = imageOrBuild(sw.Spec.Images.Harvester, ns, constants.HarvesterName)
	s.AnalyzerImage = imageOrBuild(sw.Spec.Images.Analyzer, ns, constants.AnalyzerName)
	s.CoreAIImage = imageOrBuild(sw.Spec.Images.CoreAI, ns, constants.CoreAIName)
	s.ConfigurationsImage = imageOrBuild(sw.Spec.Images.Configurations, ns, constants.ConfigurationsName)
	s.ReporterImage = imageOrBuild(sw.Spec.Images.Reporter, ns, constants.ReporterName)
	s.DashboardImage = imageOrBuild(sw.Spec.Images.Dashboard, ns, constants.DashboardName)
	if sw.Spec.Images.Postgres != "" {
		s.PostgresImage = sw.Spec.Images.Postgres
	} else {
		s.PostgresImage = constants.DefaultPostgresImage
	}
	return s
}

func imageOrBuild(override, namespace, stream string) string {
	if override != "" {
		return override
	}
	return InternalImage(namespace, stream)
}

func InternalImage(namespace, stream string) string {
	return constants.InternalRegistry + "/" + namespace + "/" + stream + ":" + constants.ImageTag
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func enabledComponents(s Settings) []string {
	names := make([]string, 0, 6)
	if s.Harvester {
		names = append(names, constants.HarvesterName)
	}
	if s.Configurations {
		names = append(names, constants.PostgresName, constants.ConfigurationsName)
	}
	if s.Analyzer {
		names = append(names, constants.AnalyzerName)
	}
	if s.CoreAI {
		names = append(names, constants.CoreAIName)
	}
	if s.Reporter {
		names = append(names, constants.ReporterName)
	}
	if s.Dashboard {
		names = append(names, constants.DashboardName)
	}
	return names
}
