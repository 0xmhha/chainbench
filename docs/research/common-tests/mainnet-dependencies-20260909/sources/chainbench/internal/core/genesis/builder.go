package genesis

// SourceMode selects how a genesis document is produced. The four modes are all
// backed by existing functions in this package (Build, MergeOverride,
// ApplyConfigOverrides, SetConfigSection); SourceMode names them for callers.
type SourceMode int

const (
	// ModeExisting uses an existing genesis file as-is.
	ModeExisting SourceMode = iota
	// ModeBuild builds from parameters via Build.
	ModeBuild
	// ModeTemplateOverride builds from a template then applies overrides.
	ModeTemplateOverride
	// ModeUpgradeInherit reuses a from-chain genesis plus an upgrade overlay.
	ModeUpgradeInherit
)

// String returns the source-mode label.
func (m SourceMode) String() string {
	switch m {
	case ModeExisting:
		return "existing"
	case ModeBuild:
		return "build"
	case ModeTemplateOverride:
		return "template-override"
	case ModeUpgradeInherit:
		return "upgrade-inherit"
	default:
		return "unknown"
	}
}
