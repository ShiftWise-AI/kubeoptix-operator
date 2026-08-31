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
	s.HarvesterImage = imageOrDefault(sw.Spec.Images.Harvester, constants.HarvesterName)
	s.AnalyzerImage = imageOrDefault(sw.Spec.Images.Analyzer, constants.AnalyzerName)
	s.CoreAIImage = imageOrDefault(sw.Spec.Images.CoreAI, constants.CoreAIName)
	s.ConfigurationsImage = imageOrDefault(sw.Spec.Images.Configurations, constants.ConfigurationsName)
	s.ReporterImage = imageOrDefault(sw.Spec.Images.Reporter, constants.ReporterName)
	s.DashboardImage = imageOrDefault(sw.Spec.Images.Dashboard, constants.DashboardName)
	if sw.Spec.Images.Postgres != "" {
		s.PostgresImage = sw.Spec.Images.Postgres
	} else {
		s.PostgresImage = constants.DefaultPostgresImage
	}
	return s
}

func imageOrDefault(override, name string) string {
	if override != "" {
		return override
	}
	return constants.QuayRegistry + "/" + name + ":" + constants.ImageTag
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
