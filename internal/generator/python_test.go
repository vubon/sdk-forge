package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

// createTestExtractedData creates a minimal ExtractedData for testing
func createTestExtractedData() *ExtractedData {
	return &ExtractedData{
		BaseURL:         "https://api.example.com/v1",
		Title:           "Test API",
		Version:         "1.0.0",
		Operations:      []APIOperation{},
		Schemas:         make(map[string]*Schema),
		SecuritySchemes: make(map[string]SecurityScheme),
	}
}

// createTestOpenAPIDoc creates a minimal openapi3.T for testing
func createTestOpenAPIDoc() *openapi3.T {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Servers: openapi3.Servers{
			&openapi3.Server{
				URL: "https://api.example.com/v1",
			},
		},
		Paths: openapi3.NewPaths(),
	}
	return doc
}

func TestGeneratePythonSDK(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := testSDKName
	httpLib := "requests"

	// Use ExtractedData for testing
	extractedData := createTestExtractedData()
	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "")
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

	extractedData := createTestExtractedData()
	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "")
	if err == nil {
		t.Error("GeneratePythonSDK() with invalid HTTP library should return error")
	}
}

func TestGeneratePythonSDK_CustomHTTPLib(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := testSDKName
	httpLib := "httpx"

	extractedData := createTestExtractedData()
	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "")
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
	if !contains(contentStr, "httpx") {
		t.Error("GeneratePythonSDK() should use httpx in client.py")
	}
}

func TestGeneratePythonSDK_RequirementsTxt(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := testSDKName
	httpLib := "requests"

	extractedData := createTestExtractedData()
	err := GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "")
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
	if !contains(contentStr, "requests") {
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
		{"test-sdk", testSDKName, "test_sdk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			extractedData := createTestExtractedData()
			err := GeneratePythonSDK(tmpDir, tt.sdkName, "requests", extractedData, nil, "")
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

func TestGeneratePythonInit(t *testing.T) {
	extractedData := createTestExtractedData()
	data := TemplateData{
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
	if !contains(content, "test") && !contains(content, "Test") {
		t.Error("generatePythonInit() should include SDK name or client class")
	}

	// Check for client import
	if !contains(content, "client") && !contains(content, "Client") {
		t.Error("generatePythonInit() should include client import")
	}
}

func TestGeneratePythonClient(t *testing.T) {
	extractedData := createTestExtractedData()
	data := TemplateData{
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

	if !contains(content, "requests") {
		t.Error("generatePythonClient() should include HTTP library import")
	}

	if !contains(content, "Test") && !contains(content, "test") {
		t.Error("generatePythonClient() should include client class name")
	}
}

func TestGeneratePythonRequirements(t *testing.T) {
	extractedData := createTestExtractedData()
	data := TemplateData{
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

	if !contains(content, "requests") {
		t.Error("generatePythonRequirements() should include HTTP library dependency")
	}
}
