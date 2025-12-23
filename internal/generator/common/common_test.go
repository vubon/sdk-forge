package common

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestGetClientClassName(t *testing.T) {
	tests := []struct {
		name     string
		sdkName  string
		expected string
	}{
		{"simple", "petstore", "Petstore"},
		{"with_sdk_suffix", "petstore-sdk", "Petstore"},
		{"with_api_suffix", "petstore-api", "Petstore"},
		{"with_client_suffix", "petstore-client", "Petstore"},
		{"with_underscore_sdk", "petstore_sdk", "Petstore"},
		{"with_underscore_api", "petstore_api", "Petstore"},
		{"with_underscore_client", "petstore_client", "Petstore"},
		{"multiple_suffixes", "petstore-sdk-client", "PetstoreSdk"}, // Removes -client first, leaving -sdk
		{"pascal_case", "PetStore", "Petstore"},
		{"camel_case", "petStore", "Petstore"},
		{"already_clean", "Petstore", "Petstore"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetClientClassName(tt.sdkName)
			if result != tt.expected {
				t.Errorf("GetClientClassName(%q) = %q, want %q", tt.sdkName, result, tt.expected)
			}
		})
	}
}

func TestGroupOperationsByTag(t *testing.T) {
	tests := []struct {
		name       string
		operations []APIOperation
		expected   map[string]int // tag -> count
	}{
		{
			name: "operations_with_tags",
			operations: []APIOperation{
				{Method: "GET", Path: "/pets", Tags: []string{"pets"}},
				{Method: "POST", Path: "/pets", Tags: []string{"pets"}},
				{Method: "GET", Path: "/users", Tags: []string{"users"}},
			},
			expected: map[string]int{
				"pets":  2,
				"users": 1,
			},
		},
		{
			name: "operations_without_tags",
			operations: []APIOperation{
				{Method: "GET", Path: "/status", Tags: []string{}},
				{Method: "GET", Path: "/health", Tags: nil},
			},
			expected: map[string]int{
				"default": 2,
			},
		},
		{
			name:       "empty_operations",
			operations: []APIOperation{},
			expected:   map[string]int{},
		},
		{
			name: "mixed_tags",
			operations: []APIOperation{
				{Method: "GET", Path: "/pets", Tags: []string{"pets"}},
				{Method: "GET", Path: "/status", Tags: []string{}},
				{Method: "GET", Path: "/users", Tags: []string{"users"}},
			},
			expected: map[string]int{
				"pets":    1,
				"default": 1,
				"users":   1,
			},
		},
		{
			name: "multiple_tags_uses_first",
			operations: []APIOperation{
				{Method: "GET", Path: "/pets", Tags: []string{"pets", "animals"}},
			},
			expected: map[string]int{
				"pets": 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GroupOperationsByTag(tt.operations)

			// Check counts
			if len(result) != len(tt.expected) {
				t.Errorf("GroupOperationsByTag() returned %d tags, want %d", len(result), len(tt.expected))
			}

			for tag, expectedCount := range tt.expected {
				if count := len(result[tag]); count != expectedCount {
					t.Errorf("GroupOperationsByTag() tag %q has %d operations, want %d", tag, count, expectedCount)
				}
			}
		})
	}
}

func TestDetermineSDKVersion(t *testing.T) {
	tests := []struct {
		name          string
		extractedData *ExtractedData
		sdkVersion    string
		expected      string
	}{
		{
			name: "extracted_data_version_takes_priority",
			extractedData: &ExtractedData{
				Version: "2.0.0",
			},
			sdkVersion: "1.0.0",
			expected:   "2.0.0",
		},
		{
			name: "user_version_when_no_extracted",
			extractedData: &ExtractedData{
				Version: "",
			},
			sdkVersion: "1.5.0",
			expected:   "1.5.0",
		},
		{
			name: "default_when_nothing_provided",
			extractedData: &ExtractedData{
				Version: "",
			},
			sdkVersion: "",
			expected:   "1.0.0",
		},
		{
			name:          "nil_extracted_data",
			extractedData: nil,
			sdkVersion:    "1.2.0",
			expected:      "1.2.0",
		},
		{
			name:          "nil_extracted_data_no_version",
			extractedData: nil,
			sdkVersion:    "",
			expected:      "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineSDKVersion(tt.extractedData, tt.sdkVersion)
			if result != tt.expected {
				t.Errorf("DetermineSDKVersion() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"contains", "hello world", "world", true},
		{"not_contains", "hello world", "foo", false},
		{"exact_match", "hello", "hello", true},
		{"empty_string", "", "", true},
		{"empty_substr", "hello", "", true},
		{"substr_longer", "hi", "hello", false},
		{"case_sensitive", "Hello", "hello", false},
		{"at_start", "hello world", "hello", true},
		{"at_end", "hello world", "world", true},
		{"in_middle", "hello world", "lo wo", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("Contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

func TestLoadTemplate(t *testing.T) {
	tests := []struct {
		name        string
		tmplContent string
		wantErr     bool
	}{
		{
			name:        "valid_template",
			tmplContent: "Hello {{.Name}}",
			wantErr:     false,
		},
		{
			name:        "template_with_functions",
			tmplContent: "{{camelCase .Name}}",
			wantErr:     false,
		},
		{
			name:        "invalid_template",
			tmplContent: "{{.Name",
			wantErr:     true,
		},
		{
			name:        "empty_template",
			tmplContent: "",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := LoadTemplate(tt.tmplContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tmpl == nil {
				t.Error("LoadTemplate() returned nil template without error")
			}
		})
	}
}

func TestRenderTemplate(t *testing.T) {
	tests := []struct {
		name        string
		tmplContent string
		data        TemplateData
		wantErr     bool
		contains    string
	}{
		{
			name:        "simple_template",
			tmplContent: "Hello {{.SDKName}}",
			data:        TemplateData{SDKName: "test-sdk"},
			wantErr:     false,
			contains:    "test-sdk",
		},
		{
			name:        "template_with_function",
			tmplContent: "{{pascalCase .SDKName}}",
			data:        TemplateData{SDKName: "test-sdk"},
			wantErr:     false,
			contains:    "TestSdk",
		},
		{
			name:        "invalid_template",
			tmplContent: "{{.Name",
			data:        TemplateData{SDKName: "test"},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderTemplate(tt.tmplContent, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tt.contains != "" && !Contains(result, tt.contains) {
					t.Errorf("RenderTemplate() result = %q, should contain %q", result, tt.contains)
				}
			}
		})
	}
}

func TestGetOperationMethodName(t *testing.T) {
	tests := []struct {
		name     string
		op       APIOperation
		expected string
	}{
		{
			name: "with_operation_id",
			op: APIOperation{
				OperationID: "listPets",
				Method:      "GET",
				Path:        "/pets",
			},
			expected: "list_pets",
		},
		{
			name: "without_operation_id",
			op: APIOperation{
				Method: "GET",
				Path:   "/pets",
			},
			expected: "get_pets",
		},
		{
			name: "with_path_params",
			op: APIOperation{
				Method: "GET",
				Path:   "/pets/{id}",
			},
			expected: "get_pets",
		},
		{
			name: "nested_path",
			op: APIOperation{
				Method: "POST",
				Path:   "/users/{userId}/pets",
			},
			expected: "post_users_pets",
		},
		{
			name: "root_path",
			op: APIOperation{
				Method: "GET",
				Path:   "/",
			},
			expected: "get",
		},
		{
			name: "empty_path",
			op: APIOperation{
				Method: "GET",
				Path:   "",
			},
			expected: "get",
		},
		{
			name: "camel_case_operation_id",
			op: APIOperation{
				OperationID: "createNewPet",
				Method:      "POST",
				Path:        "/pets",
			},
			expected: "create_new_pet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetOperationMethodName(tt.op)
			if result != tt.expected {
				t.Errorf("GetOperationMethodName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractOpenAPIData(t *testing.T) {
	tests := []struct {
		name    string
		doc     *openapi3.T
		wantErr bool
		check   func(*ExtractedData) bool
	}{
		{
			name: "minimal_doc",
			doc: &openapi3.T{
				OpenAPI: "3.0.0",
				Info: &openapi3.Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
				Paths: openapi3.NewPaths(),
			},
			wantErr: false,
			check: func(data *ExtractedData) bool {
				return data.Title == "Test API" && data.Version == "1.0.0"
			},
		},
		{
			name: "with_server",
			doc: &openapi3.T{
				OpenAPI: "3.0.0",
				Info: &openapi3.Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
				Servers: openapi3.Servers{
					&openapi3.Server{
						URL: "https://api.example.com/v1",
					},
				},
				Paths: openapi3.NewPaths(),
			},
			wantErr: false,
			check: func(data *ExtractedData) bool {
				return data.BaseURL == "https://api.example.com/v1"
			},
		},
		{
			name: "no_server_default",
			doc: &openapi3.T{
				OpenAPI: "3.0.0",
				Info: &openapi3.Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
				Paths: openapi3.NewPaths(),
			},
			wantErr: false,
			check: func(data *ExtractedData) bool {
				return data.BaseURL == "https://api.example.com/v1"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractOpenAPIData(tt.doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractOpenAPIData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result == nil {
					t.Error("ExtractOpenAPIData() returned nil")
					return
				}
				if tt.check != nil && !tt.check(result) {
					t.Error("ExtractOpenAPIData() check failed")
				}
			}
		})
	}
}
