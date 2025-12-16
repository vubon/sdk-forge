package http

import (
	"testing"
)

func TestGetDefaultLibrary(t *testing.T) {
	tests := []struct {
		name     string
		language string
		expected string
	}{
		{"python default", "python", "requests"},
		{"python alias", "py", "requests"},
		{"go default", "go", "nethttp"},
		{"go alias", "golang", "nethttp"},
		{"php default", "php", "guzzle"},
		{"javascript default", "javascript", "axios"},
		{"javascript alias", "js", "axios"},
		{"typescript default", "typescript", "axios"},
		{"typescript alias", "ts", "axios"},
		{"invalid language", "rust", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDefaultLibrary(tt.language)
			if result != tt.expected {
				t.Errorf("GetDefaultLibrary(%q) = %q, want %q", tt.language, result, tt.expected)
			}
		})
	}
}

func TestIsValidLibrary(t *testing.T) {
	tests := []struct {
		name     string
		language string
		httpLib  string
		expected bool
	}{
		{"python requests", "python", "requests", true},
		{"python httpx", "python", "httpx", true},
		{"python aiohttp", "python", "aiohttp", true},
		{"python invalid", "python", "invalid", false},
		{"go nethttp", "go", "nethttp", true},
		{"go resty", "go", "resty", true},
		{"go invalid", "go", "requests", false},
		{"php guzzle", "php", "guzzle", true},
		{"php curl", "php", "curl", true},
		{"php invalid", "php", "invalid", false},
		{"js axios", "javascript", "axios", true},
		{"js fetch", "javascript", "fetch", true},
		{"js invalid", "javascript", "invalid", false},
		{"invalid language", "rust", "requests", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidLibrary(tt.language, tt.httpLib)
			if result != tt.expected {
				t.Errorf("IsValidLibrary(%q, %q) = %v, want %v", tt.language, tt.httpLib, result, tt.expected)
			}
		})
	}
}

func TestValidateLibrary(t *testing.T) {
	tests := []struct {
		name      string
		language  string
		httpLib   string
		wantError bool
	}{
		{"valid python requests", "python", "requests", false},
		{"invalid python library", "python", "invalid", true},
		{"valid go nethttp", "go", "nethttp", false},
		{"invalid go library", "go", "requests", true},
		{"valid php guzzle", "php", "guzzle", false},
		{"invalid php library", "php", "invalid", true},
		{"valid js axios", "javascript", "axios", false},
		{"invalid js library", "javascript", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLibrary(tt.language, tt.httpLib)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateLibrary(%q, %q) error = %v, wantError %v", tt.language, tt.httpLib, err, tt.wantError)
			}
		})
	}
}

func TestValidateOrGetDefault(t *testing.T) {
	tests := []struct {
		name      string
		language  string
		httpLib   string
		expected  string
		wantError bool
	}{
		{"with library provided", "python", "httpx", "httpx", false},
		{"without library (use default)", "python", "", "requests", false},
		{"invalid library", "python", "invalid", "", true},
		{"go with library", "go", "resty", "resty", false},
		{"go without library", "go", "", "nethttp", false},
		{"invalid language", "rust", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateOrGetDefault(tt.language, tt.httpLib)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateOrGetDefault(%q, %q) error = %v, wantError %v", tt.language, tt.httpLib, err, tt.wantError)
				return
			}
			if !tt.wantError && result != tt.expected {
				t.Errorf("ValidateOrGetDefault(%q, %q) = %q, want %q", tt.language, tt.httpLib, result, tt.expected)
			}
		})
	}
}

func TestGetValidLibraries(t *testing.T) {
	tests := []struct {
		name     string
		language string
		minCount int
	}{
		{"python libraries", "python", 4},
		{"go libraries", "go", 3},
		{"php libraries", "php", 3},
		{"js libraries", "javascript", 4},
		{"invalid language", "rust", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetValidLibraries(tt.language)
			if len(result) < tt.minCount {
				t.Errorf("GetValidLibraries(%q) returned %d libraries, want at least %d", tt.language, len(result), tt.minCount)
			}
		})
	}
}

func TestGetLibraryConfig(t *testing.T) {
	tests := []struct {
		name      string
		language  string
		httpLib   string
		wantError bool
	}{
		{"valid python requests", "python", "requests", false},
		{"valid python httpx", "python", "httpx", false},
		{"valid go nethttp", "go", "nethttp", false},
		{"valid go resty", "go", "resty", false},
		{"invalid library", "python", "invalid", true},
		{"invalid language", "rust", "requests", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := GetLibraryConfig(tt.language, tt.httpLib)
			if (err != nil) != tt.wantError {
				t.Errorf("GetLibraryConfig(%q, %q) error = %v, wantError %v", tt.language, tt.httpLib, err, tt.wantError)
				return
			}
			if !tt.wantError {
				if config == nil {
					t.Errorf("GetLibraryConfig(%q, %q) returned nil config", tt.language, tt.httpLib)
					return
				}
				if config.Import == "" {
					t.Errorf("GetLibraryConfig(%q, %q) returned config with empty Import", tt.language, tt.httpLib)
				}
			}
		})
	}
}
