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

func TestGetImplementedLanguages(t *testing.T) {
	langs := GetImplementedLanguages()
	expected := map[string]bool{"python": true, "go": true, "php": true, "typescript": true, "javascript": true, "ruby": true}
	if len(langs) != len(expected) {
		t.Errorf("GetImplementedLanguages() returned %d languages, want %d", len(langs), len(expected))
	}
	for _, lang := range langs {
		if !expected[lang] {
			t.Errorf("GetImplementedLanguages() returned unexpected language: %s", lang)
		}
	}
}

func TestValidateLanguage_AllSpecialCase(t *testing.T) {
	if err := ValidateLanguage("all"); err != nil {
		t.Errorf("ValidateLanguage('all') = %v, want nil", err)
	}
	if err := ValidateLanguage("ALL"); err != nil {
		t.Errorf("ValidateLanguage('ALL') = %v, want nil", err)
	}
}

func TestValidateSDKName_EmptyName(t *testing.T) {
	_, err := ValidateSDKName("", "python")
	if err == nil {
		t.Error("ValidateSDKName should return error for empty name")
	}
}

func TestValidateSDKName_WhitespaceName(t *testing.T) {
	_, err := ValidateSDKName("   ", "go")
	if err == nil {
		t.Error("ValidateSDKName should return error for whitespace name")
	}
}

func TestValidateSDKName_ValidNames(t *testing.T) {
	validNames := []string{"sdk", "sdk-123", "sdk_abc", "SDKName"}
	for _, name := range validNames {
		result, err := ValidateSDKName(name, "go")
		if err != nil {
			t.Errorf("ValidateSDKName(%q) returned error: %v", name, err)
		}
		if result != name {
			t.Errorf("ValidateSDKName(%q) returned %q, want %q", name, result, name)
		}
	}
}

// Additional test: ValidateSDKName with invalid characters
func TestValidateSDKName_InvalidCharacters(t *testing.T) {
	invalidNames := []string{"sdk@123", "sdk!", "sdk name", "sdk#"}
	for _, name := range invalidNames {
		result, err := ValidateSDKName(name, "go")
		if err == nil {
			t.Errorf("ValidateSDKName(%q) should return error for invalid characters", name)
		}
		if result != "" {
			t.Errorf("ValidateSDKName(%q) should return empty string on error", name)
		}
	}
}

// Additional test: ValidateLanguage with whitespace and case
func TestValidateLanguage_WhitespaceAndCase(t *testing.T) {
	cases := []struct {
		input string
		valid bool
	}{
		{" PYTHON ", true},
		{" Go ", true},
		{" PHP ", true},
		{" JAVASCRIPT ", true},
		{" TYPESCRIPT ", true},
		{" Rust ", false},
	}
	for _, c := range cases {
		err := ValidateLanguage(c.input)
		if (err == nil) != c.valid {
			t.Errorf("ValidateLanguage(%q) valid=%v, got error: %v", c.input, c.valid, err)
		}
	}
}
