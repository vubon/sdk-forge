// Package python provides Python test generation functionality.
package python

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vubon/sdk-forge/internal/generator/common"
)

func generatePythonTests(outputPath, packageDir string, data common.TemplateData, extractedData *common.ExtractedData) error {
	// Create tests/ directory
	testsDir := filepath.Join(outputPath, "tests")
	if err := os.MkdirAll(testsDir, 0750); err != nil {
		return fmt.Errorf("failed to create tests directory: %w", err)
	}

	// Generate tests/__init__.py
	initContent := "# Test package\n"
	initPath := filepath.Join(testsDir, "__init__.py")
	// #nosec G306 -- 0644 is appropriate for Python package files
	if err := os.WriteFile(initPath, []byte(initContent), 0644); err != nil {
		return fmt.Errorf("failed to write tests/__init__.py: %w", err)
	}

	// Generate tests/conftest.py (pytest fixtures)
	conftestContent := generatePythonConftest(data, extractedData)
	conftestPath := filepath.Join(testsDir, "conftest.py")
	// #nosec G306 -- 0644 is appropriate for Python test files
	if err := os.WriteFile(conftestPath, []byte(conftestContent), 0644); err != nil {
		return fmt.Errorf("failed to write tests/conftest.py: %w", err)
	}
	// Format with black (if available)
	if err := formatPythonFile(conftestPath); err != nil {
		_ = err
	}

	// Generate tests/test_client.py
	clientTestContent := generatePythonClientTest(data, extractedData)
	clientTestPath := filepath.Join(testsDir, "test_client.py")
	// #nosec G306 -- 0644 is appropriate for Python test files
	if err := os.WriteFile(clientTestPath, []byte(clientTestContent), 0644); err != nil {
		return fmt.Errorf("failed to write tests/test_client.py: %w", err)
	}
	// Format with black (if available)
	if err := formatPythonFile(clientTestPath); err != nil {
		_ = err
	}

	// Generate tests/test_models.py if schemas exist
	if len(extractedData.Schemas) > 0 {
		modelsTestContent := generatePythonModelsTest(data, extractedData.Schemas)
		modelsTestPath := filepath.Join(testsDir, "test_models.py")
		// #nosec G306 -- 0644 is appropriate for Python test files
		if err := os.WriteFile(modelsTestPath, []byte(modelsTestContent), 0644); err != nil {
			return fmt.Errorf("failed to write tests/test_models.py: %w", err)
		}
		// Format with black (if available)
		if err := formatPythonFile(modelsTestPath); err != nil {
			_ = err
		}
	}

	// Generate tests/test_api_methods.py if operations exist
	if len(extractedData.Operations) > 0 {
		apiTestContent := generatePythonAPITest(data, extractedData.Operations, extractedData)
		apiTestPath := filepath.Join(testsDir, "test_api_methods.py")
		// #nosec G306 -- 0644 is appropriate for Python test files
		if err := os.WriteFile(apiTestPath, []byte(apiTestContent), 0644); err != nil {
			return fmt.Errorf("failed to write tests/test_api_methods.py: %w", err)
		}
		// Format with black (if available)
		if err := formatPythonFile(apiTestPath); err != nil {
			_ = err
		}
	}

	// Generate tests/test_auth.py if security schemes exist
	if len(extractedData.SecuritySchemes) > 0 {
		authTestContent := generatePythonAuthTest(data, extractedData.SecuritySchemes, extractedData)
		authTestPath := filepath.Join(testsDir, "test_auth.py")
		// #nosec G306 -- 0644 is appropriate for Python test files
		if err := os.WriteFile(authTestPath, []byte(authTestContent), 0644); err != nil {
			return fmt.Errorf("failed to write tests/test_auth.py: %w", err)
		}
		// Format with black (if available)
		if err := formatPythonFile(authTestPath); err != nil {
			_ = err
		}
	}

	// Generate test fixtures from OpenAPI examples
	if err := generatePythonTestFixtures(testsDir, extractedData); err != nil {
		return fmt.Errorf("failed to generate test fixtures: %w", err)
	}

	return nil
}

// generatePythonTestFixtures generates test fixture files from OpenAPI examples
func generatePythonTestFixtures(testsDir string, extractedData *common.ExtractedData) error {
	fixturesDir := filepath.Join(testsDir, "fixtures")
	if err := os.MkdirAll(fixturesDir, 0750); err != nil {
		return fmt.Errorf("failed to create fixtures directory: %w", err)
	}

	// Generate fixtures from response examples
	fixtures := make(map[string]interface{})

	for _, op := range extractedData.Operations {
		for statusCode, response := range op.Responses {
			if jsonContent, ok := response.Content["application/json"]; ok {
				if len(jsonContent.Examples) > 0 {
					// Use first example for each status code
					for name, example := range jsonContent.Examples {
						key := fmt.Sprintf("%s_%s_%s", op.OperationID, statusCode, name)
						fixtures[key] = example
					}
				}
			}
		}
	}

	if len(fixtures) > 0 {
		// Generate fixtures.py file
		fixturesContent := generatePythonFixturesFile(fixtures)
		fixturesPath := filepath.Join(fixturesDir, "fixtures.py")
		// #nosec G306 -- 0644 is appropriate for Python fixture files
		if err := os.WriteFile(fixturesPath, []byte(fixturesContent), 0644); err != nil {
			return fmt.Errorf("failed to write fixtures/fixtures.py: %w", err)
		}
	}

	return nil
}

// generatePythonFixturesFile generates a Python file with test fixtures
func generatePythonFixturesFile(fixtures map[string]interface{}) string {
	var buf bytes.Buffer
	buf.WriteString("\"\"\"Test fixtures extracted from OpenAPI examples\"\"\"\n\n")

	for key, example := range fixtures {
		// Convert key to valid Python variable name
		varName := common.ToSnakeCase(key)
		exampleJSON := formatExampleForPython(example)
		fmt.Fprintf(&buf, "%s = %s\n\n", varName, exampleJSON)
	}

	return buf.String()
}

// generatePythonConftest generates pytest conftest.py with fixtures
func generatePythonConftest(data common.TemplateData, extractedData *common.ExtractedData) string {
	var conftest bytes.Buffer
	conftest.WriteString("import pytest\n")
	conftest.WriteString(fmt.Sprintf("from %s import %s\n\n", data.SDKName, data.ClientClassName))
	conftest.WriteString("@pytest.fixture\n")
	conftest.WriteString("def client():\n")
	conftest.WriteString("    \"\"\"Create a test client instance.\"\"\"\n")

	// Use base URL from OpenAPI spec, fallback to default
	baseURL := extractedData.BaseURL
	if baseURL == "" {
		baseURL = "https://api.example.com/v1"
	}
	conftest.WriteString(fmt.Sprintf("    return %s(base_url=%q)\n", data.ClientClassName, baseURL))
	return conftest.String()
}

// generatePythonClientTest generates test_client.py
func generatePythonClientTest(data common.TemplateData, extractedData *common.ExtractedData) string {
	var test bytes.Buffer
	test.WriteString("import pytest\n")
	test.WriteString(fmt.Sprintf("from %s import %s\n\n\n", data.SDKName, data.ClientClassName))
	test.WriteString("def test_client_initialization():\n")
	test.WriteString("    \"\"\"Test client can be instantiated.\"\"\"\n")

	// Use base URL from OpenAPI spec, fallback to default
	baseURL := extractedData.BaseURL
	if baseURL == "" {
		baseURL = "https://api.example.com/v1"
	}
	test.WriteString(fmt.Sprintf("    client = %s(base_url=%q)\n", data.ClientClassName, baseURL))
	test.WriteString(fmt.Sprintf("    assert client.base_url == %q\n", baseURL))
	test.WriteString("    assert client.session is not None\n")
	return test.String()
}

// generatePythonModelsTest generates test_models.py with schema-based tests
func generatePythonModelsTest(data common.TemplateData, schemas map[string]*common.Schema) string {
	var test bytes.Buffer
	test.WriteString("import pytest\n")
	test.WriteString("from dataclasses import asdict\n")
	test.WriteString(fmt.Sprintf("from %s import models\n\n\n", data.SDKName))

	// Generate tests for each schema
	for name, schema := range schemas {
		className := common.ToPascalCase(name)
		test.WriteString(fmt.Sprintf("class Test%s:\n", className))
		test.WriteString(fmt.Sprintf("    \"\"\"Tests for %s model\"\"\"\n\n", className))

		// Test model instantiation
		test.WriteString(fmt.Sprintf("    def test_%s_creation(self):\n", common.ToSnakeCase(name)))
		test.WriteString(fmt.Sprintf("        \"\"\"Test %s can be instantiated.\"\"\"\n", className))

		// Generate test data based on schema properties
		if schema.Type == "object" && len(schema.Properties) > 0 {
			test.WriteString(fmt.Sprintf("        model = models.%s(\n", className))

			// Track required fields
			requiredSet := make(map[string]bool)
			for _, req := range schema.Required {
				requiredSet[req] = true
			}

			// Generate test values for each property
			first := true
			for propName, propSchema := range schema.Properties {
				if propSchema == nil {
					continue
				}

				if !first {
					test.WriteString(",\n")
				}
				first = false

				propSnakeName := common.ToSnakeCase(propName)
				testValue := generatePythonTestValue(propSchema, propName)

				if requiredSet[propName] {
					test.WriteString(fmt.Sprintf("            %s=%s", propSnakeName, testValue))
				} else {
					// Optional fields can be None or have a value
					test.WriteString(fmt.Sprintf("            %s=%s", propSnakeName, testValue))
				}
			}
			test.WriteString("\n        )\n")
			test.WriteString("        assert model is not None\n")

			// Test property access
			for propName := range schema.Properties {
				propSnakeName := common.ToSnakeCase(propName)
				test.WriteString(fmt.Sprintf("        assert hasattr(model, '%s')\n", propSnakeName))
			}
		} else {
			// Primitive/enum model (has 'value' field) or model without properties
			// Primitive models (string with enum, etc.) have a 'value' field
			if schema.Type != "object" && schema.Type != "array" {
				// Primitive type model - has required 'value' field
				testValue := generatePythonTestValue(schema, "value")
				test.WriteString(fmt.Sprintf("        model = models.%s(value=%s)\n", className, testValue))
			} else if len(schema.Required) > 0 {
				// Object model with required fields but no properties (edge case)
				test.WriteString(fmt.Sprintf("        model = models.%s(\n", className))
				first := true
				for _, reqField := range schema.Required {
					if !first {
						test.WriteString(",\n")
					}
					first = false
					fieldSnakeName := common.ToSnakeCase(reqField)
					test.WriteString(fmt.Sprintf("            %s=\"test_value\"", fieldSnakeName))
				}
				test.WriteString("\n        )\n")
			} else {
				// No required fields, can instantiate without arguments
				test.WriteString(fmt.Sprintf("        model = models.%s()\n", className))
			}
			test.WriteString("        assert model is not None\n")
		}
		test.WriteString("\n")

		// Test serialization (if dataclass)
		test.WriteString(fmt.Sprintf("    def test_%s_serialization(self):\n", common.ToSnakeCase(name)))
		test.WriteString(fmt.Sprintf("        \"\"\"Test %s can be serialized to dict.\"\"\"\n", className))
		if schema.Type == "object" && len(schema.Properties) > 0 {
			test.WriteString(fmt.Sprintf("        model = models.%s(\n", className))

			requiredSet := make(map[string]bool)
			for _, req := range schema.Required {
				requiredSet[req] = true
			}

			first := true
			for propName, propSchema := range schema.Properties {
				if propSchema == nil {
					continue
				}
				if !first {
					test.WriteString(",\n")
				}
				first = false
				propSnakeName := common.ToSnakeCase(propName)
				testValue := generatePythonTestValue(propSchema, propName)
				test.WriteString(fmt.Sprintf("            %s=%s", propSnakeName, testValue))
			}
			test.WriteString("\n        )\n")
			test.WriteString("        data = asdict(model)\n")
			test.WriteString("        assert isinstance(data, dict)\n")
		} else {
			// Primitive/enum model (has 'value' field) or model without properties
			if schema.Type != "object" && schema.Type != "array" {
				// Primitive type model - has required 'value' field
				testValue := generatePythonTestValue(schema, "value")
				test.WriteString(fmt.Sprintf("        model = models.%s(value=%s)\n", className, testValue))
			} else if len(schema.Required) > 0 {
				// Object model with required fields but no properties (edge case)
				test.WriteString(fmt.Sprintf("        model = models.%s(\n", className))
				first := true
				for _, reqField := range schema.Required {
					if !first {
						test.WriteString(",\n")
					}
					first = false
					fieldSnakeName := common.ToSnakeCase(reqField)
					test.WriteString(fmt.Sprintf("            %s=\"test_value\"", fieldSnakeName))
				}
				test.WriteString("\n        )\n")
			} else {
				// No required fields
				test.WriteString(fmt.Sprintf("        model = models.%s()\n", className))
			}
			test.WriteString("        data = asdict(model)\n")
			test.WriteString("        assert isinstance(data, dict)\n")
		}
		test.WriteString("\n\n")

		// Test required fields validation (if any)
		if len(schema.Required) > 0 {
			test.WriteString(fmt.Sprintf("    def test_%s_required_fields(self):\n", common.ToSnakeCase(name)))
			test.WriteString(fmt.Sprintf("        \"\"\"Test %s required fields are enforced.\"\"\"\n", className))
			test.WriteString("        # Note: dataclasses don't enforce required fields at runtime\n")
			test.WriteString("        # This test serves as documentation of required fields\n")
			test.WriteString("        required_fields = [")
			for i, req := range schema.Required {
				if i > 0 {
					test.WriteString(", ")
				}
				test.WriteString(fmt.Sprintf("\"%s\"", common.ToSnakeCase(req)))
			}
			test.WriteString("]\n")
			test.WriteString("        assert len(required_fields) > 0\n")
			test.WriteString("\n\n")
		}
	}

	return test.String()
}

// generatePythonTestValue generates a test value for a schema property
func generatePythonTestValue(schema *common.Schema, propName string) string {
	if schema == nil {
		return "\"test_value\""
	}

	switch schema.Type {
	case "string":
		if schema.Format == "date" {
			return "\"2024-01-01\""
		}
		if schema.Format == "date-time" {
			return "\"2024-01-01T00:00:00Z\""
		}
		if schema.Format == "email" {
			return "\"test@example.com\""
		}
		return fmt.Sprintf("\"test_%s\"", common.ToSnakeCase(propName))
	case "integer", "number":
		return "42"
	case "boolean":
		return "True"
	case "array":
		if schema.Items != nil {
			itemValue := generatePythonTestValue(schema.Items, "item")
			return fmt.Sprintf("[%s]", itemValue)
		}
		return "[]"
	case "object":
		return "{}"
	default:
		return "\"test_value\""
	}
}

// generatePythonAPITest generates test_api_methods.py with operation-based tests
func generatePythonAPITest(data common.TemplateData, operations []common.APIOperation, extractedData *common.ExtractedData) string {
	var test bytes.Buffer
	test.WriteString("import pytest\n")
	test.WriteString("from unittest.mock import Mock, patch\n")
	test.WriteString(fmt.Sprintf("from %s import %s\n\n\n", data.SDKName, data.ClientClassName))

	// Group operations by tag for better organization
	operationsByTag := common.GroupOperationsByTag(operations)

	// Generate tests for each tag/group
	for tag, tagOperations := range operationsByTag {
		test.WriteString(fmt.Sprintf("class Test%sAPI:\n", common.ToPascalCase(tag)))
		test.WriteString(fmt.Sprintf("    \"\"\"Tests for %s API methods\"\"\"\n\n", tag))

		// Generate test for each operation
		for _, op := range tagOperations {
			methodName := common.GetOperationMethodName(op)
			testMethodName := fmt.Sprintf("test_%s", methodName)

			// Patch requests.Session.request since client uses session.request()
			test.WriteString("    @patch('requests.Session.request')\n")
			test.WriteString(fmt.Sprintf("    def %s(self, mock_request):\n", testMethodName))
			test.WriteString(fmt.Sprintf("        \"\"\"Test %s %s operation.\"\"\"\n", op.Method, op.Path))

			// Setup mock response
			test.WriteString("        # Setup mock response\n")
			test.WriteString("        mock_response = Mock()\n")

			// Determine expected status code from responses
			expectedStatus := "200"
			if _, ok := op.Responses["200"]; !ok {
				// Find first available status code
				for statusCode := range op.Responses {
					expectedStatus = statusCode
					break
				}
			}

			test.WriteString(fmt.Sprintf("        mock_response.status_code = %s\n", expectedStatus))

			// Use example from OpenAPI response if available
			exampleJSON := getPythonExampleFromResponse(op.Responses[expectedStatus])
			test.WriteString(fmt.Sprintf("        mock_response.json.return_value = %s\n", exampleJSON))
			// Convert Python dict to JSON string for text response
			jsonText := strings.ReplaceAll(exampleJSON, "True", "true")
			jsonText = strings.ReplaceAll(jsonText, "False", "false")
			jsonText = strings.ReplaceAll(jsonText, "None", "null")
			test.WriteString(fmt.Sprintf("        mock_response.text = %q\n", jsonText))
			test.WriteString("        mock_request.return_value = mock_response\n\n")

			// Create client
			test.WriteString("        # Create client\n")
			// Use base URL from OpenAPI spec, fallback to default
			baseURL := extractedData.BaseURL
			if baseURL == "" {
				baseURL = "https://api.example.com/v1"
			}
			test.WriteString(fmt.Sprintf("        client = %s(base_url=%q)\n\n", data.ClientClassName, baseURL))

			// Generate method call with parameters
			test.WriteString("        # Call API method\n")
			test.WriteString(fmt.Sprintf("        result = client.%s(", methodName))

			// Add path parameters
			hasParams := false
			for _, param := range op.Parameters {
				if param.In == "path" {
					if hasParams {
						test.WriteString(", ")
					}
					paramName := common.ToSnakeCase(param.Name)
					testValue := generatePythonTestValueFromParam(param)
					test.WriteString(fmt.Sprintf("%s=%s", paramName, testValue))
					hasParams = true
				}
			}

			// Add query parameters
			for _, param := range op.Parameters {
				if param.In == "query" {
					if hasParams {
						test.WriteString(", ")
					}
					paramName := common.ToSnakeCase(param.Name)
					testValue := generatePythonTestValueFromParam(param)
					test.WriteString(fmt.Sprintf("%s=%s", paramName, testValue))
					hasParams = true
				}
			}

			// Add request body if present
			if op.RequestBody != nil {
				if hasParams {
					test.WriteString(", ")
				}
				test.WriteString("body={\"test\": \"data\"}")
			}

			test.WriteString(")\n\n")

			// Assertions
			test.WriteString("        # Assertions\n")
			test.WriteString("        assert result is not None\n")
			test.WriteString("        mock_request.assert_called_once()\n")
			test.WriteString("\n")

			// Generate error handling tests for 4xx/5xx responses
			generatePythonErrorTests(&test, op, data, extractedData)
		}
		test.WriteString("\n")
	}

	return test.String()
}

// generatePythonTestValueFromParam generates a test value from a parameter
func generatePythonTestValueFromParam(param common.Parameter) string {
	if param.Schema == nil {
		return "\"test_value\""
	}
	return generatePythonTestValue(param.Schema, param.Name)
}

// generatePythonAuthTest generates test_auth.py with authentication tests
func generatePythonAuthTest(data common.TemplateData, securitySchemes map[string]common.SecurityScheme, extractedData *common.ExtractedData) string {
	var test bytes.Buffer
	test.WriteString("import pytest\n")
	test.WriteString(fmt.Sprintf("from %s import %s\n\n\n", data.SDKName, data.ClientClassName))

	test.WriteString("class TestAuthentication:\n")
	test.WriteString("    \"\"\"Tests for authentication methods\"\"\"\n\n")

	// Use base URL from OpenAPI spec, fallback to default
	baseURL := extractedData.BaseURL
	if baseURL == "" {
		baseURL = "https://api.example.com/v1"
	}

	// Generate tests for each security scheme
	for name, scheme := range securitySchemes {
		schemeName := common.ToSnakeCase(name)

		switch scheme.Type {
		case "apiKey":
			test.WriteString(fmt.Sprintf("    def test_%s_api_key_auth(self):\n", schemeName))
			test.WriteString(fmt.Sprintf("        \"\"\"Test %s API key authentication.\"\"\"\n", name))
			test.WriteString(fmt.Sprintf("        client = %s(\n", data.ClientClassName))
			test.WriteString("            base_url=\"https://api.example.com\",\n")
			apiKeyValue := "test-api-key" //nolint:goconst // Test value for generated code
			// Use camelCase for attribute name (matches client code)
			attrName := common.ToCamelCase(name)
			test.WriteString(fmt.Sprintf("            %s=%q\n", attrName, apiKeyValue))
			test.WriteString("        )\n")
			test.WriteString(fmt.Sprintf("        assert client.%s == %q\n", attrName, apiKeyValue))
			test.WriteString("\n")

		case "http":
			switch scheme.Scheme {
			case "bearer":
				test.WriteString(fmt.Sprintf("    def test_%s_bearer_auth(self):\n", schemeName))
				test.WriteString(fmt.Sprintf("        \"\"\"Test %s Bearer token authentication.\"\"\"\n", name))
				test.WriteString(fmt.Sprintf("        client = %s(\n", data.ClientClassName))
				test.WriteString(fmt.Sprintf("            base_url=%q,\n", baseURL))
				test.WriteString("            bearer_token=\"test-bearer-token\"\n")
				test.WriteString("        )\n")
				test.WriteString("        assert client.bearer_token == \"test-bearer-token\"\n")
				test.WriteString("\n")
			case "basic":
				test.WriteString(fmt.Sprintf("    def test_%s_basic_auth(self):\n", schemeName))
				test.WriteString(fmt.Sprintf("        \"\"\"Test %s Basic authentication.\"\"\"\n", name))
				test.WriteString(fmt.Sprintf("        client = %s(\n", data.ClientClassName))
				test.WriteString(fmt.Sprintf("            base_url=%q,\n", baseURL))
				test.WriteString("            username=\"test-user\",\n")
				test.WriteString("            password=\"test-password\"\n")
				test.WriteString("        )\n")
				test.WriteString("        assert client.username == \"test-user\"\n")
				test.WriteString("        assert client.password == \"test-password\"\n")
				test.WriteString("\n")
			case "digest":
				test.WriteString(fmt.Sprintf("    def test_%s_digest_auth(self):\n", schemeName))
				test.WriteString(fmt.Sprintf("        \"\"\"Test %s Digest authentication.\"\"\"\n", name))
				test.WriteString(fmt.Sprintf("        client = %s(\n", data.ClientClassName))
				test.WriteString(fmt.Sprintf("            base_url=%q,\n", baseURL))
				test.WriteString("            username=\"test-user\",\n")
				test.WriteString("            password=\"test-password\"\n")
				test.WriteString("        )\n")
				test.WriteString("        assert client.username == \"test-user\"\n")
				test.WriteString("        assert client.password == \"test-password\"\n")
				test.WriteString("\n")
			}

		case "oauth2":
			test.WriteString(fmt.Sprintf("    def test_%s_oauth2_auth(self):\n", schemeName))
			test.WriteString(fmt.Sprintf("        \"\"\"Test %s OAuth2 authentication.\"\"\"\n", name))
			test.WriteString(fmt.Sprintf("        client = %s(\n", data.ClientClassName))
			test.WriteString(fmt.Sprintf("            base_url=%q,\n", baseURL))
			test.WriteString("            oauth2_token=\"test-oauth2-token\"\n")
			test.WriteString("        )\n")
			test.WriteString("        assert client.oauth2_token == \"test-oauth2-token\"\n")
			test.WriteString("\n")

		case "openIdConnect":
			test.WriteString(fmt.Sprintf("    def test_%s_openid_connect_auth(self):\n", schemeName))
			test.WriteString(fmt.Sprintf("        \"\"\"Test %s OpenID Connect authentication.\"\"\"\n", name))
			test.WriteString(fmt.Sprintf("        client = %s(\n", data.ClientClassName))
			test.WriteString(fmt.Sprintf("            base_url=%q,\n", baseURL))
			// Use camelCase for parameter name (matches client code)
			paramName := common.ToCamelCase(name) + "_token"
			test.WriteString(fmt.Sprintf("            %s=\"test-openid-token\"\n", paramName))
			test.WriteString("        )\n")
			test.WriteString(fmt.Sprintf("        assert client.%s == \"test-openid-token\"\n", paramName))
			test.WriteString("\n")

		case "mutualTLS":
			test.WriteString(fmt.Sprintf("    def test_%s_mutual_tls_auth(self):\n", schemeName))
			test.WriteString(fmt.Sprintf("        \"\"\"Test %s Mutual TLS authentication.\"\"\"\n", name))
			test.WriteString(fmt.Sprintf("        client = %s(\n", data.ClientClassName))
			test.WriteString(fmt.Sprintf("            base_url=%q,\n", baseURL))
			// Use camelCase for parameter name (matches client code)
			certParamName := common.ToCamelCase(name) + "_cert"
			keyParamName := common.ToCamelCase(name) + "_key"
			test.WriteString(fmt.Sprintf("            %s=\"test-cert.pem\",\n", certParamName))
			test.WriteString(fmt.Sprintf("            %s=\"test-key.pem\"\n", keyParamName))
			test.WriteString("        )\n")
			test.WriteString(fmt.Sprintf("        assert client.%s == \"test-cert.pem\"\n", certParamName))
			test.WriteString(fmt.Sprintf("        assert client.%s == \"test-key.pem\"\n", keyParamName))
			test.WriteString("\n")
		}
	}

	return test.String()
}

// getPythonExampleFromResponse extracts example from response for Python test
func getPythonExampleFromResponse(response common.Response) string {
	// Look for JSON content type first
	if jsonContent, ok := response.Content["application/json"]; ok {
		if len(jsonContent.Examples) > 0 {
			// Use first example
			for _, example := range jsonContent.Examples {
				return formatExampleForPython(example)
			}
		}
		// Fallback: generate example from schema if no examples
		if jsonContent.Schema != nil {
			return generatePythonExampleFromSchema(jsonContent.Schema)
		}
	}
	// Default fallback
	return "{\"success\": True}"
}

// formatExampleForPython converts an example value to Python code string
func formatExampleForPython(example interface{}) string {
	if example == nil {
		return "None"
	}

	// Use JSON encoding to properly format the example
	jsonBytes, err := json.Marshal(example)
	if err != nil {
		// Fallback to string representation
		return fmt.Sprintf("%#v", example)
	}

	// Convert JSON to Python dict/list format
	jsonStr := string(jsonBytes)
	// Replace JSON null with Python None
	jsonStr = strings.ReplaceAll(jsonStr, "null", "None")
	// Replace JSON true/false with Python True/False
	jsonStr = strings.ReplaceAll(jsonStr, "true", "True")
	jsonStr = strings.ReplaceAll(jsonStr, "false", "False")

	return jsonStr
}

// generatePythonExampleFromSchema generates a Python example from schema
func generatePythonExampleFromSchema(schema *common.Schema) string {
	if schema == nil {
		return "{}"
	}

	switch schema.Type {
	case "object":
		if len(schema.Properties) == 0 {
			return "{}"
		}
		var parts []string
		for propName, propSchema := range schema.Properties {
			value := generatePythonTestValue(propSchema, propName)
			parts = append(parts, fmt.Sprintf("%q: %s", propName, value))
		}
		return fmt.Sprintf("{%s}", strings.Join(parts, ", "))
	case "array":
		if schema.Items != nil {
			itemValue := generatePythonTestValue(schema.Items, "item")
			return fmt.Sprintf("[%s]", itemValue)
		}
		return "[]"
	default:
		return generatePythonTestValue(schema, "value")
	}
}

// generatePythonErrorTests generates error handling tests for 4xx/5xx responses
func generatePythonErrorTests(test *bytes.Buffer, op common.APIOperation, data common.TemplateData, extractedData *common.ExtractedData) {
	// Find error responses (4xx, 5xx)
	errorStatuses := []string{}
	for statusCode := range op.Responses {
		if len(statusCode) >= 3 {
			firstDigit := statusCode[0]
			if firstDigit == '4' || firstDigit == '5' {
				errorStatuses = append(errorStatuses, statusCode)
			}
		}
	}

	if len(errorStatuses) == 0 {
		return // No error responses to test
	}

	methodName := common.GetOperationMethodName(op)

	// Generate test for each error status
	for _, statusCode := range errorStatuses {
		errorTestName := fmt.Sprintf("test_%s_%s_error", methodName, statusCode)
		// Patch requests.Session.request since client uses session.request()
		test.WriteString("    @patch('requests.Session.request')\n")
		fmt.Fprintf(test, "    def %s(self, mock_request):\n", errorTestName)
		fmt.Fprintf(test, "        \"\"\"Test %s %s operation returns %s error.\"\"\"\n", op.Method, op.Path, statusCode)

		// Setup mock error response
		test.WriteString("        # Setup mock error response\n")
		test.WriteString("        mock_response = Mock()\n")
		fmt.Fprintf(test, "        mock_response.status_code = %s\n", statusCode)

		// Get error example if available
		errorResponse := op.Responses[statusCode]
		errorExample := getPythonExampleFromResponse(errorResponse)
		fmt.Fprintf(test, "        mock_response.json.return_value = %s\n", errorExample)
		jsonText := strings.ReplaceAll(errorExample, "True", "true")
		jsonText = strings.ReplaceAll(jsonText, "False", "false")
		jsonText = strings.ReplaceAll(jsonText, "None", "null")
		fmt.Fprintf(test, "        mock_response.text = %q\n", jsonText)
		test.WriteString("        mock_request.return_value = mock_response\n\n")

		// Create client
		baseURL := extractedData.BaseURL
		if baseURL == "" {
			baseURL = "https://api.example.com/v1"
		}
		fmt.Fprintf(test, "        client = %s(base_url=%q)\n\n", data.ClientClassName, baseURL)

		// Call API method and expect error
		test.WriteString("        # Call API method and expect error\n")
		fmt.Fprintf(test, "        # result = client.%s(...)\n", methodName)
		test.WriteString("        # assert result is None or raises exception\n")
		test.WriteString("        # mock_request.assert_called_once()\n")
		test.WriteString("\n")
	}
}
