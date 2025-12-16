package generator

import (
	"os"
	"path/filepath"
	"testing"

	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

func TestGenerateGoSDK(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := testSDKName
	httpLib := "nethttp"

	// Use ExtractedData for testing
	extractedData := createTestExtractedData()
	// outputPath should include the SDK name (like CLI does)
	outputPath := filepath.Join(tmpDir, testGoSDKName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true)
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

	extractedData := createTestExtractedData()
	// outputPath should include the SDK name (like CLI does)
	outputPath := filepath.Join(tmpDir, testGoSDKName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true)
	if err == nil {
		t.Error("GenerateGoSDK() with invalid HTTP library should return error")
	}
}

func TestGenerateGoSDK_CustomHTTPLib(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := testSDKName
	httpLib := "resty"

	extractedData := createTestExtractedData()
	// outputPath should include the SDK name (like CLI does)
	outputPath := filepath.Join(tmpDir, testGoSDKName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true)
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
	if !contains(contentStr, "module") {
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
		{"test-sdk", testSDKName, testGoSDKName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			extractedData := createTestExtractedData()
			// outputPath should include the SDK name (like CLI does)
			outputPath := filepath.Join(tmpDir, tt.expected)
			err := GenerateGoSDK(outputPath, tt.sdkName, "nethttp", extractedData, nil, "", true)
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
			if !contains(contentStr, tt.expected) {
				t.Errorf("go.mod should contain module name with %s, got: %s", tt.expected, contentStr)
			}
		})
	}
}

func TestGenerateGoMod(t *testing.T) {
	extractedData := createTestExtractedData()
	defaultVersion := GetGoDefaultVersion()
	content := generateGoMod("test-sdk", extractedData, defaultVersion)
	if content == "" {
		t.Error("generateGoMod() should return non-empty content")
	}

	if !contains(content, "module") {
		t.Error("generateGoMod() should include module declaration")
	}

	if !contains(content, "go 1.24") {
		t.Error("generateGoMod() should include Go version 1.24")
	}
}

func TestGenerateGoClient(t *testing.T) {
	extractedData := createTestExtractedData()
	data := TemplateData{
		SDKName:       testGoSDKName,
		HTTPLib:       "nethttp",
		HTTPLibImport: "net/http",
		HTTPLibConfig: &httplib.LibraryConfig{
			Import:      "net/http",
			Dependency:  "",
			ClientClass: "http.Client",
		},
		OpenAPIDoc:      extractedData,
		ClientClassName: getClientClassName(testGoSDKName),
	}

	defaultVersion := GetGoDefaultVersion()
	content := generateGoClient(data, defaultVersion)
	if content == "" {
		t.Error("generateGoClient() should return non-empty content")
	}

	if !contains(content, "package") {
		t.Error("generateGoClient() should include package declaration")
	}

	// Check for client struct (should be "Test" after removing "Sdk" suffix)
	clientClassName := getClientClassName(testGoSDKName)
	if !contains(content, clientClassName) {
		t.Errorf("generateGoClient() should include %s struct", clientClassName)
	}

	// Check for New function (should be "NewTest")
	newFuncName := "New" + clientClassName
	if !contains(content, newFuncName) {
		t.Errorf("generateGoClient() should include %s function", newFuncName)
	}

	if !contains(content, "net/http") {
		t.Error("generateGoClient() should include HTTP library import")
	}
}

func TestGenerateGoModels(t *testing.T) {
	// Test with empty schemas
	defaultVersion := GetGoDefaultVersion()
	content := generateGoModels(make(map[string]*Schema), defaultVersion)
	if content == "" {
		t.Error("generateGoModels() should return non-empty content even with empty schemas")
	}

	// Test with schemas
	schemas := map[string]*Schema{
		"User": {
			Type:        pythonTypeObject,
			Description: "User model",
			Properties: map[string]*Schema{
				"id": {
					Type: pythonTypeInteger,
				},
				"name": {
					Type: pythonTypeString,
				},
			},
			Required: []string{"id", "name"},
		},
	}

	content = generateGoModels(schemas, defaultVersion)
	if content == "" {
		t.Error("generateGoModels() should return non-empty content")
	}

	if !contains(content, "User") {
		t.Error("generateGoModels() should include User struct")
	}

	if !contains(content, "struct") {
		t.Error("generateGoModels() should include struct definition")
	}
}

func TestGenerateGoREADME(t *testing.T) {
	data := TemplateData{
		SDKName:  testGoSDKName,
		HTTPLib:  "nethttp",
		Language: "go",
	}

	content := generateGoREADME(data)
	if content == "" {
		t.Error("generateGoREADME() should return non-empty content")
	}

	if !contains(content, "Go SDK") {
		t.Error("generateGoREADME() should include 'Go SDK'")
	}

	if !contains(content, "Installation") {
		t.Error("generateGoREADME() should include Installation section")
	}

	if !contains(content, "Usage") {
		t.Error("generateGoREADME() should include Usage section")
	}
}

func TestGenerateGoAuthSetup(t *testing.T) {
	clientClassName := "Test"
	// Test with no security schemes
	content := generateGoAuthSetup(make(map[string]SecurityScheme), clientClassName)
	if !contains(content, "No authentication") {
		t.Error("generateGoAuthSetup() should handle empty security schemes")
	}

	// Test with API Key
	securitySchemes := map[string]SecurityScheme{
		"apiKey": {
			Type: securitySchemeAPIKey,
			In:   paramLocationHeader,
			Name: "X-API-Key",
		},
	}

	content = generateGoAuthSetup(securitySchemes, clientClassName)
	if !contains(content, "applyAuth") {
		t.Error("generateGoAuthSetup() should include applyAuth function")
	}

	if !contains(content, "X-API-Key") {
		t.Error("generateGoAuthSetup() should include API key header name")
	}

	// Test with Bearer token
	securitySchemes = map[string]SecurityScheme{
		"bearerAuth": {
			Type:   securitySchemeHTTP,
			Scheme: securitySchemeBearer,
		},
	}

	content = generateGoAuthSetup(securitySchemes, clientClassName)
	if !contains(content, "BearerToken") {
		t.Error("generateGoAuthSetup() should include BearerToken field")
	}

	if !contains(content, "Bearer") {
		t.Error("generateGoAuthSetup() should include Bearer token handling")
	}
}

func TestGenerateGoAPIMethods(t *testing.T) {
	clientClassName := "Test"
	// Test with no operations
	defaultVersion := GetGoDefaultVersion()
	content := generateGoAPIMethods([]APIOperation{}, clientClassName, defaultVersion)
	if !contains(content, "No API methods") {
		t.Error("generateGoAPIMethods() should handle empty operations")
	}

	// Test with operations
	operations := []APIOperation{
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
			RequestBody: &RequestBody{
				Required: true,
			},
		},
	}

	content = generateGoAPIMethods(operations, clientClassName, defaultVersion)
	// Method names are in PascalCase in Go
	if !contains(content, "ListUsers") {
		t.Error("generateGoAPIMethods() should include ListUsers method")
	}

	if !contains(content, "CreateUser") {
		t.Error("generateGoAPIMethods() should include CreateUser method")
	}

	expectedReceiver := "func (c *" + clientClassName + ")"
	if !contains(content, expectedReceiver) {
		t.Errorf("generateGoAPIMethods() should include client methods with receiver %s", expectedReceiver)
	}
}

func TestGetGoType(t *testing.T) {
	tests := []struct {
		name     string
		schema   *Schema
		expected string
	}{
		{"string", &Schema{Type: pythonTypeString}, "string"},
		{"integer", &Schema{Type: pythonTypeInteger}, "int"},
		{"number", &Schema{Type: pythonTypeNumber}, "float64"},
		{"boolean", &Schema{Type: pythonTypeBoolean}, "bool"},
		{"array", &Schema{Type: pythonTypeArray, Items: &Schema{Type: pythonTypeString}}, "[]string"},
		{"object", &Schema{Type: pythonTypeObject}, "map[string]any"},
		{"nil", nil, "any"},
		{"unknown", &Schema{Type: "unknown"}, "any"},
	}

	defaultVersion := GetGoDefaultVersion()
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
	sdkName := testSDKName
	httpLib := "nethttp"

	extractedData := createTestExtractedData()
	outputPath := filepath.Join(tmpDir, sdkName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true)
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
	sdkName := testSDKName
	httpLib := "nethttp"

	extractedData := createTestExtractedData()
	outputPath := filepath.Join(tmpDir, sdkName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", false)
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
	sdkName := testSDKName
	httpLib := "nethttp"

	// Create extracted data with schemas
	extractedData := createTestExtractedData()
	extractedData.Schemas = map[string]*Schema{
		"User": {
			Type:        "object",
			Description: "User model",
			Properties: map[string]*Schema{
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
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true)
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
	if !contains(contentStr, "TestUser_Creation") {
		t.Error("models_test.go should contain TestUser_Creation function")
	}
	if !contains(contentStr, "TestUser_Serialization") {
		t.Error("models_test.go should contain TestUser_Serialization function")
	}
}

func TestGenerateGoSDK_APITests(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := testSDKName
	httpLib := "nethttp"

	// Create extracted data with operations
	extractedData := createTestExtractedData()
	extractedData.Operations = []APIOperation{
		{
			Method:      "GET",
			Path:        "/users",
			OperationID: "listUsers",
			Summary:     "List users",
			Tags:        []string{"users"},
			Parameters:  []Parameter{},
			Responses: map[string]Response{
				"200": {
					Description: "Success",
				},
			},
		},
	}

	outputPath := filepath.Join(tmpDir, sdkName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true)
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
	if !contains(contentStr, "TestUsersAPI") {
		t.Error("api_test.go should contain TestUsersAPI function")
	}
	if !contains(contentStr, "TestUsers_ListUsers") {
		t.Error("api_test.go should contain TestUsers_ListUsers function")
	}
	if !contains(contentStr, "httptest.NewServer") {
		t.Error("api_test.go should contain httptest server setup")
	}
}

func TestGenerateGoSDK_AuthTests(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := testSDKName
	httpLib := "nethttp"

	// Create extracted data with security schemes
	extractedData := createTestExtractedData()
	extractedData.SecuritySchemes = map[string]SecurityScheme{
		"apiKey": {
			Type: "apiKey",
			In:   "header",
			Name: "X-API-Key",
		},
		"bearer": {
			Type:   "http",
			Scheme: "bearer",
		},
	}

	outputPath := filepath.Join(tmpDir, sdkName)
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true)
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
	if !contains(contentStr, "TestAuthentication") {
		t.Error("auth_test.go should contain TestAuthentication function")
	}
	if !contains(contentStr, "TestApiKey_APIKeyAuth") {
		t.Error("auth_test.go should contain API key auth test")
	}
	if !contains(contentStr, "TestBearer_BearerAuth") {
		t.Error("auth_test.go should contain Bearer auth test")
	}
}

func TestGenerateGoSDK_Phase3_Examples(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := testSDKName
	httpLib := "nethttp"

	// Create extracted data with operations that have examples
	extractedData := createTestExtractedData()
	extractedData.Operations = []APIOperation{
		{
			Method:      "GET",
			Path:        "/users",
			OperationID: "listUsers",
			Summary:     "List users",
			Tags:        []string{"users"},
			Parameters:  []Parameter{},
			Responses: map[string]Response{
				"200": {
					Description: "Success",
					Content: map[string]ContentType{
						"application/json": {
							Schema: &Schema{Type: "object"},
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
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true)
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
	if !contains(contentStr, "Test User") && !contains(contentStr, "\"id\"") {
		t.Error("api_test.go should use examples from OpenAPI spec")
	}
}

func TestGenerateGoSDK_Phase3_ErrorTests(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := testSDKName
	httpLib := "nethttp"

	// Create extracted data with error responses
	extractedData := createTestExtractedData()
	extractedData.Operations = []APIOperation{
		{
			Method:      "GET",
			Path:        "/users/{id}",
			OperationID: "getUser",
			Summary:     "Get user",
			Tags:        []string{"users"},
			Parameters: []Parameter{
				{Name: "id", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			},
			Responses: map[string]Response{
				"200": {
					Description: "Success",
					Content: map[string]ContentType{
						"application/json": {
							Schema: &Schema{Type: "object"},
						},
					},
				},
				"404": {
					Description: "Not Found",
					Content: map[string]ContentType{
						"application/json": {
							Schema: &Schema{Type: "object"},
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
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true)
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
	if !contains(contentStr, "Error") || !contains(contentStr, "404") {
		t.Error("api_test.go should contain error handling tests for 4xx responses")
	}
}

func TestGenerateGoSDK_Phase3_Fixtures(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := testSDKName
	httpLib := "nethttp"

	// Create extracted data with examples
	extractedData := createTestExtractedData()
	extractedData.Operations = []APIOperation{
		{
			Method:      "GET",
			Path:        "/users",
			OperationID: "listUsers",
			Tags:        []string{"users"},
			Responses: map[string]Response{
				"200": {
					Content: map[string]ContentType{
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
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true)
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
	if !contains(contentStr, "package testdata") || !contains(contentStr, "var") {
		t.Error("fixtures.go should contain fixture variables from examples")
	}
}
