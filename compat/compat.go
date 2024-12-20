package compat

// CompatibilityFlag is a type that represents a compatibility flag.
type CompatibilityFlag string

// Compatibility flags.
type CompatibilityFlags []CompatibilityFlag

// Compatibility flags.
const (
	// NodeJS compatibility flag.
	NodeJS CompatibilityFlag = "nodejs"
)

var Default = compatibilityFlags["2024-20-12"]

var compatibilityFlags = map[string]CompatibilityFlags{
	"2024-20-12": {NodeJS},
}
