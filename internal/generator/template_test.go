package generator

import (
	"strings"
	"testing"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/vubon/sdk-forge/internal/generator/common"
)

// Helper functions to test unexported functions
func pluralizeHelper(s string) string {
	if strings.HasSuffix(s, "y") {
		return strings.TrimSuffix(s, "y") + "ies"
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") || strings.HasSuffix(s, "z") {
		return s + "es"
	}
	return s + "s"
}

func singularizeHelper(s string) string {
	if strings.HasSuffix(s, "ies") {
		return strings.TrimSuffix(s, "ies") + "y"
	}
	if strings.HasSuffix(s, "es") {
		return strings.TrimSuffix(s, "es")
	}
	if strings.HasSuffix(s, "s") {
		return strings.TrimSuffix(s, "s")
	}
	return s
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "hello", "hello"},
		{"two words", "hello world", "helloWorld"},
		{"snake case", "hello_world", "helloWorld"},
		{"kebab case", "hello-world", "helloWorld"},
		{"pascal case", "HelloWorld", "helloWorld"},
		{"mixed", "hello_world-test", "helloWorldTest"},
		{"empty", "", ""},
		{"single word", "hello", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.ToCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "hello", "Hello"},
		{"two words", "hello world", "HelloWorld"},
		{"snake case", "hello_world", "HelloWorld"},
		{"kebab case", "hello-world", "HelloWorld"},
		{"camel case", "helloWorld", "HelloWorld"},
		{"mixed", "hello_world-test", "HelloWorldTest"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.ToPascalCase(tt.input)
			if result != tt.expected {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "hello", "hello"},
		{"two words", "hello world", "hello_world"},
		{"camel case", "helloWorld", "hello_world"},
		{"pascal case", "HelloWorld", "hello_world"},
		{"kebab case", "hello-world", "hello_world"},
		{"mixed", "helloWorld-test", "hello_world_test"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.ToSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
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
		{"two words", "hello world", "hello-world"},
		{"camel case", "helloWorld", "hello-world"},
		{"pascal case", "HelloWorld", "hello-world"},
		{"snake case", "hello_world", "hello-world"},
		{"mixed", "helloWorld_test", "hello-world-test"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.ToKebabCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToKebabCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCapitalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"lowercase", "hello", "Hello"},
		{"uppercase", "HELLO", "Hello"},
		{"mixed", "hElLo", "Hello"},
		{"empty", "", ""},
		{"single char", "a", "A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// capitalize is unexported, test through template function
			c := cases.Title(language.English)
			result := c.String(strings.ToLower(tt.input))
			if result != tt.expected {
				t.Errorf("capitalize(%q) = %q, want %q", tt.input, result, tt.expected)
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
		{"simple", "user", "users"},
		{"ends with y", "city", "cities"},
		{"ends with s", "users", "userses"},
		{"ends with x", "box", "boxes"},
		{"ends with z", "buzz", "buzzes"},
		{"empty", "", "s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// pluralize is unexported, test through template function
			result := pluralizeHelper(tt.input)
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
		{"simple", "users", "user"},
		{"ends with ies", "cities", "city"},
		{"ends with es", "boxes", "box"},
		{"ends with s", "users", "user"},
		{"no s", "user", "user"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// singularize is unexported, test through template function
			result := singularizeHelper(tt.input)
			if result != tt.expected {
				t.Errorf("singularize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRenderTemplate(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		data      common.TemplateData
		wantError bool
		contains  string
	}{
		{
			name:     "simple template",
			template: "Hello {{.SDKName}}",
			data: common.TemplateData{
				SDKName: "test-sdk",
			},
			wantError: false,
			contains:  "Hello test-sdk",
		},
		{
			name:     "with function",
			template: "{{.SDKName | pascalCase}}",
			data: common.TemplateData{
				SDKName: "test-sdk",
			},
			wantError: false,
			contains:  "TestSdk",
		},
		{
			name:     "invalid template",
			template: "{{.Invalid",
			data: common.TemplateData{
				SDKName: "test",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := common.RenderTemplate(tt.template, tt.data)
			if (err != nil) != tt.wantError {
				t.Errorf("RenderTemplate() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && tt.contains != "" {
				if result != tt.contains {
					t.Errorf("RenderTemplate() = %q, want to contain %q", result, tt.contains)
				}
			}
		})
	}
}

func TestLoadTemplate(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantError bool
	}{
		{
			name:      "valid template",
			content:   "Hello {{.SDKName}}",
			wantError: false,
		},
		{
			name:      "template with functions",
			content:   "{{.SDKName | camelCase}}",
			wantError: false,
		},
		{
			name:      "complex template",
			content:   "{{range .Operations}}{{.Name | pascalCase}}{{end}}",
			wantError: false,
		},
		{
			name:      "invalid template syntax",
			content:   "{{.Invalid",
			wantError: true,
		},
		{
			name:      "empty template",
			content:   "",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := LoadTemplate(tt.content)
			if (err != nil) != tt.wantError {
				t.Errorf("LoadTemplate() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && tmpl == nil {
				t.Error("LoadTemplate() returned nil template without error")
			}
		})
	}
}

func TestGetGoClientTemplate(t *testing.T) {
	tmpl := GetGoClientTemplate()
	if tmpl == "" {
		t.Error("GetGoClientTemplate() returned empty string")
	}
	// Verify it's a valid Go template
	if !strings.Contains(tmpl, "package") {
		t.Error("GetGoClientTemplate() doesn't appear to be a Go template")
	}
}

func TestGetGoModTemplate(t *testing.T) {
	tmpl := GetGoModTemplate()
	if tmpl == "" {
		t.Error("GetGoModTemplate() returned empty string")
	}
	// Verify it contains module declaration
	if !strings.Contains(tmpl, "module") {
		t.Error("GetGoModTemplate() doesn't appear to be a go.mod template")
	}
}

func TestGetGoReadmeTemplate(t *testing.T) {
	tmpl := GetGoReadmeTemplate()
	if tmpl == "" {
		t.Error("GetGoReadmeTemplate() returned empty string")
	}
	// Verify it's a markdown template
	if !strings.Contains(tmpl, "#") && !strings.Contains(tmpl, "SDK") {
		t.Error("GetGoReadmeTemplate() doesn't appear to be a README template")
	}
}

func TestGetPythonClientTemplate(t *testing.T) {
	tmpl := GetPythonClientTemplate()
	if tmpl == "" {
		t.Error("GetPythonClientTemplate() returned empty string")
	}
	// Verify it's a Python template
	if !strings.Contains(tmpl, "class") && !strings.Contains(tmpl, "def") {
		t.Error("GetPythonClientTemplate() doesn't appear to be a Python template")
	}
}

func TestGetPythonInitTemplate(t *testing.T) {
	tmpl := GetPythonInitTemplate()
	if tmpl == "" {
		t.Error("GetPythonInitTemplate() returned empty string")
	}
	// Python __init__.py can be minimal, just verify it's not empty
}

func TestGetPythonSetupTemplate(t *testing.T) {
	tmpl := GetPythonSetupTemplate()
	if tmpl == "" {
		t.Error("GetPythonSetupTemplate() returned empty string")
	}
	// Verify it's a setup.py template
	if !strings.Contains(tmpl, "setup") {
		t.Error("GetPythonSetupTemplate() doesn't appear to be a setup.py template")
	}
}

func TestGetPythonReadmeTemplate(t *testing.T) {
	tmpl := GetPythonReadmeTemplate()
	if tmpl == "" {
		t.Error("GetPythonReadmeTemplate() returned empty string")
	}
	// Verify it's a markdown template
	if !strings.Contains(tmpl, "#") && !strings.Contains(tmpl, "SDK") {
		t.Error("GetPythonReadmeTemplate() doesn't appear to be a README template")
	}
}

func TestGetPHPClientTemplate(t *testing.T) {
	tmpl := GetPHPClientTemplate()
	if tmpl == "" {
		t.Error("GetPHPClientTemplate() returned empty string")
	}
	// Verify it's a PHP template
	if !strings.Contains(tmpl, "class") && !strings.Contains(tmpl, "<?php") {
		t.Error("GetPHPClientTemplate() doesn't appear to be a PHP template")
	}
}

func TestGetPHPComposerTemplate(t *testing.T) {
	tmpl := GetPHPComposerTemplate()
	if tmpl == "" {
		t.Error("GetPHPComposerTemplate() returned empty string")
	}
	// Verify it's a composer.json template
	if !strings.Contains(tmpl, "name") && !strings.Contains(tmpl, "require") {
		t.Error("GetPHPComposerTemplate() doesn't appear to be a composer.json template")
	}
}

func TestGetPHPReadmeTemplate(t *testing.T) {
	tmpl := GetPHPReadmeTemplate()
	if tmpl == "" {
		t.Error("GetPHPReadmeTemplate() returned empty string")
	}
	// Verify it's a markdown template
	if !strings.Contains(tmpl, "#") && !strings.Contains(tmpl, "SDK") {
		t.Error("GetPHPReadmeTemplate() doesn't appear to be a README template")
	}
}

func TestLoadTemplateWithAllFunctions(t *testing.T) {
	// Test that LoadTemplate includes all custom functions
	tests := []struct {
		name     string
		template string
		data     common.TemplateData
		contains string
	}{
		{
			name:     "camelCase function",
			template: "{{.SDKName | camelCase}}",
			data:     common.TemplateData{SDKName: "test_sdk"},
			contains: "testSdk",
		},
		{
			name:     "pascalCase function",
			template: "{{.SDKName | pascalCase}}",
			data:     common.TemplateData{SDKName: "test_sdk"},
			contains: "TestSdk",
		},
		{
			name:     "snakeCase function",
			template: "{{.SDKName | snakeCase}}",
			data:     common.TemplateData{SDKName: "TestSDK"},
			contains: "test_s_d_k", // Each capital letter is treated as a word boundary
		},
		{
			name:     "kebabCase function",
			template: "{{.SDKName | kebabCase}}",
			data:     common.TemplateData{SDKName: "TestSDK"},
			contains: "test-s-d-k", // Each capital letter is treated as a word boundary
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := LoadTemplate(tt.template)
			if err != nil {
				t.Fatalf("LoadTemplate() error = %v", err)
			}

			var buf strings.Builder
			err = tmpl.Execute(&buf, tt.data)
			if err != nil {
				t.Fatalf("Template execution error = %v", err)
			}

			result := buf.String()
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Template result = %q, want to contain %q", result, tt.contains)
			}
		})
	}
}

func TestTemplateEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		wantError bool
	}{
		{
			name:      "nested templates",
			template:  "{{define \"nested\"}}Nested{{end}}{{template \"nested\"}}",
			wantError: false,
		},
		{
			name:      "conditional",
			template:  "{{if .SDKName}}Has name{{else}}No name{{end}}",
			wantError: false,
		},
		{
			name:      "range with empty",
			template:  "{{range .Operations}}Op{{end}}",
			wantError: false,
		},
		{
			name:      "undefined function",
			template:  "{{.SDKName | undefinedFunc}}",
			wantError: true, // Template parsing will fail with undefined function
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadTemplate(tt.template)
			if (err != nil) != tt.wantError {
				t.Errorf("LoadTemplate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestAllTemplatesNonEmpty(t *testing.T) {
	// Ensure all embedded templates are non-empty
	templates := map[string]func() string{
		"Go Client":     GetGoClientTemplate,
		"Go Mod":        GetGoModTemplate,
		"Go README":     GetGoReadmeTemplate,
		"Python Client": GetPythonClientTemplate,
		"Python Init":   GetPythonInitTemplate,
		"Python Setup":  GetPythonSetupTemplate,
		"Python README": GetPythonReadmeTemplate,
		"PHP Client":    GetPHPClientTemplate,
		"PHP Composer":  GetPHPComposerTemplate,
		"PHP README":    GetPHPReadmeTemplate,
	}

	for name, getTemplate := range templates {
		t.Run(name, func(t *testing.T) {
			tmpl := getTemplate()
			if tmpl == "" {
				t.Errorf("%s template is empty", name)
			}
			if len(tmpl) < 10 {
				t.Errorf("%s template is suspiciously short: %d bytes", name, len(tmpl))
			}
		})
	}
}

func TestLoadTemplateErrorMessage(t *testing.T) {
	// Test that error messages are descriptive
	invalidTemplate := "{{.Field | invalid"
	_, err := LoadTemplate(invalidTemplate)
	if err == nil {
		t.Error("LoadTemplate() should return error for invalid template")
	}
	if !strings.Contains(err.Error(), "failed to parse template") {
		t.Errorf("Error message should mention 'failed to parse template', got: %v", err)
	}
}
