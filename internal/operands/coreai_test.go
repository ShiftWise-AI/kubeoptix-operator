package operands

import "testing"

func TestCoreAIUsesConfigurationsSystemSettings(t *testing.T) {
	sts := coreAISTS(Settings{Namespace: "test"}, nil)
	for _, container := range sts.Spec.Template.Spec.Containers {
		if container.Name != "api" {
			continue
		}
		for _, variable := range container.Env {
			if variable.Name != "SYSTEM_SETTINGS_URL" {
				continue
			}
			if variable.Value != "http://configurations-api:8000/system-settings" {
				t.Fatalf("SYSTEM_SETTINGS_URL = %q, want configurations service endpoint", variable.Value)
			}
			return
		}
	}
	t.Fatal("core-ai API container is missing SYSTEM_SETTINGS_URL")
}