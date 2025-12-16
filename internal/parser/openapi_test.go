package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseOpenAPI_File(t *testing.T) {
	// Create a temporary OpenAPI file
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "test.yaml")
	schemaContent := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      summary: Test endpoint
      responses:
        '200':
          description: Success
`
	// #nosec G306 -- 0644 is appropriate for test files
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("Failed to create test schema file: %v", err)
	}

	doc, err := ParseOpenAPI(schemaPath)
	if err != nil {
		t.Fatalf("ParseOpenAPI() error = %v", err)
	}

	if doc == nil {
		t.Fatal("ParseOpenAPI() returned nil document")
	}

	if doc.Info == nil {
		t.Fatal("ParseOpenAPI() returned document with nil Info")
	}

	if doc.Info.Title != "Test API" {
		t.Errorf("ParseOpenAPI() Info.Title = %q, want %q", doc.Info.Title, "Test API")
	}

	if doc.Info.Version != "1.0.0" {
		t.Errorf("ParseOpenAPI() Info.Version = %q, want %q", doc.Info.Version, "1.0.0")
	}
}

func TestParseOpenAPI_InvalidFile(t *testing.T) {
	_, err := ParseOpenAPI("/nonexistent/file.yaml")
	if err == nil {
		t.Error("ParseOpenAPI() with invalid file should return error")
	}
}

func TestValidateOpenAPI(t *testing.T) {
	// Create a valid OpenAPI document
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "valid.yaml")
	schemaContent := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      summary: Test endpoint
      responses:
        '200':
          description: Success
`
	// #nosec G306 -- 0644 is appropriate for test files
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("Failed to create test schema file: %v", err)
	}

	doc, err := ParseOpenAPI(schemaPath)
	if err != nil {
		t.Fatalf("ParseOpenAPI() error = %v", err)
	}

	result, err := ValidateOpenAPI(doc, false)
	if err != nil {
		t.Fatalf("ValidateOpenAPI() error = %v", err)
	}

	if result == nil {
		t.Fatal("ValidateOpenAPI() returned nil result")
	}

	// Note: kin-openapi validation may find some issues even with valid schemas
	// We check that major blocking errors are not present
	majorErrors := 0
	for _, e := range result.Errors {
		if e.Severity == "major" {
			majorErrors++
		}
	}
	if majorErrors > 0 {
		t.Errorf("ValidateOpenAPI() found %d major errors, expected 0. Errors: %v", majorErrors, result.Errors)
	}
}

func TestValidateOpenAPI_MissingRequiredFields(t *testing.T) {
	// Create an invalid OpenAPI document (missing info)
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "invalid.yaml")
	schemaContent := `openapi: 3.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: Success
`
	// #nosec G306 -- 0644 is appropriate for test files
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("Failed to create test schema file: %v", err)
	}

	doc, err := ParseOpenAPI(schemaPath)
	if err != nil {
		t.Fatalf("ParseOpenAPI() error = %v", err)
	}

	result, err := ValidateOpenAPI(doc, false)
	if err != nil {
		t.Fatalf("ValidateOpenAPI() error = %v", err)
	}

	if len(result.Errors) == 0 {
		t.Error("ValidateOpenAPI() should find errors for missing required fields")
	}
}

func TestFormatValidationErrors(t *testing.T) {
	result := &ParseResult{
		Errors: []ValidationError{
			{Severity: "major", Message: "Missing required field 'info'", Path: "/info"},
			{Severity: "minor", Message: "Missing description", Path: "/info/description"},
		},
		Warnings: []ValidationError{
			{Severity: "minor", Message: "Deprecated feature", Path: "/paths"},
		},
	}

	formatted := FormatValidationErrors(result)
	if formatted == "" {
		t.Error("FormatValidationErrors() should return formatted error message")
	}

	if !contains(formatted, "major") {
		t.Error("FormatValidationErrors() should include major errors")
	}
}

func contains(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	if s == substr {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
