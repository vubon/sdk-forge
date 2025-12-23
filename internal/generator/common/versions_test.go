package common

import (
	"testing"
)

func TestGetTypeScriptDefaultVersion(t *testing.T) {
	t.Parallel()
	version := GetTypeScriptDefaultVersion()
	expected := LanguageVersion{Major: 5, Minor: 0}
	if version.Major != expected.Major || version.Minor != expected.Minor {
		t.Errorf("GetTypeScriptDefaultVersion() = %v, want %v", version, expected)
	}
}

func TestGetTypeScriptAvailableVersions(t *testing.T) {
	t.Parallel()
	versions := GetTypeScriptAvailableVersions()
	expectedVersions := []LanguageVersion{
		{Major: 4, Minor: 9},
		{Major: 5, Minor: 0},
		{Major: 5, Minor: 1},
		{Major: 5, Minor: 2},
		{Major: 5, Minor: 3},
		{Major: 5, Minor: 4},
		{Major: 5, Minor: 5},
		{Major: 5, Minor: 6},
	}

	if len(versions) != len(expectedVersions) {
		t.Errorf("GetTypeScriptAvailableVersions() returned %d versions, want %d", len(versions), len(expectedVersions))
	}

	for i, v := range versions {
		if v.Major != expectedVersions[i].Major || v.Minor != expectedVersions[i].Minor {
			t.Errorf("GetTypeScriptAvailableVersions()[%d] = %v, want %v", i, v, expectedVersions[i])
		}
	}
}

func TestValidateTypeScriptVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		version LanguageVersion
		wantErr bool
	}{
		{"valid version 4.9", LanguageVersion{Major: 4, Minor: 9}, false},
		{"valid version 5.0", LanguageVersion{Major: 5, Minor: 0}, false},
		{"valid version 5.3", LanguageVersion{Major: 5, Minor: 3}, false},
		{"valid version 5.6", LanguageVersion{Major: 5, Minor: 6}, false},
		{"invalid version 3.0", LanguageVersion{Major: 3, Minor: 0}, true},
		{"invalid version 4.8", LanguageVersion{Major: 4, Minor: 8}, true},
		{"invalid version 5.7", LanguageVersion{Major: 5, Minor: 7}, true},
		{"invalid version 6.0", LanguageVersion{Major: 6, Minor: 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateTypeScriptVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTypeScriptVersion(%v) error = %v, wantErr %v", tt.version, err, tt.wantErr)
			}
		})
	}
}

func TestGetTypeScriptVersionString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		version  LanguageVersion
		expected string
	}{
		{"version 4.9", LanguageVersion{Major: 4, Minor: 9}, "^4.9.0"},
		{"version 5.0", LanguageVersion{Major: 5, Minor: 0}, "^5.0.0"},
		{"version 5.3", LanguageVersion{Major: 5, Minor: 3}, "^5.3.0"},
		{"version 5.6", LanguageVersion{Major: 5, Minor: 6}, "^5.6.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.version.GetTypeScriptVersionString()
			if result != tt.expected {
				t.Errorf("GetTypeScriptVersionString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetPHPAvailableVersions(t *testing.T) {
	t.Parallel()
	versions := GetPHPAvailableVersions()
	expectedVersions := []LanguageVersion{
		{Major: 8, Minor: 0},
		{Major: 8, Minor: 1},
		{Major: 8, Minor: 2},
		{Major: 8, Minor: 3},
	}

	if len(versions) != len(expectedVersions) {
		t.Errorf("GetPHPAvailableVersions() returned %d versions, want %d", len(versions), len(expectedVersions))
	}

	for i, v := range versions {
		if v.Major != expectedVersions[i].Major || v.Minor != expectedVersions[i].Minor {
			t.Errorf("GetPHPAvailableVersions()[%d] = %v, want %v", i, v, expectedVersions[i])
		}
	}
}

func TestParseVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		versionStr string
		expected   LanguageVersion
		wantErr    bool
	}{
		{"valid version", "1.24", LanguageVersion{Major: 1, Minor: 24}, false},
		{"valid version 3.11", "3.11", LanguageVersion{Major: 3, Minor: 11}, false},
		{"valid version 8.1", "8.1", LanguageVersion{Major: 8, Minor: 1}, false},
		{"invalid format single", "1", LanguageVersion{}, true},
		{"invalid format triple", "1.2.3", LanguageVersion{}, true},
		{"invalid major", "x.24", LanguageVersion{}, true},
		{"invalid minor", "1.x", LanguageVersion{}, true},
		{"empty string", "", LanguageVersion{}, true},
		{"no dot", "124", LanguageVersion{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ParseVersion(tt.versionStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion(%q) error = %v, wantErr %v", tt.versionStr, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result.Major != tt.expected.Major || result.Minor != tt.expected.Minor {
					t.Errorf("ParseVersion(%q) = %v, want %v", tt.versionStr, result, tt.expected)
				}
			}
		})
	}
}

func TestValidateGoVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		version LanguageVersion
		wantErr bool
	}{
		{"valid version 1.24", LanguageVersion{Major: 1, Minor: 24}, false},
		{"valid version 1.25", LanguageVersion{Major: 1, Minor: 25}, false},
		{"invalid version 1.23", LanguageVersion{Major: 1, Minor: 23}, true},
		{"invalid version 1.26", LanguageVersion{Major: 1, Minor: 26}, true},
		{"invalid version 2.0", LanguageVersion{Major: 2, Minor: 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateGoVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGoVersion(%v) error = %v, wantErr %v", tt.version, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePythonVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		version LanguageVersion
		wantErr bool
	}{
		{"valid version 3.11", LanguageVersion{Major: 3, Minor: 11}, false},
		{"valid version 3.12", LanguageVersion{Major: 3, Minor: 12}, false},
		{"valid version 3.13", LanguageVersion{Major: 3, Minor: 13}, false},
		{"valid version 3.14", LanguageVersion{Major: 3, Minor: 14}, false},
		{"invalid version 3.10", LanguageVersion{Major: 3, Minor: 10}, true},
		{"invalid version 3.15", LanguageVersion{Major: 3, Minor: 15}, true},
		{"invalid version 2.7", LanguageVersion{Major: 2, Minor: 7}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePythonVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePythonVersion(%v) error = %v, wantErr %v", tt.version, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePHPVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		version LanguageVersion
		wantErr bool
	}{
		{"valid version 8.0", LanguageVersion{Major: 8, Minor: 0}, false},
		{"valid version 8.1", LanguageVersion{Major: 8, Minor: 1}, false},
		{"valid version 8.2", LanguageVersion{Major: 8, Minor: 2}, false},
		{"valid version 8.3", LanguageVersion{Major: 8, Minor: 3}, false},
		{"invalid version 7.4", LanguageVersion{Major: 7, Minor: 4}, true},
		{"invalid version 8.4", LanguageVersion{Major: 8, Minor: 4}, true},
		{"invalid version 9.0", LanguageVersion{Major: 9, Minor: 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePHPVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePHPVersion(%v) error = %v, wantErr %v", tt.version, err, tt.wantErr)
			}
		})
	}
}

func TestGoUsesAny(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		version  LanguageVersion
		expected bool
	}{
		{"go 1.18", LanguageVersion{Major: 1, Minor: 18}, true},
		{"go 1.19", LanguageVersion{Major: 1, Minor: 19}, true},
		{"go 1.24", LanguageVersion{Major: 1, Minor: 24}, true},
		{"go 2.0", LanguageVersion{Major: 2, Minor: 0}, true},
		{"go 1.17", LanguageVersion{Major: 1, Minor: 17}, false},
		{"go 1.16", LanguageVersion{Major: 1, Minor: 16}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.version.GoUsesAny()
			if result != tt.expected {
				t.Errorf("GoUsesAny() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPythonUsesModernTypeHints(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		version  LanguageVersion
		expected bool
	}{
		{"python 3.8", LanguageVersion{Major: 3, Minor: 8}, true},
		{"python 3.9", LanguageVersion{Major: 3, Minor: 9}, true},
		{"python 3.11", LanguageVersion{Major: 3, Minor: 11}, true},
		{"python 3.7", LanguageVersion{Major: 3, Minor: 7}, false},
		{"python 2.7", LanguageVersion{Major: 2, Minor: 7}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.version.PythonUsesModernTypeHints()
			if result != tt.expected {
				t.Errorf("PythonUsesModernTypeHints() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetGoEmptyInterface(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		version  LanguageVersion
		expected string
	}{
		{"go 1.18+", LanguageVersion{Major: 1, Minor: 18}, "any"},
		{"go 1.24", LanguageVersion{Major: 1, Minor: 24}, "any"},
		{"go 2.0", LanguageVersion{Major: 2, Minor: 0}, "any"},
		{"go 1.17", LanguageVersion{Major: 1, Minor: 17}, "interface{}"},
		{"go 1.16", LanguageVersion{Major: 1, Minor: 16}, "interface{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.version.GetGoEmptyInterface()
			if result != tt.expected {
				t.Errorf("GetGoEmptyInterface() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetGoVersionString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		version  LanguageVersion
		expected string
	}{
		{"go 1.24", LanguageVersion{Major: 1, Minor: 24}, "go 1.24"},
		{"go 1.25", LanguageVersion{Major: 1, Minor: 25}, "go 1.25"},
		{"go 2.0", LanguageVersion{Major: 2, Minor: 0}, "go 2.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.version.GetGoVersionString()
			if result != tt.expected {
				t.Errorf("GetGoVersionString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPHPUsesTypedProperties(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		version  LanguageVersion
		expected bool
	}{
		{"php 7.4", LanguageVersion{Major: 7, Minor: 4}, true},
		{"php 8.0", LanguageVersion{Major: 8, Minor: 0}, true},
		{"php 8.1", LanguageVersion{Major: 8, Minor: 1}, true},
		{"php 7.3", LanguageVersion{Major: 7, Minor: 3}, false},
		{"php 7.0", LanguageVersion{Major: 7, Minor: 0}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.version.PHPUsesTypedProperties()
			if result != tt.expected {
				t.Errorf("PHPUsesTypedProperties() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPHPUsesEnums(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		version  LanguageVersion
		expected bool
	}{
		{"php 8.1", LanguageVersion{Major: 8, Minor: 1}, true},
		{"php 8.2", LanguageVersion{Major: 8, Minor: 2}, true},
		{"php 8.3", LanguageVersion{Major: 8, Minor: 3}, true},
		{"php 8.0", LanguageVersion{Major: 8, Minor: 0}, false},
		{"php 7.4", LanguageVersion{Major: 7, Minor: 4}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.version.PHPUsesEnums()
			if result != tt.expected {
				t.Errorf("PHPUsesEnums() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPHPUsesReadonlyProperties(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		version  LanguageVersion
		expected bool
	}{
		{"php 8.1", LanguageVersion{Major: 8, Minor: 1}, true},
		{"php 8.2", LanguageVersion{Major: 8, Minor: 2}, true},
		{"php 8.3", LanguageVersion{Major: 8, Minor: 3}, true},
		{"php 8.0", LanguageVersion{Major: 8, Minor: 0}, false},
		{"php 7.4", LanguageVersion{Major: 7, Minor: 4}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.version.PHPUsesReadonlyProperties()
			if result != tt.expected {
				t.Errorf("PHPUsesReadonlyProperties() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetPHPVersionString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		version  LanguageVersion
		expected string
	}{
		{"php 8.0", LanguageVersion{Major: 8, Minor: 0}, "^8.0"},
		{"php 8.1", LanguageVersion{Major: 8, Minor: 1}, "^8.1"},
		{"php 8.2", LanguageVersion{Major: 8, Minor: 2}, "^8.2"},
		{"php 8.3", LanguageVersion{Major: 8, Minor: 3}, "^8.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.version.GetPHPVersionString()
			if result != tt.expected {
				t.Errorf("GetPHPVersionString() = %q, want %q", result, tt.expected)
			}
		})
	}
}
