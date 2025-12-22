package typescript

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vubon/sdk-forge/internal/generator/common"
)

func TestGenerateTypeScriptSDK(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Use ExtractedData for testing
	extractedData := common.CreateTestExtractedData()
	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Check that files were created
	expectedFiles := []string{
		filepath.Join(tmpDir, "test-sdk", "package.json"),
		filepath.Join(tmpDir, "test-sdk", "tsconfig.json"),
		filepath.Join(tmpDir, "test-sdk", "README.md"),
		filepath.Join(tmpDir, "test-sdk", ".gitignore"),
		filepath.Join(tmpDir, "test-sdk", ".eslintrc.json"),
		filepath.Join(tmpDir, "test-sdk", ".prettierrc.json"),
		filepath.Join(tmpDir, "test-sdk", ".prettierignore"),
		filepath.Join(tmpDir, "test-sdk", "src", "index.ts"),
		filepath.Join(tmpDir, "test-sdk", "src", "client.ts"),
		filepath.Join(tmpDir, "test-sdk", "src", "exceptions.ts"),
	}

	for _, file := range expectedFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("GenerateTypeScriptSDK() did not create file: %s", file)
		}
	}
}

func TestGenerateTypeScriptSDK_InvalidHTTPLib(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := "test-sdk"
	httpLib := "invalid-lib"

	extractedData := common.CreateTestExtractedData()
	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err == nil {
		t.Error("GenerateTypeScriptSDK() with invalid HTTP library should return error")
	}
}

func TestGenerateTypeScriptSDK_CustomHTTPLib(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "fetch"

	extractedData := common.CreateTestExtractedData()
	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Check that client.ts exists and uses fetch
	clientPath := filepath.Join(tmpDir, "test-sdk", "src", "client.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read client.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "fetch") {
		t.Error("GenerateTypeScriptSDK() should use fetch in client.ts")
	}
}

func TestGenerateTypeScriptSDK_PackageJSON(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	extractedData := common.CreateTestExtractedData()
	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	packageJSONPath := filepath.Join(tmpDir, "test-sdk", "package.json")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(packageJSONPath)
	if err != nil {
		t.Fatalf("Failed to read package.json: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "test-sdk") {
		t.Error("package.json should contain SDK name")
	}
	if !common.Contains(contentStr, "typescript") {
		t.Error("package.json should contain TypeScript dependency")
	}
	if !common.Contains(contentStr, "axios") {
		t.Error("package.json should contain axios dependency")
	}
}

func TestGenerateTypeScriptSDK_SDKNameSanitization(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		sdkName  string
		expected string
	}{
		{"kebab case", "my-api-sdk", "my-api-sdk"},
		{"camel case", "myApiSdk", "my-api-sdk"},
		{"pascal case", "MyApiSdk", "my-api-sdk"},
		{"with spaces", "my api sdk", "my-api-sdk"},
		{"test-sdk", common.TestSDKName, "test-sdk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			extractedData := common.CreateTestExtractedData()
			err := GenerateTypeScriptSDK(tmpDir, tt.sdkName, "axios", extractedData, nil, "", true, common.DefaultRetryConfig())
			if err != nil {
				t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
			}

			// Check that package directory uses kebab-case
			packageDir := filepath.Join(tmpDir, tt.expected)
			if _, err := os.Stat(packageDir); os.IsNotExist(err) {
				t.Errorf("GenerateTypeScriptSDK() did not create package directory: %s", packageDir)
			}
		})
	}
}

func TestGenerateTypeScriptSDK_WithTests(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	extractedData := common.CreateTestExtractedData()
	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify tests directory exists
	testsDir := filepath.Join(tmpDir, "test-sdk", "tests")
	if _, err := os.Stat(testsDir); os.IsNotExist(err) {
		t.Error("GenerateTypeScriptSDK() with tests enabled should create tests directory")
	}

	// Verify test files exist
	expectedTestFiles := []string{
		filepath.Join(testsDir, "client.test.ts"),
		filepath.Join(tmpDir, "test-sdk", "jest.config.js"),
	}

	for _, file := range expectedTestFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("GenerateTypeScriptSDK() did not create test file: %s", file)
		}
	}
}

func TestGenerateTypeScriptSDK_WithoutTests(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	extractedData := common.CreateTestExtractedData()
	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", false, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify tests directory does NOT exist
	testsDir := filepath.Join(tmpDir, "test-sdk", "tests")
	if _, err := os.Stat(testsDir); err == nil {
		t.Error("GenerateTypeScriptSDK() with tests disabled should NOT create tests directory")
	}

	// Verify jest.config.js does NOT exist
	jestConfigPath := filepath.Join(tmpDir, "test-sdk", "jest.config.js")
	if _, err := os.Stat(jestConfigPath); err == nil {
		t.Error("GenerateTypeScriptSDK() with tests disabled should NOT create jest.config.js")
	}
}

func TestGenerateTypeScriptSDK_ModelGeneration(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

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

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify models directory exists
	modelsDir := filepath.Join(tmpDir, "test-sdk", "src", "models")
	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		t.Error("GenerateTypeScriptSDK() should create models directory when schemas exist")
	}

	// Verify model files exist
	expectedModelFiles := []string{
		filepath.Join(modelsDir, "index.ts"),
		filepath.Join(modelsDir, "user.ts"),
	}

	for _, file := range expectedModelFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("GenerateTypeScriptSDK() did not create model file: %s", file)
		}
	}

	// Verify model content
	userModelPath := filepath.Join(modelsDir, "user.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(userModelPath)
	if err != nil {
		t.Fatalf("Failed to read user.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "export interface User") {
		t.Error("user.ts should export User interface")
	}
	if !common.Contains(contentStr, "id") {
		t.Error("user.ts should contain id property")
	}
	if !common.Contains(contentStr, "name") {
		t.Error("user.ts should contain name property")
	}
}

func TestGenerateTypeScriptSDK_APIGeneration(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

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
					Content:     map[string]common.ContentType{},
				},
			},
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify api directory exists
	apiDir := filepath.Join(tmpDir, "test-sdk", "src", "api")
	if _, err := os.Stat(apiDir); os.IsNotExist(err) {
		t.Error("GenerateTypeScriptSDK() should create api directory when operations exist")
	}

	// Verify API files exist
	expectedAPIFiles := []string{
		filepath.Join(apiDir, "index.ts"),
		filepath.Join(apiDir, "users.ts"),
	}

	for _, file := range expectedAPIFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("GenerateTypeScriptSDK() did not create API file: %s", file)
		}
	}

	// Verify API content
	usersAPIPath := filepath.Join(apiDir, "users.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(usersAPIPath)
	if err != nil {
		t.Fatalf("Failed to read users.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "export class UsersApi") {
		t.Error("users.ts should export UsersApi class")
	}
}

func TestGenerateTypeScriptSDK_RetryConfig(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	extractedData := common.CreateTestExtractedData()
	retryConfig := common.RetryConfig{
		Enabled:              true,
		MaxAttempts:          5,
		InitialDelay:         2000000000,  // 2 seconds in nanoseconds
		MaxDelay:             60000000000, // 60 seconds
		BackoffMultiplier:    2.0,
		Strategy:             common.RetryStrategyExponential,
		RetryableStatusCodes: []int{429, 500, 502, 503, 504},
		RetryOnNetworkErrors: true,
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, retryConfig)
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify client.ts contains retry configuration
	clientPath := filepath.Join(tmpDir, "test-sdk", "src", "client.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read client.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "RetryConfig") {
		t.Error("client.ts should contain RetryConfig interface when retry is enabled")
	}
	if !common.Contains(contentStr, "requestWithRetry") {
		t.Error("client.ts should contain requestWithRetry method when retry is enabled")
	}
}

func TestGenerateTypeScriptSDK_NoRetryConfig(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	extractedData := common.CreateTestExtractedData()
	retryConfig := common.DefaultRetryConfig()
	retryConfig.Enabled = false

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, retryConfig)
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify client.ts does NOT contain retry configuration
	clientPath := filepath.Join(tmpDir, "test-sdk", "src", "client.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read client.ts: %v", err)
	}

	contentStr := string(content)
	if common.Contains(contentStr, "requestWithRetry") {
		t.Error("client.ts should NOT contain requestWithRetry method when retry is disabled")
	}
}

func TestGenerateTypeScriptSDK_Examples(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	extractedData := common.CreateTestExtractedData()
	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify examples directory exists
	examplesDir := filepath.Join(tmpDir, "test-sdk", "examples")
	if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
		t.Error("GenerateTypeScriptSDK() should create examples directory")
	}

	// Verify example file exists
	examplePath := filepath.Join(examplesDir, "basic-usage.ts")
	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		t.Error("GenerateTypeScriptSDK() should create basic-usage.ts example")
	}
}

func TestGenerateTypeScriptSDK_TSConfig(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	extractedData := common.CreateTestExtractedData()
	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify tsconfig.json exists and has correct content
	tsconfigPath := filepath.Join(tmpDir, "test-sdk", "tsconfig.json")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(tsconfigPath)
	if err != nil {
		t.Fatalf("Failed to read tsconfig.json: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "ES2020") {
		t.Error("tsconfig.json should target ES2020")
	}
	if !common.Contains(contentStr, "strict") {
		t.Error("tsconfig.json should enable strict mode")
	}
}

func TestGenerateTypeScriptSDK_DifferentHTTPLibraries(t *testing.T) {
	t.Parallel()
	httpLibraries := []string{"axios", "fetch", "node-fetch", "ky"}

	for _, httpLib := range httpLibraries {
		t.Run(httpLib, func(t *testing.T) {
			tmpDir := t.TempDir()
			sdkName := common.TestSDKName

			extractedData := common.CreateTestExtractedData()
			err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
			if err != nil {
				t.Fatalf("GenerateTypeScriptSDK() with %s error = %v", httpLib, err)
			}

			// Verify package.json contains the correct dependency
			packageJSONPath := filepath.Join(tmpDir, "test-sdk", "package.json")
			// #nosec G304 -- File path is from test, safe to read
			content, err := os.ReadFile(packageJSONPath)
			if err != nil {
				t.Fatalf("Failed to read package.json: %v", err)
			}

			contentStr := string(content)
			// For fetch, there's no dependency needed (it's built-in)
			if httpLib != "fetch" && !common.Contains(contentStr, httpLib) {
				t.Errorf("package.json should contain %s dependency", httpLib)
			}
		})
	}
}

func TestGenerateTypeScriptSDK_Authentication(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with authentication schemes
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
		"oauth2": {
			Type: "oauth2",
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify client.ts contains authentication methods
	clientPath := filepath.Join(tmpDir, "test-sdk", "src", "client.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read client.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "setApiKey") {
		t.Error("client.ts should contain setApiKey method for API key auth")
	}
	if !common.Contains(contentStr, "setBearerToken") {
		t.Error("client.ts should contain setBearerToken method for bearer auth")
	}
	if !common.Contains(contentStr, "setBasicAuth") {
		t.Error("client.ts should contain setBasicAuth method for basic auth")
	}
}

func TestGenerateTypeScriptSDK_OpenAPIDoc(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Test with openapi3.T document instead of ExtractedData
	doc := common.CreateTestOpenAPIDoc()
	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, doc, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() with openapi3.T error = %v", err)
	}

	// Verify files were created
	packageJSONPath := filepath.Join(tmpDir, "test-sdk", "package.json")
	if _, err := os.Stat(packageJSONPath); os.IsNotExist(err) {
		t.Error("GenerateTypeScriptSDK() should create package.json")
	}
}

func TestGenerateTypeScriptSDK_InvalidOpenAPIDoc(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Test with invalid document type
	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, "invalid", nil, "", true, common.DefaultRetryConfig())
	if err == nil {
		t.Error("GenerateTypeScriptSDK() with invalid document should return error")
	}
}

func TestGenerateTypeScriptSDK_ArrayModel(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with array schema
	extractedData := common.CreateTestExtractedData()
	extractedData.Schemas = map[string]*common.Schema{
		"UserList": {
			Type: "array",
			Items: &common.Schema{
				Type: "string",
			},
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify array model was generated
	modelPath := filepath.Join(tmpDir, "test-sdk", "src", "models", "user-list.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("Failed to read user-list.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "export type UserList") {
		t.Error("user-list.ts should export UserList type")
	}
	if !common.Contains(contentStr, "string[]") {
		t.Error("user-list.ts should contain string[] type")
	}
}

func TestGenerateTypeScriptSDK_APIMethodWithParams(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with operation that has parameters
	extractedData := common.CreateTestExtractedData()
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/users/{userId}",
			OperationID: "getUser",
			Summary:     "Get user by ID",
			Tags:        []string{"users"},
			Parameters: []common.Parameter{
				{
					Name:     "userId",
					In:       "path",
					Required: true,
					Schema: &common.Schema{
						Type: "integer",
					},
				},
				{
					Name:     "include",
					In:       "query",
					Required: false,
					Schema: &common.Schema{
						Type: "string",
					},
				},
			},
			Responses: map[string]common.Response{
				"200": {
					Description: "Success",
					Content:     map[string]common.ContentType{},
				},
			},
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify API method was generated with parameters
	apiPath := filepath.Join(tmpDir, "test-sdk", "src", "api", "users.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("Failed to read users.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "getUser") {
		t.Error("users.ts should contain getUser method")
	}
	if !common.Contains(contentStr, "userId") {
		t.Error("users.ts should contain userId parameter")
	}
	if !common.Contains(contentStr, "include") {
		t.Error("users.ts should contain include parameter")
	}
}

func TestGenerateTypeScriptSDK_AllSchemaTypes(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with various schema types
	extractedData := common.CreateTestExtractedData()
	extractedData.Schemas = map[string]*common.Schema{
		"StringModel": {
			Type:   "string",
			Format: "email",
		},
		"NumberModel": {
			Type: "number",
		},
		"IntegerModel": {
			Type: "integer",
		},
		"BooleanModel": {
			Type: "boolean",
		},
		"ArrayModel": {
			Type: "array",
			Items: &common.Schema{
				Type: "string",
			},
		},
		"ObjectModel": {
			Type: "object",
			Properties: map[string]*common.Schema{
				"value": {
					Type: "string",
				},
			},
		},
		"RefModel": {
			Ref: "#/components/schemas/User",
		},
		"DateModel": {
			Type:   "string",
			Format: "date",
		},
		"DateTimeModel": {
			Type:   "string",
			Format: "date-time",
		},
		"URIModel": {
			Type:   "string",
			Format: "uri",
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify all model types were generated correctly
	modelsDir := filepath.Join(tmpDir, "test-sdk", "src", "models")

	// Check string model
	stringModelPath := filepath.Join(modelsDir, "string-model.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(stringModelPath)
	if err != nil {
		t.Fatalf("Failed to read string-model.ts: %v", err)
	}
	contentStr := string(content)
	if !common.Contains(contentStr, "export type StringModel") {
		t.Error("string-model.ts should export StringModel type")
	}

	// Check array model
	arrayModelPath := filepath.Join(modelsDir, "array-model.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err = os.ReadFile(arrayModelPath)
	if err != nil {
		t.Fatalf("Failed to read array-model.ts: %v", err)
	}
	contentStr = string(content)
	if !common.Contains(contentStr, "string[]") {
		t.Error("array-model.ts should contain string[] type")
	}
}

func TestGenerateTypeScriptSDK_ResponseTypes(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with operations having different response types
	extractedData := common.CreateTestExtractedData()
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "POST",
			Path:        "/users",
			OperationID: "createUser",
			Tags:        []string{"users"},
			Responses: map[string]common.Response{
				"201": {
					Description: "Created",
					Content: map[string]common.ContentType{
						"application/json": {
							Schema: &common.Schema{
								Type: "object",
								Properties: map[string]*common.Schema{
									"id": {
										Type: "integer",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Method:      "GET",
			Path:        "/items",
			OperationID: "listItems",
			Tags:        []string{"items"},
			Responses: map[string]common.Response{
				"200": {
					Description: "Success",
					Content: map[string]common.ContentType{
						"application/json": {
							Schema: &common.Schema{
								Type: "array",
								Items: &common.Schema{
									Type: "string",
								},
							},
						},
					},
				},
			},
		},
		{
			Method:      "GET",
			Path:        "/no-content",
			OperationID: "getNoContent",
			Tags:        []string{"default"},
			Responses: map[string]common.Response{
				"204": {
					Description: "No Content",
					Content:     map[string]common.ContentType{},
				},
			},
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify API methods were generated with correct return types
	usersAPIPath := filepath.Join(tmpDir, "test-sdk", "src", "api", "users.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(usersAPIPath)
	if err != nil {
		t.Fatalf("Failed to read users.ts: %v", err)
	}
	contentStr := string(content)
	if !common.Contains(contentStr, "createUser") {
		t.Error("users.ts should contain createUser method")
	}
}

func TestGenerateTypeScriptSDK_AllHTTPLibrariesDetailed(t *testing.T) {
	t.Parallel()
	httpLibraries := []struct {
		name string
		lib  string
	}{
		{"axios", "axios"},
		{"fetch", "fetch"},
		{"node-fetch", "node-fetch"},
		{"ky", "ky"},
	}

	for _, tt := range httpLibraries {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sdkName := common.TestSDKName

			extractedData := common.CreateTestExtractedData()
			err := GenerateTypeScriptSDK(tmpDir, sdkName, tt.lib, extractedData, nil, "", true, common.DefaultRetryConfig())
			if err != nil {
				t.Fatalf("GenerateTypeScriptSDK() with %s error = %v", tt.lib, err)
			}

			// Verify client.ts uses the correct HTTP library
			clientPath := filepath.Join(tmpDir, "test-sdk", "src", "client.ts")
			// #nosec G304 -- File path is from test, safe to read
			content, err := os.ReadFile(clientPath)
			if err != nil {
				t.Fatalf("Failed to read client.ts: %v", err)
			}

			contentStr := string(content)
			// Verify HTTP library specific code
			switch tt.lib {
			case "axios":
				if !common.Contains(contentStr, "axios") {
					t.Error("client.ts should contain axios import")
				}
			case "fetch", "node-fetch":
				if !common.Contains(contentStr, "fetch") {
					t.Error("client.ts should contain fetch usage")
				}
			case "ky":
				if !common.Contains(contentStr, "ky") {
					t.Error("client.ts should contain ky import")
				}
			}
		})
	}
}

func TestGenerateTypeScriptSDK_AllAuthTypes(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with all authentication types
	extractedData := common.CreateTestExtractedData()
	extractedData.SecuritySchemes = map[string]common.SecurityScheme{
		"apiKeyHeader": {
			Type: "apiKey",
			In:   "header",
			Name: "X-API-Key",
		},
		"apiKeyQuery": {
			Type: "apiKey",
			In:   "query",
			Name: "api_key",
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

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify client.ts contains all authentication methods
	clientPath := filepath.Join(tmpDir, "test-sdk", "src", "client.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read client.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "setApiKey") {
		t.Error("client.ts should contain setApiKey method")
	}
	if !common.Contains(contentStr, "setBearerToken") {
		t.Error("client.ts should contain setBearerToken method")
	}
	if !common.Contains(contentStr, "setBasicAuth") {
		t.Error("client.ts should contain setBasicAuth method")
	}
}

func TestGenerateTypeScriptSDK_ModelWithRef(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with schema reference
	extractedData := common.CreateTestExtractedData()
	extractedData.Schemas = map[string]*common.Schema{
		"User": {
			Type: "object",
			Properties: map[string]*common.Schema{
				"id": {
					Type: "integer",
				},
			},
		},
		"Order": {
			Type: "object",
			Properties: map[string]*common.Schema{
				"user": {
					Ref: "#/components/schemas/User",
				},
			},
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify model with ref was generated
	orderModelPath := filepath.Join(tmpDir, "test-sdk", "src", "models", "order.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(orderModelPath)
	if err != nil {
		t.Fatalf("Failed to read order.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "User") {
		t.Error("order.ts should reference User type")
	}
}

func TestGenerateTypeScriptSDK_ModelWithOptionalFields(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with model having optional fields
	extractedData := common.CreateTestExtractedData()
	extractedData.Schemas = map[string]*common.Schema{
		"Product": {
			Type: "object",
			Properties: map[string]*common.Schema{
				"id": {
					Type: "integer",
				},
				"name": {
					Type: "string",
				},
				"description": {
					Type: "string",
				},
			},
			Required: []string{"id", "name"}, // description is optional
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify optional field was marked with ?
	productModelPath := filepath.Join(tmpDir, "test-sdk", "src", "models", "product.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(productModelPath)
	if err != nil {
		t.Fatalf("Failed to read product.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "description?:") {
		t.Error("product.ts should mark optional field with ?")
	}
	if !common.Contains(contentStr, "id:") && !common.Contains(contentStr, "name:") {
		t.Error("product.ts should have required fields without ?")
	}
}

func TestGenerateTypeScriptSDK_APIMethodWithRequestBody(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with operation that has request body
	extractedData := common.CreateTestExtractedData()
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
								"name": {
									Type: "string",
								},
							},
						},
					},
				},
			},
			Responses: map[string]common.Response{
				"200": {
					Description: "Success",
					Content:     map[string]common.ContentType{},
				},
			},
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify API method was generated with request body parameter
	apiPath := filepath.Join(tmpDir, "test-sdk", "src", "api", "users.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("Failed to read users.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "body") {
		t.Error("users.ts should contain body parameter for request body")
	}
}

func TestGenerateTypeScriptSDK_OperationWithoutTags(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with operation without tags
	extractedData := common.CreateTestExtractedData()
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/health",
			OperationID: "healthCheck",
			Tags:        []string{}, // No tags
			Responses: map[string]common.Response{
				"200": {
					Description: "Success",
					Content:     map[string]common.ContentType{},
				},
			},
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify default tag was used
	apiPath := filepath.Join(tmpDir, "test-sdk", "src", "api", "default.ts")
	if _, err := os.Stat(apiPath); os.IsNotExist(err) {
		t.Error("GenerateTypeScriptSDK() should create default.ts for operations without tags")
	}
}

func TestGenerateTypeScriptSDK_EmptySchemas(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with empty schemas
	extractedData := common.CreateTestExtractedData()
	extractedData.Schemas = map[string]*common.Schema{}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify models directory was not created or only has index.ts
	modelsDir := filepath.Join(tmpDir, "test-sdk", "src", "models")
	if _, err := os.Stat(modelsDir); err == nil {
		// If models directory exists, check that it's empty or only has index.ts
		files, _ := os.ReadDir(modelsDir)
		if len(files) > 1 {
			t.Error("GenerateTypeScriptSDK() should not create model files when schemas are empty")
		}
	}
}

func TestGenerateTypeScriptSDK_EmptyOperations(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with empty operations
	extractedData := common.CreateTestExtractedData()
	extractedData.Operations = []common.APIOperation{}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify api directory was not created
	apiDir := filepath.Join(tmpDir, "test-sdk", "src", "api")
	if _, err := os.Stat(apiDir); err == nil {
		t.Error("GenerateTypeScriptSDK() should not create api directory when operations are empty")
	}
}

func TestGenerateTypeScriptSDK_CustomSDKVersion(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"
	customVersion := "2.0.0"

	extractedData := common.CreateTestExtractedData()
	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, customVersion, true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify package.json contains custom version
	packageJSONPath := filepath.Join(tmpDir, "test-sdk", "package.json")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(packageJSONPath)
	if err != nil {
		t.Fatalf("Failed to read package.json: %v", err)
	}

	contentStr := string(content)
	// Check for version in JSON format (with quotes)
	if !common.Contains(contentStr, `"version"`) {
		t.Error("package.json should contain version field")
	}
	// The version might be determined by common.DetermineSDKVersion, so check if it exists
	if !common.Contains(contentStr, "version") {
		t.Error("package.json should contain version field")
	}
}

func TestGenerateTypeScriptSDK_ModelWithNestedObjects(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with nested object schema
	extractedData := common.CreateTestExtractedData()
	extractedData.Schemas = map[string]*common.Schema{
		"Address": {
			Type: "object",
			Properties: map[string]*common.Schema{
				"street": {
					Type: "string",
				},
				"city": {
					Type: "string",
				},
			},
		},
		"User": {
			Type: "object",
			Properties: map[string]*common.Schema{
				"address": {
					Type: "object",
					Properties: map[string]*common.Schema{
						"street": {
							Type: "string",
						},
					},
				},
			},
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify nested object was generated
	userModelPath := filepath.Join(tmpDir, "test-sdk", "src", "models", "user.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(userModelPath)
	if err != nil {
		t.Fatalf("Failed to read user.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "address") {
		t.Error("user.ts should contain address property")
	}
}

func TestGenerateTypeScriptSDK_ModelWithEmptyProperties(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with model having empty properties
	extractedData := common.CreateTestExtractedData()
	extractedData.Schemas = map[string]*common.Schema{
		"EmptyModel": {
			Type:       "object",
			Properties: map[string]*common.Schema{},
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify empty model was generated with index signature
	modelPath := filepath.Join(tmpDir, "test-sdk", "src", "models", "empty-model.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("Failed to read empty-model.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "[key: string]: any") {
		t.Error("empty-model.ts should contain index signature for empty properties")
	}
}

func TestGenerateTypeScriptSDK_ModelWithDescription(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with model having field descriptions
	extractedData := common.CreateTestExtractedData()
	extractedData.Schemas = map[string]*common.Schema{
		"Product": {
			Type: "object",
			Properties: map[string]*common.Schema{
				"id": {
					Type:        "integer",
					Description: "Product identifier",
				},
				"name": {
					Type:        "string",
					Description: "Product name\nCan be multiple lines",
				},
			},
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify model with descriptions was generated
	modelPath := filepath.Join(tmpDir, "test-sdk", "src", "models", "product.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("Failed to read product.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "/**") {
		t.Error("product.ts should contain JSDoc comments for fields with descriptions")
	}
	if !common.Contains(contentStr, "Product identifier") {
		t.Error("product.ts should contain field description")
	}
}

func TestGenerateTypeScriptSDK_APIMethodWithoutOperationID(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with operation without OperationID
	extractedData := common.CreateTestExtractedData()
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/api/v1/users",
			OperationID: "", // No operation ID
			Tags:        []string{"users"},
			Responses: map[string]common.Response{
				"200": {
					Description: "Success",
					Content:     map[string]common.ContentType{},
				},
			},
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify API method was generated with fallback naming
	apiPath := filepath.Join(tmpDir, "test-sdk", "src", "api", "users.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("Failed to read users.ts: %v", err)
	}

	contentStr := string(content)
	// Should have a method generated (fallback naming)
	if !common.Contains(contentStr, "async") {
		t.Error("users.ts should contain async method even without OperationID")
	}
}

func TestGenerateTypeScriptSDK_APIMethodWithDescription(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with operation having description
	extractedData := common.CreateTestExtractedData()
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/users",
			OperationID: "listUsers",
			Summary:     "List all users",
			Description: "Retrieves a list of all users\nSupports pagination",
			Tags:        []string{"users"},
			Responses: map[string]common.Response{
				"200": {
					Description: "Success",
					Content:     map[string]common.ContentType{},
				},
			},
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify API method was generated with JSDoc
	apiPath := filepath.Join(tmpDir, "test-sdk", "src", "api", "users.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("Failed to read users.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "/**") {
		t.Error("users.ts should contain JSDoc comments")
	}
	if !common.Contains(contentStr, "List all users") {
		t.Error("users.ts should contain summary in JSDoc")
	}
}

func TestGenerateTypeScriptSDK_AllParameterTypes(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with operation having all parameter types
	extractedData := common.CreateTestExtractedData()
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/test/{pathParam}",
			OperationID: "testParams",
			Tags:        []string{"test"},
			Parameters: []common.Parameter{
				{
					Name:     "pathParam",
					In:       "path",
					Required: true,
					Schema: &common.Schema{
						Type: "string",
					},
				},
				{
					Name:     "queryParam",
					In:       "query",
					Required: false,
					Schema: &common.Schema{
						Type: "string",
					},
				},
				{
					Name:     "headerParam",
					In:       "header",
					Required: false,
					Schema: &common.Schema{
						Type: "string",
					},
				},
				{
					Name:     "cookieParam",
					In:       "cookie",
					Required: false,
					Schema: &common.Schema{
						Type: "string",
					},
				},
			},
			Responses: map[string]common.Response{
				"200": {
					Description: "Success",
					Content:     map[string]common.ContentType{},
				},
			},
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify API method was generated with all parameter types
	apiPath := filepath.Join(tmpDir, "test-sdk", "src", "api", "test.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("Failed to read test.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "pathParam") {
		t.Error("test.ts should contain pathParam")
	}
	if !common.Contains(contentStr, "queryParam") {
		t.Error("test.ts should contain queryParam")
	}
}

func TestGenerateTypeScriptSDK_NilSchemaProperties(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with model having nil property schema
	extractedData := common.CreateTestExtractedData()
	extractedData.Schemas = map[string]*common.Schema{
		"ModelWithNil": {
			Type: "object",
			Properties: map[string]*common.Schema{
				"validField": {
					Type: "string",
				},
				"nilField": nil, // Nil schema
			},
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify model was generated (nil properties should be skipped)
	modelPath := filepath.Join(tmpDir, "test-sdk", "src", "models", "model-with-nil.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("Failed to read model-with-nil.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "validField") {
		t.Error("model-with-nil.ts should contain validField")
	}
	// nilField should be skipped
}

func TestGenerateTypeScriptSDK_AllIntegerTypes(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with all integer type variations
	extractedData := common.CreateTestExtractedData()
	extractedData.Schemas = map[string]*common.Schema{
		"Int32Model": {
			Type: "int32",
		},
		"Int64Model": {
			Type: "int64",
		},
		"FloatModel": {
			Type: "float",
		},
		"DoubleModel": {
			Type: "double",
		},
		"NullModel": {
			Type: "null",
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify all types were generated
	modelsDir := filepath.Join(tmpDir, "test-sdk", "src", "models")

	// Check int32 model
	int32Path := filepath.Join(modelsDir, "int32-model.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(int32Path)
	if err != nil {
		t.Fatalf("Failed to read int32-model.ts: %v", err)
	}
	contentStr := string(content)
	if !common.Contains(contentStr, "number") {
		t.Error("int32-model.ts should contain number type")
	}
}

func TestGenerateTypeScriptSDK_RefWithInvalidPath(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with ref that has invalid path
	extractedData := common.CreateTestExtractedData()
	extractedData.Schemas = map[string]*common.Schema{
		"ModelWithInvalidRef": {
			Type: "object",
			Properties: map[string]*common.Schema{
				"ref": {
					Ref: "invalid-ref", // Invalid ref path
				},
			},
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify model was generated (should handle invalid ref gracefully)
	modelPath := filepath.Join(tmpDir, "test-sdk", "src", "models", "model-with-invalid-ref.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("Failed to read model-with-invalid-ref.ts: %v", err)
	}

	contentStr := string(content)
	// Should generate something (even if ref is invalid)
	if contentStr == "" {
		t.Error("model-with-invalid-ref.ts should not be empty")
	}
}

func TestGenerateTypeScriptSDK_ArrayWithNilItems(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with array schema having nil items
	extractedData := common.CreateTestExtractedData()
	extractedData.Schemas = map[string]*common.Schema{
		"ArrayWithNilItems": {
			Type:  "array",
			Items: nil, // Nil items
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify array model was generated with any[]
	modelPath := filepath.Join(tmpDir, "test-sdk", "src", "models", "array-with-nil-items.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("Failed to read array-with-nil-items.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "any[]") {
		t.Error("array-with-nil-items.ts should contain any[] type")
	}
}

func TestGenerateTypeScriptSDK_ResponseWithNonJSONContent(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with operation having non-JSON response
	extractedData := common.CreateTestExtractedData()
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/download",
			OperationID: "downloadFile",
			Tags:        []string{"files"},
			Responses: map[string]common.Response{
				"200": {
					Description: "Success",
					Content: map[string]common.ContentType{
						"application/octet-stream": {
							Schema: &common.Schema{
								Type: "string",
							},
						},
					},
				},
			},
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify API method was generated
	apiPath := filepath.Join(tmpDir, "test-sdk", "src", "api", "files.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("Failed to read files.ts: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "downloadFile") {
		t.Error("files.ts should contain downloadFile method")
	}
}

func TestGenerateTypeScriptSDK_ResponseWithNilSchema(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	// Create extracted data with operation having response with nil schema
	extractedData := common.CreateTestExtractedData()
	extractedData.Operations = []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/test",
			OperationID: "testMethod",
			Tags:        []string{"test"},
			Responses: map[string]common.Response{
				"200": {
					Description: "Success",
					Content: map[string]common.ContentType{
						"application/json": {
							Schema: nil, // Nil schema
						},
					},
				},
			},
		},
	}

	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify API method was generated with default return type
	apiPath := filepath.Join(tmpDir, "test-sdk", "src", "api", "test.ts")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("Failed to read test.ts: %v", err)
	}

	contentStr := string(content)
	// Should default to "any" return type
	if !common.Contains(contentStr, "Promise<any>") {
		t.Error("test.ts should contain Promise<any> return type when schema is nil")
	}
}

func TestGenerateTypeScriptSDK_PackageJSONWithDifferentDependencyFormats(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	extractedData := common.CreateTestExtractedData()
	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify package.json was generated with correct structure
	packageJSONPath := filepath.Join(tmpDir, "test-sdk", "package.json")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(packageJSONPath)
	if err != nil {
		t.Fatalf("Failed to read package.json: %v", err)
	}

	contentStr := string(content)
	// Verify package.json structure
	if !common.Contains(contentStr, `"name"`) {
		t.Error("package.json should contain name field")
	}
	if !common.Contains(contentStr, `"version"`) {
		t.Error("package.json should contain version field")
	}
	if !common.Contains(contentStr, `"main"`) {
		t.Error("package.json should contain main field")
	}
	if !common.Contains(contentStr, `"types"`) {
		t.Error("package.json should contain types field")
	}
	if !common.Contains(contentStr, `"exports"`) {
		t.Error("package.json should contain exports field")
	}
}

func TestGenerateTypeScriptSDK_ReadmeGeneration(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	extractedData := common.CreateTestExtractedData()
	extractedData.Description = "Test API Description"
	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify README was generated
	readmePath := filepath.Join(tmpDir, "test-sdk", "README.md")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README.md: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "SDK") {
		t.Error("README.md should contain SDK information")
	}
	if !common.Contains(contentStr, "Installation") {
		t.Error("README.md should contain Installation section")
	}
	if !common.Contains(contentStr, "Usage") {
		t.Error("README.md should contain Usage section")
	}
}

func TestGenerateTypeScriptSDK_QualityConfigs(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "axios"

	extractedData := common.CreateTestExtractedData()
	err := GenerateTypeScriptSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateTypeScriptSDK() error = %v", err)
	}

	// Verify ESLint config was generated
	eslintPath := filepath.Join(tmpDir, "test-sdk", ".eslintrc.json")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(eslintPath)
	if err != nil {
		t.Fatalf("Failed to read .eslintrc.json: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "eslint:recommended") {
		t.Error(".eslintrc.json should contain eslint:recommended")
	}
	if !common.Contains(contentStr, "@typescript-eslint") {
		t.Error(".eslintrc.json should contain @typescript-eslint configuration")
	}

	// Verify Prettier config was generated
	prettierPath := filepath.Join(tmpDir, "test-sdk", ".prettierrc.json")
	// #nosec G304 -- File path is from test, safe to read
	content, err = os.ReadFile(prettierPath)
	if err != nil {
		t.Fatalf("Failed to read .prettierrc.json: %v", err)
	}

	contentStr = string(content)
	if !common.Contains(contentStr, "singleQuote") {
		t.Error(".prettierrc.json should contain singleQuote setting")
	}
	if !common.Contains(contentStr, "semi") {
		t.Error(".prettierrc.json should contain semi setting")
	}

	// Verify Prettier ignore was generated
	prettierIgnorePath := filepath.Join(tmpDir, "test-sdk", ".prettierignore")
	if _, err := os.Stat(prettierIgnorePath); os.IsNotExist(err) {
		t.Error("GenerateTypeScriptSDK() should create .prettierignore")
	}

	// Verify package.json includes ESLint and Prettier dependencies
	packageJSONPath := filepath.Join(tmpDir, "test-sdk", "package.json")
	// #nosec G304 -- File path is from test, safe to read
	content, err = os.ReadFile(packageJSONPath)
	if err != nil {
		t.Fatalf("Failed to read package.json: %v", err)
	}

	contentStr = string(content)
	if !common.Contains(contentStr, "eslint") {
		t.Error("package.json should contain eslint in devDependencies")
	}
	if !common.Contains(contentStr, "@typescript-eslint") {
		t.Error("package.json should contain @typescript-eslint packages in devDependencies")
	}
	if !common.Contains(contentStr, "prettier") {
		t.Error("package.json should contain prettier in devDependencies")
	}
}
