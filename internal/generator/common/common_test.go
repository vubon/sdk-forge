package common

import (
	"strings"
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
		{
			name:        "template_with_multiple_functions",
			tmplContent: "{{camelCase .SDKName}} {{snakeCase .SDKName}}",
			data:        TemplateData{SDKName: "test-sdk"},
			wantErr:     false,
			contains:    "testSdk",
		},
		{
			name:        "template_with_kebab_case",
			tmplContent: "{{kebabCase .SDKName}}",
			data:        TemplateData{SDKName: "testSdk"},
			wantErr:     false,
			contains:    "test-sdk",
		},
		{
			name:        "template_with_capitalize",
			tmplContent: "{{capitalize .SDKName}}",
			data:        TemplateData{SDKName: "test"},
			wantErr:     false,
			contains:    "Test",
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

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "hello", "hello"},
		{"two_words", "hello world", "helloWorld"},
		{"snake_case", "hello_world", "helloWorld"},
		{"kebab_case", "hello-world", "helloWorld"},
		{"pascal_case", "HelloWorld", "helloWorld"},
		{"mixed", "helloWorld", "helloWorld"},
		{"with_numbers", "hello123world", "hello123world"},
		{"empty", "", ""},
		{"single_char", "a", "a"},
		{"uppercase", "HELLO", "hELLO"}, // splitWords splits uppercase into individual letters
		{"with_dots", "hello.world", "helloWorld"},
		{"camelCase_boundary", "helloWorldTest", "helloWorldTest"},
		{"multiple_uppercase", "XMLParser", "xMLParser"},
		{"starts_with_number", "123test", "123test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "hello", "hello"},
		{"two_words", "hello world", "hello-world"},
		{"snake_case", "hello_world", "hello-world"},
		{"pascal_case", "HelloWorld", "hello-world"},
		{"camel_case", "helloWorld", "hello-world"},
		{"with_numbers", "hello123world", "hello123world"},
		{"empty", "", ""},
		{"single_char", "a", "a"},
		{"uppercase", "HELLO", "h-e-l-l-o"}, // splitWords splits uppercase into individual letters
		{"with_dots", "hello.world", "hello-world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToKebabCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToKebabCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToTitleCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "hello", "Hello"},
		{"two_words", "hello world", "Hello World"},
		{"uppercase", "HELLO", "Hello"},
		{"mixed", "hELLo", "Hello"},
		{"empty", "", ""},
		{"single_char", "a", "A"},
		{"with_numbers", "hello123", "Hello123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// toTitleCase is not exported, test via template
			tmpl := "{{title .SDKName}}"
			result, err := RenderTemplate(tmpl, TemplateData{SDKName: tt.input})
			if err != nil {
				t.Fatalf("RenderTemplate() error = %v", err)
			}
			// Remove newlines and trim
			result = strings.TrimSpace(result)
			if result != tt.expected {
				t.Errorf("toTitleCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"ends_with_y", "city", "cities"},
		{"ends_with_s", "class", "classes"},
		{"ends_with_x", "box", "boxes"},
		{"ends_with_z", "buzz", "buzzes"},
		{"regular", "dog", "dogs"},
		{"empty", "", "s"},
		{"single_char", "a", "as"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// pluralize is not exported, test via template
			tmpl := "{{plural .SDKName}}"
			result, err := RenderTemplate(tmpl, TemplateData{SDKName: tt.input})
			if err != nil {
				t.Fatalf("RenderTemplate() error = %v", err)
			}
			result = strings.TrimSpace(result)
			if result != tt.expected {
				t.Errorf("pluralize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSingularize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"ends_with_ies", "cities", "city"},
		{"ends_with_es", "classes", "class"},
		{"ends_with_s", "dogs", "dog"},
		{"no_s", "dog", "dog"},
		{"empty", "", ""},
		{"single_char", "a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// singularize is not exported, test via template
			tmpl := "{{singular .SDKName}}"
			result, err := RenderTemplate(tmpl, TemplateData{SDKName: tt.input})
			if err != nil {
				t.Fatalf("RenderTemplate() error = %v", err)
			}
			result = strings.TrimSpace(result)
			if result != tt.expected {
				t.Errorf("singularize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCreateTestExtractedData(t *testing.T) {
	data := CreateTestExtractedData()
	if data == nil {
		t.Fatal("CreateTestExtractedData() returned nil")
	}
	if data.BaseURL != "https://api.example.com/v1" {
		t.Errorf("CreateTestExtractedData() BaseURL = %q, want %q", data.BaseURL, "https://api.example.com/v1")
	}
	if data.Title != "Test API" {
		t.Errorf("CreateTestExtractedData() Title = %q, want %q", data.Title, "Test API")
	}
	if data.Version != "1.0.0" {
		t.Errorf("CreateTestExtractedData() Version = %q, want %q", data.Version, "1.0.0")
	}
	if data.Operations == nil {
		t.Error("CreateTestExtractedData() Operations should not be nil")
	}
	if data.Schemas == nil {
		t.Error("CreateTestExtractedData() Schemas should not be nil")
	}
	if data.SecuritySchemes == nil {
		t.Error("CreateTestExtractedData() SecuritySchemes should not be nil")
	}
}

func TestCreateTestOpenAPIDoc(t *testing.T) {
	doc := CreateTestOpenAPIDoc()
	if doc == nil {
		t.Fatal("CreateTestOpenAPIDoc() returned nil")
	}
	if doc.OpenAPI != "3.0.0" {
		t.Errorf("CreateTestOpenAPIDoc() OpenAPI = %q, want %q", doc.OpenAPI, "3.0.0")
	}
	if doc.Info == nil {
		t.Fatal("CreateTestOpenAPIDoc() Info should not be nil")
	}
	if doc.Info.Title != "Test API" {
		t.Errorf("CreateTestOpenAPIDoc() Info.Title = %q, want %q", doc.Info.Title, "Test API")
	}
	if doc.Info.Version != "1.0.0" {
		t.Errorf("CreateTestOpenAPIDoc() Info.Version = %q, want %q", doc.Info.Version, "1.0.0")
	}
	if len(doc.Servers) == 0 {
		t.Error("CreateTestOpenAPIDoc() should have at least one server")
	}
	if doc.Servers[0].URL != "https://api.example.com/v1" {
		t.Errorf("CreateTestOpenAPIDoc() Server.URL = %q, want %q", doc.Servers[0].URL, "https://api.example.com/v1")
	}
}

func TestGetGoDefaultVersion(t *testing.T) {
	version := GetGoDefaultVersion()
	if version.Major != 1 || version.Minor != 24 {
		t.Errorf("GetGoDefaultVersion() = %v, want {Major: 1, Minor: 24}", version)
	}
}

func TestGetPythonDefaultVersion(t *testing.T) {
	version := GetPythonDefaultVersion()
	if version.Major != 3 || version.Minor != 11 {
		t.Errorf("GetPythonDefaultVersion() = %v, want {Major: 3, Minor: 11}", version)
	}
}

func TestGetPHPDefaultVersion(t *testing.T) {
	version := GetPHPDefaultVersion()
	if version.Major != 8 || version.Minor != 1 {
		t.Errorf("GetPHPDefaultVersion() = %v, want {Major: 8, Minor: 1}", version)
	}
}
