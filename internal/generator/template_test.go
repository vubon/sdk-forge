package generator

import (
	"testing"
)

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
			result := toCamelCase(tt.input)
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
			result := toPascalCase(tt.input)
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
			result := toSnakeCase(tt.input)
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
			result := toKebabCase(tt.input)
			if result != tt.expected {
				t.Errorf("toKebabCase(%q) = %q, want %q", tt.input, result, tt.expected)
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
			result := capitalize(tt.input)
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
			result := pluralize(tt.input)
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
			result := singularize(tt.input)
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
		data      TemplateData
		wantError bool
		contains  string
	}{
		{
			name:     "simple template",
			template: "Hello {{.SDKName}}",
			data: TemplateData{
				SDKName: "test-sdk",
			},
			wantError: false,
			contains:  "Hello test-sdk",
		},
		{
			name:     "with function",
			template: "{{.SDKName | pascalCase}}",
			data: TemplateData{
				SDKName: "test-sdk",
			},
			wantError: false,
			contains:  "TestSdk",
		},
		{
			name:     "invalid template",
			template: "{{.Invalid",
			data: TemplateData{
				SDKName: "test",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderTemplate(tt.template, tt.data)
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
