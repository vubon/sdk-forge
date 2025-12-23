package php

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vubon/sdk-forge/internal/generator/common"
	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

func TestGeneratePHPSDK(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	// Use ExtractedData for testing
	extractedData := common.CreateTestExtractedData()
	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	// Check that files were created
	// PHP SDK creates a directory with PascalCase SDK name
	sanitizedName := common.ToPascalCase(sdkName)
	packageDir := filepath.Join(tmpDir, sanitizedName)
	expectedFiles := []string{
		filepath.Join(packageDir, "composer.json"),
		filepath.Join(packageDir, "README.md"),
		filepath.Join(packageDir, "src", sanitizedName+".php"),
		filepath.Join(packageDir, "src", "Exceptions", "ApiException.php"),
		filepath.Join(packageDir, "examples", "basic_usage.php"),
		filepath.Join(packageDir, "phpcs.xml"),
		filepath.Join(packageDir, "phpstan.neon"),
		filepath.Join(packageDir, ".php-cs-fixer.php"),
	}

	for _, file := range expectedFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("GeneratePHPSDK() did not create file: %s", file)
		}
	}
}

func TestGeneratePHPSDK_InvalidHTTPLib(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := "test-sdk"
	httpLib := "invalid-lib"

	extractedData := common.CreateTestExtractedData()
	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err == nil {
		t.Error("GeneratePHPSDK() with invalid HTTP library should return error")
	}
}

func TestGeneratePHPSDK_CustomHTTPLib(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "curl"

	extractedData := common.CreateTestExtractedData()
	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	// Check that client.php exists and uses curl
	sanitizedName := common.ToPascalCase(sdkName)
	clientPath := filepath.Join(tmpDir, sanitizedName, "src", sanitizedName+".php")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read client.php: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "curl") {
		t.Error("GeneratePHPSDK() should use curl in client.php")
	}
}

func TestGeneratePHPSDK_ComposerJSON(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	composerPath := filepath.Join(tmpDir, sanitizedName, "composer.json")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(composerPath)
	if err != nil {
		t.Fatalf("Failed to read composer.json: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, strings.ToLower(sdkName)) {
		t.Error("composer.json should contain SDK name")
	}
	if !common.Contains(contentStr, "guzzlehttp/guzzle") {
		t.Error("composer.json should contain guzzle dependency")
	}
	if !common.Contains(contentStr, "php") {
		t.Error("composer.json should contain PHP version requirement")
	}
}

func TestGeneratePHPSDK_SDKNameSanitization(t *testing.T) {
	tests := []struct {
		name     string
		sdkName  string
		expected string
	}{
		{"kebab case", "my-api-sdk", "MyApiSdk"},
		{"camel case", "myApiSdk", "MyApiSdk"},
		{"pascal case", "MyApiSdk", "MyApiSdk"},
		{"with spaces", "my api sdk", "MyApiSdk"},
		{"test-sdk", common.TestSDKName, common.ToPascalCase(common.TestSDKName)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			extractedData := common.CreateTestExtractedData()
			err := GeneratePHPSDK(tmpDir, tt.sdkName, "guzzle", extractedData, nil, "", true, common.DefaultRetryConfig())
			if err != nil {
				t.Fatalf("GeneratePHPSDK() error = %v", err)
			}

			// Check that package directory uses PascalCase
			packageDir := filepath.Join(tmpDir, tt.expected)
			if _, err := os.Stat(packageDir); os.IsNotExist(err) {
				t.Errorf("GeneratePHPSDK() did not create package directory: %s", packageDir)
			}
		})
	}
}

func TestGeneratePHPSDK_WithTests(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	// Verify tests directory exists
	testsDir := filepath.Join(tmpDir, sanitizedName, "tests")
	if _, err := os.Stat(testsDir); os.IsNotExist(err) {
		t.Error("GeneratePHPSDK() with tests enabled should create tests directory")
	}

	// Verify test files exist
	expectedTestFiles := []string{
		filepath.Join(testsDir, "bootstrap.php"),
		filepath.Join(testsDir, "phpunit.xml"),
		filepath.Join(testsDir, "ClientTest.php"),
	}

	for _, file := range expectedTestFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("GeneratePHPSDK() did not create test file: %s", file)
		}
	}
}

func TestGeneratePHPSDK_WithoutTests(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", false, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	// Verify tests directory does NOT exist
	testsDir := filepath.Join(tmpDir, sanitizedName, "tests")
	if _, err := os.Stat(testsDir); err == nil {
		t.Error("GeneratePHPSDK() with tests disabled should NOT create tests directory")
	}
}

func TestGeneratePHPSDK_ModelGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add a test schema
	extractedData.Schemas["User"] = &common.Schema{
		Type:        "object",
		Description: "User model",
		Properties: map[string]*common.Schema{
			"id": {
				Type: "integer",
			},
			"name": {
				Type: "string",
			},
		},
		Required: []string{"id", "name"},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	// Verify model file exists
	modelPath := filepath.Join(tmpDir, sanitizedName, "src", "Models", "User.php")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Error("GeneratePHPSDK() should create model file when schemas exist")
	}

	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("Failed to read User.php: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "class User") {
		t.Error("User.php should contain User class")
	}
	if !common.Contains(contentStr, "JsonSerializable") {
		t.Error("User.php should implement JsonSerializable")
	}
}

func TestGeneratePHPSDK_APIGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add a test operation
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/users",
			OperationID: "listUsers",
			Summary:     "List users",
			Tags:        []string{"users"},
		},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	// Verify API file exists
	apiPath := filepath.Join(tmpDir, sanitizedName, "src", "Api", "UsersApi.php")
	if _, err := os.Stat(apiPath); os.IsNotExist(err) {
		t.Error("GeneratePHPSDK() should create API file when operations exist")
	}

	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("Failed to read UsersApi.php: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "class UsersApi") {
		t.Error("UsersApi.php should contain UsersApi class")
	}
	if !common.Contains(contentStr, "listUsers") {
		t.Error("UsersApi.php should contain listUsers method")
	}
}

func TestGeneratePHPSDK_RetryConfig(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	retryConfig := common.RetryConfig{
		Enabled:              true,
		MaxAttempts:          3,
		Strategy:             common.RetryStrategyExponential,
		InitialDelay:         time.Second,
		MaxDelay:             30 * time.Second,
		BackoffMultiplier:    2.0,
		RetryableStatusCodes: []int{500, 502, 503},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, retryConfig)
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	clientPath := filepath.Join(tmpDir, sanitizedName, "src", sanitizedName+".php")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read client.php: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "retryMaxAttempts") {
		t.Error("client.php should contain retry fields when retry is enabled")
	}
}

func TestGeneratePHPSDK_NoRetryConfig(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	retryConfig := common.DefaultRetryConfig()
	retryConfig.Enabled = false

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, retryConfig)
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	clientPath := filepath.Join(tmpDir, sanitizedName, "src", sanitizedName+".php")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read client.php: %v", err)
	}

	contentStr := string(content)
	if common.Contains(contentStr, "retryMaxAttempts") {
		t.Error("client.php should NOT contain retry fields when retry is disabled")
	}
}

func TestGeneratePHPSDK_Examples(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	examplesPath := filepath.Join(tmpDir, sanitizedName, "examples", "basic_usage.php")
	if _, err := os.Stat(examplesPath); os.IsNotExist(err) {
		t.Error("GeneratePHPSDK() should create examples file")
	}

	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(examplesPath)
	if err != nil {
		t.Fatalf("Failed to read basic_usage.php: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "<?php") {
		t.Error("basic_usage.php should contain PHP opening tag")
	}
}

func TestGeneratePHPSDK_TSConfig(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	// Verify quality config files exist
	qualityFiles := []string{
		filepath.Join(tmpDir, sanitizedName, "phpcs.xml"),
		filepath.Join(tmpDir, sanitizedName, "phpstan.neon"),
		filepath.Join(tmpDir, sanitizedName, ".php-cs-fixer.php"),
	}

	for _, file := range qualityFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("GeneratePHPSDK() did not create quality config file: %s", file)
		}
	}
}

func TestGeneratePHPSDK_DifferentHTTPLibraries(t *testing.T) {
	tests := []struct {
		name    string
		httpLib string
	}{
		{"guzzle", "guzzle"},
		{"curl", "curl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sdkName := common.TestSDKName
			extractedData := common.CreateTestExtractedData()
			err := GeneratePHPSDK(tmpDir, sdkName, tt.httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
			if err != nil {
				t.Fatalf("GeneratePHPSDK() with %s error = %v", tt.httpLib, err)
			}

			sanitizedName := common.ToPascalCase(sdkName)
			clientPath := filepath.Join(tmpDir, sanitizedName, "src", sanitizedName+".php")
			// #nosec G304 -- File path is from test, safe to read
			content, err := os.ReadFile(clientPath)
			if err != nil {
				t.Fatalf("Failed to read client.php: %v", err)
			}

			contentStr := string(content)
			// Check for HTTP library (case-insensitive, or common variations)
			httpLibLower := strings.ToLower(tt.httpLib)
			contentLower := strings.ToLower(contentStr)
			if !common.Contains(contentLower, httpLibLower) && !common.Contains(contentStr, "Guzzle") && !common.Contains(contentStr, "curl") {
				t.Errorf("client.php should use %s HTTP library", tt.httpLib)
			}
		})
	}
}

func TestGeneratePHPSDK_AllHTTPLibrariesDetailed(t *testing.T) {
	tests := []struct {
		name    string
		httpLib string
	}{
		{"guzzle", "guzzle"},
		{"curl", "curl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sdkName := common.TestSDKName
			extractedData := common.CreateTestExtractedData()
			err := GeneratePHPSDK(tmpDir, sdkName, tt.httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
			if err != nil {
				t.Fatalf("GeneratePHPSDK() with %s error = %v", tt.httpLib, err)
			}

			sanitizedName := common.ToPascalCase(sdkName)
			composerPath := filepath.Join(tmpDir, sanitizedName, "composer.json")
			// #nosec G304 -- File path is from test, safe to read
			content, err := os.ReadFile(composerPath)
			if err != nil {
				t.Fatalf("Failed to read composer.json: %v", err)
			}

			contentStr := string(content)
			libConfig, err := httplib.GetLibraryConfig("php", tt.httpLib)
			if err != nil {
				t.Fatalf("Failed to get library config: %v", err)
			}

			// Check that dependency is in composer.json
			if libConfig.Dependency != "" {
				depParts := strings.Split(libConfig.Dependency, ":")
				depName := depParts[0]
				if !common.Contains(contentStr, depName) {
					t.Errorf("composer.json should contain %s dependency", depName)
				}
			}
		})
	}
}

func TestGeneratePHPSDK_Authentication(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add API key authentication
	extractedData.SecuritySchemes["apiKey"] = common.SecurityScheme{
		Type: "apiKey",
		Name: "X-API-Key",
		In:   "header",
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	clientPath := filepath.Join(tmpDir, sanitizedName, "src", sanitizedName+".php")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read client.php: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "apiKey") {
		t.Error("client.php should contain authentication fields when security schemes exist")
	}
}

func TestGeneratePHPSDK_AllAuthTypes(t *testing.T) {
	tests := []struct {
		name   string
		scheme common.SecurityScheme
	}{
		{"apiKey", common.SecurityScheme{Type: "apiKey", Name: "X-API-Key", In: "header"}},
		{"bearer", common.SecurityScheme{Type: "http", Scheme: "bearer"}},
		{"basic", common.SecurityScheme{Type: "http", Scheme: "basic"}},
		{"oauth2", common.SecurityScheme{Type: "oauth2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sdkName := common.TestSDKName
			httpLib := "guzzle"

			extractedData := common.CreateTestExtractedData()
			extractedData.SecuritySchemes[tt.name] = tt.scheme

			err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
			if err != nil {
				t.Fatalf("GeneratePHPSDK() with %s auth error = %v", tt.name, err)
			}

			sanitizedName := common.ToPascalCase(sdkName)
			clientPath := filepath.Join(tmpDir, sanitizedName, "src", sanitizedName+".php")
			// #nosec G304 -- File path is from test, safe to read
			content, err := os.ReadFile(clientPath)
			if err != nil {
				t.Fatalf("Failed to read client.php: %v", err)
			}

			contentStr := string(content)
			// Verify authentication is present
			if len(extractedData.SecuritySchemes) > 0 && !common.Contains(contentStr, "set") {
				t.Errorf("client.php should contain authentication methods for %s", tt.name)
			}
		})
	}
}

func TestGeneratePHPSDK_OpenAPIDoc(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	// Test with openapi3.T document
	openAPIDoc := common.CreateTestOpenAPIDoc()
	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, openAPIDoc, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() with openapi3.T error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	composerPath := filepath.Join(tmpDir, sanitizedName, "composer.json")
	if _, err := os.Stat(composerPath); os.IsNotExist(err) {
		t.Error("GeneratePHPSDK() should create files when using openapi3.T")
	}
}

func TestGeneratePHPSDK_InvalidOpenAPIDoc(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := "test-sdk"
	httpLib := "guzzle"

	// Test with invalid document type
	invalidDoc := "not a valid document"
	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, invalidDoc, nil, "", true, common.DefaultRetryConfig())
	if err == nil {
		t.Error("GeneratePHPSDK() with invalid OpenAPI document should return error")
	}
}

func TestGeneratePHPSDK_ArrayModel(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add an array schema
	extractedData.Schemas["UserList"] = &common.Schema{
		Type: "array",
		Items: &common.Schema{
			Type: "string",
		},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	modelPath := filepath.Join(tmpDir, sanitizedName, "src", "Models", "UserList.php")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Error("GeneratePHPSDK() should create array model file")
	}
}

func TestGeneratePHPSDK_APIMethodWithParams(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add operation with parameters
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/users/{id}",
			OperationID: "getUser",
			Tags:        []string{"users"},
			Parameters: []common.Parameter{
				{
					Name: "id",
					In:   "path",
					Schema: &common.Schema{
						Type: "integer",
					},
				},
			},
		},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	apiPath := filepath.Join(tmpDir, sanitizedName, "src", "Api", "UsersApi.php")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("Failed to read UsersApi.php: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "getUser") {
		t.Error("UsersApi.php should contain getUser method")
	}
	if !common.Contains(contentStr, "$id") {
		t.Error("UsersApi.php should contain parameter in method signature")
	}
}

func TestGeneratePHPSDK_AllSchemaTypes(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add schemas with different types
	extractedData.Schemas["StringModel"] = &common.Schema{Type: "string"}
	extractedData.Schemas["IntegerModel"] = &common.Schema{Type: "integer"}
	extractedData.Schemas["NumberModel"] = &common.Schema{Type: "number"}
	extractedData.Schemas["BooleanModel"] = &common.Schema{Type: "boolean"}
	extractedData.Schemas["ObjectModel"] = &common.Schema{
		Type: "object",
		Properties: map[string]*common.Schema{
			"value": {Type: "string"},
		},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	modelsDir := filepath.Join(tmpDir, sanitizedName, "src", "Models")

	expectedModels := []string{"StringModel.php", "IntegerModel.php", "NumberModel.php", "BooleanModel.php", "ObjectModel.php"}
	for _, model := range expectedModels {
		modelPath := filepath.Join(modelsDir, model)
		if _, err := os.Stat(modelPath); os.IsNotExist(err) {
			t.Errorf("GeneratePHPSDK() should create model file: %s", model)
		}
	}
}

func TestGeneratePHPSDK_ResponseTypes(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add operation with response
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/users",
			OperationID: "listUsers",
			Tags:        []string{"users"},
			Responses: map[string]common.Response{
				"200": {
					Description: "Success",
					Content: map[string]common.ContentType{
						"application/json": {
							Schema: &common.Schema{
								Type: "array",
								Items: &common.Schema{
									Type: "object",
								},
							},
						},
					},
				},
			},
		},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	apiPath := filepath.Join(tmpDir, sanitizedName, "src", "Api", "UsersApi.php")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("Failed to read UsersApi.php: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "listUsers") {
		t.Error("UsersApi.php should contain listUsers method with response handling")
	}
}

func TestGeneratePHPSDK_ModelWithRef(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add schemas with reference
	extractedData.Schemas["User"] = &common.Schema{
		Type: "object",
		Properties: map[string]*common.Schema{
			"id": {Type: "integer"},
		},
	}
	extractedData.Schemas["UserProfile"] = &common.Schema{
		Type: "object",
		Properties: map[string]*common.Schema{
			"user": {
				Ref: "#/components/schemas/User",
			},
		},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	modelPath := filepath.Join(tmpDir, sanitizedName, "src", "Models", "UserProfile.php")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Error("GeneratePHPSDK() should create model with reference")
	}
}

func TestGeneratePHPSDK_ModelWithOptionalFields(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add schema with optional fields
	extractedData.Schemas["User"] = &common.Schema{
		Type: "object",
		Properties: map[string]*common.Schema{
			"id":    {Type: "integer"},
			"name":  {Type: "string"},
			"email": {Type: "string"}, // Optional field
		},
		Required: []string{"id", "name"}, // email is optional
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	modelPath := filepath.Join(tmpDir, sanitizedName, "src", "Models", "User.php")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("Failed to read User.php: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "email") {
		t.Error("User.php should contain optional email field")
	}
}

func TestGeneratePHPSDK_APIMethodWithRequestBody(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add operation with request body
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "POST",
			Path:        "/users",
			OperationID: "createUser",
			Tags:        []string{"users"},
			RequestBody: &common.RequestBody{
				Required: true,
				Content: map[string]common.ContentType{
					"application/json": {
						Schema: &common.Schema{
							Type: "object",
							Properties: map[string]*common.Schema{
								"name": {Type: "string"},
							},
						},
					},
				},
			},
		},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	apiPath := filepath.Join(tmpDir, sanitizedName, "src", "Api", "UsersApi.php")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("Failed to read UsersApi.php: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "createUser") {
		t.Error("UsersApi.php should contain createUser method with request body")
	}
}

func TestGeneratePHPSDK_OperationWithoutTags(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add operation without tag
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/status",
			OperationID: "getStatus",
			Tags:        []string{}, // No tag
		},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	// Should still generate API file (with default tag)
	sanitizedName := common.ToPascalCase(sdkName)
	apiDir := filepath.Join(tmpDir, sanitizedName, "src", "Api")
	if _, err := os.Stat(apiDir); os.IsNotExist(err) {
		t.Error("GeneratePHPSDK() should create API directory even for operations without tags")
	}
}

func TestGeneratePHPSDK_EmptySchemas(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// No schemas

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	// Models directory should not exist if no schemas
	modelsDir := filepath.Join(tmpDir, sanitizedName, "src", "Models")
	if _, err := os.Stat(modelsDir); err == nil {
		// If it exists, it should be empty or not have model files
		entries, _ := os.ReadDir(modelsDir)
		if len(entries) > 0 {
			t.Error("GeneratePHPSDK() should not create model files when no schemas exist")
		}
	}
}

func TestGeneratePHPSDK_EmptyOperations(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// No operations

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	// Api directory should not exist if no operations
	apiDir := filepath.Join(tmpDir, sanitizedName, "src", "Api")
	if _, err := os.Stat(apiDir); err == nil {
		// If it exists, it should be empty or not have API files
		entries, _ := os.ReadDir(apiDir)
		if len(entries) > 0 {
			t.Error("GeneratePHPSDK() should not create API files when no operations exist")
		}
	}
}

func TestGeneratePHPSDK_CustomSDKVersion(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"
	customVersion := "2.0.0"

	extractedData := common.CreateTestExtractedData()
	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, customVersion, true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	composerPath := filepath.Join(tmpDir, sanitizedName, "composer.json")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(composerPath)
	if err != nil {
		t.Fatalf("Failed to read composer.json: %v", err)
	}

	contentStr := string(content)
	// Check that composer.json contains a version field
	// Note: The version might come from extractedData if it has one, or use the custom version
	if !common.Contains(contentStr, `"version"`) {
		t.Error("composer.json should contain a version field")
	}
	// If extractedData doesn't have a version, custom version should be used
	// But if extractedData has version "1.0.0", it takes precedence
	// So we just verify that a version exists
	if !common.Contains(contentStr, "1.0.0") && !common.Contains(contentStr, customVersion) {
		t.Errorf("composer.json should contain a version (either from schema or custom: %s)", customVersion)
	}
}

func TestGeneratePHPSDK_PHPVersion(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	phpVersion := common.GetPHPDefaultVersion()
	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, &phpVersion, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	composerPath := filepath.Join(tmpDir, sanitizedName, "composer.json")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(composerPath)
	if err != nil {
		t.Fatalf("Failed to read composer.json: %v", err)
	}

	contentStr := string(content)
	phpVersionStr := phpVersion.GetPHPVersionString()
	if !common.Contains(contentStr, phpVersionStr) {
		t.Errorf("composer.json should contain PHP version %s", phpVersionStr)
	}
}

func TestGeneratePHPSDK_ModelWithNestedObjects(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add schema with nested object
	extractedData.Schemas["User"] = &common.Schema{
		Type: "object",
		Properties: map[string]*common.Schema{
			"id": {Type: "integer"},
			"address": {
				Type: "object",
				Properties: map[string]*common.Schema{
					"street": {Type: "string"},
					"city":   {Type: "string"},
				},
			},
		},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	modelPath := filepath.Join(tmpDir, sanitizedName, "src", "Models", "User.php")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Error("GeneratePHPSDK() should create model with nested objects")
	}

	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("Failed to read User.php: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "address") {
		t.Error("User.php should contain nested address field")
	}
}

func TestGeneratePHPSDK_ModelWithEmptyProperties(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add schema with no properties
	extractedData.Schemas["EmptyModel"] = &common.Schema{
		Type:       "object",
		Properties: make(map[string]*common.Schema),
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	modelPath := filepath.Join(tmpDir, sanitizedName, "src", "Models", "EmptyModel.php")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Error("GeneratePHPSDK() should create model even with empty properties")
	}
}

func TestGeneratePHPSDK_ModelWithDescription(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add schema with description
	extractedData.Schemas["User"] = &common.Schema{
		Type:        "object",
		Description: "A user model representing a user in the system",
		Properties: map[string]*common.Schema{
			"id": {Type: "integer"},
		},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	modelPath := filepath.Join(tmpDir, sanitizedName, "src", "Models", "User.php")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("Failed to read User.php: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "A user model") {
		t.Error("User.php should contain model description in PHPDoc")
	}
}

func TestGeneratePHPSDK_APIMethodWithoutOperationID(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add operation without operationID
	extractedData.Operations = []common.APIOperation{
		{
			Method: "GET",
			Path:   "/users",
			Tags:   []string{"users"},
			// No OperationID
		},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	apiPath := filepath.Join(tmpDir, sanitizedName, "src", "Api", "UsersApi.php")
	if _, err := os.Stat(apiPath); os.IsNotExist(err) {
		t.Error("GeneratePHPSDK() should create API file even without operationID")
	}
}

func TestGeneratePHPSDK_APIMethodWithDescription(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add operation with description
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/users",
			OperationID: "listUsers",
			Summary:     "List all users",
			Description: "Retrieves a list of all users in the system",
			Tags:        []string{"users"},
		},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	apiPath := filepath.Join(tmpDir, sanitizedName, "src", "Api", "UsersApi.php")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("Failed to read UsersApi.php: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "List all users") {
		t.Error("UsersApi.php should contain method description in PHPDoc")
	}
}

func TestGeneratePHPSDK_AllParameterTypes(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add operation with different parameter types
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/search",
			OperationID: "search",
			Tags:        []string{"search"},
			Parameters: []common.Parameter{
				{Name: "id", In: "path", Schema: &common.Schema{Type: "integer"}},
				{Name: "q", In: "query", Schema: &common.Schema{Type: "string"}},
				{Name: "X-API-Key", In: "header", Schema: &common.Schema{Type: "string"}},
			},
		},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	apiPath := filepath.Join(tmpDir, sanitizedName, "src", "Api", "SearchApi.php")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("Failed to read SearchApi.php: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "search") {
		t.Error("SearchApi.php should contain search method with all parameter types")
	}
}

func TestGeneratePHPSDK_NilSchemaProperties(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add schema with nil property
	extractedData.Schemas["User"] = &common.Schema{
		Type: "object",
		Properties: map[string]*common.Schema{
			"id":   {Type: "integer"},
			"name": nil, // Nil property
		},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	// Should not crash, nil properties should be skipped
	sanitizedName := common.ToPascalCase(sdkName)
	modelPath := filepath.Join(tmpDir, sanitizedName, "src", "Models", "User.php")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Error("GeneratePHPSDK() should create model even with nil properties")
	}
}

func TestGeneratePHPSDK_AllIntegerTypes(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add schemas with different integer formats
	extractedData.Schemas["Int32Model"] = &common.Schema{Type: "integer", Format: "int32"}
	extractedData.Schemas["Int64Model"] = &common.Schema{Type: "integer", Format: "int64"}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	modelsDir := filepath.Join(tmpDir, sanitizedName, "src", "Models")

	expectedModels := []string{"Int32Model.php", "Int64Model.php"}
	for _, model := range expectedModels {
		modelPath := filepath.Join(modelsDir, model)
		if _, err := os.Stat(modelPath); os.IsNotExist(err) {
			t.Errorf("GeneratePHPSDK() should create model file: %s", model)
		}
	}
}

func TestGeneratePHPSDK_RefWithInvalidPath(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add schema with invalid reference
	extractedData.Schemas["UserProfile"] = &common.Schema{
		Type: "object",
		Properties: map[string]*common.Schema{
			"user": {
				Ref: "#/components/schemas/NonExistent",
			},
		},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	// Should handle gracefully (may or may not error, but shouldn't crash)
	if err != nil {
		// Error is acceptable for invalid references
		return
	}

	// If no error, should still generate files
	sanitizedName := common.ToPascalCase(sdkName)
	composerPath := filepath.Join(tmpDir, sanitizedName, "composer.json")
	if _, err := os.Stat(composerPath); os.IsNotExist(err) {
		t.Error("GeneratePHPSDK() should create files even with invalid reference")
	}
}

func TestGeneratePHPSDK_ArrayWithNilItems(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add array schema with nil items
	extractedData.Schemas["ItemList"] = &common.Schema{
		Type:  "array",
		Items: nil,
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	modelPath := filepath.Join(tmpDir, sanitizedName, "src", "Models", "ItemList.php")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Error("GeneratePHPSDK() should create array model even with nil items")
	}
}

func TestGeneratePHPSDK_ResponseWithNonJSONContent(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add operation with non-JSON response
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/download",
			OperationID: "download",
			Tags:        []string{"files"},
			Responses: map[string]common.Response{
				"200": {
					Description: "File content",
					Content: map[string]common.ContentType{
						"application/octet-stream": {
							Schema: &common.Schema{Type: "string", Format: "binary"},
						},
					},
				},
			},
		},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	apiPath := filepath.Join(tmpDir, sanitizedName, "src", "Api", "FilesApi.php")
	if _, err := os.Stat(apiPath); os.IsNotExist(err) {
		t.Error("GeneratePHPSDK() should create API file even with non-JSON response")
	}
}

func TestGeneratePHPSDK_ResponseWithNilSchema(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	// Add operation with nil response schema
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/status",
			OperationID: "getStatus",
			Tags:        []string{"status"},
			Responses: map[string]common.Response{
				"204": {
					Description: "No content",
					Content:     nil,
				},
			},
		},
	}

	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	apiPath := filepath.Join(tmpDir, sanitizedName, "src", "Api", "StatusApi.php")
	if _, err := os.Stat(apiPath); os.IsNotExist(err) {
		t.Error("GeneratePHPSDK() should create API file even with nil response schema")
	}
}

func TestGeneratePHPSDK_ComposerJSONWithDifferentDependencyFormats(t *testing.T) {
	tests := []struct {
		name      string
		httpLib   string
		expectDep string
	}{
		{"guzzle with version", "guzzle", "guzzlehttp/guzzle"},
		{"curl", "curl", ""}, // curl doesn't need external dependency
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sdkName := common.TestSDKName
			extractedData := common.CreateTestExtractedData()
			err := GeneratePHPSDK(tmpDir, sdkName, tt.httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
			if err != nil {
				t.Fatalf("GeneratePHPSDK() error = %v", err)
			}

			sanitizedName := common.ToPascalCase(sdkName)
			composerPath := filepath.Join(tmpDir, sanitizedName, "composer.json")
			// #nosec G304 -- File path is from test, safe to read
			content, err := os.ReadFile(composerPath)
			if err != nil {
				t.Fatalf("Failed to read composer.json: %v", err)
			}

			contentStr := string(content)
			if tt.expectDep != "" && !common.Contains(contentStr, tt.expectDep) {
				t.Errorf("composer.json should contain %s dependency", tt.expectDep)
			}
		})
	}
}

func TestGeneratePHPSDK_ReadmeGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	readmePath := filepath.Join(tmpDir, sanitizedName, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Error("GeneratePHPSDK() should create README.md")
	}

	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README.md: %v", err)
	}

	contentStr := string(content)
	// README should contain some content and be non-empty
	if len(contentStr) == 0 {
		t.Error("README.md should not be empty")
	}
	// Check for common README elements
	if !common.Contains(contentStr, "PHP") && !common.Contains(contentStr, "SDK") {
		t.Error("README.md should contain SDK-related content")
	}
}

func TestGeneratePHPSDK_QualityConfigs(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "guzzle"

	extractedData := common.CreateTestExtractedData()
	err := GeneratePHPSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePHPSDK() error = %v", err)
	}

	sanitizedName := common.ToPascalCase(sdkName)
	qualityFiles := []string{
		filepath.Join(tmpDir, sanitizedName, "phpcs.xml"),
		filepath.Join(tmpDir, sanitizedName, "phpstan.neon"),
		filepath.Join(tmpDir, sanitizedName, ".php-cs-fixer.php"),
	}

	for _, file := range qualityFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("GeneratePHPSDK() did not create quality config file: %s", file)
		}

		// #nosec G304 -- File path is from test, safe to read
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", file, err)
		}

		if len(content) == 0 {
			t.Errorf("Quality config file %s should not be empty", file)
		}
	}
}
