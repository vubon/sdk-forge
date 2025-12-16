package validator

import (
	"testing"
)

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"python lowercase", "python", "python"},
		{"python alias", "py", "python"},
		{"go lowercase", "go", "go"},
		{"go alias", "golang", "go"},
		{"javascript lowercase", "javascript", "javascript"},
		{"javascript alias", "js", "javascript"},
		{"typescript lowercase", "typescript", "typescript"},
		{"typescript alias", "ts", "typescript"},
		{"php lowercase", "php", "php"},
		{"uppercase python", "PYTHON", "python"},
		{"mixed case", "PyThOn", "python"},
		{"with spaces", "  python  ", "python"},
		{"unknown language", "rust", "rust"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeLanguage(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeLanguage(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateLanguage(t *testing.T) {
	tests := []struct {
		name      string
		language  string
		wantError bool
	}{
		{"valid python", "python", false},
		{"valid python alias", "py", false},
		{"valid go", "go", false},
		{"valid go alias", "golang", false},
		{"valid php", "php", false},
		{"valid javascript", "javascript", false},
		{"valid javascript alias", "js", false},
		{"valid typescript", "typescript", false},
		{"valid typescript alias", "ts", false},
		{"invalid language", "rust", true},
		{"empty string", "", true},
		{"unknown", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLanguage(tt.language)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateLanguage(%q) error = %v, wantError %v", tt.language, err, tt.wantError)
			}
		})
	}
}

func TestValidateSDKName(t *testing.T) {
	tests := []struct {
		name      string
		sdkName   string
		language  string
		wantError bool
	}{
		{"valid name", "my-sdk", "python", false},
		{"valid with underscores", "my_sdk", "python", false},
		{"valid alphanumeric", "mySdk123", "python", false},
		{"empty name", "", "python", true},
		{"valid go name", "my-client", "go", false},
		{"valid php name", "my-api", "php", false},
		{"valid js name", "my-sdk", "javascript", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateSDKName(tt.sdkName, tt.language)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateSDKName(%q, %q) error = %v, wantError %v", tt.sdkName, tt.language, err, tt.wantError)
				return
			}
			if !tt.wantError && result == "" {
				t.Errorf("ValidateSDKName(%q, %q) returned empty string, expected non-empty", tt.sdkName, tt.language)
			}
		})
	}
}
