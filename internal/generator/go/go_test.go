package gogen

import (
	"os"
	"path/filepath"
	"testing"

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
