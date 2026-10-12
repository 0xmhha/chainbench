package nodeconfig

// ConfigOptionSchemas describes conservative browser choices for the native
// config knobs. Other CLI bind addresses remain supported by ApplyConfigOverride;
// importing them preserves their value for review rather than widening these
// browser choices. Launch option names identify higher-priority conflicts.
func ConfigOptionSchemas() map[string]any {
	return map[string]any{
		"syncMode":    map[string]any{"type": "string", "enum": []string{"full", "snap", "archive"}, "default": "full", "x-launch-options": []string{string(KeySyncMode), string(KeyGCMode)}},
		"httpHost":    map[string]any{"type": "string", "enum": []string{"0.0.0.0", "127.0.0.1"}, "default": "0.0.0.0", "x-launch-options": []string{string(KeyHTTPAddr), string(KeyWSAddr)}},
		"metricsHost": map[string]any{"type": "string", "enum": []string{"0.0.0.0", "127.0.0.1"}, "default": "0.0.0.0", "x-launch-options": []string{string(KeyMetricsAddr)}},
	}
}
