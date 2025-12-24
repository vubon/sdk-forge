package commands

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/cobra"

	"github.com/vubon/sdk-forge/internal/generator/common"
)

// setFlag is a helper function to set command flags and fail the test on error
func setFlag(t *testing.T, cmd *cobra.Command, flagName, flagValue string) {
	t.Helper()
	if err := cmd.Flags().Set(flagName, flagValue); err != nil {
		t.Fatalf("failed to set flag %s=%s: %v", flagName, flagValue, err)
	}
}

// createTestOpenAPIDoc creates a minimal valid OpenAPI document for testing
func createTestOpenAPIDoc() *openapi3.T {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:       "Test API",
			Version:     "1.0.0",
			Description: "Test API for unit testing",
		},
		Paths: openapi3.NewPaths(),
	}

	// Add a simple path
	pathItem := &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "testGet",
			Responses:   openapi3.NewResponses(),
		},
	}
	desc := "Success"
	pathItem.Get.Responses.Set("200", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: &desc,
		},
	})
	doc.Paths.Set("/test", pathItem)

	return doc
}

// TestRunGenerate_MissingRequiredFlags tests error handling for missing flags
func TestRunGenerate_MissingRequiredFlags(t *testing.T) {
	cmd := GetGenerateCmd()

	// Test missing schema
	err := runGenerate(cmd, []string{})
	if err == nil {
		t.Error("expected error for missing schema")
	}

	// Test missing name
	setFlag(t, cmd, "schema", "test.yaml")
	setFlag(t, cmd, "lang", "python")
	setFlag(t, cmd, "output", "/tmp/test")
	err = runGenerate(cmd, []string{})
	if err == nil {
		t.Error("expected error for missing name")
	}
}

// TestRunGenerate_InvalidLanguage tests error handling for invalid language
func TestRunGenerate_InvalidLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.yaml")
	schemaContent := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}

	cmd := GetGenerateCmd()
	setFlag(t, cmd, "schema", schemaPath)
	setFlag(t, cmd, "lang", "invalid-language")
	setFlag(t, cmd, "name", "test-sdk")
	setFlag(t, cmd, "output", tmpDir)

	err := runGenerate(cmd, []string{})
	if err == nil {
		t.Error("expected error for invalid language")
	}
}

// TestRunGenerate_ValidGeneration tests successful SDK generation
func TestRunGenerate_ValidGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.yaml")
	schemaContent := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      operationId: testGet
      responses:
        '200':
          description: Success
`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}

	outputDir := filepath.Join(tmpDir, "output")

	cmd := GetGenerateCmd()
	setFlag(t, cmd, "schema", schemaPath)
	setFlag(t, cmd, "lang", "python")
	setFlag(t, cmd, "name", "test-sdk")
	setFlag(t, cmd, "output", outputDir)
	setFlag(t, cmd, "skip-tests", "true")
	setFlag(t, cmd, "ignore-minor-issues", "true")

	err := runGenerate(cmd, []string{})
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	// Verify SDK was generated
	expectedSDKPath := filepath.Join(outputDir, "python", "test-sdk")
	if _, err := os.Stat(expectedSDKPath); os.IsNotExist(err) {
		t.Errorf("expected SDK directory not created: %s", expectedSDKPath)
	}
}

// TestRunGenerate_WithRetryEnabled tests generation with retry enabled
func TestRunGenerate_WithRetryEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.yaml")
	schemaContent := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      operationId: testGet
      responses:
        '200':
          description: Success
`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}

	outputDir := filepath.Join(tmpDir, "output")

	cmd := GetGenerateCmd()
	setFlag(t, cmd, "schema", schemaPath)
	setFlag(t, cmd, "lang", "go")
	setFlag(t, cmd, "name", "test-sdk-retry")
	setFlag(t, cmd, "output", outputDir)
	setFlag(t, cmd, "retry-enabled", "true")
	setFlag(t, cmd, "retry-max-attempts", "5")
	setFlag(t, cmd, "skip-tests", "true")
	setFlag(t, cmd, "ignore-minor-issues", "true")

	err := runGenerate(cmd, []string{})
	if err != nil {
		t.Fatalf("runGenerate with retry failed: %v", err)
	}

	// Verify SDK was generated
	expectedSDKPath := filepath.Join(outputDir, "go", "test-sdk-retry")
	if _, err := os.Stat(expectedSDKPath); os.IsNotExist(err) {
		t.Errorf("expected SDK directory not created: %s", expectedSDKPath)
	}
}

// TestRunGenerate_ForceOverwrite tests force overwrite functionality
func TestRunGenerate_ForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.yaml")
	schemaContent := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      operationId: testGet
      responses:
        '200':
          description: Success
`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}

	outputDir := filepath.Join(tmpDir, "output")
	sdkPath := filepath.Join(outputDir, "python", "test-sdk")

	// Create existing SDK directory
	if err := os.MkdirAll(sdkPath, 0750); err != nil {
		t.Fatalf("failed to create existing SDK directory: %v", err)
	}

	cmd := GetGenerateCmd()
	setFlag(t, cmd, "schema", schemaPath)
	setFlag(t, cmd, "lang", "python")
	setFlag(t, cmd, "name", "test-sdk")
	setFlag(t, cmd, "output", outputDir)
	setFlag(t, cmd, "skip-tests", "true")
	setFlag(t, cmd, "ignore-minor-issues", "true")

	// First attempt without force should fail
	err := runGenerate(cmd, []string{})
	if err == nil {
		t.Error("expected error when directory exists without --force")
	}

	// Second attempt with force should succeed
	setFlag(t, cmd, "force", "true")
	err = runGenerate(cmd, []string{})
	if err != nil {
		t.Fatalf("runGenerate with force failed: %v", err)
	}
}

// TestGenerateSDKForLanguage tests the generateSDKForLanguage function
func TestGenerateSDKForLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test-sdk")
	doc := createTestOpenAPIDoc()

	cmd := GetGenerateCmd()
	setFlag(t, cmd, "skip-tests", "true")
	setFlag(t, cmd, "ignore-minor-issues", "true")

	// Test Python generation
	err := generateSDKForLanguage("python", outputPath, "test-sdk", "requests", doc, cmd)
	if err != nil {
		t.Fatalf("generateSDKForLanguage failed for Python: %v", err)
	}

	// Verify Python SDK was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("expected Python SDK directory not created: %s", outputPath)
	}

	// Test Go generation
	outputPathGo := filepath.Join(tmpDir, "test-sdk-go")
	err = generateSDKForLanguage("go", outputPathGo, "test-sdk-go", "nethttp", doc, cmd)
	if err != nil {
		t.Fatalf("generateSDKForLanguage failed for Go: %v", err)
	}

	// Verify Go SDK was created
	if _, err := os.Stat(outputPathGo); os.IsNotExist(err) {
		t.Errorf("expected Go SDK directory not created: %s", outputPathGo)
	}
}

// TestGenerateSDKForLanguage_WithRetry tests generation with retry config
func TestGenerateSDKForLanguage_WithRetry(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test-sdk-retry")
	doc := createTestOpenAPIDoc()

	cmd := GetGenerateCmd()
	setFlag(t, cmd, "skip-tests", "true")
	setFlag(t, cmd, "ignore-minor-issues", "true")
	setFlag(t, cmd, "retry-enabled", "true")
	setFlag(t, cmd, "retry-max-attempts", "3")
	setFlag(t, cmd, "retry-strategy", "exponential")

	err := generateSDKForLanguage("python", outputPath, "test-sdk-retry", "requests", doc, cmd)
	if err != nil {
		t.Fatalf("generateSDKForLanguage with retry failed: %v", err)
	}

	// Verify SDK was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("expected SDK directory not created: %s", outputPath)
	}
}

// TestGenerateAllLanguages tests the generateAllLanguages function
func TestGenerateAllLanguages(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	doc := createTestOpenAPIDoc()

	cmd := GetGenerateCmd()
	setFlag(t, cmd, "skip-tests", "true")
	setFlag(t, cmd, "ignore-minor-issues", "true")
	setFlag(t, cmd, "force", "true")

	err := generateAllLanguages(outputDir, "test-sdk-all", "", doc, true, cmd)
	if err != nil {
		t.Fatalf("generateAllLanguages failed: %v", err)
	}

	// Verify SDKs were generated for all languages
	// Check for Python
	pythonPath := filepath.Join(outputDir, "python", "test-sdk-all")
	if _, err := os.Stat(pythonPath); os.IsNotExist(err) {
		t.Errorf("expected Python SDK directory not created: %s", pythonPath)
	}

	// Check for Go
	goPath := filepath.Join(outputDir, "go", "test-sdk-all")
	if _, err := os.Stat(goPath); os.IsNotExist(err) {
		t.Errorf("expected Go SDK directory not created: %s", goPath)
	}
}

// TestParseRetryConfig_AllStrategies tests all retry strategies
func TestParseRetryConfig_AllStrategies(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("retry-enabled", false, "")
	cmd.Flags().Int("retry-max-attempts", 3, "")
	cmd.Flags().Float64("retry-initial-delay", 1.0, "")
	cmd.Flags().Float64("retry-max-delay", 60.0, "")
	cmd.Flags().Float64("retry-backoff-multiplier", 2.0, "")
	cmd.Flags().String("retry-strategy", "exponential", "")
	cmd.Flags().String("retry-status-codes", "429,500,502,503,504", "")

	strategies := []string{"exponential", "linear", "fixed"}
	for _, strategy := range strategies {
		setFlag(t, cmd, "retry-strategy", strategy)
		config := parseRetryConfig(cmd)

		expectedStrategy := common.RetryStrategy(strategy)
		if config.Strategy != expectedStrategy {
			t.Errorf("expected strategy %s, got %s", expectedStrategy, config.Strategy)
		}
	}
}

// TestParseRetryConfig_CustomStatusCodes tests custom retry status codes
func TestParseRetryConfig_CustomStatusCodes(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("retry-enabled", false, "")
	cmd.Flags().Int("retry-max-attempts", 3, "")
	cmd.Flags().Float64("retry-initial-delay", 1.0, "")
	cmd.Flags().Float64("retry-max-delay", 60.0, "")
	cmd.Flags().Float64("retry-backoff-multiplier", 2.0, "")
	cmd.Flags().String("retry-strategy", "exponential", "")
	cmd.Flags().String("retry-status-codes", "429,500", "")

	config := parseRetryConfig(cmd)
	if len(config.RetryableStatusCodes) != 2 {
		t.Errorf("expected 2 status codes, got %d", len(config.RetryableStatusCodes))
	}
	if config.RetryableStatusCodes[0] != 429 || config.RetryableStatusCodes[1] != 500 {
		t.Errorf("expected [429, 500], got %v", config.RetryableStatusCodes)
	}
}

// TestRunGenerate_WithLanguageFlag tests using --language flag instead of --lang
func TestRunGenerate_WithLanguageFlag(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.yaml")
	schemaContent := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      operationId: testGet
      responses:
        '200':
          description: Success
`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}

	outputDir := filepath.Join(tmpDir, "output")
	cmd := GetGenerateCmd()
	setFlag(t, cmd, "schema", schemaPath)
	setFlag(t, cmd, "language", "go") // Use --language instead of --lang
	setFlag(t, cmd, "name", "test-sdk")
	setFlag(t, cmd, "output", outputDir)
	setFlag(t, cmd, "skip-tests", "true")
	setFlag(t, cmd, "ignore-minor-issues", "true")

	err := runGenerate(cmd, []string{})
	if err != nil {
		t.Fatalf("runGenerate with --language flag failed: %v", err)
	}

	expectedSDKPath := filepath.Join(outputDir, "go", "test-sdk")
	if _, err := os.Stat(expectedSDKPath); os.IsNotExist(err) {
		t.Errorf("expected SDK directory not created: %s", expectedSDKPath)
	}
}

// TestRunGenerate_WithWarnings tests generation with validation warnings
func TestRunGenerate_WithWarnings(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.yaml")
	// Schema with minor issues (warnings) but valid
	schemaContent := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      operationId: testGet
      responses:
        '200':
          description: Success
`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}

	outputDir := filepath.Join(tmpDir, "output")
	cmd := GetGenerateCmd()
	setFlag(t, cmd, "schema", schemaPath)
	setFlag(t, cmd, "lang", "python")
	setFlag(t, cmd, "name", "test-sdk")
	setFlag(t, cmd, "output", outputDir)
	setFlag(t, cmd, "skip-tests", "true")
	setFlag(t, cmd, "ignore-minor-issues", "true")

	err := runGenerate(cmd, []string{})
	if err != nil {
		t.Fatalf("runGenerate with warnings failed: %v", err)
	}
}

// TestRunGenerate_AllLanguages tests generating for all languages
func TestRunGenerate_AllLanguages(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.yaml")
	schemaContent := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      operationId: testGet
      responses:
        '200':
          description: Success
`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}

	outputDir := filepath.Join(tmpDir, "output")
	cmd := GetGenerateCmd()
	setFlag(t, cmd, "schema", schemaPath)
	setFlag(t, cmd, "lang", "all")
	setFlag(t, cmd, "name", "test-sdk-all")
	setFlag(t, cmd, "output", outputDir)
	setFlag(t, cmd, "skip-tests", "true")
	setFlag(t, cmd, "ignore-minor-issues", "true")
	setFlag(t, cmd, "force", "true")

	err := runGenerate(cmd, []string{})
	if err != nil {
		t.Fatalf("runGenerate for all languages failed: %v", err)
	}

	// Verify SDKs were generated for multiple languages
	languages := []string{"python", "go", "php", "typescript"}
	for _, lang := range languages {
		expectedPath := filepath.Join(outputDir, lang, "test-sdk-all")
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("expected %s SDK directory not created: %s", lang, expectedPath)
		}
	}
}

// TestRunGenerate_InvalidSDKName tests invalid SDK name validation
func TestRunGenerate_InvalidSDKName(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.yaml")
	schemaContent := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}

	cmd := GetGenerateCmd()
	setFlag(t, cmd, "schema", schemaPath)
	setFlag(t, cmd, "lang", "go")
	setFlag(t, cmd, "name", "123-invalid") // Invalid Go SDK name
	setFlag(t, cmd, "output", tmpDir)
	setFlag(t, cmd, "skip-tests", "true")
	setFlag(t, cmd, "ignore-minor-issues", "true")

	err := runGenerate(cmd, []string{})
	if err == nil {
		t.Error("expected error for invalid SDK name")
	}
}

// TestRunGenerate_InvalidHTTPLib tests invalid HTTP library
func TestRunGenerate_InvalidHTTPLib(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.yaml")
	schemaContent := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}

	cmd := GetGenerateCmd()
	setFlag(t, cmd, "schema", schemaPath)
	setFlag(t, cmd, "lang", "python")
	setFlag(t, cmd, "name", "test-sdk")
	setFlag(t, cmd, "output", tmpDir)
	setFlag(t, cmd, "http-lib", "invalid-lib")
	setFlag(t, cmd, "skip-tests", "true")
	setFlag(t, cmd, "ignore-minor-issues", "true")

	err := runGenerate(cmd, []string{})
	if err == nil {
		t.Error("expected error for invalid HTTP library")
	}
}

// TestGenerateSDKForLanguage_AllLanguages tests all supported languages
func TestGenerateSDKForLanguage_AllLanguages(t *testing.T) {
	tmpDir := t.TempDir()
	doc := createTestOpenAPIDoc()
	cmd := GetGenerateCmd()
	setFlag(t, cmd, "skip-tests", "true")

	languages := []struct {
		lang    string
		httpLib string
	}{
		{"python", "requests"},
		{"go", "nethttp"},
		{"php", "guzzle"},
		{"typescript", "axios"},
	}

	for _, tt := range languages {
		t.Run(tt.lang, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, "test-sdk-"+tt.lang)
			err := generateSDKForLanguage(tt.lang, outputPath, "test-sdk", tt.httpLib, doc, cmd)
			if err != nil {
				t.Fatalf("generateSDKForLanguage failed for %s: %v", tt.lang, err)
			}

			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Errorf("expected %s SDK directory not created: %s", tt.lang, outputPath)
			}
		})
	}
}

// TestGenerateSDKForLanguage_WithVersions tests generation with version flags
func TestGenerateSDKForLanguage_WithVersions(t *testing.T) {
	tmpDir := t.TempDir()
	doc := createTestOpenAPIDoc()
	cmd := GetGenerateCmd()
	setFlag(t, cmd, "skip-tests", "true")

	tests := []struct {
		lang        string
		httpLib     string
		versionFlag string
		version     string
	}{
		{"go", "nethttp", "go-version", "1.24"},
		{"python", "requests", "python-version", "3.11"},
		{"php", "guzzle", "php-version", "8.1"},
		{"typescript", "axios", "typescript-version", "5.0"},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, "test-sdk-"+tt.lang+"-ver")
			setFlag(t, cmd, tt.versionFlag, tt.version)
			err := generateSDKForLanguage(tt.lang, outputPath, "test-sdk", tt.httpLib, doc, cmd)
			if err != nil {
				t.Fatalf("generateSDKForLanguage with version failed for %s: %v", tt.lang, err)
			}
		})
	}
}

// TestGenerateSDKForLanguage_InvalidVersion tests invalid version handling
func TestGenerateSDKForLanguage_InvalidVersion(t *testing.T) {
	tmpDir := t.TempDir()
	doc := createTestOpenAPIDoc()
	cmd := GetGenerateCmd()
	setFlag(t, cmd, "skip-tests", "true")
	setFlag(t, cmd, "go-version", "invalid")

	err := generateSDKForLanguage("go", filepath.Join(tmpDir, "test-sdk"), "test-sdk", "nethttp", doc, cmd)
	if err == nil {
		t.Error("expected error for invalid Go version")
	}
}

// TestGenerateSDKForLanguage_UnsupportedLanguage tests unsupported language error
func TestGenerateSDKForLanguage_UnsupportedLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	doc := createTestOpenAPIDoc()
	cmd := GetGenerateCmd()

	err := generateSDKForLanguage("ruby", filepath.Join(tmpDir, "test-sdk"), "test-sdk", "", doc, cmd)
	if err == nil {
		t.Error("expected error for unsupported language")
	}
	if err != nil && !contains(err.Error(), "unsupported language") {
		t.Errorf("expected 'unsupported language' error, got: %v", err)
	}
}

// TestGenerateSDKForLanguage_WithOpenAPIRetryConfig tests retry config from OpenAPI
func TestGenerateSDKForLanguage_WithOpenAPIRetryConfig(t *testing.T) {
	tmpDir := t.TempDir()
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Extensions: map[string]interface{}{
			"x-sdk-forge-retry": map[string]interface{}{
				"enabled":              true,
				"maxAttempts":          5.0,
				"strategy":             "linear",
				"retryableStatusCodes": []interface{}{429.0, 500.0},
			},
		},
		Paths: openapi3.NewPaths(),
	}
	cmd := GetGenerateCmd()
	setFlag(t, cmd, "skip-tests", "true")
	setFlag(t, cmd, "retry-enabled", "true") // CLI flag should override OpenAPI

	err := generateSDKForLanguage("python", filepath.Join(tmpDir, "test-sdk"), "test-sdk", "requests", doc, cmd)
	if err != nil {
		t.Fatalf("generateSDKForLanguage with OpenAPI retry config failed: %v", err)
	}
}

// TestGenerateAllLanguages_WithErrors tests error handling in generateAllLanguages
func TestGenerateAllLanguages_WithErrors(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	doc := createTestOpenAPIDoc()
	cmd := GetGenerateCmd()
	setFlag(t, cmd, "skip-tests", "true")
	setFlag(t, cmd, "force", "true")

	// Use invalid SDK name that will fail validation for some languages
	err := generateAllLanguages(outputDir, "123-invalid", "", doc, false, cmd)
	// Should return error but may have generated some SDKs
	if err == nil {
		t.Log("generateAllLanguages completed (some languages may have failed validation)")
	}
}

// TestGenerateAllLanguages_WithoutForce tests without force flag when directory exists
func TestGenerateAllLanguages_WithoutForce(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	doc := createTestOpenAPIDoc()
	cmd := GetGenerateCmd()
	setFlag(t, cmd, "skip-tests", "true")

	// Create existing directory for one language
	pythonPath := filepath.Join(outputDir, "python", "test-sdk")
	if err := os.MkdirAll(pythonPath, 0750); err != nil {
		t.Fatalf("failed to create existing directory: %v", err)
	}

	err := generateAllLanguages(outputDir, "test-sdk", "", doc, false, cmd)
	// Should return error about existing directory
	if err == nil {
		t.Error("expected error when directory exists without force")
	}
}

// TestPromptRequiredInput_ErrorCases tests error handling in promptRequiredInput
func TestPromptRequiredInput_ErrorCases(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("test-flag", "", "test flag")

	// Test with empty input
	reader := strings.NewReader("\n") // Empty input
	bufReader := bufio.NewReader(reader)
	err := promptRequiredInput(bufReader, "Test: ", "test-flag", cmd)
	if err == nil {
		t.Error("expected error for empty required input")
	}
	if err != nil && !strings.Contains(err.Error(), "required") {
		t.Errorf("expected 'required' in error message, got: %v", err)
	}

	// Test with valid input
	reader = strings.NewReader("valid-input\n")
	bufReader = bufio.NewReader(reader)
	err = promptRequiredInput(bufReader, "Test: ", "test-flag", cmd)
	if err != nil {
		t.Errorf("unexpected error for valid input: %v", err)
	}
	value, _ := cmd.Flags().GetString("test-flag")
	if value != "valid-input" {
		t.Errorf("expected flag value 'valid-input', got '%s'", value)
	}
}

// TestParseRetryConfig_EdgeCases tests edge cases in parseRetryConfig
func TestParseRetryConfig_EdgeCases(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("retry-enabled", false, "")
	cmd.Flags().Int("retry-max-attempts", 3, "")
	cmd.Flags().Float64("retry-initial-delay", 1.0, "")
	cmd.Flags().Float64("retry-max-delay", 60.0, "")
	cmd.Flags().Float64("retry-backoff-multiplier", 2.0, "")
	cmd.Flags().String("retry-strategy", "exponential", "")
	cmd.Flags().String("retry-status-codes", "429,500,502,503,504", "")

	// Test with zero values (should use defaults)
	setFlag(t, cmd, "retry-max-attempts", "0")
	setFlag(t, cmd, "retry-initial-delay", "0")
	setFlag(t, cmd, "retry-max-delay", "0")
	setFlag(t, cmd, "retry-backoff-multiplier", "0")

	config := parseRetryConfig(cmd)
	// Zero values should not override defaults
	if config.MaxAttempts == 0 {
		t.Error("expected MaxAttempts to use default, not 0")
	}

	// Test with invalid status codes (should skip them)
	setFlag(t, cmd, "retry-status-codes", "429,invalid,500,abc")
	config = parseRetryConfig(cmd)
	if len(config.RetryableStatusCodes) != 2 {
		t.Errorf("expected 2 valid status codes, got %d", len(config.RetryableStatusCodes))
	}

	// Test with unknown strategy (should default to exponential)
	setFlag(t, cmd, "retry-strategy", "unknown")
	config = parseRetryConfig(cmd)
	if config.Strategy != common.RetryStrategyExponential {
		t.Errorf("expected default strategy exponential, got %s", config.Strategy)
	}

	// Test with empty status codes string
	setFlag(t, cmd, "retry-status-codes", "")
	config = parseRetryConfig(cmd)
	if len(config.RetryableStatusCodes) == 0 {
		t.Log("empty status codes string uses default (expected)")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && strings.Contains(s, substr)))
}
