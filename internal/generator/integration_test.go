package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/vubon/sdk-forge/internal/generator/common"
	gogen "github.com/vubon/sdk-forge/internal/generator/go"
	"github.com/vubon/sdk-forge/internal/generator/python"
)

const (
	testHTTPLib = "requests"
	testSDKName = "test-sdk"
)

// TestIntegration_PythonSDKGeneration tests the complete Python SDK generation flow
func TestIntegration_PythonSDKGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := "my-api-sdk"
	httpLib := testHTTPLib

	// Create a minimal OpenAPI document
	doc := common.CreateTestOpenAPIDoc()

	// Add a simple path
	pathItem := &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "listItems",
			Summary:     "List items",
			Responses:   openapi3.NewResponses(),
		},
	}
	desc := "Success"
	pathItem.Get.Responses.Set("200", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: &desc,
		},
	})
	doc.Paths.Set("/items", pathItem)

	err := python.GeneratePythonSDK(tmpDir, sdkName, httpLib, doc, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() error = %v", err)
	}

	// Verify directory structure
	expectedDir := filepath.Join(tmpDir, "my_api_sdk")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("Expected package directory not found: %s", expectedDir)
	}

	// Verify all required files exist
	requiredFiles := []string{
		filepath.Join(expectedDir, "__init__.py"),
		filepath.Join(expectedDir, "client.py"),
		filepath.Join(tmpDir, "requirements.txt"),
		filepath.Join(tmpDir, "README.md"),
	}

	for _, file := range requiredFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Required file not generated: %s", file)
		}
	}

	// Verify file contents are not empty
	for _, file := range requiredFiles {
		// #nosec G304 -- File path is from test, safe to read
		content, err := os.ReadFile(file)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", file, err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("File %s is empty", file)
		}
	}

	// Verify requirements.txt contains the HTTP library
	// #nosec G304 -- File path is from test, safe to read
	requirementsContent, _ := os.ReadFile(filepath.Join(tmpDir, "requirements.txt"))
	if !common.Contains(string(requirementsContent), "requests") {
		t.Error("requirements.txt should contain 'requests' dependency")
	}

	// Verify client.py contains the HTTP library import
	// #nosec G304 -- File path is from test, safe to read
	clientContent, _ := os.ReadFile(filepath.Join(expectedDir, "client.py"))
	if !common.Contains(string(clientContent), "import requests") {
		t.Error("client.py should import requests")
	}
}

// TestIntegration_PythonSDKWithCustomHTTPLib tests SDK generation with custom HTTP library
func TestIntegration_PythonSDKWithCustomHTTPLib(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := testSDKName
	httpLib := "httpx"

	extractedData := common.CreateTestExtractedData()
	err := python.GeneratePythonSDK(tmpDir, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GeneratePythonSDK() error = %v", err)
	}

	// Verify client.py uses httpx
	clientPath := filepath.Join(tmpDir, "test_sdk", "client.py")
	// #nosec G304 -- File path is from test, safe to read
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("Failed to read client.py: %v", err)
	}

	contentStr := string(content)
	if !common.Contains(contentStr, "httpx") {
		t.Error("client.py should use httpx when specified")
	}

	// Verify requirements.txt contains httpx
	requirementsPath := filepath.Join(tmpDir, "requirements.txt")
	// #nosec G304 -- File path is from test, safe to read
	reqContent, err := os.ReadFile(requirementsPath)
	if err != nil {
		t.Fatalf("Failed to read requirements.txt: %v", err)
	}

	if !common.Contains(string(reqContent), "httpx") {
		t.Error("requirements.txt should contain httpx dependency")
	}
}

// TestIntegration_GoSDKGeneration tests the complete Go SDK generation flow
func TestIntegration_GoSDKGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := "my-api-sdk"
	httpLib := "nethttp"

	// Create a minimal OpenAPI document
	doc := common.CreateTestOpenAPIDoc()

	// Add a simple path
	pathItem := &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "listItems",
			Summary:     "List items",
			Responses:   openapi3.NewResponses(),
		},
	}
	desc := "Success"
	pathItem.Get.Responses.Set("200", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: &desc,
		},
	})
	doc.Paths.Set("/items", pathItem)

	// outputPath should include the SDK name (like CLI does)
	const expectedGoSDKName = "myapisdk" // "my-api-sdk" -> "myapisdk"
	outputPath := filepath.Join(tmpDir, expectedGoSDKName)
	err := gogen.GenerateGoSDK(outputPath, sdkName, httpLib, doc, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateGoSDK() error = %v", err)
	}

	// Verify all required files exist (files are in outputPath directly)
	requiredFiles := []string{
		filepath.Join(outputPath, "go.mod"),
		filepath.Join(outputPath, "client.go"),
		filepath.Join(outputPath, "README.md"),
	}

	for _, file := range requiredFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Required file not generated: %s", file)
		}
	}

	// Verify file contents are not empty
	for _, file := range requiredFiles {
		// #nosec G304 -- File path is from test, safe to read
		content, err := os.ReadFile(file)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", file, err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("File %s is empty", file)
		}
	}

	// Verify go.mod contains module declaration
	// #nosec G304 -- File path is from test, safe to read
	goModContent, _ := os.ReadFile(filepath.Join(outputPath, "go.mod"))
	if !common.Contains(string(goModContent), "module") {
		t.Error("go.mod should contain 'module' declaration")
	}

	// Verify client.go contains package declaration
	// #nosec G304 -- File path is from test, safe to read
	clientContent, _ := os.ReadFile(filepath.Join(outputPath, "client.go"))
	if !common.Contains(string(clientContent), "package") {
		t.Error("client.go should contain package declaration")
	}

	if !common.Contains(string(clientContent), "Client") {
		t.Error("client.go should contain Client struct")
	}
}

// TestIntegration_GoSDKWithCustomHTTPLib tests Go SDK generation with custom HTTP library
func TestIntegration_GoSDKWithCustomHTTPLib(t *testing.T) {
	tmpDir := t.TempDir()
	sdkName := testSDKName
	httpLib := "resty"

	extractedData := common.CreateTestExtractedData()
	// outputPath should include the SDK name (like CLI does)
	const expectedGoSDKName = "testsdk"
	outputPath := filepath.Join(tmpDir, expectedGoSDKName)
	err := gogen.GenerateGoSDK(outputPath, sdkName, httpLib, extractedData, nil, "", true, common.DefaultRetryConfig())
	if err != nil {
		t.Fatalf("GenerateGoSDK() error = %v", err)
	}

	// Verify go.mod exists (files are in outputPath directly)
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
