package parser

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
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

func TestParseOpenAPI_URL(t *testing.T) {
	// Start a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		yaml := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: Success`
		_, err := fmt.Fprint(w, yaml)
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	doc, err := ParseOpenAPI(ts.URL)
	if err != nil {
		t.Fatalf("ParseOpenAPI() error = %v", err)
	}
	if doc == nil || doc.Info == nil || doc.Info.Title != "Test API" {
		t.Error("ParseOpenAPI() did not parse from URL correctly")
	}
}

func TestParseOpenAPI_URLNon200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer ts.Close()
	_, err := ParseOpenAPI(ts.URL)
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Error("ParseOpenAPI() should error on non-200 HTTP status")
	}
}

func TestParseOpenAPI_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "bad.yaml")
	err := os.WriteFile(file, []byte("not: valid: yaml: :"), 0644)
	if err != nil {
		t.Fatalf("Failed to write bad yaml: %v", err)
	}
	_, err = ParseOpenAPI(file)
	if err == nil || !strings.Contains(err.Error(), "failed to parse") {
		t.Error("ParseOpenAPI() should error on invalid YAML")
	}
}

func TestParseOpenAPI_FileReadError(t *testing.T) {
	// Try to open a directory as a file
	dir := t.TempDir()
	_, err := ParseOpenAPI(dir)
	if err == nil || !strings.Contains(err.Error(), "failed to read schema file") {
		t.Error("ParseOpenAPI() should error on file read error")
	}
}

func TestValidateOpenAPI_CustomRules(t *testing.T) {
	// Missing info
	doc := &openapi3.T{}
	res, _ := ValidateOpenAPI(doc, false)
	if len(res.Errors) == 0 {
		t.Error("ValidateOpenAPI() should catch missing info")
	} else {
		// Check if any error is about missing info
		found := false
		for _, e := range res.Errors {
			if e.Path == "/info" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ValidateOpenAPI() should catch missing info, got errors: %v", res.Errors)
		}
	}

	// Missing title
	doc = &openapi3.T{Info: &openapi3.Info{Version: "1.0.0"}, Paths: &openapi3.Paths{}}
	res, _ = ValidateOpenAPI(doc, false)
	found := false
	for _, e := range res.Errors {
		if e.Path == "/info/title" {
			found = true
		}
	}
	if !found {
		t.Error("ValidateOpenAPI() should catch missing info.title")
	}

	// Missing version
	doc = &openapi3.T{Info: &openapi3.Info{Title: "T"}, Paths: &openapi3.Paths{}}
	res, _ = ValidateOpenAPI(doc, false)
	found = false
	for _, e := range res.Errors {
		if e.Path == "/info/version" {
			found = true
		}
	}
	if !found {
		t.Error("ValidateOpenAPI() should catch missing info.version")
	}

	// Empty paths
	doc = &openapi3.T{Info: &openapi3.Info{Title: "T", Version: "1.0.0"}, Paths: &openapi3.Paths{}}
	res, _ = ValidateOpenAPI(doc, false)
	found = false
	for _, e := range res.Errors {
		if e.Path == "/paths" {
			found = true
		}
	}
	if !found {
		t.Error("ValidateOpenAPI() should catch empty paths")
	}

	// Minor: missing description, not ignored
	paths2 := openapi3.NewPaths()
	paths2.Set("/", &openapi3.PathItem{})
	doc = &openapi3.T{Info: &openapi3.Info{Title: "T", Version: "1.0.0"}, Paths: paths2}
	res, _ = ValidateOpenAPI(doc, false)
	found = false
	for _, e := range res.Errors {
		if e.Path == "/info/description" && e.Severity == "minor" {
			found = true
		}
	}
	if !found {
		t.Error("ValidateOpenAPI() should catch missing info.description as minor error")
	}

	// Minor: missing description, ignored
	res, _ = ValidateOpenAPI(doc, true)
	found = false
	for _, w := range res.Warnings {
		if w.Path == "/info/description" && w.Severity == "minor" {
			found = true
		}
	}
	if !found {
		t.Error("ValidateOpenAPI() should put missing info.description in warnings if ignoreMinor")
	}
}

func TestParseValidationErrors(t *testing.T) {
	major := parseValidationErrors(errors.New("field is required"))
	if len(major) == 0 || major[0].Severity != "major" {
		t.Error("parseValidationErrors() should mark 'required' as major")
	}
	minor := parseValidationErrors(errors.New("something else"))
	if len(minor) == 0 || minor[0].Severity != "minor" {
		t.Error("parseValidationErrors() should mark other errors as minor")
	}
}

func TestFormatValidationErrors_Empty(t *testing.T) {
	res := &ParseResult{}
	if FormatValidationErrors(res) != "" {
		t.Error("FormatValidationErrors() should return empty string for no errors/warnings")
	}
}

func TestFormatValidationErrors_Combinations(t *testing.T) {
	res := &ParseResult{
		Errors: []ValidationError{
			{Severity: "major", Message: "Major error", Path: "/a"},
			{Severity: "minor", Message: "Minor error", Path: "/b"},
		},
		Warnings: []ValidationError{
			{Severity: "minor", Message: "Warn", Path: "/c"},
		},
	}
	out := FormatValidationErrors(res)
	if !strings.Contains(out, "Major Issues") || !strings.Contains(out, "Minor Issues") || !strings.Contains(out, "Warnings") {
		t.Error("FormatValidationErrors() should format all error types")
	}
}
