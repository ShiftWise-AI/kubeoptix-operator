package operands

import (
	shiftwisev1alpha1 "github.com/ShiftWise-AI/kubeoptix-operator/api/v1alpha1"
	"github.com/ShiftWise-AI/kubeoptix-operator/internal/constants"
)

// Settings is the resolved inventory for one ShiftWise instance.
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

	PostgresSecret   string
	PostgresUser     string
	PostgresDatabase string

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

		PostgresSecret:   constants.PostgresSecretName,
		PostgresUser:     constants.DefaultPostgresUser,
		PostgresDatabase: constants.DefaultPostgresDB,

		Harvester:      true,
		Configurations: true,
		Analyzer:       true,
		CoreAI:         true,
		Reporter:       true,
		Dashboard:      true,

		HarvesterImage:      quayImage(constants.HarvesterName),
		AnalyzerImage:       quayImage(constants.AnalyzerName),
		CoreAIImage:         quayImage(constants.CoreAIName),
		ConfigurationsImage: quayImage(constants.ConfigurationsName),
		ReporterImage:       quayImage(constants.ReporterName),
		DashboardImage:      quayImage(constants.DashboardName),
		PostgresImage:       constants.DefaultPostgresImage,
	}
	if len(s.AccessModes) == 0 {
		s.AccessModes = []string{"ReadWriteOnce"}
	}
	return s
}

func quayImage(name string) string {
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
