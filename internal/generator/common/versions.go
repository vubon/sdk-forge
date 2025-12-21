// Package generator provides language version configuration.
package generator

import (
	"fmt"
	"strconv"
	"strings"
)

// LanguageVersion holds version information for generated code
type LanguageVersion struct {
	Major int
	Minor int
}

// GetGoDefaultVersion returns the default Go version for generated SDKs
func GetGoDefaultVersion() LanguageVersion {
	return LanguageVersion{Major: 1, Minor: 24}
}

// GetGoAvailableVersions returns all available Go versions
func GetGoAvailableVersions() []LanguageVersion {
	return []LanguageVersion{
		{Major: 1, Minor: 24},
		{Major: 1, Minor: 25},
	}
}

// GetPythonDefaultVersion returns the default Python version for generated SDKs
func GetPythonDefaultVersion() LanguageVersion {
	return LanguageVersion{Major: 3, Minor: 11}
}

// GetPythonAvailableVersions returns all available Python versions
func GetPythonAvailableVersions() []LanguageVersion {
	return []LanguageVersion{
		{Major: 3, Minor: 11},
		{Major: 3, Minor: 12},
		{Major: 3, Minor: 13},
		{Major: 3, Minor: 14},
	}
}

// GetPHPDefaultVersion returns the default PHP version for generated SDKs
func GetPHPDefaultVersion() LanguageVersion {
	return LanguageVersion{Major: 8, Minor: 1}
}

// GetPHPAvailableVersions returns all available PHP versions
func GetPHPAvailableVersions() []LanguageVersion {
	return []LanguageVersion{
		{Major: 8, Minor: 0},
		{Major: 8, Minor: 1},
		{Major: 8, Minor: 2},
		{Major: 8, Minor: 3},
	}
}

// ParseVersion parses a version string (e.g., "1.24", "3.11") into LanguageVersion
func ParseVersion(versionStr string) (LanguageVersion, error) {
	parts := strings.Split(versionStr, ".")
	if len(parts) != 2 {
		return LanguageVersion{}, fmt.Errorf("invalid version format: %s (expected format: X.Y)", versionStr)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return LanguageVersion{}, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return LanguageVersion{}, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	return LanguageVersion{Major: major, Minor: minor}, nil
}

// String returns the version as a string (e.g., "1.24")
func (v LanguageVersion) String() string {
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

// ValidateGoVersion checks if the version is in the list of available Go versions
func ValidateGoVersion(version LanguageVersion) error {
	available := GetGoAvailableVersions()
	for _, v := range available {
		if v.Major == version.Major && v.Minor == version.Minor {
			return nil
		}
	}
	return fmt.Errorf("unsupported Go version: %s. Available versions: %v", version.String(), formatVersions(available))
}

// ValidatePythonVersion checks if the version is in the list of available Python versions
func ValidatePythonVersion(version LanguageVersion) error {
	available := GetPythonAvailableVersions()
	for _, v := range available {
		if v.Major == version.Major && v.Minor == version.Minor {
			return nil
		}
	}
	return fmt.Errorf(
		"unsupported Python version: %s. Available versions: %v",
		version.String(),
		formatVersions(available),
	)
}

// ValidatePHPVersion checks if the version is in the list of available PHP versions
func ValidatePHPVersion(version LanguageVersion) error {
	if version.Major < 8 {
		return fmt.Errorf("PHP 8.0+ required, got %s", version.String())
	}
	available := GetPHPAvailableVersions()
	for _, v := range available {
		if v.Major == version.Major && v.Minor == version.Minor {
			return nil
		}
	}
	return fmt.Errorf(
		"unsupported PHP version: %s. Available versions: %v",
		version.String(),
		formatVersions(available),
	)
}

// formatVersions formats a slice of versions as strings
func formatVersions(versions []LanguageVersion) []string {
	result := make([]string, len(versions))
	for i, v := range versions {
		result[i] = v.String()
	}
	return result
}

// GoUsesAny returns true if Go version supports 'any' type alias (Go 1.18+)
func (v LanguageVersion) GoUsesAny() bool {
	return v.Major > 1 || (v.Major == 1 && v.Minor >= 18)
}

// PythonUsesModernTypeHints returns true if Python version supports modern type hints
func (v LanguageVersion) PythonUsesModernTypeHints() bool {
	return v.Major >= 3 && v.Minor >= 8
}

// GetGoEmptyInterface returns the appropriate empty interface type for the Go version
func (v LanguageVersion) GetGoEmptyInterface() string {
	if v.GoUsesAny() {
		return "any"
	}
	return "interface{}"
}

// GetGoVersionString returns the Go version string for go.mod
func (v LanguageVersion) GetGoVersionString() string {
	return fmt.Sprintf("go %d.%d", v.Major, v.Minor)
}

// PHPUsesTypedProperties returns true if PHP version supports typed properties (PHP 7.4+)
func (v LanguageVersion) PHPUsesTypedProperties() bool {
	return v.Major > 7 || (v.Major == 7 && v.Minor >= 4)
}

// PHPUsesEnums returns true if PHP version supports enums (PHP 8.1+)
func (v LanguageVersion) PHPUsesEnums() bool {
	return v.Major > 8 || (v.Major == 8 && v.Minor >= 1)
}

// PHPUsesReadonlyProperties returns true if PHP version supports readonly properties (PHP 8.1+)
func (v LanguageVersion) PHPUsesReadonlyProperties() bool {
	return v.Major > 8 || (v.Major == 8 && v.Minor >= 1)
}

// GetPHPVersionString returns the PHP version string for composer.json
func (v LanguageVersion) GetPHPVersionString() string {
	return fmt.Sprintf("^%d.%d", v.Major, v.Minor)
}
