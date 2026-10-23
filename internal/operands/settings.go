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

		HarvesterImage:      registryImage(constants.HarvesterName),
		AnalyzerImage:       "quay.io/parraes/kubeoptix-analyzer:1.0.7@sha256:84de93180097e16ce0acb008347a644c3f7b1822407442729c5b8907df4ac1bd",
		CoreAIImage:         "quay.io/parraes/kubeoptix-core-ai:1.0.6@sha256:5ce0acd5b8d644072da33dfa380b4ff68e4bbaa458bf9699038dd4d5f4af698a",
		ConfigurationsImage: "quay.io/parraes/kubeoptix-configurations:1.0.7@sha256:d111a4e05ef7813060815ef77182d8249538828e382a93e2f45ae8287164ac03",
		ReporterImage:       "quay.io/parraes/kubeoptix-reporter:1.0.6@sha256:c2a555ff8cec8ebd14f180191c599d92c73c58438ee1e1346c0a8985f7ba3e34",
		DashboardImage:      "quay.io/parraes/kubeoptix-dashboard:1.0.6@sha256:636ec01b4794e3a10c3c29121747b541dde43c5e833e7c83b1c74ee80f8919de",
		PostgresImage:       constants.DefaultPostgresImage,
	}
	if len(s.AccessModes) == 0 {
		s.AccessModes = []string{"ReadWriteOnce"}
	}
	return s
}

func registryImage(name string) string {
	return constants.ImageRegistry + "/" + name + ":" + constants.ImageTag
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
