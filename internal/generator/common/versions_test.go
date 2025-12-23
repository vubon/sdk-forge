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
