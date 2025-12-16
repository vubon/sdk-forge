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
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "")
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
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "")
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
	err := GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "")
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
			err := GenerateGoSDK(outputPath, tt.sdkName, "nethttp", extractedData, nil, "")
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
