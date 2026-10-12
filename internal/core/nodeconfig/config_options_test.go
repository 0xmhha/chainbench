package nodeconfig

import "testing"

func TestConfigChoicesCoverNativeKnobsAndApplyToSpec(t *testing.T) {
	schema := ConfigOptionSchemas()
	if len(schema) != len(ConfigKnobs) {
		t.Fatal("configuration choices diverge from native knobs")
	}
	for _, key := range ConfigKnobs {
		field, ok := schema[key].(map[string]any)
		if !ok {
			t.Fatal("native knob absent", key)
		}
		for _, value := range field["enum"].([]string) {
			spec := Spec{}
			if err := ApplyConfigOverride(&spec, key, value); err != nil {
				t.Fatal("published choice rejected by engine", err)
			}
			got := map[string]string{"syncMode": spec.SyncMode, "httpHost": spec.HTTPHost, "metricsHost": spec.MetricsHost}[key]
			if got != value {
				t.Fatal("choice did not reach native spec", key, value, got)
			}
		}
	}
}
