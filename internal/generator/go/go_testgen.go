// Package gogen provides Go test generation functionality.
package gogen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vubon/sdk-forge/internal/generator/common"
)

// generateGoTests generates test files for Go SDK
func generateGoTests(packageDir string, data common.TemplateData, extractedData *common.ExtractedData, version common.LanguageVersion) error {
	// Generate client_test.go
	clientTestContent := generateGoClientTest(data, extractedData, version)
	clientTestPath := filepath.Join(packageDir, "client_test.go")
	// #nosec G306 -- 0644 is appropriate for Go test files
	if err := os.WriteFile(clientTestPath, []byte(clientTestContent), 0644); err != nil {
		return fmt.Errorf("failed to write client_test.go: %w", err)
	}
	// Format with gofmt
	if err := formatGoFile(clientTestPath); err != nil {
		_ = err
	}

	// Generate models_test.go if schemas exist
	if len(extractedData.Schemas) > 0 {
		modelsTestContent := generateGoModelsTest(data, extractedData.Schemas, version)
		modelsTestPath := filepath.Join(packageDir, "models_test.go")
		// #nosec G306 -- 0644 is appropriate for Go test files
		if err := os.WriteFile(modelsTestPath, []byte(modelsTestContent), 0644); err != nil {
			return fmt.Errorf("failed to write models_test.go: %w", err)
		}
		// Format with gofmt
		if err := formatGoFile(modelsTestPath); err != nil {
			_ = err
		}
	}

	// Generate api_test.go if operations exist
	if len(extractedData.Operations) > 0 {
		apiTestContent := generateGoAPITest(data, extractedData.Operations, extractedData, version)
		apiTestPath := filepath.Join(packageDir, "api_test.go")
		// #nosec G306 -- 0644 is appropriate for Go test files
		if err := os.WriteFile(apiTestPath, []byte(apiTestContent), 0644); err != nil {
			return fmt.Errorf("failed to write api_test.go: %w", err)
		}
		// Format with gofmt
		if err := formatGoFile(apiTestPath); err != nil {
			_ = err
		}
	}

	// Generate auth_test.go if security schemes exist
	if len(extractedData.SecuritySchemes) > 0 {
		authTestContent := generateGoAuthTest(data, extractedData.SecuritySchemes, extractedData, version)
		authTestPath := filepath.Join(packageDir, "auth_test.go")
		// #nosec G306 -- 0644 is appropriate for Go test files
		if err := os.WriteFile(authTestPath, []byte(authTestContent), 0644); err != nil {
			return fmt.Errorf("failed to write auth_test.go: %w", err)
		}
		// Format with gofmt
		if err := formatGoFile(authTestPath); err != nil {
			_ = err
		}
	}

	// Generate test fixtures from OpenAPI examples
	if err := generateGoTestFixtures(packageDir, extractedData); err != nil {
		return fmt.Errorf("failed to generate test fixtures: %w", err)
	}

	return nil
}

// generateGoTestFixtures generates test fixture files from OpenAPI examples
func generateGoTestFixtures(packageDir string, extractedData *common.ExtractedData) error {
	fixturesDir := filepath.Join(packageDir, "testdata")
	if err := os.MkdirAll(fixturesDir, 0750); err != nil {
		return fmt.Errorf("failed to create testdata directory: %w", err)
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
		// Generate fixtures.go file
		fixturesContent := generateGoFixturesFile(fixtures)
		fixturesPath := filepath.Join(fixturesDir, "fixtures.go")
		// #nosec G306 -- 0644 is appropriate for Go fixture files
		if err := os.WriteFile(fixturesPath, []byte(fixturesContent), 0644); err != nil {
			return fmt.Errorf("failed to write testdata/fixtures.go: %w", err)
		}
		// Format with gofmt
		if err := formatGoFile(fixturesPath); err != nil {
			_ = err
		}
	}

	return nil
}

// generateGoFixturesFile generates a Go file with test fixtures
func generateGoFixturesFile(fixtures map[string]interface{}) string {
	var buf bytes.Buffer
	buf.WriteString("package testdata\n\n")
	buf.WriteString("// Test fixtures extracted from OpenAPI examples\n\n")

	for key, example := range fixtures {
		// Convert key to valid Go variable name
		varName := common.ToPascalCase(key)
		exampleJSON := formatExampleForGo(example)
		fmt.Fprintf(&buf, "var %s = %s\n\n", varName, exampleJSON)
	}

	return buf.String()
}

// generateGoClientTest generates client_test.go
func generateGoClientTest(data common.TemplateData, extractedData *common.ExtractedData, version common.LanguageVersion) string {
	var test bytes.Buffer
	test.WriteString(fmt.Sprintf("package %s\n\n", data.SDKName))
	test.WriteString("import (\n")
	test.WriteString("\t\"testing\"\n")
	test.WriteString(")\n\n")

	// Use base URL from OpenAPI spec, fallback to default
	baseURL := extractedData.BaseURL
	if baseURL == "" {
		baseURL = "https://api.example.com/v1"
	}

	test.WriteString(fmt.Sprintf("func TestNew%s(t *testing.T) {\n", data.ClientClassName))
	test.WriteString(fmt.Sprintf("\tclient := New%s(%q)\n", data.ClientClassName, baseURL))
	test.WriteString("\tif client == nil {\n")
	test.WriteString(fmt.Sprintf("\t\tt.Fatal(\"New%s() returned nil\")\n", data.ClientClassName))
	test.WriteString("\t}\n")
	test.WriteString(fmt.Sprintf("\tif client.BaseURL != %q {\n", baseURL))
	test.WriteString(fmt.Sprintf("\t\tt.Errorf(\"BaseURL = %%q, want %%q\", client.BaseURL, %q)\n", baseURL))
	test.WriteString("\t}\n")
	test.WriteString("}\n")
	return test.String()
}

// generateGoModelsTest generates models_test.go with schema-based tests
func generateGoModelsTest(data common.TemplateData, schemas map[string]*common.Schema, version common.LanguageVersion) string {
	var test bytes.Buffer
	test.WriteString(fmt.Sprintf("package %s\n\n", data.SDKName))
	test.WriteString("import (\n")
	test.WriteString("\t\"encoding/json\"\n")
	test.WriteString("\t\"testing\"\n")
	test.WriteString(")\n\n")

	// Generate tests for each schema
	for name, schema := range schemas {
		structName := common.ToPascalCase(name)

		// Test struct creation
		test.WriteString(fmt.Sprintf("func Test%s_Creation(t *testing.T) {\n", structName))
		test.WriteString(fmt.Sprintf("\t// Test %s can be instantiated\n", structName))

		if schema.Type == "object" && len(schema.Properties) > 0 {
			// Track required fields
			requiredSet := make(map[string]bool)
			for _, req := range schema.Required {
				requiredSet[req] = true
			}

			// Generate struct literal with test values
			test.WriteString(fmt.Sprintf("\tmodel := %s{\n", structName))

			for propName, propSchema := range schema.Properties {
				if propSchema == nil {
					continue
				}

				fieldName := common.ToPascalCase(propName)
				isRequired := requiredSet[propName]
				isPointer := !isRequired // Optional fields are pointers

				testValue := generateGoTestValue(propSchema, propName, version, isPointer)
				test.WriteString(fmt.Sprintf("\t\t%s: %s,\n", fieldName, testValue))
			}
			test.WriteString("\t}\n")
			test.WriteString("\t// Verify model can be instantiated\n")
			test.WriteString("\t_ = model\n")
		} else {
			// Simple model without properties
			test.WriteString(fmt.Sprintf("\tmodel := %s{}\n", structName))
			test.WriteString("\t// Verify model can be instantiated\n")
			test.WriteString("\t_ = model\n")
		}
		test.WriteString("}\n\n")

		// Test JSON serialization/deserialization
		if schema.Type == "object" && len(schema.Properties) > 0 {
			test.WriteString(fmt.Sprintf("func Test%s_Serialization(t *testing.T) {\n", structName))
			test.WriteString(fmt.Sprintf("\t// Test %s JSON serialization\n", structName))

			requiredSet := make(map[string]bool)
			for _, req := range schema.Required {
				requiredSet[req] = true
			}

			test.WriteString(fmt.Sprintf("\tmodel := %s{\n", structName))
			requiredSet2 := make(map[string]bool)
			for _, req := range schema.Required {
				requiredSet2[req] = true
			}
			for propName, propSchema := range schema.Properties {
				if propSchema == nil {
					continue
				}
				fieldName := common.ToPascalCase(propName)
				isRequired := requiredSet2[propName]
				isPointer := !isRequired
				testValue := generateGoTestValue(propSchema, propName, version, isPointer)
				test.WriteString(fmt.Sprintf("\t\t%s: %s,\n", fieldName, testValue))
			}
			test.WriteString("\t}\n\n")

			test.WriteString("\t// Marshal to JSON\n")
			test.WriteString("\tdata, err := json.Marshal(model)\n")
			test.WriteString("\tif err != nil {\n")
			test.WriteString("\t\tt.Fatalf(\"Marshal() error = %v\", err)\n")
			test.WriteString("\t}\n")
			test.WriteString("\tif len(data) == 0 {\n")
			test.WriteString("\t\tt.Error(\"Marshal() should return non-empty data\")\n")
			test.WriteString("\t}\n\n")

			test.WriteString("\t// Unmarshal from JSON\n")
			test.WriteString(fmt.Sprintf("\tvar unmarshaled %s\n", structName))
			test.WriteString("\tif err := json.Unmarshal(data, &unmarshaled); err != nil {\n")
			test.WriteString("\t\tt.Fatalf(\"Unmarshal() error = %v\", err)\n")
			test.WriteString("\t}\n")
			test.WriteString("}\n\n")
		}
	}

	return test.String()
}

// generateGoTestValue generates a test value for a schema property in Go
// isPointer indicates if the field is a pointer type (for optional fields)
func generateGoTestValue(schema *common.Schema, propName string, version common.LanguageVersion, isPointer bool) string {
	if schema == nil {
		if isPointer {
			return "nil"
		}
		return "\"test_value\""
	}

	emptyInterface := version.GetGoEmptyInterface()
	var value string

	switch schema.Type {
	case "string":
		switch schema.Format {
		case "date":
			value = "\"2024-01-01\""
		case "date-time":
			value = "\"2024-01-01T00:00:00Z\""
		case "email":
			value = "\"test@example.com\""
		default:
			value = fmt.Sprintf("%q", "test_"+common.ToSnakeCase(propName))
		}
	case "integer", "number":
		value = "42"
	case "boolean":
		value = "true"
	case "array":
		if schema.Items != nil {
			itemType := getGoType(schema.Items, version)
			// For arrays, generate a simple item value
			itemValue := generateGoTestValue(schema.Items, "item", version, false)
			arrayValue := fmt.Sprintf("[]%s{%s}", itemType, itemValue)
			if isPointer {
				// For pointer to array, we need to create the array first
				value = fmt.Sprintf("&%s", arrayValue)
			} else {
				value = arrayValue
			}
		} else {
			value = "nil"
		}
	case "object":
		// Use map[string]interface{} instead of any{}
		mapValue := fmt.Sprintf("map[string]%s{}", emptyInterface)
		if isPointer {
			value = fmt.Sprintf("&%s", mapValue)
		} else {
			value = mapValue
		}
	default:
		value = "\"test_value\""
	}

	// If field is a pointer, wrap with address-of operator
	if isPointer {
		// For pointer types, we need to take address of the value
		if value == "nil" {
			return "nil"
		}
		// Arrays and maps are already handled above with pointer logic
		// For primitives (string, int, bool), we need to create a variable first
		// In Go, we can't use &"string" directly, we need to use a helper variable
		// For struct literals, we can use: &value where value is a variable
		// But for string literals, we need: strPtr := "value"; &strPtr
		// Actually, in struct literals we can use: &[]string{...} or &map[string]interface{}{}
		// For string/int/bool, we need to create a temporary variable
		// However, Go allows: &"string" in some contexts but not in struct literals
		// The safest approach is to use a helper function or create variables
		// For now, let's use a pattern that works: create a variable name
		// Actually, we can use: &[]type{value} for arrays, &map[...]{...} for maps
		// For strings/ints/bools, we need to avoid &"literal" - use a temp variable pattern
		// But that's complex. Let's check if the value already has & prefix
		if strings.HasPrefix(value, "&") {
			return value // Already has address-of
		}
		// For string literals, we can't use &"string" in struct literals
		// We need to use a workaround: create a helper variable
		// But for now, let's use nil for optional string fields in tests
		// Or we can use: stringPtr := "value"; &stringPtr
		// Actually, the simplest is to use nil for optional fields in tests
		// Or create a helper: func stringPtr(s string) *string { return &s }
		// For now, let's just use nil for optional primitive fields
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			// It's a string literal - can't use &"string" in struct literal
			// Use nil instead for optional string fields
			return "nil"
		}
		// For other types (int, bool), same issue - use nil
		if value == "42" || value == "true" || value == "false" {
			return "nil"
		}
		// For arrays and maps, we already handled them above
		return fmt.Sprintf("&%s", value)
	}

	return value
}

// generateGoAPITest generates api_test.go with operation-based tests
func generateGoAPITest(data common.TemplateData, operations []common.APIOperation, extractedData *common.ExtractedData, version common.LanguageVersion) string {
	var test bytes.Buffer
	test.WriteString(fmt.Sprintf("package %s\n\n", data.SDKName))
	test.WriteString("import (\n")
	test.WriteString("\t\"net/http\"\n")
	test.WriteString("\t\"net/http/httptest\"\n")
	test.WriteString("\t\"testing\"\n")
	test.WriteString(")\n\n")

	// Group operations by tag for better organization
	operationsByTag := common.GroupOperationsByTag(operations)

	// Generate tests for each tag/group
	for tag, tagOperations := range operationsByTag {
		test.WriteString(fmt.Sprintf("// Test%sAPI tests for %s API methods\n", common.ToPascalCase(tag), tag))
		test.WriteString(fmt.Sprintf("func Test%sAPI(t *testing.T) {\n", common.ToPascalCase(tag)))
		test.WriteString("\tt.Skip(\"TODO: Implement tests for this API group\")\n")
		test.WriteString("}\n\n")

		// Generate test for each operation
		for _, op := range tagOperations {
			methodName := common.GetOperationMethodName(op)
			testMethodName := fmt.Sprintf("Test%s_%s", common.ToPascalCase(tag), common.ToPascalCase(methodName))

			test.WriteString(fmt.Sprintf("func %s(t *testing.T) {\n", testMethodName))
			test.WriteString(fmt.Sprintf("\t// Test %s %s operation\n", op.Method, op.Path))
			test.WriteString("\tserver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {\n")

			// Determine expected status code from responses
			expectedStatus := 200 // http.StatusOK
			if _, ok := op.Responses["200"]; !ok {
				// Find first available status code
				for statusCode := range op.Responses {
					switch statusCode {
					case "201":
						expectedStatus = 201 // http.StatusCreated
					case "204":
						expectedStatus = 204 // http.StatusNoContent
					}
					break
				}
			}

			test.WriteString(fmt.Sprintf("\t\tw.WriteHeader(%d)\n", expectedStatus))

			// Use example from OpenAPI response if available
			statusCodeStr := fmt.Sprintf("%d", expectedStatus)
			exampleJSON := getGoExampleFromResponse(op.Responses[statusCodeStr])
			test.WriteString(fmt.Sprintf("\t\tw.Write([]byte(%s))\n", exampleJSON))
			test.WriteString("\t}))\n")
			test.WriteString("\tdefer server.Close()\n\n")

			test.WriteString(fmt.Sprintf("\tclient := New%s(server.URL)\n", data.ClientClassName))
			test.WriteString("\tif client == nil {\n")
			test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
			test.WriteString("\t}\n\n")

			// Generate method call with parameters
			test.WriteString("\t// Call API method\n")
			test.WriteString("\t// TODO: Uncomment and implement based on your API method signature\n")
			test.WriteString(fmt.Sprintf("\t// result, err := client.%s(", common.ToPascalCase(methodName)))

			// Add path parameters
			hasParams := false
			for _, param := range op.Parameters {
				if param.In == "path" {
					if hasParams {
						test.WriteString(", ")
					}
					paramName := common.ToPascalCase(param.Name)
					testValue := generateGoTestValueFromParam(param, version)
					test.WriteString(fmt.Sprintf("%s: %s", paramName, testValue))
					hasParams = true
				}
			}

			// Add query parameters
			for _, param := range op.Parameters {
				if param.In == "query" {
					if hasParams {
						test.WriteString(", ")
					}
					paramName := common.ToPascalCase(param.Name)
					testValue := generateGoTestValueFromParam(param, version)
					test.WriteString(fmt.Sprintf("%s: %s", paramName, testValue))
					hasParams = true
				}
			}

			test.WriteString(")\n")
			test.WriteString("\t// if err != nil {\n")
			test.WriteString("\t//     t.Fatalf(\"Method call error = %v\", err)\n")
			test.WriteString("\t// }\n")
			test.WriteString("\t// if result == nil {\n")
			test.WriteString("\t//     t.Error(\"Result should not be nil\")\n")
			test.WriteString("\t// }\n")
			test.WriteString("}\n\n")

			// Generate error handling tests for 4xx/5xx responses
			generateGoErrorTests(&test, op, data, extractedData)
		}
	}

	return test.String()
}

// generateGoErrorTests generates error handling tests for 4xx/5xx responses
func generateGoErrorTests(test *bytes.Buffer, op common.APIOperation, data common.TemplateData, extractedData *common.ExtractedData) {
	// Find error responses (4xx, 5xx)
	var errorStatuses []string
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
		statusCodeInt := 0
		if _, err := fmt.Sscanf(statusCode, "%d", &statusCodeInt); err != nil || statusCodeInt == 0 {
			continue
		}

		testMethodName := fmt.Sprintf("Test%s_%s_%sError", common.ToPascalCase(op.Tags[0]), common.ToPascalCase(methodName), statusCode)
		fmt.Fprintf(test, "func %s(t *testing.T) {\n", testMethodName)
		fmt.Fprintf(test, "\t// Test %s %s operation returns %s error\n", op.Method, op.Path, statusCode)
		fmt.Fprintf(test, "\tserver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {\n")
		fmt.Fprintf(test, "\t\tw.WriteHeader(%d)\n", statusCodeInt)

		// Get error example if available
		errorResponse := op.Responses[statusCode]
		exampleJSON := getGoExampleFromResponse(errorResponse)
		fmt.Fprintf(test, "\t\tw.Write([]byte(%s))\n", exampleJSON)
		test.WriteString("\t}))\n")
		test.WriteString("\tdefer server.Close()\n\n")

		fmt.Fprintf(test, "\tclient := New%s(server.URL)\n", data.ClientClassName)
		test.WriteString("\tif client == nil {\n")
		test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
		test.WriteString("\t}\n\n")

		test.WriteString("\t// Call API method and expect error\n")
		fmt.Fprintf(test, "\t// result, err := client.%s(...)\n", common.ToPascalCase(methodName))
		test.WriteString("\t// if err == nil {\n")
		test.WriteString("\t//     t.Error(\"Expected error but got nil\")\n")
		test.WriteString("\t// }\n")
		test.WriteString("}\n\n")
	}
}

// generateGoTestValueFromParam generates a test value from a parameter in Go
func generateGoTestValueFromParam(param common.Parameter, version common.LanguageVersion) string {
	if param.Schema == nil {
		return "\"test_value\""
	}
	// Parameters are typically not pointers (they're function parameters)
	return generateGoTestValue(param.Schema, param.Name, version, false)
}

// generateGoAuthTest generates auth_test.go with authentication tests
func generateGoAuthTest(data common.TemplateData, securitySchemes map[string]common.SecurityScheme, extractedData *common.ExtractedData, version common.LanguageVersion) string {
	var test bytes.Buffer
	test.WriteString(fmt.Sprintf("package %s\n\n", data.SDKName))
	test.WriteString("import (\n")
	test.WriteString("\t\"testing\"\n")
	test.WriteString(")\n\n")

	// Use base URL from OpenAPI spec, fallback to default
	baseURL := extractedData.BaseURL
	if baseURL == "" {
		baseURL = "https://api.example.com/v1"
	}

	test.WriteString("// TestAuthentication tests for authentication methods\n")
	test.WriteString("func TestAuthentication(t *testing.T) {\n")
	test.WriteString("\tt.Skip(\"TODO: Implement authentication tests\")\n")
	test.WriteString("}\n\n")

	// Generate tests for each security scheme
	for name, scheme := range securitySchemes {
		schemeName := common.ToPascalCase(name)

		switch scheme.Type {
		case "apiKey":
			test.WriteString(fmt.Sprintf("func Test%s_APIKeyAuth(t *testing.T) {\n", schemeName))
			test.WriteString(fmt.Sprintf("\t// Test %s API key authentication\n", name))
			test.WriteString(fmt.Sprintf("\tclient := New%s(%q)\n", data.ClientClassName, baseURL))
			test.WriteString("\tif client == nil {\n")
			test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
			test.WriteString("\t}\n")
			// Use the security scheme name (not the header name) to match client field
			apiKeyField := common.ToPascalCase(name)
			test.WriteString(fmt.Sprintf("\tclient.%s = \"test-api-key\"\n", apiKeyField))
			test.WriteString(fmt.Sprintf("\tif client.%s != \"test-api-key\" {\n", apiKeyField))
			test.WriteString("\t\tt.Error(\"API key should be set\")\n")
			test.WriteString("\t}\n")
			test.WriteString("}\n\n")

		case "http":
			switch scheme.Scheme {
			case "bearer", "Bearer":
				test.WriteString(fmt.Sprintf("func Test%s_BearerAuth(t *testing.T) {\n", schemeName))
				test.WriteString(fmt.Sprintf("\t// Test %s Bearer token authentication\n", name))
				test.WriteString(fmt.Sprintf("\tclient := New%s(%q)\n", data.ClientClassName, baseURL))
				test.WriteString("\tif client == nil {\n")
				test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
				test.WriteString("\t}\n")
				test.WriteString("\tclient.BearerToken = \"test-bearer-token\"\n")
				test.WriteString("\tif client.BearerToken != \"test-bearer-token\" {\n")
				test.WriteString("\t\tt.Error(\"Bearer token should be set\")\n")
				test.WriteString("\t}\n")
				test.WriteString("}\n\n")
			case "basic":
				test.WriteString(fmt.Sprintf("func Test%s_BasicAuth(t *testing.T) {\n", schemeName))
				test.WriteString(fmt.Sprintf("\t// Test %s Basic authentication\n", name))
				test.WriteString(fmt.Sprintf("\tclient := New%s(%q)\n", data.ClientClassName, baseURL))
				test.WriteString("\tif client == nil {\n")
				test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
				test.WriteString("\t}\n")
				test.WriteString("\tclient.Username = \"test-user\"\n")
				test.WriteString("\tclient.Password = \"test-password\"\n")
				test.WriteString("\tif client.Username != \"test-user\" {\n")
				test.WriteString("\t\tt.Error(\"Username should be set\")\n")
				test.WriteString("\t}\n")
				test.WriteString("\tif client.Password != \"test-password\" {\n")
				test.WriteString("\t\tt.Error(\"Password should be set\")\n")
				test.WriteString("\t}\n")
				test.WriteString("}\n\n")
			case "digest":
				test.WriteString(fmt.Sprintf("func Test%s_DigestAuth(t *testing.T) {\n", schemeName))
				test.WriteString(fmt.Sprintf("\t// Test %s Digest authentication\n", name))
				test.WriteString(fmt.Sprintf("\tclient := New%s(%q)\n", data.ClientClassName, baseURL))
				test.WriteString("\tif client == nil {\n")
				test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
				test.WriteString("\t}\n")
				test.WriteString("\tclient.Username = \"test-user\"\n")
				test.WriteString("\tclient.Password = \"test-password\"\n")
				test.WriteString("\tif client.Username != \"test-user\" {\n")
				test.WriteString("\t\tt.Error(\"Username should be set\")\n")
				test.WriteString("\t}\n")
				test.WriteString("\tif client.Password != \"test-password\" {\n")
				test.WriteString("\t\tt.Error(\"Password should be set\")\n")
				test.WriteString("\t}\n")
				test.WriteString("}\n\n")
			}

		case "oauth2":
			test.WriteString(fmt.Sprintf("func Test%s_OAuth2Auth(t *testing.T) {\n", schemeName))
			test.WriteString(fmt.Sprintf("\t// Test %s OAuth2 authentication\n", name))
			test.WriteString(fmt.Sprintf("\tclient := New%s(%q)\n", data.ClientClassName, baseURL))
			test.WriteString("\tif client == nil {\n")
			test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
			test.WriteString("\t}\n")
			test.WriteString("\tclient.Oauth2Token = \"test-oauth2-token\"\n")
			test.WriteString("\tif client.Oauth2Token != \"test-oauth2-token\" {\n")
			test.WriteString("\t\tt.Error(\"OAuth2 token should be set\")\n")
			test.WriteString("\t}\n")
			test.WriteString("}\n\n")

		case "openIdConnect":
			test.WriteString(fmt.Sprintf("func Test%s_OpenIDConnectAuth(t *testing.T) {\n", schemeName))
			test.WriteString(fmt.Sprintf("\t// Test %s OpenID Connect authentication\n", name))
			test.WriteString(fmt.Sprintf("\tclient := New%s(%q)\n", data.ClientClassName, baseURL))
			test.WriteString("\tif client == nil {\n")
			test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
			test.WriteString("\t}\n")
			test.WriteString("\tclient.OpenIdConnectToken = \"test-openid-token\"\n")
			test.WriteString("\tif client.OpenIdConnectToken != \"test-openid-token\" {\n")
			test.WriteString("\t\tt.Error(\"OpenID Connect token should be set\")\n")
			test.WriteString("\t}\n")
			test.WriteString("}\n\n")

		case "mutualTLS":
			test.WriteString(fmt.Sprintf("func Test%s_MutualTLSAuth(t *testing.T) {\n", schemeName))
			test.WriteString(fmt.Sprintf("\t// Test %s Mutual TLS authentication\n", name))
			test.WriteString(fmt.Sprintf("\tclient := New%s(%q)\n", data.ClientClassName, baseURL))
			test.WriteString("\tif client == nil {\n")
			test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
			test.WriteString("\t}\n")
			test.WriteString("\t// Note: Mutual TLS requires certificate configuration\n")
			test.WriteString("\t// This test verifies the client can be instantiated\n")
			test.WriteString("\t_ = client\n")
			test.WriteString("}\n\n")
		}
	}

	return test.String()
}

// getGoExampleFromResponse extracts example from response for Go test
func getGoExampleFromResponse(response common.Response) string {
	// Look for JSON content type first
	if jsonContent, ok := response.Content["application/json"]; ok {
		if len(jsonContent.Examples) > 0 {
			// Use first example
			for _, example := range jsonContent.Examples {
				return formatExampleForGo(example)
			}
		}
		// Fallback: generate example from schema if no examples
		if jsonContent.Schema != nil {
			return generateGoExampleFromSchema(jsonContent.Schema)
		}
	}
	// Default fallback
	return "`{\"success\": true}`"
}

// formatExampleForGo converts an example value to Go code string
func formatExampleForGo(example interface{}) string {
	if example == nil {
		return "`null`"
	}

	// Use JSON encoding to properly format the example
	jsonBytes, err := json.Marshal(example)
	if err != nil {
		// Fallback: escape and quote
		return fmt.Sprintf("%q", fmt.Sprintf("%v", example))
	}

	// Return as Go raw string literal
	return fmt.Sprintf("`%s`", string(jsonBytes))
}

// generateGoExampleFromSchema generates a Go example from schema
func generateGoExampleFromSchema(schema *common.Schema) string {
	if schema == nil {
		return "`{}`"
	}

	// For now, return a simple example based on type
	// This can be enhanced to generate more complex examples
	switch schema.Type {
	case "object":
		return "`{}`"
	case "array":
		return "`[]`"
	default:
		return "`{\"value\": \"test\"}`"
	}
}
