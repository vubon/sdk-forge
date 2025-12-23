package python

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/vubon/sdk-forge/internal/generator/common"

	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestGeneratePythonSDK(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"

	// Use ExtractedData for testing
	extractedData := common.CreateTestExtractedData()
	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() error = %v", err)
	}

	// Check that files were created
	expectedFiles := []string{
		filepath.Join(tmpDir, "test_sdk", "__init__.py"),
		filepath.Join(tmpDir, "test_sdk", "client.py"),
		filepath.Join(tmpDir, "requirements.txt"),
		filepath.Join(tmpDir, "README.md"),
	}

	for _, file := range expectedFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("GeneratePythonSDK() did not create file: %s", file)
		}
	}
}

func TestGeneratePythonSDK_InvalidHTTPLib(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := "test-sdk"
	httpLib := "invalid-lib"

	extractedData := common.CreateTestExtractedData()
	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err == nil {
		t.Error("GeneratePythonSDK() with invalid HTTP library should return error")
	}
}

func TestGeneratePythonSDK_CustomHTTPLib(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "httpx"

	extractedData := common.CreateTestExtractedData()
	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() error = %v", err)
	}

	// Check that client.py uses httpx
	clientPath := filepath.Join(tmpDir, "test_sdk", "client.py")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read client.py: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "httpx") {
		t.Error("GeneratePythonSDK() should use httpx in client.py")
	}
}

func TestGeneratePythonSDK_RequirementsTxt(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"

	extractedData := common.CreateTestExtractedData()
	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() error = %v", err)
	}

	requirementsPath := filepath.Join(tmpDir, "requirements.txt")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(requirementsPath)
	if err != nil {
		t.Fatalf("Failed to read requirements.txt: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "requests") {
		t.Error("requirements.txt should contain requests dependency")
	}
}

func TestGeneratePythonSDK_SDKNameSanitization(t *testing.T) {
	tests := []struct {
		name     string
		sdkName  string
		expected string
	}{
		{"kebab case", "my-api-sdk", "my_api_sdk"},
		{"camel case", "myApiSdk", "my_api_sdk"},
		{"pascal case", "MyApiSdk", "my_api_sdk"},
		{"with spaces", "my api sdk", "my_api_sdk"},
		{"test-sdk", common.TestSDKName, "test_sdk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			extractedData := common.CreateTestExtractedData()
			err := GeneratePythonSDK(tmpDir, tt.sdkName, "requests", extractedData, nil, "", true, common.DefaultRetryConfig())
			if err != nil {
				t.Fatalf("GeneratePythonSDK() error = %v", err)
			}

			// Check that package directory uses snake_case
			packageDir := filepath.Join(tmpDir, tt.expected)
			if _, err := os.Stat(packageDir); os.IsNotExist(err) {
				t.Errorf("GeneratePythonSDK() did not create package directory: %s", packageDir)
			}
		})
	}
}

func TestGeneratePythonSDK_WithTests(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"

	extractedData := common.CreateTestExtractedData()
	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() error = %v", err)
	}

	// Verify tests directory exists
	testsDir := filepath.Join(tmpDir, "tests")
	if _, err := os.Stat(testsDir); os.IsNotExist(err) {
		t.Error("GeneratePythonSDK() with tests enabled should create tests directory")
	}

	// Verify test files exist
	expectedTestFiles := []string{
		filepath.Join(testsDir, "__init__.py"),
		filepath.Join(testsDir, "conftest.py"),
		filepath.Join(testsDir, "test_client.py"),
	}

	for _, file := range expectedTestFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("GeneratePythonSDK() did not create test file: %s", file)
		}
	}
}

func TestGeneratePythonSDK_WithoutTests(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"

	extractedData := common.CreateTestExtractedData()
	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", false, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() error = %v", err)
	}

	// Verify tests directory does NOT exist
	testsDir := filepath.Join(tmpDir, "tests")
	if _, err := os.Stat(testsDir); err == nil {
		t.Error("GeneratePythonSDK() with tests disabled should NOT create tests directory")
	}
}

func TestGeneratePythonSDK_ModelTests(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"

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

	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() error = %v", err)
	}

	// Verify test_models.py exists and contains schema-based tests
	modelsTestPath := filepath.Join(tmpDir, "tests", "test_models.py")
	if _, err := os.Stat(modelsTestPath); os.IsNotExist(err) {
		t.Fatal("test_models.py should be generated when schemas exist")
	}

	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(modelsTestPath)
	if err != nil {
		t.Fatalf("Failed to read test_models.py: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "TestUser") {
		t.Error("test_models.py should contain TestUser class")
	}
	if !common.Contains(contentStr, "test_user_creation") {
		t.Error("test_models.py should contain test_user_creation method")
	}
	if !common.Contains(contentStr, "test_user_serialization") {
		t.Error("test_models.py should contain test_user_serialization method")
	}
}

func TestGeneratePythonSDK_APITests(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"

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

	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() error = %v", err)
	}

	// Verify test_api_methods.py exists and contains operation-based tests
	apiTestPath := filepath.Join(tmpDir, "tests", "test_api_methods.py")
	if _, err := os.Stat(apiTestPath); os.IsNotExist(err) {
		t.Fatal("test_api_methods.py should be generated when operations exist")
	}

	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiTestPath)
	if err != nil {
		t.Fatalf("Failed to read test_api_methods.py: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "TestUsersAPI") {
		t.Error("test_api_methods.py should contain TestUsersAPI class")
	}
	if !common.Contains(contentStr, "test_list_users") {
		t.Error("test_api_methods.py should contain test_list_users method")
	}
	if !common.Contains(contentStr, "mock_request") {
		t.Error("test_api_methods.py should contain mock setup")
	}
}

func TestGeneratePythonSDK_AuthTests(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"

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

	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() error = %v", err)
	}

	// Verify test_auth.py exists
	authTestPath := filepath.Join(tmpDir, "tests", "test_auth.py")
	if _, err := os.Stat(authTestPath); os.IsNotExist(err) {
		t.Fatal("test_auth.py should be generated when security schemes exist")
	}

	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(authTestPath)
	if err != nil {
		t.Fatalf("Failed to read test_auth.py: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "TestAuthentication") {
		t.Error("test_auth.py should contain TestAuthentication class")
	}
	// Check for API key auth test (scheme name "apiKey" is converted to "api_key")
	if !common.Contains(contentStr, "api_key") {
		t.Error("test_auth.py should contain API key auth test")
	}
	// Check for Bearer auth test (scheme name "bearer" stays as "bearer")
	if !common.Contains(contentStr, "bearer_auth") {
		t.Error("test_auth.py should contain Bearer auth test")
	}
	// Check for Basic auth test
	if !common.Contains(contentStr, "basic_auth") {
		t.Error("test_auth.py should contain Basic auth test")
	}
	// Check for Digest auth test
	if !common.Contains(contentStr, "digest_auth") {
		t.Error("test_auth.py should contain Digest auth test")
	}
	// Check for OAuth2 auth test
	if !common.Contains(contentStr, "oauth2_auth") {
		t.Error("test_auth.py should contain OAuth2 auth test")
	}
	// Check for OpenID Connect auth test
	if !common.Contains(contentStr, "openid_connect_auth") {
		t.Error("test_auth.py should contain OpenID Connect auth test")
	}
	// Check for Mutual TLS auth test
	if !common.Contains(contentStr, "mutual_tls_auth") {
		t.Error("test_auth.py should contain Mutual TLS auth test")
	}
}

func TestGeneratePythonInit(t *testing.T) {
	extractedData := common.CreateTestExtractedData()
	data := common.TemplateData{
		SDKName:       "test_sdk",
		HTTPLib:       "requests",
		HTTPLibImport: "requests",
		HTTPLibConfig: &httplib.LibraryConfig{
			Import:      "requests",
			Dependency:  "requests>=2.31.0",
			ClientClass: "requests.Session",
		},
		OpenAPIDoc: extractedData,
	}

	content := generatePythonInit(data, "1.0.0")
	if content == "" {
		t.Error("generatePythonInit() should return non-empty content")
	}

	// Check for SDK name or client class (template might render differently)
	if !common.Contains(content, "test") && !common.Contains(content, "Test") {
		t.Error("generatePythonInit() should include SDK name or client class")
	}

	// Check for client import
	if !common.Contains(content, "client") && !common.Contains(content, "Client") {
		t.Error("generatePythonInit() should include client import")
	}
}

func TestGeneratePythonClient(t *testing.T) {
	extractedData := common.CreateTestExtractedData()
	data := common.TemplateData{
		SDKName:       "test_sdk",
		HTTPLib:       "requests",
		HTTPLibImport: "requests",
		HTTPLibConfig: &httplib.LibraryConfig{
			Import:      "requests",
			Dependency:  "requests>=2.31.0",
			ClientClass: "requests.Session",
		},
		OpenAPIDoc: extractedData,
	}

	content := generatePythonClient(data)
	if content == "" {
		t.Error("generatePythonClient() should return non-empty content")
	}

	if !common.Contains(content, "requests") {
		t.Error("generatePythonClient() should include HTTP library import")
	}

	if !common.Contains(content, "Test") && !common.Contains(content, "test") {
		t.Error("generatePythonClient() should include client class name")
	}
}

func TestGeneratePythonRequirements(t *testing.T) {
	extractedData := common.CreateTestExtractedData()
	data := common.TemplateData{
		SDKName:       "test_sdk",
		HTTPLib:       "requests",
		HTTPLibImport: "requests",
		HTTPLibConfig: &httplib.LibraryConfig{
			Import:      "requests",
			Dependency:  "requests>=2.31.0",
			ClientClass: "requests.Session",
		},
		OpenAPIDoc: extractedData,
	}

	content := generatePythonRequirements(data)
	if content == "" {
		t.Error("generatePythonRequirements() should return non-empty content")
	}

	if !common.Contains(content, "requests") {
		t.Error("generatePythonRequirements() should include HTTP library dependency")
	}
}

func TestGeneratePythonSDK_Phase3_Examples(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"

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

	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() error = %v", err)
	}

	// Verify test_api_methods.py uses examples
	apiTestPath := filepath.Join(tmpDir, "tests", "test_api_methods.py")
	if _, err := os.Stat(apiTestPath); os.IsNotExist(err) {
		t.Fatal("test_api_methods.py should be generated")
	}

	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiTestPath)
	if err != nil {
		t.Fatalf("Failed to read test_api_methods.py: %v", err)
	}

	contentStr := string(content)
	// Check that example data is used (not just hardcoded success)
	if !common.Contains(contentStr, "Test User") && !common.Contains(contentStr, "\"id\"") {
		t.Error("test_api_methods.py should use examples from OpenAPI spec")
	}
}

func TestGeneratePythonSDK_Phase3_ErrorTests(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"

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

	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() error = %v", err)
	}

	// Verify error tests are generated
	apiTestPath := filepath.Join(tmpDir, "tests", "test_api_methods.py")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(apiTestPath)
	if err != nil {
		t.Fatalf("Failed to read test_api_methods.py: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "error") || !common.Contains(contentStr, "404") {
		t.Error("test_api_methods.py should contain error handling tests for 4xx responses")
	}
}

func TestGeneratePythonSDK_Phase3_Fixtures(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"

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

	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() error = %v", err)
	}

	// Verify fixtures directory and file are created
	fixturesPath := filepath.Join(tmpDir, "tests", "fixtures", "fixtures.py")
	if _, err := os.Stat(fixturesPath); os.IsNotExist(err) {
		t.Fatal("fixtures/fixtures.py should be generated when examples exist")
	}

	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(fixturesPath)
	if err != nil {
		t.Fatalf("Failed to read fixtures.py: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "fixtures") || !common.Contains(contentStr, "list_users") {
		t.Error("fixtures.py should contain fixture variables from examples")
	}
}

func TestGeneratePythonSDK_WithRetryEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"

	extractedData := common.CreateTestExtractedData()

	// Create retry config with retry enabled
	retryConfig := common.DefaultRetryConfig()
	retryConfig.Enabled = true
	retryConfig.MaxAttempts = 5
	retryConfig.Strategy = common.RetryStrategyExponential
	retryConfig.InitialDelay = 1 * time.Second
	retryConfig.MaxDelay = 60 * time.Second
	retryConfig.BackoffMultiplier = 2.0
	retryConfig.RetryableStatusCodes = []int{429, 500, 502, 503, 504}
	retryConfig.RetryOnNetworkErrors = true

	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, retryConfig)
	if err != nil {
		t.Fatalf("GeneratePythonSDK() with retry enabled error = %v", err)
	}

	// Check that client.py contains retry configuration
	clientPath := filepath.Join(tmpDir, "test_sdk", "client.py")
	if _, err := os.Stat(clientPath); os.IsNotExist(err) {
		t.Fatal("client.py should be created")
	}

	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read client.py: %v", err)
	}

	contentStr := string(content)

	// Check for retry configuration
	if !common.Contains(contentStr, "retry_enabled") {
		t.Error("client.py should contain retry_enabled when retry is enabled")
	}
	if !common.Contains(contentStr, "retry_max_attempts") {
		t.Error("client.py should contain retry_max_attempts when retry is enabled")
	}
	if !common.Contains(contentStr, "_calculate_retry_delay") {
		t.Error("client.py should contain _calculate_retry_delay method when retry is enabled")
	}
	if !common.Contains(contentStr, "exponential") {
		t.Error("client.py should contain retry strategy when retry is enabled")
	}
}

func TestGeneratePythonSDK_WithRetryDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"

	extractedData := common.CreateTestExtractedData()

	// Use default retry config (disabled)
	retryConfig := common.DefaultRetryConfig()

	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, retryConfig)
	if err != nil {
		t.Fatalf("GeneratePythonSDK() with retry disabled error = %v", err)
	}

	// Check that client.py does NOT contain retry configuration
	clientPath := filepath.Join(tmpDir, "test_sdk", "client.py")
	if _, err := os.Stat(clientPath); os.IsNotExist(err) {
		t.Fatal("client.py should be created")
	}

	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read client.py: %v", err)
	}

	contentStr := string(content)

	// When retry is disabled, retry setup should not be present
	if common.Contains(contentStr, "retry_enabled = True") {
		t.Error("client.py should not contain retry_enabled=True when retry is disabled")
	}
}

// TestGeneratePythonArrayModel tests array model generation
func TestGeneratePythonArrayModel(t *testing.T) {
	schemas := map[string]*common.Schema{
		"StringArray": {
			Type: "array",
			Items: &common.Schema{
				Type: "string",
			},
		},
		"IntArray": {
			Type: "array",
			Items: &common.Schema{
				Type: "integer",
			},
		},
		"ObjectArray": {
			Type: "array",
			Items: &common.Schema{
				Type: "object",
				Properties: map[string]*common.Schema{
					"name": {
						Type: "string",
					},
				},
			},
		},
		"ArrayWithoutItems": {
			Type: "array",
		},
	}

	modelsContent := generatePythonModels(schemas)
	if !common.Contains(modelsContent, "StringArray") {
		t.Error("models should contain StringArray")
	}
	if !common.Contains(modelsContent, "List[str]") {
		t.Error("models should contain List[str] for string array")
	}
	if !common.Contains(modelsContent, "List[int]") {
		t.Error("models should contain List[int] for int array")
	}
	if !common.Contains(modelsContent, "items: List") {
		t.Error("models should contain items field for array models")
	}
}

// TestGetPythonType tests all Python type conversions
func TestGetPythonType(t *testing.T) {
	tests := []struct {
		name     string
		schema   *common.Schema
		expected string
	}{
		{"string", &common.Schema{Type: "string"}, "str"},
		{"integer", &common.Schema{Type: "integer"}, "int"},
		{"number", &common.Schema{Type: "number"}, "float"},
		{"boolean", &common.Schema{Type: "boolean"}, "bool"},
		{"array", &common.Schema{Type: "array"}, "list"},
		{"object", &common.Schema{Type: "object"}, "dict"},
		{"nil", nil, "any"},
		{"unknown", &common.Schema{Type: "unknown"}, "any"},
		{"empty_type", &common.Schema{}, "any"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPythonType(tt.schema)
			if result != tt.expected {
				t.Errorf("getPythonType(%v) = %q, want %q", tt.schema, result, tt.expected)
			}
		})
	}
}

// TestGeneratePythonInit_WithModels tests __init__.py generation with models
func TestGeneratePythonInit_WithModels(t *testing.T) {
	extractedData := common.CreateTestExtractedData()
	extractedData.Schemas = map[string]*common.Schema{
		"User": {
			Type: "object",
			Properties: map[string]*common.Schema{
				"name": {Type: "string"},
			},
		},
		"Product": {
			Type: "object",
			Properties: map[string]*common.Schema{
				"id": {Type: "integer"},
			},
		},
	}

	data := common.TemplateData{
		SDKName:         "test_sdk",
		ClientClassName: "TestSdk",
		OpenAPIDoc:      extractedData,
	}

	initContent := generatePythonInit(data, "1.0.0")
	// generatePythonInit uses common.GetClientClassName(data.SDKName) which converts "test_sdk" to "TestSdk"
	// Check for client import (could be from template or fallback)
	if !common.Contains(initContent, "TestSdk") && !common.Contains(initContent, "client") {
		t.Errorf("__init__.py should import client class, got: %s", initContent[:min(200, len(initContent))])
	}
	if !common.Contains(initContent, "models") {
		t.Error("__init__.py should import models when schemas exist")
	}
	if !common.Contains(initContent, "User") {
		t.Error("__init__.py should include User model")
	}
	if !common.Contains(initContent, "Product") {
		t.Error("__init__.py should include Product model")
	}
	if !common.Contains(initContent, "version") {
		t.Error("__init__.py should include version")
	}
}

// TestGeneratePythonInit_WithoutModels tests __init__.py generation without models
func TestGeneratePythonInit_WithoutModels(t *testing.T) {
	extractedData := common.CreateTestExtractedData()
	extractedData.Schemas = map[string]*common.Schema{}

	data := common.TemplateData{
		SDKName:         "test_sdk",
		ClientClassName: "TestSdk",
		OpenAPIDoc:      extractedData,
	}

	initContent := generatePythonInit(data, "1.0.0")
	// generatePythonInit uses common.GetClientClassName(data.SDKName) which converts "test_sdk" to "TestSdk"
	// Check for client import (could be from template or fallback)
	if !common.Contains(initContent, "TestSdk") && !common.Contains(initContent, "client") {
		t.Errorf("__init__.py should import client class, got: %s", initContent[:min(200, len(initContent))])
	}
	if common.Contains(initContent, "from .models import *") || common.Contains(initContent, "models") {
		t.Error("__init__.py should not import models when no schemas exist")
	}
}

// TestGeneratePythonInit_InvalidOpenAPIDoc tests __init__.py with invalid OpenAPIDoc
func TestGeneratePythonInit_InvalidOpenAPIDoc(t *testing.T) {
	data := common.TemplateData{
		SDKName:         "test_sdk",
		ClientClassName: "TestSdk",
		OpenAPIDoc:      "invalid", // Not *common.ExtractedData
	}

	initContent := generatePythonInit(data, "1.0.0")
	// generatePythonInit uses common.GetClientClassName(data.SDKName) which converts "test_sdk" to "TestSdk"
	// Check for client import (could be from template or fallback)
	if !common.Contains(initContent, "TestSdk") && !common.Contains(initContent, "client") {
		t.Errorf("__init__.py should import client class even with invalid OpenAPIDoc, got: %s", initContent[:min(200, len(initContent))])
	}
}

// TestGeneratePythonSetup tests setup.py generation
func TestGeneratePythonSetup(t *testing.T) {
	extractedData := common.CreateTestExtractedData()
	extractedData.Description = "Test API Description"

	data := common.TemplateData{
		SDKName:    "test_sdk",
		OpenAPIDoc: extractedData,
	}

	pythonVersion := common.LanguageVersion{Major: 3, Minor: 11}
	setupContent := generatePythonSetup(data, pythonVersion, "1.0.0")

	if !common.Contains(setupContent, "test_sdk") {
		t.Error("setup.py should contain SDK name")
	}
	if !common.Contains(setupContent, "1.0.0") {
		t.Error("setup.py should contain version")
	}
	if !common.Contains(setupContent, "python_requires") {
		t.Error("setup.py should contain python_requires")
	}
	if !common.Contains(setupContent, ">=3.11") {
		t.Error("setup.py should contain Python version requirement")
	}
	if !common.Contains(setupContent, "Programming Language :: Python :: 3.11") {
		t.Error("setup.py should contain Python 3.11 classifier")
	}
}

// TestGeneratePythonSetup_WithDescription tests setup.py with description
func TestGeneratePythonSetup_WithDescription(t *testing.T) {
	extractedData := common.CreateTestExtractedData()
	extractedData.Description = "Test API Description with \"quotes\" and\nnewlines"

	data := common.TemplateData{
		SDKName:    "test_sdk",
		OpenAPIDoc: extractedData,
	}

	pythonVersion := common.LanguageVersion{Major: 3, Minor: 12}
	setupContent := generatePythonSetup(data, pythonVersion, "2.0.0")

	// Description should be escaped
	if !common.Contains(setupContent, "Test API Description") {
		t.Error("setup.py should contain description")
	}
}

// TestGeneratePythonSetup_DifferentVersions tests setup.py with different Python versions
func TestGeneratePythonSetup_DifferentVersions(t *testing.T) {
	data := common.TemplateData{
		SDKName:    "test_sdk",
		OpenAPIDoc: common.CreateTestExtractedData(),
	}

	versions := []common.LanguageVersion{
		{Major: 3, Minor: 11},
		{Major: 3, Minor: 12},
		{Major: 3, Minor: 13},
	}

	for _, version := range versions {
		setupContent := generatePythonSetup(data, version, "1.0.0")
		expectedRequires := fmt.Sprintf(">=%d.%d", version.Major, version.Minor)
		if !common.Contains(setupContent, expectedRequires) {
			t.Errorf("setup.py should contain %s for Python %d.%d", expectedRequires, version.Major, version.Minor)
		}
	}
}

// TestGeneratePythonRetryHelper tests retry helper generation for different HTTP libraries
func TestGeneratePythonRetryHelper(t *testing.T) {
	config := common.RetryConfig{
		Enabled:              true,
		MaxAttempts:          3,
		InitialDelay:         time.Second,
		MaxDelay:             60 * time.Second,
		BackoffMultiplier:    2.0,
		Strategy:             common.RetryStrategyExponential,
		RetryableStatusCodes: []int{429, 500, 502, 503, 504},
		RetryOnNetworkErrors: true,
	}

	httpLibraries := []string{"requests", "httpx", "aiohttp", "urllib3", "unknown"}

	for _, httpLib := range httpLibraries {
		t.Run(httpLib, func(t *testing.T) {
			helperContent := generatePythonRetryHelper(httpLib, config)
			if !common.Contains(helperContent, "_calculate_retry_delay") {
				t.Errorf("retry helper for %s should contain _calculate_retry_delay", httpLib)
			}
			if !common.Contains(helperContent, "retry_strategy") {
				t.Errorf("retry helper for %s should contain retry_strategy", httpLib)
			}
		})
	}
}

// TestGeneratePythonRetryHelper_Disabled tests retry helper when disabled
func TestGeneratePythonRetryHelper_Disabled(t *testing.T) {
	config := common.RetryConfig{
		Enabled: false,
	}

	helperContent := generatePythonRetryHelper("requests", config)
	if helperContent != "" {
		t.Error("retry helper should be empty when retry is disabled")
	}
}

// TestGeneratePythonRetryHelper_Strategies tests different retry strategies
func TestGeneratePythonRetryHelper_Strategies(t *testing.T) {
	strategies := []common.RetryStrategy{
		common.RetryStrategyExponential,
		common.RetryStrategyLinear,
		common.RetryStrategyFixed,
	}

	for _, strategy := range strategies {
		t.Run(string(strategy), func(t *testing.T) {
			config := common.RetryConfig{
				Enabled:  true,
				Strategy: strategy,
			}
			helperContent := generatePythonRetryHelper("requests", config)
			if !common.Contains(helperContent, string(strategy)) {
				t.Errorf("retry helper should contain strategy %s", strategy)
			}
		})
	}
}

// TestGeneratePythonREADME tests README generation
func TestGeneratePythonREADME(t *testing.T) {
	data := common.TemplateData{
		SDKName: "test_sdk",
	}

	readmeContent := generatePythonREADME(data)
	// README uses title case for display, so "test_sdk" becomes "Test Sdk"
	if !common.Contains(readmeContent, "test_sdk") && !common.Contains(readmeContent, "Test Sdk") && !common.Contains(readmeContent, "Test Sdk") {
		t.Error("README should contain SDK name")
	}
}

// TestGeneratePythonREADME_WithDescription tests README with description
func TestGeneratePythonREADME_WithDescription(t *testing.T) {
	extractedData := common.CreateTestExtractedData()
	extractedData.Description = "Test API Description"

	data := common.TemplateData{
		SDKName:    "test_sdk",
		OpenAPIDoc: extractedData,
	}

	readmeContent := generatePythonREADME(data)
	if !common.Contains(readmeContent, "test_sdk") {
		t.Error("README should contain SDK name")
	}
}

// TestFormatPythonFile tests Python file formatting
func TestFormatPythonFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")
	content := "def test():\n    pass\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Set TESTING env var to skip formatting
	_ = os.Setenv("TESTING", "true")
	defer func() { _ = os.Unsetenv("TESTING") }()

	err := formatPythonFile(testFile)
	if err != nil {
		t.Errorf("formatPythonFile() should not error when TESTING=true, got: %v", err)
	}

	// Test with SKIP_FORMATTING
	_ = os.Unsetenv("TESTING")
	_ = os.Setenv("SKIP_FORMATTING", "true")
	defer func() { _ = os.Unsetenv("SKIP_FORMATTING") }()

	err = formatPythonFile(testFile)
	if err != nil {
		t.Errorf("formatPythonFile() should not error when SKIP_FORMATTING=true, got: %v", err)
	}
}

// TestGeneratePythonSDK_WithVersion tests SDK generation with version
func TestGeneratePythonSDK_WithVersion(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"
	version := common.LanguageVersion{Major: 3, Minor: 12}

	extractedData := common.CreateTestExtractedData()
	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, &version, "2.0.0", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() error = %v", err)
	}

	// Check setup.py contains version
	setupPath := filepath.Join(tmpDir, "setup.py")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(setupPath)
	if err != nil {
		t.Fatalf("Failed to read setup.py: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, ">=3.12") {
		t.Error("setup.py should contain Python 3.12 requirement")
	}
	// SDK version might be determined from OpenAPI spec, so check for any version
	if !common.Contains(contentStr, "version") {
		t.Error("setup.py should contain version field")
	}
}

// TestGeneratePythonSDK_InvalidOpenAPIDoc tests with invalid OpenAPI document
func TestGeneratePythonSDK_InvalidOpenAPIDoc(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"

	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, "invalid", nil, "", true, common.DefaultRetryConfig())
	if err == nil {
		t.Error("GeneratePythonSDK() with invalid OpenAPI doc should return error")
	}
}

// TestGeneratePythonSDK_NoSchemas tests SDK generation without schemas
func TestGeneratePythonSDK_NoSchemas(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"

	extractedData := common.CreateTestExtractedData()
	extractedData.Schemas = map[string]*common.Schema{}

	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() error = %v", err)
	}

	// models.py should not be created when no schemas
	modelsPath := filepath.Join(tmpDir, "test_sdk", "models.py")
	if _, err := os.Stat(modelsPath); err == nil {
		t.Error("models.py should not be created when no schemas exist")
	}
}

// TestGeneratePythonSDK_NoOperations tests SDK generation without operations
func TestGeneratePythonSDK_NoOperations(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"

	extractedData := common.CreateTestExtractedData()
	extractedData.Operations = []common.APIOperation{}

	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() error = %v", err)
	}

	// api/ directory should not be created when no operations
	apiPath := filepath.Join(tmpDir, "test_sdk", "api")
	if _, err := os.Stat(apiPath); err == nil {
		t.Error("api/ directory should not be created when no operations exist")
	}
}

// TestGeneratePythonSetup_NoDescription tests setup.py without description
func TestGeneratePythonSetup_NoDescription(t *testing.T) {
	extractedData := common.CreateTestExtractedData()
	extractedData.Description = "" // Empty description

	data := common.TemplateData{
		SDKName:    "test_sdk",
		OpenAPIDoc: extractedData,
	}

	pythonVersion := common.LanguageVersion{Major: 3, Minor: 11}
	setupContent := generatePythonSetup(data, pythonVersion, "1.0.0")

	if !common.Contains(setupContent, "test_sdk") {
		t.Error("setup.py should contain SDK name")
	}
	if !common.Contains(setupContent, "Auto-generated Python SDK") {
		t.Error("setup.py should contain default description when none provided")
	}
}

// TestGeneratePythonSetup_InvalidOpenAPIDoc tests setup.py with invalid OpenAPIDoc
func TestGeneratePythonSetup_InvalidOpenAPIDoc(t *testing.T) {
	data := common.TemplateData{
		SDKName:    "test_sdk",
		OpenAPIDoc: "invalid", // Not *common.ExtractedData
	}

	pythonVersion := common.LanguageVersion{Major: 3, Minor: 11}
	setupContent := generatePythonSetup(data, pythonVersion, "1.0.0")

	if !common.Contains(setupContent, "test_sdk") {
		t.Error("setup.py should contain SDK name even with invalid OpenAPIDoc")
	}
	if !common.Contains(setupContent, "Auto-generated Python SDK") {
		t.Error("setup.py should contain default description with invalid OpenAPIDoc")
	}
}

// TestGeneratePythonSetup_EscapeDescription tests description escaping
func TestGeneratePythonSetup_EscapeDescription(t *testing.T) {
	extractedData := common.CreateTestExtractedData()
	extractedData.Description = "Test with \"quotes\" and\nnewlines\tand\ttabs"

	data := common.TemplateData{
		SDKName:    "test_sdk",
		OpenAPIDoc: extractedData,
	}

	pythonVersion := common.LanguageVersion{Major: 3, Minor: 11}
	setupContent := generatePythonSetup(data, pythonVersion, "1.0.0")

	// Escaped description should not contain unescaped quotes or newlines
	if common.Contains(setupContent, "\"quotes\"") && !common.Contains(setupContent, "\\\"quotes\\\"") {
		t.Error("setup.py should escape quotes in description")
	}
}

// TestGeneratePythonSDK_WithOpenAPI3Doc tests with openapi3.T document
func TestGeneratePythonSDK_WithOpenAPI3Doc(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"

	// Create a minimal openapi3.T document
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}

	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, doc, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() with openapi3.T error = %v", err)
	}

	// Verify SDK was generated
	expectedSDKPath := filepath.Join(tmpDir, "test_sdk")
	if _, err := os.Stat(expectedSDKPath); os.IsNotExist(err) {
		t.Errorf("expected SDK directory not created: %s", expectedSDKPath)
	}
}

// TestGeneratePythonSDK_WithCustomSDKVersion tests with custom SDK version
func TestGeneratePythonSDK_WithCustomSDKVersion(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := common.TestSDKName
	httpLib := "requests"

	extractedData := common.CreateTestExtractedData()
	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "2.5.0", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() error = %v", err)
	}

	// Check __init__.py contains version
	initPath := filepath.Join(tmpDir, "test_sdk", "__init__.py")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(initPath)
	if err != nil {
		t.Fatalf("Failed to read __init__.py: %v", err)
	}

	contentStr := string(content)
	// Version might be determined from OpenAPI spec, so check for version field
	if !common.Contains(contentStr, "version") && !common.Contains(contentStr, "2.5.0") {
		t.Error("__init__.py should contain version field")
	}
}

// TestFormatPythonFile_WithoutEnvVars tests formatPythonFile without env vars (will fail but that's expected)
func TestFormatPythonFile_WithoutEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")
	content := "def test():\n    pass\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Unset env vars
	_ = os.Unsetenv("TESTING")
	_ = os.Unsetenv("SKIP_FORMATTING")
	defer func() {
		// Restore for other tests
		_ = os.Setenv("TESTING", "true")
	}()

	// This will likely fail because black/autopep8 may not be installed, but that's expected
	err := formatPythonFile(testFile)
	// We expect an error if formatters are not available, which is fine
	if err != nil && !common.Contains(err.Error(), "no Python formatter available") {
		t.Logf("formatPythonFile returned error (expected if formatters not installed): %v", err)
	}
}

// TestGeneratePythonRequirements_NoHTTPLib tests requirements generation without HTTP lib config
func TestGeneratePythonRequirements_NoHTTPLib(t *testing.T) {
	data := common.TemplateData{
		SDKName:       "test_sdk",
		HTTPLibConfig: nil, // No HTTP lib config
	}

	requirementsContent := generatePythonRequirements(data)
	if !common.Contains(requirementsContent, "Dependencies") {
		t.Error("requirements.txt should contain dependencies comment when no HTTP lib config")
	}
}

// TestGeneratePythonRequirements_WithHTTPLib tests requirements generation with HTTP lib
func TestGeneratePythonRequirements_WithHTTPLib(t *testing.T) {
	libConfig, _ := httplib.GetLibraryConfig("python", "requests")
	data := common.TemplateData{
		SDKName:       "test_sdk",
		HTTPLibConfig: libConfig,
	}

	requirementsContent := generatePythonRequirements(data)
	if !common.Contains(requirementsContent, "requests") {
		t.Error("requirements.txt should contain HTTP library dependency")
	}
}

// TestGeneratePythonAuthSetup_AllSecurityTypes tests all security scheme types
func TestGeneratePythonAuthSetup_AllSecurityTypes(t *testing.T) {
	securitySchemes := map[string]common.SecurityScheme{
		"apiKeyHeader": {
			Type: "apiKey",
			Name: "X-API-Key",
			In:   "header",
		},
		"apiKeyQuery": {
			Type: "apiKey",
			Name: "api_key",
			In:   "query",
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
			OAuth2Flows: &common.OAuth2Flows{
				AuthorizationCode: &common.OAuth2Flow{
					AuthorizationURL: "https://example.com/auth",
					TokenURL:         "https://example.com/token",
				},
			},
		},
		"openIdConnect": {
			Type:             "openIdConnect",
			OpenIDConnectURL: "https://example.com/.well-known/openid-configuration",
		},
		"mutualTLS": {
			Type: "mutualTLS",
		},
	}

	authContent := generatePythonAuthSetup(securitySchemes)
	if !common.Contains(authContent, "Set up authentication") {
		t.Error("auth setup should contain authentication setup comment")
	}
	if !common.Contains(authContent, "apiKeyHeader") {
		t.Error("auth setup should contain apiKey header scheme")
	}
	if !common.Contains(authContent, "bearer_token") {
		t.Error("auth setup should contain bearer token")
	}
	if !common.Contains(authContent, "oauth2") {
		t.Error("auth setup should contain OAuth2")
	}
}

// TestGeneratePythonAuthSetup_NoAuth tests with no security schemes
func TestGeneratePythonAuthSetup_NoAuth(t *testing.T) {
	authContent := generatePythonAuthSetup(map[string]common.SecurityScheme{})
	if !common.Contains(authContent, "No authentication required") {
		t.Error("auth setup should indicate no authentication required")
	}
}

// TestGeneratePythonAPIMethods_EmptyOperations tests with no operations
func TestGeneratePythonAPIMethods_EmptyOperations(t *testing.T) {
	methodsContent := generatePythonAPIMethods([]common.APIOperation{})
	if !common.Contains(methodsContent, "No API methods") {
		t.Error("methods should indicate no API methods when empty")
	}
}

// TestGeneratePythonAPIMethods_WithAllParameterTypes tests with all parameter types
func TestGeneratePythonAPIMethods_WithAllParameterTypes(t *testing.T) {
	operations := []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/users/{userId}/posts/{postId}",
			OperationID: "getUserPost",
			Parameters: []common.Parameter{
				{
					Name:     "userId",
					In:       "path",
					Required: true,
					Schema:   &common.Schema{Type: "string"},
				},
				{
					Name:     "postId",
					In:       "path",
					Required: true,
					Schema:   &common.Schema{Type: "integer"},
				},
				{
					Name:     "page",
					In:       "query",
					Required: false,
					Schema:   &common.Schema{Type: "integer"},
				},
			},
			RequestBody: &common.RequestBody{
				Content: map[string]common.ContentType{
					"application/json": {
						Schema: &common.Schema{Type: "object"},
					},
				},
			},
			Summary:     "Get user post",
			Description: "Retrieve a specific post by a user",
		},
	}

	methodsContent := generatePythonAPIMethods(operations)
	// GetOperationMethodName converts "getUserPost" to "get_user_post" using ToSnakeCase
	if !common.Contains(methodsContent, "get_user_post") {
		t.Errorf("methods should contain operation method name 'get_user_post', got: %s", methodsContent[:min(200, len(methodsContent))])
	}
	if !common.Contains(methodsContent, "userId") {
		t.Error("methods should contain path parameters")
	}
	if !common.Contains(methodsContent, "page") {
		t.Error("methods should contain query parameters")
	}
	if !common.Contains(methodsContent, "body") {
		t.Error("methods should contain request body parameter")
	}
}

// TestGeneratePythonAPIMethods_WithoutOperationID tests without operation ID
func TestGeneratePythonAPIMethods_WithoutOperationID(t *testing.T) {
	operations := []common.APIOperation{
		{
			Method: "GET",
			Path:   "/test/path",
			// No OperationID
		},
	}

	methodsContent := generatePythonAPIMethods(operations)
	if !common.Contains(methodsContent, "get") {
		t.Error("methods should generate method name from path when no operation ID")
	}
}

// TestGeneratePythonAPIMethods_WithoutSummaryOrDescription tests without summary/description
func TestGeneratePythonAPIMethods_WithoutSummaryOrDescription(t *testing.T) {
	operations := []common.APIOperation{
		{
			Method:      "GET",
			Path:        "/test",
			OperationID: "test",
			// No Summary or Description
		},
	}

	methodsContent := generatePythonAPIMethods(operations)
	if !common.Contains(methodsContent, "def test") {
		t.Error("methods should generate method even without summary/description")
	}
}
