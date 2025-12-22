package commands

import (
	"os"
	"path/filepath"
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
