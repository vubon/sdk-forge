package python

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vubon/sdk-forge/internal/generator/common"

	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

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
