package gogen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vubon/sdk-forge/internal/generator/common"
	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

func TestGenerateGoSDK(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "nethttp"

	// Use ExtractedData for testing
	extractedData := common.CreateTestExtractedData()
	// outputPath should include the SDK name (like CLI does)
	outputPath := filepath.Join(tmpDir, common.TestGoSDKName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateGoSDK() error = %v", err)
	}

	// Check that files were created
	// Files are created directly in outputPath (no subdirectory)
	expectedFiles := []string{
		filepath.Join(outputPath, "go.mod"),
		filepath.Join(outputPath, "version.go"),
		filepath.Join(outputPath, "client.go"),
		filepath.Join(outputPath, "README.md"),
	}

	for _, file := range expectedFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("GenerateGoSDK() did not create file: %s", file)
		}
	}
}

func TestGenerateGoSDK_InvalidHTTPLib(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := "test-sdk"
	httpLib := "invalid-lib"

	extractedData := common.CreateTestExtractedData()
	// outputPath should include the SDK name (like CLI does)
	outputPath := filepath.Join(tmpDir, common.TestGoSDKName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err == nil {
		t.Error("GenerateGoSDK() with invalid HTTP library should return error")
	}
}

func TestGenerateGoSDK_CustomHTTPLib(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "resty"

	extractedData := common.CreateTestExtractedData()
	// outputPath should include the SDK name (like CLI does)
	outputPath := filepath.Join(tmpDir, common.TestGoSDKName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateGoSDK() error = %v", err)
	}

	// Check that go.mod exists (files are in outputPath directly)
	goModPath := filepath.Join(outputPath, "go.mod")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "module") {
		t.Error("go.mod should contain module declaration")
	}
}

func TestGenerateGoSDK_SDKNameSanitization(t *testing.T) {
	tests := []struct {
		name     string
		sdkName  string
		expected string
	}{
		{"kebab case", "my-api-sdk", "myapisdk"},
		{"camel case", "myApiSdk", "myapisdk"},
		{"pascal case", "MyApiSdk", "myapisdk"},
		{"with spaces", "my api sdk", "myapisdk"},
		{"test-sdk", common.TestSDKName, common.TestGoSDKName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			extractedData := common.CreateTestExtractedData()
			// outputPath should include the SDK name (like CLI does)
			outputPath := filepath.Join(tmpDir, tt.expected)
			err := GenerateGoSDK(outputPath, tt.sdkName, "nethttp", extractedData, nil, "", true, common.DefaultRetryConfig())
			if err != nil {
				t.Fatalf("GenerateGoSDK() error = %v", err)
			}

			// Check that go.mod uses lowercase package name (files are in outputPath directly)
			goModPath := filepath.Join(outputPath, "go.mod")
			// #nosec G304 -- File path is from test, safe to read
			content, err := os.ReadFile(goModPath)
			if err != nil {
				t.Fatalf("Failed to read go.mod: %v", err)
			}

			contentStr := string(content)
			if !common.Contains(contentStr, tt.expected) {
				t.Errorf("go.mod should contain module name with %s, got: %s", tt.expected, contentStr)
			}
		})
	}
}

func TestGenerateGoMod(t *testing.T) {
	extractedData := common.CreateTestExtractedData()
	defaultVersion := common.GetGoDefaultVersion()
	content := generateGoMod("test-sdk", extractedData, defaultVersion)
	if content == "" {
		t.Error("generateGoMod() should return non-empty content")
	}

	if !common.Contains(content, "module") {
		t.Error("generateGoMod() should include module declaration")
	}

	if !common.Contains(content, "go 1.24") {
		t.Error("generateGoMod() should include Go version 1.24")
	}
}

func TestGenerateGoClient(t *testing.T) {
	extractedData := common.CreateTestExtractedData()
	data := common.TemplateData{
		SDKName:       common.TestGoSDKName,
		HTTPLib:       "nethttp",
		HTTPLibImport: "net/http",
		HTTPLibConfig: &httplib.LibraryConfig{
			Import:      "net/http",
			Dependency:  "",
			ClientClass: "http.Client",
		},
		OpenAPIDoc:      extractedData,
		ClientClassName: common.GetClientClassName(common.TestGoSDKName),
	}

	defaultVersion := common.GetGoDefaultVersion()
	content := generateGoClient(data, defaultVersion)
	if content == "" {
		t.Error("generateGoClient() should return non-empty content")
	}

	if !common.Contains(content, "package") {
		t.Error("generateGoClient() should include package declaration")
	}

	// Check for client struct (should be "Test" after removing "Sdk" suffix)
	clientClassName := common.GetClientClassName(common.TestGoSDKName)
	if !common.Contains(content, clientClassName) {
		t.Errorf("generateGoClient() should include %s struct", clientClassName)
	}

	// Check for New function (should be "NewTest")
	newFuncName := "New" + clientClassName
	if !common.Contains(content, newFuncName) {
		t.Errorf("generateGoClient() should include %s function", newFuncName)
	}

	if !common.Contains(content, "net/http") {
		t.Error("generateGoClient() should include HTTP library import")
	}
}

func TestGenerateGoModels(t *testing.T) {
	// Test with empty schemas
	defaultVersion := common.GetGoDefaultVersion()
	content := generateGoModels(make(map[string]*common.Schema), defaultVersion)
	if content == "" {
		t.Error("generateGoModels() should return non-empty content even with empty schemas")
	}

	// Test with schemas
	schemas := map[string]*common.Schema{
		"User": {
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
		},
	}

	content = generateGoModels(schemas, defaultVersion)
	if content == "" {
		t.Error("generateGoModels() should return non-empty content")
	}

	if !common.Contains(content, "User") {
		t.Error("generateGoModels() should include User struct")
	}

	if !common.Contains(content, "struct") {
		t.Error("generateGoModels() should include struct definition")
	}
}

func TestGenerateGoREADME(t *testing.T) {
	data := common.TemplateData{
		SDKName:  common.TestGoSDKName,
		HTTPLib:  "nethttp",
		Language: "go",
	}

	content := generateGoREADME(data)
	if content == "" {
		t.Error("generateGoREADME() should return non-empty content")
	}

	if !common.Contains(content, "Go SDK") {
		t.Error("generateGoREADME() should include 'Go SDK'")
	}

	if !common.Contains(content, "Installation") {
		t.Error("generateGoREADME() should include Installation section")
	}

	if !common.Contains(content, "Usage") {
		t.Error("generateGoREADME() should include Usage section")
	}
}

func TestGenerateGoAuthSetup(t *testing.T) {
	clientClassName := "Test"
	// Test with no security schemes
	content := generateGoAuthSetup(make(map[string]common.SecurityScheme), clientClassName)
	if !common.Contains(content, "No authentication") {
		t.Error("generateGoAuthSetup() should handle empty security schemes")
	}

	// Test with API Key
	securitySchemes := map[string]common.SecurityScheme{
		"apiKey": {
			Type: "apiKey",
			In:   "header",
			Name: "X-API-Key",
		},
	}

	content = generateGoAuthSetup(securitySchemes, clientClassName)
	if !common.Contains(content, "applyAuth") {
		t.Error("generateGoAuthSetup() should include applyAuth function")
	}

	if !common.Contains(content, "X-API-Key") {
		t.Error("generateGoAuthSetup() should include API key header name")
	}

	// Test with Bearer token
	securitySchemes = map[string]common.SecurityScheme{
		"bearerAuth": {
			Type:   "http",
			Scheme: "bearer",
		},
	}

	content = generateGoAuthSetup(securitySchemes, clientClassName)
	if !common.Contains(content, "BearerToken") {
		t.Error("generateGoAuthSetup() should include BearerToken field")
	}

	if !common.Contains(content, "Bearer") {
		t.Error("generateGoAuthSetup() should include Bearer token handling")
	}
}

func TestGenerateGoAPIMethods(t *testing.T) {
	clientClassName := "Test"
	// Test with no operations
	defaultVersion := common.GetGoDefaultVersion()
	content := generateGoAPIMethods([]common.APIOperation{}, clientClassName, defaultVersion)
	if !common.Contains(content, "No API methods") {
		t.Error("generateGoAPIMethods() should handle empty operations")
	}

	// Test with operations
	operations := []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/users",
			OperationID: "listUsers",
			Summary:     "List users",
		},
		{
			Method:      "POST",
			Path:        "/users",
			OperationID: "createUser",
			Summary:     "Create user",
			RequestBody: &common.RequestBody{
				Required: true,
			},
		},
	}

	content = generateGoAPIMethods(operations, clientClassName, defaultVersion)
	// Method names are in PascalCase in Go
	if !common.Contains(content, "ListUsers") {
		t.Error("generateGoAPIMethods() should include ListUsers method")
	}

	if !common.Contains(content, "CreateUser") {
		t.Error("generateGoAPIMethods() should include CreateUser method")
	}

	expectedReceiver := "func (c *" + clientClassName + ")"
	if !common.Contains(content, expectedReceiver) {
		t.Errorf("generateGoAPIMethods() should include client methods with receiver %s", expectedReceiver)
	}
}

func TestGetGoType(t *testing.T) {
	tests := []struct {
		name     string
		schema   *common.Schema
		expected string
	}{
		{"string", &common.Schema{Type: "string"}, "string"},
		{"integer", &common.Schema{Type: "integer"}, "int"},
		{"number", &common.Schema{Type: "number"}, "float64"},
		{"boolean", &common.Schema{Type: "boolean"}, "bool"},
		{"array", &common.Schema{Type: "array", Items: &common.Schema{Type: "string"}}, "[]string"},
		{"object", &common.Schema{Type: "object"}, "map[string]any"},
		{"nil", nil, "any"},
		{"unknown", &common.Schema{Type: "unknown"}, "any"},
	}

	defaultVersion := common.GetGoDefaultVersion()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getGoType(tt.schema, defaultVersion)
			if result != tt.expected {
				t.Errorf("getGoType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateGoSDK_WithTests(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "nethttp"

	extractedData := common.CreateTestExtractedData()
	outputPath := filepath.Join(tmpDir, sdkName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateGoSDK() error = %v", err)
	}

	// Verify test files exist
	expectedTestFiles := []string{
		filepath.Join(outputPath, "client_test.go"),
	}

	for _, file := range expectedTestFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("GenerateGoSDK() did not create test file: %s", file)
		}
	}
}

func TestGenerateGoSDK_WithoutTests(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "nethttp"

	extractedData := common.CreateTestExtractedData()
	outputPath := filepath.Join(tmpDir, sdkName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", false, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateGoSDK() error = %v", err)
	}

	// Verify test files do NOT exist
	testFile := filepath.Join(outputPath, "client_test.go")
	if _, err := os.Stat(testFile); err == nil {
		t.Error("GenerateGoSDK() with tests disabled should NOT create test files")
	}
}

func TestGenerateGoSDK_ModelTests(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "nethttp"

	// Create extracted data with schemas
	extractedData := common.CreateTestExtractedData()
	extractedData.Schemas = map[string]*common.Schema{
		"User": {
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
		},
	}

	outputPath := filepath.Join(tmpDir, sdkName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateGoSDK() error = %v", err)
	}

	// Verify models_test.go exists and contains schema-based tests
	modelsTestPath := filepath.Join(outputPath, "models_test.go")
	if _, err := os.Stat(modelsTestPath); os.IsNotExist(err) {
		t.Fatal("models_test.go should be generated when schemas exist")
	}

	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(modelsTestPath)
	if err != nil {
		t.Fatalf("Failed to read models_test.go: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "TestUser_Creation") {
		t.Error("models_test.go should contain TestUser_Creation function")
	}
	if !common.Contains(contentStr, "TestUser_Serialization") {
		t.Error("models_test.go should contain TestUser_Serialization function")
	}
}

func TestGenerateGoSDK_APITests(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "nethttp"

	// Create extracted data with operations
	extractedData := common.CreateTestExtractedData()
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/users",
			OperationID: "listUsers",
			Summary:     "List users",
			Tags:        []string{"users"},
			Parameters:  []common.Parameter{},
			Responses: map[string]common.Response{
				"200": {
					Description: "Success",
				},
			},
		},
	}

	outputPath := filepath.Join(tmpDir, sdkName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateGoSDK() error = %v", err)
	}

	// Verify api_test.go exists and contains operation-based tests
	apiTestPath := filepath.Join(outputPath, "api_test.go")
	if _, err := os.Stat(apiTestPath); os.IsNotExist(err) {
		t.Fatal("api_test.go should be generated when operations exist")
	}

	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiTestPath)
	if err != nil {
		t.Fatalf("Failed to read api_test.go: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "TestUsersAPI") {
		t.Error("api_test.go should contain TestUsersAPI function")
	}
	if !common.Contains(contentStr, "TestUsers_ListUsers") {
		t.Error("api_test.go should contain TestUsers_ListUsers function")
	}
	if !common.Contains(contentStr, "httptest.NewServer") {
		t.Error("api_test.go should contain httptest server setup")
	}
}

func TestGenerateGoSDK_AuthTests(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "nethttp"

	// Create extracted data with security schemes
	extractedData := common.CreateTestExtractedData()
	extractedData.SecuritySchemes = map[string]common.SecurityScheme{
		"apiKey": {
			Type: "apiKey",
			In:   "header",
			Name: "X-API-Key",
		},
		"bearer": {
			Type:   "http",
			Scheme: "bearer",
		},
		"basic": {
			Type:   "http",
			Scheme: "basic",
		},
		"digest": {
			Type:   "http",
			Scheme: "digest",
		},
		"oauth2": {
			Type: "oauth2",
		},
		"openIdConnect": {
			Type: "openIdConnect",
		},
		"mutualTLS": {
			Type: "mutualTLS",
		},
	}

	outputPath := filepath.Join(tmpDir, sdkName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateGoSDK() error = %v", err)
	}

	// Verify auth_test.go exists
	authTestPath := filepath.Join(outputPath, "auth_test.go")
	if _, err := os.Stat(authTestPath); os.IsNotExist(err) {
		t.Fatal("auth_test.go should be generated when security schemes exist")
	}

	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(authTestPath)
	if err != nil {
		t.Fatalf("Failed to read auth_test.go: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "TestAuthentication") {
		t.Error("auth_test.go should contain TestAuthentication function")
	}
	if !common.Contains(contentStr, "TestApiKey_APIKeyAuth") {
		t.Error("auth_test.go should contain API key auth test")
	}
	if !common.Contains(contentStr, "TestBearer_BearerAuth") {
		t.Error("auth_test.go should contain Bearer auth test")
	}
	if !common.Contains(contentStr, "TestBasic_BasicAuth") {
		t.Error("auth_test.go should contain Basic auth test")
	}
	if !common.Contains(contentStr, "TestDigest_DigestAuth") {
		t.Error("auth_test.go should contain Digest auth test")
	}
	if !common.Contains(contentStr, "TestOauth2_OAuth2Auth") {
		t.Error("auth_test.go should contain OAuth2 auth test")
	}
	if !common.Contains(contentStr, "TestOpenIdConnect_OpenIDConnectAuth") {
		t.Error("auth_test.go should contain OpenID Connect auth test")
	}
	if !common.Contains(contentStr, "TestMutualTLS_MutualTLSAuth") {
		t.Error("auth_test.go should contain Mutual TLS auth test")
	}
}

func TestGenerateGoSDK_Phase3_Examples(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "nethttp"

	// Create extracted data with operations that have examples
	extractedData := common.CreateTestExtractedData()
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/users",
			OperationID: "listUsers",
			Summary:     "List users",
			Tags:        []string{"users"},
			Parameters:  []common.Parameter{},
			Responses: map[string]common.Response{
				"200": {
					Description: "Success",
					Content: map[string]common.ContentType{
						"application/json": {
							Schema: &common.Schema{Type: "object"},
							Examples: map[string]interface{}{
								"default": map[string]interface{}{
									"id":   1,
									"name": "Test User",
								},
							},
						},
					},
				},
			},
		},
	}

	outputPath := filepath.Join(tmpDir, sdkName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateGoSDK() error = %v", err)
	}

	// Verify api_test.go uses examples
	apiTestPath := filepath.Join(outputPath, "api_test.go")
	if _, err := os.Stat(apiTestPath); os.IsNotExist(err) {
		t.Fatal("api_test.go should be generated")
	}

	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiTestPath)
	if err != nil {
		t.Fatalf("Failed to read api_test.go: %v", err)
	}

	contentStr := string(content)
	// Check that example data is used (not just hardcoded success)
	if !common.Contains(contentStr, "Test User") && !common.Contains(contentStr, "\"id\"") {
		t.Error("api_test.go should use examples from OpenAPI spec")
	}
}

func TestGenerateGoSDK_Phase3_ErrorTests(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "nethttp"

	// Create extracted data with error responses
	extractedData := common.CreateTestExtractedData()
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/users/{id}",
			OperationID: "getUser",
			Summary:     "Get user",
			Tags:        []string{"users"},
			Parameters: []common.Parameter{
				{Name: "id", In: "path", Required: true, Schema: &common.Schema{Type: "string"}},
			},
			Responses: map[string]common.Response{
				"200": {
					Description: "Success",
					Content: map[string]common.ContentType{
						"application/json": {
							Schema: &common.Schema{Type: "object"},
						},
					},
				},
				"404": {
					Description: "Not Found",
					Content: map[string]common.ContentType{
						"application/json": {
							Schema: &common.Schema{Type: "object"},
							Examples: map[string]interface{}{
								"default": map[string]interface{}{
									"error": "User not found",
								},
							},
						},
					},
				},
			},
		},
	}

	outputPath := filepath.Join(tmpDir, sdkName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateGoSDK() error = %v", err)
	}

	// Verify error tests are generated
	apiTestPath := filepath.Join(outputPath, "api_test.go")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiTestPath)
	if err != nil {
		t.Fatalf("Failed to read api_test.go: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "Error") || !common.Contains(contentStr, "404") {
		t.Error("api_test.go should contain error handling tests for 4xx responses")
	}
}

func TestGenerateGoSDK_Phase3_Fixtures(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "nethttp"

	// Create extracted data with examples
	extractedData := common.CreateTestExtractedData()
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/users",
			OperationID: "listUsers",
			Tags:        []string{"users"},
			Responses: map[string]common.Response{
				"200": {
					Content: map[string]common.ContentType{
						"application/json": {
							Examples: map[string]interface{}{
								"default": map[string]interface{}{
									"users": []interface{}{map[string]interface{}{"id": 1}},
								},
							},
						},
					},
				},
			},
		},
	}

	outputPath := filepath.Join(tmpDir, sdkName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateGoSDK() error = %v", err)
	}

	// Verify testdata directory and file are created
	fixturesPath := filepath.Join(outputPath, "testdata", "fixtures.go")
	if _, err := os.Stat(fixturesPath); os.IsNotExist(err) {
		t.Fatal("testdata/fixtures.go should be generated when examples exist")
	}

	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(fixturesPath)
	if err != nil {
		t.Fatalf("Failed to read fixtures.go: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "package testdata") || !common.Contains(contentStr, "var") {
		t.Error("fixtures.go should contain fixture variables from examples")
	}
}

// TestGenerateGoSDK_WithRetry tests retry configuration
func TestGenerateGoSDK_WithRetry(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "nethttp"

	extractedData := common.CreateTestExtractedData()
	outputPath := filepath.Join(tmpDir, sdkName)

	// Create retry config with all strategies
	retryConfig := common.RetryConfig{
		Enabled:              true,
		MaxAttempts:          3,
		InitialDelay:         1 * time.Second,
		MaxDelay:             10 * time.Second,
		BackoffMultiplier:    2.0,
		Strategy:             common.RetryStrategyExponential,
		RetryableStatusCodes: []int{429, 500, 502, 503, 504},
		RetryOnNetworkErrors: true,
	}

	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true, retryConfig)
	if err != nil {
		t.Fatalf("GenerateGoSDK() error = %v", err)
	}

	// Check that client.go contains retry logic
	clientPath := filepath.Join(outputPath, "client.go")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read client.go: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "retryMaxAttempts") {
		t.Error("client.go should contain retry configuration fields")
	}
	if !common.Contains(contentStr, "calculateRetryDelay") {
		t.Error("client.go should contain calculateRetryDelay function")
	}
	if !common.Contains(contentStr, "isRetryableStatusCode") {
		t.Error("client.go should contain isRetryableStatusCode function")
	}
	if !common.Contains(contentStr, "requestWithRetry") {
		t.Error("client.go should contain requestWithRetry function")
	}
}

// TestGenerateGoSDK_RetryStrategies tests different retry strategies
func TestGenerateGoSDK_RetryStrategies(t *testing.T) {
	strategies := []common.RetryStrategy{
		common.RetryStrategyExponential,
		common.RetryStrategyLinear,
		common.RetryStrategyFixed,
	}

	for _, strategy := range strategies {
		t.Run(string(strategy), func(t *testing.T) {
			tmpDir := t.TempDir()
			sdkName := common.TestSDKName
			httpLib := "nethttp"

			extractedData := common.CreateTestExtractedData()
			outputPath := filepath.Join(tmpDir, sdkName)

			retryConfig := common.RetryConfig{
				Enabled:              true,
				MaxAttempts:          3,
				InitialDelay:         1 * time.Second,
				MaxDelay:             10 * time.Second,
				BackoffMultiplier:    2.0,
				Strategy:             strategy,
				RetryableStatusCodes: []int{429, 500},
				RetryOnNetworkErrors: true,
			}

			err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", false, retryConfig)
			if err != nil {
				t.Fatalf("GenerateGoSDK() error = %v", err)
			}

			// Check that client.go contains the strategy
			clientPath := filepath.Join(outputPath, "client.go")
			// #nosec G304 -- File path is from test, safe to read
			content, err := os.ReadFile(clientPath)
			if err != nil {
				t.Fatalf("Failed to read client.go: %v", err)
			}

			contentStr := string(content)
			if !common.Contains(contentStr, string(strategy)) {
				t.Errorf("client.go should contain retry strategy %s", strategy)
			}
		})
	}
}

// TestGenerateGoSDK_WithoutRetry tests SDK generation without retry
func TestGenerateGoSDK_WithoutRetry(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "nethttp"

	extractedData := common.CreateTestExtractedData()
	outputPath := filepath.Join(tmpDir, sdkName)

	retryConfig := common.RetryConfig{
		Enabled: false,
	}

	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", false, retryConfig)
	if err != nil {
		t.Fatalf("GenerateGoSDK() error = %v", err)
	}

	// Check that client.go does NOT contain retry logic
	clientPath := filepath.Join(outputPath, "client.go")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read client.go: %v", err)
	}

	contentStr := string(content)
	if common.Contains(contentStr, "retryMaxAttempts") {
		t.Error("client.go should NOT contain retry configuration when retry is disabled")
	}
}

// TestGenerateGoREADME_WithExamples tests README generation with examples
func TestGenerateGoREADME_WithExamples(t *testing.T) {
	extractedData := common.CreateTestExtractedData()
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/users",
			OperationID: "listUsers",
			Summary:     "List users",
			Tags:        []string{"users"},
		},
	}

	data := common.TemplateData{
		SDKName:         common.TestGoSDKName,
		HTTPLib:         "nethttp",
		Language:        "go",
		OpenAPIDoc:      extractedData,
		ClientClassName: common.GetClientClassName(common.TestGoSDKName),
	}

	content := generateGoREADME(data)
	if content == "" {
		t.Error("generateGoREADME() should return non-empty content")
	}

	if !common.Contains(content, "Go SDK") {
		t.Error("generateGoREADME() should include 'Go SDK'")
	}

	if !common.Contains(content, "Installation") {
		t.Error("generateGoREADME() should include Installation section")
	}

	if !common.Contains(content, "Usage") {
		t.Error("generateGoREADME() should include Usage section")
	}
}

// TestGenerateGoREADME_EmptyOperations tests README with no operations
func TestGenerateGoREADME_EmptyOperations(t *testing.T) {
	extractedData := common.CreateTestExtractedData()
	extractedData.Operations = []common.APIOperation{}

	data := common.TemplateData{
		SDKName:         common.TestGoSDKName,
		HTTPLib:         "nethttp",
		Language:        "go",
		OpenAPIDoc:      extractedData,
		ClientClassName: common.GetClientClassName(common.TestGoSDKName),
	}

	content := generateGoREADME(data)
	if content == "" {
		t.Error("generateGoREADME() should return non-empty content even without operations")
	}
}

// TestGenerateGoREADME_TemplateSuccess tests successful template-based README generation
func TestGenerateGoREADME_TemplateSuccess(t *testing.T) {
	data := common.TemplateData{
		SDKName:         "my-api-sdk",
		HTTPLib:         "nethttp",
		Language:        "go",
		ClientClassName: "MyApiSdk",
	}

	content := generateGoREADME(data)

	// Check all major sections
	expectedSections := []string{
		"# MyApiSdk Go SDK",
		"Auto-generated Go SDK from OpenAPI schema",
		"## Installation",
		"go get github.com/example/my-api-sdk",
		"## Usage",
		"package main",
		"import",
		"github.com/example/my-api-sdk",
		"func main()",
		"my-api-sdk.NewMyApiSdk",
		"## HTTP Library",
		"net/http",
		"## Authentication",
		"BearerToken",
	}

	for _, section := range expectedSections {
		if !common.Contains(content, section) {
			t.Errorf("generateGoREADME() should contain '%s'", section)
		}
	}
}

// TestGenerateGoREADME_DifferentSDKNames tests README with various SDK name formats
func TestGenerateGoREADME_DifferentSDKNames(t *testing.T) {
	tests := []struct {
		name            string
		sdkName         string
		clientClassName string
		expectedPkg     string
		expectedDisplay string
	}{
		{
			name:            "kebab-case",
			sdkName:         "my-api-sdk",
			clientClassName: "MyApiSdk",
			expectedPkg:     "my-api-sdk",
			expectedDisplay: "MyApiSdk",
		},
		{
			name:            "snake_case",
			sdkName:         "my_api_sdk",
			clientClassName: "MyApiSdk",
			expectedPkg:     "my_api_sdk",
			expectedDisplay: "MyApiSdk",
		},
		{
			name:            "PascalCase",
			sdkName:         "MyApi",
			clientClassName: "MyApi",
			expectedPkg:     "myapi",
			expectedDisplay: "MyApi",
		},
		{
			name:            "single word",
			sdkName:         "petstore",
			clientClassName: "Petstore",
			expectedPkg:     "petstore",
			expectedDisplay: "Petstore",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := common.TemplateData{
				SDKName:         tt.sdkName,
				HTTPLib:         "nethttp",
				Language:        "go",
				ClientClassName: tt.clientClassName,
			}

			content := generateGoREADME(data)

			// Check package name in import
			if !common.Contains(content, "github.com/example/"+tt.expectedPkg) {
				t.Errorf("README should contain package import 'github.com/example/%s'", tt.expectedPkg)
			}

			// Check display name in title
			if !common.Contains(content, "# "+tt.expectedDisplay+" Go SDK") {
				t.Errorf("README should contain title '# %s Go SDK'", tt.expectedDisplay)
			}

			// Check client constructor
			expectedConstructor := tt.expectedPkg + ".New" + tt.clientClassName
			if !common.Contains(content, expectedConstructor) {
				t.Errorf("README should contain constructor '%s'", expectedConstructor)
			}
		})
	}
}

// TestGenerateGoREADME_AllSections tests that all expected sections are present
func TestGenerateGoREADME_AllSections(t *testing.T) {
	data := common.TemplateData{
		SDKName:         "test-sdk",
		HTTPLib:         "nethttp",
		Language:        "go",
		ClientClassName: "Test",
	}

	content := generateGoREADME(data)

	// Define all sections that should be present
	sections := map[string][]string{
		"Title": {
			"# TestSdk Go SDK",
		},
		"Description": {
			"Auto-generated Go SDK from OpenAPI schema",
		},
		"Installation": {
			"## Installation",
			"```bash",
			"go get github.com/example/test-sdk",
			"```",
		},
		"Usage": {
			"## Usage",
			"```go",
			"package main",
			"import (",
			"github.com/example/test-sdk",
			"func main() {",
			"test-sdk.NewTest",
			"https://api.example.com/v1",
			"// Use client methods",
		},
		"HTTP Library": {
			"## HTTP Library",
			"net/http",
			"Go standard library",
		},
		"Authentication": {
			"## Authentication",
			"Configure authentication",
			"BearerToken",
			"your-token",
		},
	}

	for sectionName, keywords := range sections {
		for _, keyword := range keywords {
			if !common.Contains(content, keyword) {
				t.Errorf("Section '%s' should contain '%s'", sectionName, keyword)
			}
		}
	}
}

// TestGenerateGoREADME_CodeBlocks tests that code blocks are properly formatted
func TestGenerateGoREADME_CodeBlocks(t *testing.T) {
	data := common.TemplateData{
		SDKName:         "example-sdk",
		HTTPLib:         "nethttp",
		Language:        "go",
		ClientClassName: "Example",
	}

	content := generateGoREADME(data)

	// Check for bash code block
	if !common.Contains(content, "```bash") {
		t.Error("README should contain bash code block")
	}

	// Check for Go code blocks
	goCodeBlockCount := 0
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, "```go") {
			goCodeBlockCount++
		}
	}

	if goCodeBlockCount < 2 {
		t.Errorf("README should contain at least 2 Go code blocks, found %d", goCodeBlockCount)
	}

	// Check that code blocks are closed
	openBlocks := strings.Count(content, "```")
	if openBlocks%2 != 0 {
		t.Error("README should have balanced code block markers (```)")
	}
}

// TestGenerateGoREADME_ClientClassName tests different client class names
func TestGenerateGoREADME_ClientClassName(t *testing.T) {
	tests := []struct {
		name            string
		clientClassName string
		expectedNew     string
	}{
		{
			name:            "simple",
			clientClassName: "Client",
			expectedNew:     "NewClient",
		},
		{
			name:            "with SDK suffix",
			clientClassName: "TestSdk",
			expectedNew:     "NewTestSdk",
		},
		{
			name:            "multi-word",
			clientClassName: "MyApiClient",
			expectedNew:     "NewMyApiClient",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := common.TemplateData{
				SDKName:         "test",
				HTTPLib:         "nethttp",
				Language:        "go",
				ClientClassName: tt.clientClassName,
			}

			content := generateGoREADME(data)

			if !common.Contains(content, tt.expectedNew) {
				t.Errorf("README should contain constructor '%s'", tt.expectedNew)
			}
		})
	}
}

// TestGenerateGoREADME_MarkdownFormatting tests markdown formatting
func TestGenerateGoREADME_MarkdownFormatting(t *testing.T) {
	data := common.TemplateData{
		SDKName:         "test-sdk",
		HTTPLib:         "nethttp",
		Language:        "go",
		ClientClassName: "Test",
	}

	content := generateGoREADME(data)

	// Check for proper markdown headers
	if !common.Contains(content, "# ") {
		t.Error("README should contain h1 header (#)")
	}

	if !common.Contains(content, "## ") {
		t.Error("README should contain h2 headers (##)")
	}

	// Check for backticks for inline code
	if !common.Contains(content, "`net/http`") {
		t.Error("README should use backticks for inline code")
	}

	// Check for proper line breaks (double newline for paragraphs)
	if !common.Contains(content, "\n\n") {
		t.Error("README should have proper paragraph spacing")
	}
}

// TestGenerateGoTestValue_ComplexTypes tests test value generation for complex types
func TestGenerateGoTestValue_ComplexTypes(t *testing.T) {
	version := common.GetGoDefaultVersion()

	tests := []struct {
		name      string
		schema    *common.Schema
		propName  string
		isPointer bool
		wantType  string // substring to check in result
	}{
		{
			name: "date format",
			schema: &common.Schema{
				Type:   "string",
				Format: "date",
			},
			propName:  "birthDate",
			isPointer: false,
			wantType:  "2024-01-01",
		},
		{
			name: "date-time format",
			schema: &common.Schema{
				Type:   "string",
				Format: "date-time",
			},
			propName:  "createdAt",
			isPointer: false,
			wantType:  "2024-01-01T00:00:00Z",
		},
		{
			name: "email format",
			schema: &common.Schema{
				Type:   "string",
				Format: "email",
			},
			propName:  "email",
			isPointer: false,
			wantType:  "test@example.com",
		},
		{
			name: "array of strings",
			schema: &common.Schema{
				Type: "array",
				Items: &common.Schema{
					Type: "string",
				},
			},
			propName:  "tags",
			isPointer: false,
			wantType:  "[]string",
		},
		{
			name: "array of integers",
			schema: &common.Schema{
				Type: "array",
				Items: &common.Schema{
					Type: "integer",
				},
			},
			propName:  "ids",
			isPointer: false,
			wantType:  "[]int",
		},
		{
			name: "object type",
			schema: &common.Schema{
				Type: "object",
			},
			propName:  "metadata",
			isPointer: false,
			wantType:  "map[string]",
		},
		{
			name: "pointer to string",
			schema: &common.Schema{
				Type: "string",
			},
			propName:  "optionalField",
			isPointer: true,
			wantType:  "nil",
		},
		{
			name: "pointer to integer",
			schema: &common.Schema{
				Type: "integer",
			},
			propName:  "optionalCount",
			isPointer: true,
			wantType:  "nil",
		},
		{
			name: "pointer to boolean",
			schema: &common.Schema{
				Type: "boolean",
			},
			propName:  "optionalFlag",
			isPointer: true,
			wantType:  "nil",
		},
		{
			name: "pointer to array",
			schema: &common.Schema{
				Type: "array",
				Items: &common.Schema{
					Type: "string",
				},
			},
			propName:  "optionalTags",
			isPointer: true,
			wantType:  "&[]string",
		},
		{
			name: "pointer to object",
			schema: &common.Schema{
				Type: "object",
			},
			propName:  "optionalMetadata",
			isPointer: true,
			wantType:  "&map[string]",
		},
		{
			name:      "nil schema",
			schema:    nil,
			propName:  "unknown",
			isPointer: false,
			wantType:  "test_value",
		},
		{
			name:      "nil schema pointer",
			schema:    nil,
			propName:  "unknown",
			isPointer: true,
			wantType:  "nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateGoTestValue(tt.schema, tt.propName, version, tt.isPointer)
			if !common.Contains(result, tt.wantType) {
				t.Errorf("generateGoTestValue() = %v, want to contain %v", result, tt.wantType)
			}
		})
	}
}

// TestGenerateGoExampleFromSchema tests example generation from schema
func TestGenerateGoExampleFromSchema(t *testing.T) {
	tests := []struct {
		name     string
		schema   *common.Schema
		expected string
	}{
		{
			name:     "nil schema",
			schema:   nil,
			expected: "`{}`",
		},
		{
			name: "object schema",
			schema: &common.Schema{
				Type: "object",
			},
			expected: "`{}`",
		},
		{
			name: "array schema",
			schema: &common.Schema{
				Type: "array",
			},
			expected: "`[]`",
		},
		{
			name: "string schema",
			schema: &common.Schema{
				Type: "string",
			},
			expected: "`{\"value\": \"test\"}`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateGoExampleFromSchema(tt.schema)
			if result != tt.expected {
				t.Errorf("generateGoExampleFromSchema() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestFormatExampleForGo tests example formatting
func TestFormatExampleForGo(t *testing.T) {
	tests := []struct {
		name     string
		example  interface{}
		contains string
	}{
		{
			name:     "nil example",
			example:  nil,
			contains: "null",
		},
		{
			name: "map example",
			example: map[string]interface{}{
				"id":   1,
				"name": "test",
			},
			contains: "id",
		},
		{
			name: "array example",
			example: []interface{}{
				map[string]interface{}{"id": 1},
				map[string]interface{}{"id": 2},
			},
			contains: "[",
		},
		{
			name:     "string example",
			example:  "test string",
			contains: "test string",
		},
		{
			name:     "number example",
			example:  42,
			contains: "42",
		},
		{
			name:     "boolean example",
			example:  true,
			contains: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatExampleForGo(tt.example)
			if !common.Contains(result, tt.contains) {
				t.Errorf("formatExampleForGo() = %v, want to contain %v", result, tt.contains)
			}
		})
	}
}

// TestGenerateGoTestValueFromParam tests generating test values from params
func TestGenerateGoTestValueFromParam(t *testing.T) {
	version := common.GetGoDefaultVersion()

	tests := []struct {
		name     string
		param    common.Parameter
		contains string
	}{
		{
			name: "Test param with nil schema",
			param: common.Parameter{
				Name:   "testParam",
				In:     "query",
				Schema: nil,
			},
			contains: "\"test_value\"",
		},
		{
			name: "Test param with string schema",
			param: common.Parameter{
				Name: "stringParam",
				In:   "query",
				Schema: &common.Schema{
					Type: "string",
				},
			},
			contains: "\"test_string_param\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateGoTestValueFromParam(tt.param, version)
			if !common.Contains(result, tt.contains) {
				t.Errorf("generateGoTestValueFromParam() = %v, want to contain %v", result, tt.contains)
			}
		})
	}
}

// TestGenerateGoClientFallback tests the fallback client generation
func TestGenerateGoClientFallback(t *testing.T) {
	extractedData := common.CreateTestExtractedData()
	data := common.TemplateData{
		SDKName:         common.TestSDKName,
		HTTPLib:         "nethttp",
		Language:        "go",
		OpenAPIDoc:      extractedData,
		ClientClassName: common.GetClientClassName(common.TestSDKName),
		RetryConfig: common.RetryConfig{
			Enabled: true,
		},
	}

	baseURLDefault := "https://api.example.com/v1"
	authSetup := "// Auth setup"
	apiMethods := "// API methods"
	displayName := "TestSdk"
	packageName := "testsdk"
	imports := `import (
		"bytes"
		"encoding/json"
		"fmt"
		"io"
		"net/http"
	)`

	content := generateGoClientFallback(
		data,
		extractedData,
		baseURLDefault,
		authSetup,
		apiMethods,
		displayName,
		packageName,
		imports,
	)

	// Check for essential components
	expectedComponents := []string{
		"package testsdk",
		"type Test struct",
		"func NewTest",
		"func (c *Test) Request",
		"// Auth setup",
		"// API methods",
	}

	for _, component := range expectedComponents {
		if !common.Contains(content, component) {
			t.Errorf("Fallback client should contain '%s'", component)
		}
	}
}


// TestGenerateGoAPIMethods_Comprehensive tests various API method generation scenarios
func TestGenerateGoAPIMethods_Comprehensive(t *testing.T) {
	clientClassName := "Client"
	version := common.GetGoDefaultVersion()

	tests := []struct {
		name       string
		operations []common.APIOperation
		contains   []string
		excludes   []string
	}{
		{
			name: "Simple GET Operation",
			operations: []common.APIOperation{
				{
					Method:      "GET",
					Path:        "/status",
					OperationID: "getStatus",
					Summary:     "Get system status",
				},
			},
			contains: []string{
				"func (c *Client) GetStatus() ([]byte, error) {",
				"path := \"/status\"",
				"return c.Request(\"GET\", path, nil)",
			},
		},
		{
			name: "Path Parameters (String & Integer)",
			operations: []common.APIOperation{
				{
					Method:      "GET",
					Path:        "/users/{userId}/posts/{postId}",
					OperationID: "getUserPost",
					Parameters: []common.Parameter{
						{
							Name: "userId",
							In:   "path",
							Schema: &common.Schema{Type: "string"},
						},
						{
							Name: "postId",
							In:   "path",
							Schema: &common.Schema{Type: "integer"},
						},
					},
				},
			},
			contains: []string{
				"func (c *Client) GetUserPost(userId string, postId int) ([]byte, error) {",
				"path := fmt.Sprintf(\"/users/%s/posts/%d\", userId, postId)",
			},
		},
		{
			name: "Query Parameters",
			operations: []common.APIOperation{
				{
					Method:      "GET",
					Path:        "/users",
					OperationID: "listUsers",
					Parameters: []common.Parameter{
						{
							Name: "limit",
							In:   "query",
							Schema: &common.Schema{Type: "integer"},
						},
						{
							Name: "active",
							In:   "query",
							Schema: &common.Schema{Type: "boolean"},
						},
					},
				},
			},
			contains: []string{
				"func (c *Client) ListUsers(limit *int, active *bool) ([]byte, error) {",
				"queryParts := []string{}",
				"if limit != nil {",
				"queryParts = append(queryParts, fmt.Sprintf(\"limit=%v\", *limit))",
				"if active != nil {",
				"queryParts = append(queryParts, fmt.Sprintf(\"active=%v\", *active))",
				"path += \"?\" + strings.Join(queryParts, \"&\")",
			},
		},
		{
			name: "POST with Request Body",
			operations: []common.APIOperation{
				{
					Method:      "POST",
					Path:        "/users",
					OperationID: "createUser",
					RequestBody: &common.RequestBody{
						Required: true,
						Content: map[string]common.ContentType{
							"application/json": {},
						},
					},
				},
			},
			contains: []string{
				"func (c *Client) CreateUser(body any) ([]byte, error) {",
				"return c.Request(\"POST\", path, body)",
			},
		},
		{
			name: "Complex Path Params (Boolean & Number)",
			operations: []common.APIOperation{
				{
					Method:      "GET",
					Path:        "/config/{flag}/{threshold}",
					OperationID: "getConfig",
					Parameters: []common.Parameter{
						{
							Name: "flag",
							In:   "path",
							Schema: &common.Schema{Type: "boolean"},
						},
						{
							Name: "threshold",
							In:   "path",
							Schema: &common.Schema{Type: "number"},
						},
					},
				},
			},
			contains: []string{
				"func (c *Client) GetConfig(flag bool, threshold float64) ([]byte, error) {",
				"path := fmt.Sprintf(\"/config/%t/%f\", flag, threshold)",
			},
		},
		{
			name: "Fallback Method Naming",
			operations: []common.APIOperation{
				{
					Method:      "GET",
					Path:        "/admin/system-stats",
					OperationID: "", // Empty OperationID triggers fallback
				},
			},
			contains: []string{
				"func (c *Client) GetAdminSystemStats() ([]byte, error) {",
			},
		},
		{
			name: "Multi-line Description",
			operations: []common.APIOperation{
				{
					Method:      "GET",
					Path:        "/info",
					OperationID: "getInfo",
					Summary:     "Get Info",
					Description: "Line 1\nLine 2",
				},
			},
			contains: []string{
				"// GetInfo Get Info",
				"// Line 1",
				"// Line 2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := generateGoAPIMethods(tt.operations, clientClassName, version)

			for _, s := range tt.contains {
				if !strings.Contains(content, s) {
					t.Errorf("Content missing expected string: %s\nGenerated content:\n%s", s, content)
				}
			}
		})
	}
}
