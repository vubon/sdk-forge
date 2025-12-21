// Package generator provides PHP test generation functionality.
package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// generatePHPTests generates PHPUnit tests for PHP SDK
func generatePHPTests(packageDir, srcDir string, data TemplateData, extractedData *ExtractedData) error {
	// Create tests/ directory
	testsDir := filepath.Join(packageDir, "tests")
	if err := os.MkdirAll(testsDir, 0750); err != nil {
		return fmt.Errorf("failed to create tests directory: %w", err)
	}

	// Generate tests/bootstrap.php (PHPUnit bootstrap)
	bootstrapContent := generatePHPBootstrap(data)
	bootstrapPath := filepath.Join(testsDir, "bootstrap.php")
	// #nosec G306 -- 0644 is appropriate for PHP bootstrap files
	if err := os.WriteFile(bootstrapPath, []byte(bootstrapContent), 0644); err != nil {
		return fmt.Errorf("failed to write tests/bootstrap.php: %w", err)
	}

	// Generate tests/phpunit.xml (PHPUnit configuration)
	phpunitContent := generatePHPPHPUnitConfig(data)
	phpunitPath := filepath.Join(testsDir, "phpunit.xml")
	// #nosec G306 -- 0644 is appropriate for PHPUnit config files
	if err := os.WriteFile(phpunitPath, []byte(phpunitContent), 0644); err != nil {
		return fmt.Errorf("failed to write tests/phpunit.xml: %w", err)
	}

	// Generate tests/ClientTest.php
	clientTestContent := generatePHPClientTest(data, extractedData)
	clientTestPath := filepath.Join(testsDir, "ClientTest.php")
	// #nosec G306 -- 0644 is appropriate for PHP test files
	if err := os.WriteFile(clientTestPath, []byte(clientTestContent), 0644); err != nil {
		return fmt.Errorf("failed to write tests/ClientTest.php: %w", err)
	}
	// Format with PHP-CS-Fixer (if available)
	if err := formatPHPFile(clientTestPath); err != nil {
		_ = err
	}

	// Generate tests/ModelsTest.php if schemas exist
	if len(extractedData.Schemas) > 0 {
		modelsTestContent := generatePHPModelsTest(data, extractedData.Schemas)
		modelsTestPath := filepath.Join(testsDir, "ModelsTest.php")
		// #nosec G306 -- 0644 is appropriate for PHP test files
		if err := os.WriteFile(modelsTestPath, []byte(modelsTestContent), 0644); err != nil {
			return fmt.Errorf("failed to write tests/ModelsTest.php: %w", err)
		}
		// Format with PHP-CS-Fixer (if available)
		if err := formatPHPFile(modelsTestPath); err != nil {
			_ = err
		}
	}

	// Generate tests/ApiMethodsTest.php if operations exist
	if len(extractedData.Operations) > 0 {
		apiTestContent := generatePHPApiMethodsTest(data, extractedData.Operations, extractedData)
		apiTestPath := filepath.Join(testsDir, "ApiMethodsTest.php")
		// #nosec G306 -- 0644 is appropriate for PHP test files
		if err := os.WriteFile(apiTestPath, []byte(apiTestContent), 0644); err != nil {
			return fmt.Errorf("failed to write tests/ApiMethodsTest.php: %w", err)
		}
		// Format with PHP-CS-Fixer (if available)
		if err := formatPHPFile(apiTestPath); err != nil {
			_ = err
		}
	}

	// Generate tests/AuthTest.php if security schemes exist
	if len(extractedData.SecuritySchemes) > 0 {
		authTestContent := generatePHPAuthTest(data, extractedData.SecuritySchemes, extractedData)
		authTestPath := filepath.Join(testsDir, "AuthTest.php")
		// #nosec G306 -- 0644 is appropriate for PHP test files
		if err := os.WriteFile(authTestPath, []byte(authTestContent), 0644); err != nil {
			return fmt.Errorf("failed to write tests/AuthTest.php: %w", err)
		}
		// Format with PHP-CS-Fixer (if available)
		if err := formatPHPFile(authTestPath); err != nil {
			_ = err
		}
	}

	return nil
}

// generatePHPBootstrap generates PHPUnit bootstrap file
func generatePHPBootstrap(data TemplateData) string {
	return `<?php

require_once __DIR__ . '/../vendor/autoload.php';

// Bootstrap for PHPUnit tests
// Auto-generated from OpenAPI schema
`
}

// generatePHPPHPUnitConfig generates PHPUnit configuration file
func generatePHPPHPUnitConfig(data TemplateData) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<phpunit xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:noNamespaceSchemaLocation="https://schema.phpunit.de/10.0/phpunit.xsd"
         bootstrap="bootstrap.php"
         colors="true"
         cacheDirectory=".phpunit.cache"
         executionOrder="depends,defects"
         forceCoversAnnotation="false"
         beStrictAboutCoversAnnotation="true"
         beStrictAboutOutputDuringTests="true"
         beStrictAboutTodoAnnotatedTests="true"
         failOnRisky="true"
         failOnWarning="false"
         verbose="true">
    <testsuites>
        <testsuite name="%s SDK Test Suite">
            <directory>.</directory>
        </testsuite>
    </testsuites>
    <source>
        <include>
            <directory suffix=".php">../src</directory>
        </include>
    </source>
</phpunit>
`, data.SDKName)
}

// generatePHPClientTest generates ClientTest.php
func generatePHPClientTest(data TemplateData, extractedData *ExtractedData) string {
	namespace := getPHPNamespace(data.SDKName)
	clientClassName := getClientClassName(data.SDKName)

	baseURL := extractedData.BaseURL
	if baseURL == "" {
		baseURL = "https://api.example.com/v1"
	}

	var test strings.Builder
	test.WriteString("<?php\n\n")
	test.WriteString(fmt.Sprintf("namespace %s\\Tests;\n\n", namespace))
	test.WriteString("use PHPUnit\\Framework\\TestCase;\n")
	test.WriteString(fmt.Sprintf("use %s\\%s;\n\n", namespace, clientClassName))
	test.WriteString("/**\n")
	test.WriteString(" * Tests for API client\n")
	test.WriteString(" * Auto-generated from OpenAPI schema\n")
	test.WriteString(" */\n")
	test.WriteString("class ClientTest extends TestCase\n")
	test.WriteString("{\n")
	test.WriteString("    /**\n")
	test.WriteString("     * Test client can be instantiated\n")
	test.WriteString("     */\n")
	test.WriteString("    public function testClientInitialization(): void\n")
	test.WriteString("    {\n")
	test.WriteString(fmt.Sprintf("        $client = new %s(baseUrl: \"%s\");\n", clientClassName, baseURL))
	test.WriteString(fmt.Sprintf("        $this->assertInstanceOf(%s::class, $client);\n", clientClassName))
	test.WriteString("    }\n")
	test.WriteString("}\n")

	return test.String()
}

// generatePHPModelsTest generates ModelsTest.php
func generatePHPModelsTest(data TemplateData, schemas map[string]*Schema) string {
	namespace := getPHPNamespace(data.SDKName)

	var test strings.Builder
	test.WriteString("<?php\n\n")
	test.WriteString(fmt.Sprintf("namespace %s\\Tests;\n\n", namespace))
	test.WriteString("use PHPUnit\\Framework\\TestCase;\n")
	test.WriteString(fmt.Sprintf("use %s\\Models\\*;\n\n", namespace))
	test.WriteString("/**\n")
	test.WriteString(" * Tests for data models\n")
	test.WriteString(" * Auto-generated from OpenAPI schema\n")
	test.WriteString(" */\n")
	test.WriteString("class ModelsTest extends TestCase\n")
	test.WriteString("{\n")

	// Generate tests for each schema
	for name, schema := range schemas {
		className := toPascalCase(name)
		test.WriteString("    /**\n")
		test.WriteString(fmt.Sprintf("     * Test %s model creation\n", className))
		test.WriteString("     */\n")
		test.WriteString(fmt.Sprintf("    public function test%sCreation(): void\n", className))
		test.WriteString("    {\n")

		if schema.Type == "object" && len(schema.Properties) > 0 {
			// Generate test data based on schema properties
			requiredSet := make(map[string]bool)
			for _, req := range schema.Required {
				requiredSet[req] = true
			}

			test.WriteString(fmt.Sprintf("        $model = new %s(\n", className))
			var params []string
			for propName, propSchema := range schema.Properties {
				if propSchema == nil {
					continue
				}
				if requiredSet[propName] {
					testValue := getPHPTestValue(propSchema)
					params = append(params, fmt.Sprintf("            %s: %s", propName, testValue))
				}
			}
			test.WriteString(strings.Join(params, ",\n"))
			test.WriteString("\n        );\n")
			test.WriteString(fmt.Sprintf("        $this->assertInstanceOf(%s::class, $model);\n", className))
		} else {
			// Primitive or array model
			testValue := getPHPTestValue(schema)
			test.WriteString(fmt.Sprintf("        $model = new %s(%s);\n", className, testValue))
			test.WriteString(fmt.Sprintf("        $this->assertInstanceOf(%s::class, $model);\n", className))
		}

		test.WriteString("    }\n\n")

		// Test fromArray method
		test.WriteString("    /**\n")
		test.WriteString(fmt.Sprintf("     * Test %s fromArray method\n", className))
		test.WriteString("     */\n")
		test.WriteString(fmt.Sprintf("    public function test%sFromArray(): void\n", className))
		test.WriteString("    {\n")
		test.WriteString("        $data = [];\n")
		if schema.Type == "object" && len(schema.Properties) > 0 {
			for propName, propSchema := range schema.Properties {
				if propSchema == nil {
					continue
				}
				testValue := getPHPTestValue(propSchema)
				test.WriteString(fmt.Sprintf("        $data['%s'] = %s;\n", propName, testValue))
			}
		} else {
			testValue := getPHPTestValue(schema)
			test.WriteString(fmt.Sprintf("        $data = %s;\n", testValue))
		}
		test.WriteString(fmt.Sprintf("        $model = %s::fromArray($data);\n", className))
		test.WriteString(fmt.Sprintf("        $this->assertInstanceOf(%s::class, $model);\n", className))
		test.WriteString("    }\n\n")

		// Test jsonSerialize method
		test.WriteString("    /**\n")
		test.WriteString(fmt.Sprintf("     * Test %s jsonSerialize method\n", className))
		test.WriteString("     */\n")
		test.WriteString(fmt.Sprintf("    public function test%sJsonSerialize(): void\n", className))
		test.WriteString("    {\n")
		if schema.Type == "object" && len(schema.Properties) > 0 {
			requiredSet := make(map[string]bool)
			for _, req := range schema.Required {
				requiredSet[req] = true
			}
			test.WriteString(fmt.Sprintf("        $model = new %s(\n", className))
			var params []string
			for propName, propSchema := range schema.Properties {
				if propSchema == nil {
					continue
				}
				if requiredSet[propName] {
					testValue := getPHPTestValue(propSchema)
					params = append(params, fmt.Sprintf("            %s: %s", propName, testValue))
				}
			}
			test.WriteString(strings.Join(params, ",\n"))
			test.WriteString("\n        );\n")
		} else {
			testValue := getPHPTestValue(schema)
			test.WriteString(fmt.Sprintf("        $model = new %s(%s);\n", className, testValue))
		}
		test.WriteString("        $serialized = $model->jsonSerialize();\n")
		test.WriteString("        $this->assertIsArray($serialized);\n")
		test.WriteString("    }\n\n")
	}

	test.WriteString("}\n")

	return test.String()
}

// generatePHPApiMethodsTest generates ApiMethodsTest.php
func generatePHPApiMethodsTest(data TemplateData, operations []APIOperation, extractedData *ExtractedData) string {
	namespace := getPHPNamespace(data.SDKName)
	clientClassName := getClientClassName(data.SDKName)

	var test strings.Builder
	test.WriteString("<?php\n\n")
	test.WriteString(fmt.Sprintf("namespace %s\\Tests;\n\n", namespace))
	test.WriteString("use PHPUnit\\Framework\\TestCase;\n")
	test.WriteString("use PHPUnit\\Framework\\MockObject\\MockObject;\n")
	test.WriteString(fmt.Sprintf("use %s\\%s;\n", namespace, clientClassName))
	test.WriteString(fmt.Sprintf("use %s\\Api\\*;\n\n", namespace))
	test.WriteString("/**\n")
	test.WriteString(" * Tests for API methods\n")
	test.WriteString(" * Auto-generated from OpenAPI schema\n")
	test.WriteString(" */\n")
	test.WriteString("class ApiMethodsTest extends TestCase\n")
	test.WriteString("{\n")
	test.WriteString(fmt.Sprintf("    private %s $client;\n\n", clientClassName))
	test.WriteString("    protected function setUp(): void\n")
	test.WriteString("    {\n")
	test.WriteString("        $baseUrl = 'https://api.example.com/v1';\n")
	test.WriteString(fmt.Sprintf("        $this->client = new %s(baseUrl: $baseUrl);\n", clientClassName))
	test.WriteString("    }\n\n")

	// Group operations by tag
	operationsByTag := groupOperationsByTag(operations)

	// Generate tests for each operation
	for tag, tagOperations := range operationsByTag {
		apiClassName := toPascalCase(tag) + "Api"
		test.WriteString(fmt.Sprintf("    /**\n     * Tests for %s API\n     */\n", tag))
		for _, op := range tagOperations {
			methodName := GetOperationMethodName(op)
			if methodName == "" {
				pathPart := strings.ReplaceAll(strings.Trim(op.Path, "/"), "/", "_")
				methodName = strings.ToLower(op.Method) + toPascalCase(pathPart)
			}
			methodName = toCamelCase(methodName)
			testMethodName := "test" + toPascalCase(methodName)

			test.WriteString(fmt.Sprintf("    public function %s(): void\n", testMethodName))
			test.WriteString("    {\n")
			test.WriteString(fmt.Sprintf("        $api = new %s($this->client);\n", apiClassName))
			test.WriteString("        // TODO: Implement test with mocked HTTP client\n")
			test.WriteString("        $this->markTestIncomplete('Test needs HTTP client mocking');\n")
			test.WriteString("    }\n\n")
		}
	}

	test.WriteString("}\n")

	return test.String()
}

// generatePHPAuthTest generates AuthTest.php
func generatePHPAuthTest(data TemplateData, securitySchemes map[string]SecurityScheme, extractedData *ExtractedData) string {
	namespace := getPHPNamespace(data.SDKName)
	clientClassName := getClientClassName(data.SDKName)

	var test strings.Builder
	test.WriteString("<?php\n\n")
	test.WriteString(fmt.Sprintf("namespace %s\\Tests;\n\n", namespace))
	test.WriteString("use PHPUnit\\Framework\\TestCase;\n")
	test.WriteString(fmt.Sprintf("use %s\\%s;\n\n", namespace, clientClassName))
	test.WriteString("/**\n")
	test.WriteString(" * Tests for authentication\n")
	test.WriteString(" * Auto-generated from OpenAPI schema\n")
	test.WriteString(" */\n")
	test.WriteString("class AuthTest extends TestCase\n")
	test.WriteString("{\n")

	// Generate tests for each security scheme
	for name, scheme := range securitySchemes {
		testName := toPascalCase(name) + "Auth"
		test.WriteString("    /**\n")
		test.WriteString(fmt.Sprintf("     * Test %s authentication\n", name))
		test.WriteString("     */\n")
		test.WriteString(fmt.Sprintf("    public function test%s(): void\n", testName))
		test.WriteString("    {\n")
		test.WriteString("        $baseUrl = 'https://api.example.com/v1';\n")
		test.WriteString("        $options = [];\n")

		switch scheme.Type {
		case securitySchemeAPIKey:
			test.WriteString(fmt.Sprintf("        $options['%s'] = 'test-api-key';\n", name))
		case securitySchemeHTTP:
			switch scheme.Scheme {
			case securitySchemeBearer:
				test.WriteString("        $options['bearer_token'] = 'test-token';\n")
			case securitySchemeBasic:
				test.WriteString("        $options['username'] = 'test-user';\n")
				test.WriteString("        $options['password'] = 'test-pass';\n")
			}
		case securitySchemeOAuth2, securitySchemeOpenIDConnect:
			test.WriteString(fmt.Sprintf("        $options['%s_token'] = 'test-token';\n", name))
		}

		test.WriteString(fmt.Sprintf("        $client = new %s(baseUrl: $baseUrl, options: $options);\n", clientClassName))
		test.WriteString(fmt.Sprintf("        $this->assertInstanceOf(%s::class, $client);\n", clientClassName))
		test.WriteString("    }\n\n")
	}

	test.WriteString("}\n")

	return test.String()
}

// getPHPTestValue generates a test value for a schema
func getPHPTestValue(schema *Schema) string {
	if schema == nil {
		return "null"
	}

	switch schema.Type {
	case "string":
		return `"test-string"`
	case "integer", "int32", "int64":
		return "42"
	case "number", "float", "double":
		return "3.14"
	case "boolean":
		return "true"
	case "array":
		return "[]"
	case "object":
		return "[]"
	default:
		return "null"
	}
}
