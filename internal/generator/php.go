// Package generator provides code generation functionality for SDKs.
package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/getkin/kin-openapi/openapi3"

	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

// GeneratePHPSDK generates a PHP SDK
// If version is nil, uses the default PHP version
// If sdkVersion is empty, extracts from OpenAPI schema or defaults to "1.0.0"
// retryConfig specifies retry behavior for HTTP requests (can be disabled)
func GeneratePHPSDK(
	outputPath, sdkName, httpLib string,
	openAPIDoc interface{},
	version *LanguageVersion,
	sdkVersion string,
	generateTests bool,
	retryConfig RetryConfig,
) error {
	// Use default version if not provided
	if version == nil {
		defaultVersion := GetPHPDefaultVersion()
		version = &defaultVersion
	}

	// Convert openAPIDoc to *openapi3.T
	doc, ok := openAPIDoc.(*openapi3.T)
	if !ok {
		// If not an openapi3.T, try to extract from ExtractedData
		if extractedData, ok := openAPIDoc.(*ExtractedData); ok {
			return generatePHPSDKFromExtracted(outputPath, sdkName, httpLib, extractedData, *version, sdkVersion, generateTests, retryConfig)
		}
		return fmt.Errorf("invalid OpenAPI document type")
	}

	// Extract data from OpenAPI document
	extractedData, err := ExtractOpenAPIData(doc)
	if err != nil {
		return fmt.Errorf("failed to extract OpenAPI data: %w", err)
	}

	return generatePHPSDKFromExtracted(outputPath, sdkName, httpLib, extractedData, *version, sdkVersion, generateTests, retryConfig)
}

// generatePHPSDKFromExtracted generates SDK from extracted data
func generatePHPSDKFromExtracted(
	outputPath, sdkName, httpLib string,
	extractedData *ExtractedData,
	version LanguageVersion,
	sdkVersion string,
	generateTests bool,
	retryConfig RetryConfig,
) error {
	// Get HTTP library config
	libConfig, err := httplib.GetLibraryConfig("php", httpLib)
	if err != nil {
		return fmt.Errorf("failed to get HTTP library config: %w", err)
	}

	// Sanitize SDK name for PHP (PascalCase for namespace, but lowercase for directory)
	sanitizedName := toPascalCase(sdkName)
	packageDir := filepath.Join(outputPath, sanitizedName)

	// Create package directory
	if err := os.MkdirAll(packageDir, 0750); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Template data
	data := TemplateData{
		SDKName:         sanitizedName,
		Language:        "php",
		HTTPLib:         httpLib,
		HTTPLibImport:   libConfig.Import,
		HTTPLibConfig:   libConfig,
		OpenAPIDoc:      extractedData,
		ClientClassName: getClientClassName(sanitizedName),
		RetryConfig:     retryConfig,
	}

	// Determine SDK version using common utility
	finalSDKVersion := determineSDKVersion(extractedData, sdkVersion)

	// Generate composer.json
	composerContent := generatePHPComposerJSON(sdkName, finalSDKVersion, httpLib, version, libConfig)
	composerPath := filepath.Join(packageDir, "composer.json")
	// #nosec G306 -- 0644 is appropriate for composer.json file
	if err := os.WriteFile(composerPath, []byte(composerContent), 0644); err != nil {
		return fmt.Errorf("failed to write composer.json: %w", err)
	}

	// Create src/ directory
	srcDir := filepath.Join(packageDir, "src")
	if err := os.MkdirAll(srcDir, 0750); err != nil {
		return fmt.Errorf("failed to create src directory: %w", err)
	}

	// Generate client class
	clientContent := generatePHPClient(data, version)
	clientPath := filepath.Join(srcDir, fmt.Sprintf("%sClient.php", sanitizedName))
	// #nosec G306 -- 0644 is appropriate for PHP source files
	if err := os.WriteFile(clientPath, []byte(clientContent), 0644); err != nil {
		return fmt.Errorf("failed to write client class: %w", err)
	}
	// Format with PHP-CS-Fixer (if available)
	if err := formatPHPFile(clientPath); err != nil {
		// Log but don't fail - formatting is nice-to-have
		_ = err
	}

	// Generate models if schemas exist
	if len(extractedData.Schemas) > 0 {
		modelsDir := filepath.Join(srcDir, "Models")
		if err := os.MkdirAll(modelsDir, 0750); err != nil {
			return fmt.Errorf("failed to create Models directory: %w", err)
		}

		// Generate Models/Model.php for each schema
		for name, schema := range extractedData.Schemas {
			// Pass SDK name for namespace generation
			modelContent := generatePHPModelWithNamespace(name, schema, extractedData.Schemas, data.SDKName, version)
			modelFileName := fmt.Sprintf("%s.php", toPascalCase(name))
			modelPath := filepath.Join(modelsDir, modelFileName)
			// #nosec G306 -- 0644 is appropriate for PHP source files
			if err := os.WriteFile(modelPath, []byte(modelContent), 0644); err != nil {
				return fmt.Errorf("failed to write model %s: %w", name, err)
			}
			// Format with PHP-CS-Fixer (if available)
			if err := formatPHPFile(modelPath); err != nil {
				_ = err
			}
		}
	}

	// Generate API methods if operations exist
	if len(extractedData.Operations) > 0 {
		apiDir := filepath.Join(srcDir, "Api")
		if err := os.MkdirAll(apiDir, 0750); err != nil {
			return fmt.Errorf("failed to create Api directory: %w", err)
		}

		// Group operations by tags
		operationsByTag := groupOperationsByTag(extractedData.Operations)
		for tag, operations := range operationsByTag {
			apiContent := generatePHPAPIModule(tag, operations, data, version)
			apiFileName := fmt.Sprintf("%sApi.php", toPascalCase(tag))
			apiPath := filepath.Join(apiDir, apiFileName)
			// #nosec G306 -- 0644 is appropriate for PHP source files
			if err := os.WriteFile(apiPath, []byte(apiContent), 0644); err != nil {
				return fmt.Errorf("failed to write API module %s: %w", tag, err)
			}
			// Format with PHP-CS-Fixer (if available)
			if err := formatPHPFile(apiPath); err != nil {
				_ = err
			}
		}
	}

	// Generate Exceptions directory
	exceptionsDir := filepath.Join(srcDir, "Exceptions")
	if err := os.MkdirAll(exceptionsDir, 0750); err != nil {
		return fmt.Errorf("failed to create Exceptions directory: %w", err)
	}

	// Generate ApiException.php
	exceptionContent := generatePHPException()
	exceptionPath := filepath.Join(exceptionsDir, "ApiException.php")
	// #nosec G306 -- 0644 is appropriate for PHP source files
	if err := os.WriteFile(exceptionPath, []byte(exceptionContent), 0644); err != nil {
		return fmt.Errorf("failed to write ApiException: %w", err)
	}
	// Format with PHP-CS-Fixer (if available)
	if err := formatPHPFile(exceptionPath); err != nil {
		_ = err
	}

	// Generate README.md
	readmeContent := generatePHPReadme(data, finalSDKVersion)
	readmePath := filepath.Join(packageDir, "README.md")
	// #nosec G306 -- 0644 is appropriate for README file
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	// Generate examples/ directory
	examplesDir := filepath.Join(packageDir, "examples")
	if err := os.MkdirAll(examplesDir, 0750); err != nil {
		return fmt.Errorf("failed to create examples directory: %w", err)
	}

	// Generate examples/basic_usage.php
	examplesContent := generatePHPExamples(data)
	examplesPath := filepath.Join(examplesDir, "basic_usage.php")
	// #nosec G306 -- 0644 is appropriate for PHP example files
	if err := os.WriteFile(examplesPath, []byte(examplesContent), 0644); err != nil {
		return fmt.Errorf("failed to write examples: %w", err)
	}
	// Format with PHP-CS-Fixer (if available)
	if err := formatPHPFile(examplesPath); err != nil {
		_ = err
	}

	// Generate tests/ directory if test generation is enabled
	if generateTests {
		if err := generatePHPTests(packageDir, srcDir, data, extractedData); err != nil {
			return fmt.Errorf("failed to generate tests: %w", err)
		}
	}

	// Generate code quality configuration files
	if err := generatePHPQualityConfigs(packageDir); err != nil {
		return fmt.Errorf("failed to generate quality configs: %w", err)
	}

	return nil
}

// generatePHPComposerJSON generates composer.json file
func generatePHPComposerJSON(sdkName, sdkVersion, httpLib string, version LanguageVersion, libConfig *httplib.LibraryConfig) string {
	// Parse dependency string (format: "package:version" or "package")
	dependency := libConfig.Dependency
	if dependency == "" {
		// Default to guzzle if no dependency specified
		dependency = "guzzlehttp/guzzle:^7.0"
	}

	// Split dependency into package and version if needed
	parts := strings.Split(dependency, ":")
	packageName := parts[0]
	packageVersion := ""
	if len(parts) > 1 {
		packageVersion = parts[1]
	}

	// Build require section
	requireSection := fmt.Sprintf(`        "php": "%s"`, version.GetPHPVersionString())
	if packageName != "" {
		if packageVersion != "" {
			requireSection += fmt.Sprintf(",\n        \"%s\": \"%s\"", packageName, packageVersion)
		} else {
			requireSection += fmt.Sprintf(",\n        \"%s\"", packageName)
		}
	}

	// Add PHPUnit for dev dependencies if tests will be generated
	requireDevSection := `        "phpunit/phpunit": "^10.0"`

	namespace := fmt.Sprintf("Vendor\\%s", sdkName)

	return fmt.Sprintf(`{
    "name": "vendor/%s",
    "description": "PHP SDK for %s - Auto-generated from OpenAPI schema",
    "type": "library",
    "license": "MIT",
    "version": "%s",
    "require": {
%s
    },
    "require-dev": {
%s
    },
    "autoload": {
        "psr-4": {
            "%s\\": "src/"
        }
    },
    "autoload-dev": {
        "psr-4": {
            "%s\\Tests\\": "tests/"
        }
    }
}
`, strings.ToLower(sdkName), sdkName, sdkVersion, requireSection, requireDevSection, namespace, namespace)
}

// generatePHPClient generates PHP client class
// Implementation moved to php_client.go

// generatePHPModel is now generatePHPModelWithNamespace in php_models.go

// generatePHPAPIModule is now implemented in php_api.go

// generatePHPException generates PHP exception class
func generatePHPException() string {
	return `<?php

namespace Vendor\SdkName\Exceptions;

class ApiException extends \Exception
{
    public function __construct(
        string $message,
        int $code = 0,
        ?\Throwable $previous = null,
        private ?array $responseBody = null
    ) {
        parent::__construct($message, $code, $previous);
    }
    
    public function getResponseBody(): ?array
    {
        return $this->responseBody;
    }
}
`
}

// generatePHPReadme generates README.md for PHP SDK
func generatePHPReadme(data TemplateData, sdkVersion string) string {
	extractedData, ok := data.OpenAPIDoc.(*ExtractedData)
	namespace := getPHPNamespace(data.SDKName)
	clientClassName := getClientClassName(data.SDKName)

	// Format display name
	displayName := strings.ReplaceAll(data.SDKName, "_", " ")
	c := cases.Title(language.English)
	displayName = c.String(strings.ToLower(displayName))

	// Get base URL
	baseURL := "https://api.example.com/v1"
	if ok && extractedData != nil && extractedData.BaseURL != "" {
		baseURL = extractedData.BaseURL
	}

	// Get description
	description := "Auto-generated PHP SDK from OpenAPI schema"
	if ok && extractedData != nil && extractedData.Description != "" {
		description = extractedData.Description
	}

	var readme strings.Builder
	readme.WriteString(fmt.Sprintf("# %s PHP SDK\n\n", displayName))
	readme.WriteString(fmt.Sprintf("%s\n\n", description))
	readme.WriteString(fmt.Sprintf("**Version:** %s\n\n", sdkVersion))
	readme.WriteString("## Installation\n\n")
	readme.WriteString("Install via Composer:\n\n")
	readme.WriteString("```bash\n")
	readme.WriteString("composer require vendor/" + strings.ToLower(data.SDKName) + "\n")
	readme.WriteString("```\n\n")
	readme.WriteString("Or add to your `composer.json`:\n\n")
	readme.WriteString("```json\n")
	readme.WriteString(fmt.Sprintf("{\n    \"require\": {\n        \"vendor/%s\": \"%s\"\n    }\n}\n", strings.ToLower(data.SDKName), sdkVersion))
	readme.WriteString("```\n\n")

	readme.WriteString("## Quick Start\n\n")
	readme.WriteString("```php\n")
	readme.WriteString("<?php\n\n")
	readme.WriteString("require_once __DIR__ . '/vendor/autoload.php';\n\n")
	readme.WriteString(fmt.Sprintf("use %s\\%s;\n", namespace, clientClassName))

	// Add API namespace if operations exist
	if ok && extractedData != nil && len(extractedData.Operations) > 0 {
		operationsByTag := groupOperationsByTag(extractedData.Operations)
		for tag := range operationsByTag {
			apiClassName := toPascalCase(tag) + "Api"
			readme.WriteString(fmt.Sprintf("use %s\\Api\\%s;\n", namespace, apiClassName))
			break // Just show first one as example
		}
	}

	readme.WriteString("\n")
	readme.WriteString("// Initialize client\n")
	readme.WriteString(fmt.Sprintf("$client = new %s(\n", clientClassName))
	readme.WriteString(fmt.Sprintf("    baseUrl: \"%s\"\n", baseURL))

	// Add authentication example if available
	if ok && extractedData != nil && len(extractedData.SecuritySchemes) > 0 {
		for name, scheme := range extractedData.SecuritySchemes {
			switch scheme.Type {
			case securitySchemeAPIKey:
				readme.WriteString(fmt.Sprintf("    options: [\n        '%s' => 'your-api-key'\n    ]\n", name))
			case securitySchemeHTTP:
				switch scheme.Scheme {
				case securitySchemeBearer:
					readme.WriteString("    options: [\n        'bearer_token' => 'your-token'\n    ]\n")
				case securitySchemeBasic:
					readme.WriteString("    options: [\n        'username' => 'your-username',\n        'password' => 'your-password'\n    ]\n")
				}
			case securitySchemeOAuth2, securitySchemeOpenIDConnect:
				readme.WriteString(fmt.Sprintf("    options: [\n        '%s_token' => 'your-token'\n    ]\n", name))
			}
			break // Just show first one as example
		}
	} else {
		readme.WriteString(")\n")
	}

	readme.WriteString(");\n\n")

	// Add API usage example if operations exist
	if ok && extractedData != nil && len(extractedData.Operations) > 0 {
		operationsByTag := groupOperationsByTag(extractedData.Operations)
		for tag, operations := range operationsByTag {
			if len(operations) > 0 {
				apiClassName := toPascalCase(tag) + "Api"
				op := operations[0]
				methodName := GetOperationMethodName(op)
				if methodName == "" {
					pathPart := strings.ReplaceAll(strings.Trim(op.Path, "/"), "/", "_")
					methodName = strings.ToLower(op.Method) + toPascalCase(pathPart)
				}
				methodName = toCamelCase(methodName)

				readme.WriteString("// Use API methods\n")
				readme.WriteString(fmt.Sprintf("$api = new %s($client);\n", apiClassName))
				readme.WriteString(fmt.Sprintf("$response = $api->%s(", methodName))

				// Add example parameters
				var params []string
				for _, param := range op.Parameters {
					if param.In == paramLocationPath {
						params = append(params, fmt.Sprintf("\"example-%s\"", param.Name))
					}
				}
				if len(params) > 0 {
					readme.WriteString(strings.Join(params, ", "))
				}
				readme.WriteString(");\n")
				readme.WriteString("print_r($response);\n")
				break // Just show first operation as example
			}
		}
	}

	readme.WriteString("```\n\n")

	// HTTP Library section
	readme.WriteString("## HTTP Library\n\n")
	readme.WriteString(fmt.Sprintf("This SDK uses **%s** for HTTP requests.\n\n", data.HTTPLib))

	// Authentication section
	if ok && extractedData != nil && len(extractedData.SecuritySchemes) > 0 {
		readme.WriteString("## Authentication\n\n")
		readme.WriteString("Configure authentication when creating the client:\n\n")
		readme.WriteString("```php\n")
		for name, scheme := range extractedData.SecuritySchemes {
			switch scheme.Type {
			case securitySchemeAPIKey:
				readme.WriteString(fmt.Sprintf("$client = new %s(\n", clientClassName))
				readme.WriteString(fmt.Sprintf("    baseUrl: \"%s\",\n", baseURL))
				readme.WriteString(fmt.Sprintf("    options: ['%s' => 'your-api-key']\n", name))
				readme.WriteString(");\n")
			case securitySchemeHTTP:
				switch scheme.Scheme {
				case securitySchemeBearer:
					readme.WriteString(fmt.Sprintf("$client = new %s(\n", clientClassName))
					readme.WriteString(fmt.Sprintf("    baseUrl: \"%s\",\n", baseURL))
					readme.WriteString("    options: ['bearer_token' => 'your-token']\n")
					readme.WriteString(");\n")
				case securitySchemeBasic:
					readme.WriteString(fmt.Sprintf("$client = new %s(\n", clientClassName))
					readme.WriteString(fmt.Sprintf("    baseUrl: \"%s\",\n", baseURL))
					readme.WriteString("    options: [\n")
					readme.WriteString("        'username' => 'your-username',\n")
					readme.WriteString("        'password' => 'your-password'\n")
					readme.WriteString("    ]\n")
					readme.WriteString(");\n")
				}
			case securitySchemeOAuth2, securitySchemeOpenIDConnect:
				readme.WriteString(fmt.Sprintf("$client = new %s(\n", clientClassName))
				readme.WriteString(fmt.Sprintf("    baseUrl: \"%s\",\n", baseURL))
				readme.WriteString(fmt.Sprintf("    options: ['%s_token' => 'your-token']\n", name))
				readme.WriteString(");\n")
			}
			break // Just show first one
		}
		readme.WriteString("```\n\n")
	}

	// Retry mechanism section (if enabled)
	if data.RetryConfig.Enabled {
		readme.WriteString("## Retry Mechanism\n\n")
		readme.WriteString("This SDK includes automatic retry logic for transient failures.\n\n")
		readme.WriteString("**Configuration:**\n")
		readme.WriteString(fmt.Sprintf("- Max attempts: %d\n", data.RetryConfig.MaxAttempts))
		readme.WriteString(fmt.Sprintf("- Strategy: %s\n", data.RetryConfig.Strategy))
		readme.WriteString(fmt.Sprintf("- Initial delay: %.1fs\n", data.RetryConfig.InitialDelay.Seconds()))
		readme.WriteString(fmt.Sprintf("- Max delay: %.1fs\n", data.RetryConfig.MaxDelay.Seconds()))
		readme.WriteString(fmt.Sprintf("- Backoff multiplier: %.1f\n", data.RetryConfig.BackoffMultiplier))
		readme.WriteString("\n")
	}

	// Testing section
	readme.WriteString("## Testing\n\n")
	readme.WriteString("Run PHPUnit tests:\n\n")
	readme.WriteString("```bash\n")
	readme.WriteString("composer install --dev\n")
	readme.WriteString("vendor/bin/phpunit\n")
	readme.WriteString("```\n\n")

	// Code Quality section
	readme.WriteString("## Code Quality\n\n")
	readme.WriteString("This SDK includes code quality tools:\n\n")
	readme.WriteString("- **PHP_CodeSniffer** (PSR-12): `vendor/bin/phpcs`\n")
	readme.WriteString("- **PHPStan** (static analysis): `vendor/bin/phpstan analyse`\n")
	readme.WriteString("- **PHP-CS-Fixer** (formatting): `vendor/bin/php-cs-fixer fix`\n\n")

	// License section
	readme.WriteString("## License\n\n")
	readme.WriteString("Generated by [SDK Forge](https://github.com/vubon/sdk-forge)\n")

	return readme.String()
}

// generatePHPExamples generates PHP usage examples
func generatePHPExamples(data TemplateData) string {
	extractedData, ok := data.OpenAPIDoc.(*ExtractedData)
	namespace := getPHPNamespace(data.SDKName)
	clientClassName := getClientClassName(data.SDKName)

	baseURL := "https://api.example.com/v1"
	if ok && extractedData != nil && extractedData.BaseURL != "" {
		baseURL = extractedData.BaseURL
	}

	var examples strings.Builder
	examples.WriteString("<?php\n\n")
	examples.WriteString("/**\n")
	examples.WriteString(" * Usage examples for PHP SDK\n")
	examples.WriteString(" * Auto-generated from OpenAPI schema\n")
	examples.WriteString(" */\n\n")
	examples.WriteString("require_once __DIR__ . '/../vendor/autoload.php';\n\n")
	examples.WriteString(fmt.Sprintf("use %s\\%s;\n", namespace, clientClassName))

	// Add API and Models namespaces if they exist
	if ok && extractedData != nil {
		if len(extractedData.Operations) > 0 {
			operationsByTag := groupOperationsByTag(extractedData.Operations)
			for tag := range operationsByTag {
				apiClassName := toPascalCase(tag) + "Api"
				examples.WriteString(fmt.Sprintf("use %s\\Api\\%s;\n", namespace, apiClassName))
				break // Just show first one
			}
		}
		if len(extractedData.Schemas) > 0 {
			examples.WriteString(fmt.Sprintf("use %s\\Models\\*;\n", namespace))
		}
	}

	examples.WriteString("\n")
	examples.WriteString("// Example 1: Initialize client\n")
	examples.WriteString(fmt.Sprintf("$client = new %s(\n", clientClassName))
	examples.WriteString(fmt.Sprintf("    baseUrl: \"%s\"\n", baseURL))

	// Add authentication example
	if ok && extractedData != nil && len(extractedData.SecuritySchemes) > 0 {
		for name, scheme := range extractedData.SecuritySchemes {
			switch scheme.Type {
			case securitySchemeAPIKey:
				examples.WriteString(fmt.Sprintf("    options: [\n        '%s' => 'your-api-key'\n    ]\n", name))
			case securitySchemeHTTP:
				switch scheme.Scheme {
				case securitySchemeBearer:
					examples.WriteString("    options: [\n        'bearer_token' => 'your-token'\n    ]\n")
				case securitySchemeBasic:
					examples.WriteString("    options: [\n        'username' => 'your-username',\n        'password' => 'your-password'\n    ]\n")
				}
			case securitySchemeOAuth2, securitySchemeOpenIDConnect:
				examples.WriteString(fmt.Sprintf("    options: [\n        '%s_token' => 'your-token'\n    ]\n", name))
			}
			break
		}
	} else {
		examples.WriteString(")\n")
	}

	examples.WriteString(");\n\n")

	// Add API usage examples
	if ok && extractedData != nil && len(extractedData.Operations) > 0 {
		operationsByTag := groupOperationsByTag(extractedData.Operations)
		exampleCount := 0
		for tag, operations := range operationsByTag {
			if exampleCount >= 3 {
				break // Limit to 3 examples
			}
			if len(operations) > 0 {
				apiClassName := toPascalCase(tag) + "Api"
				op := operations[0]
				methodName := GetOperationMethodName(op)
				if methodName == "" {
					pathPart := strings.ReplaceAll(strings.Trim(op.Path, "/"), "/", "_")
					methodName = strings.ToLower(op.Method) + toPascalCase(pathPart)
				}
				methodName = toCamelCase(methodName)

				examples.WriteString(fmt.Sprintf("// Example %d: Use %s API\n", exampleCount+2, tag))
				examples.WriteString(fmt.Sprintf("$api = new %s($client);\n", apiClassName))
				examples.WriteString("try {\n")
				examples.WriteString(fmt.Sprintf("    $response = $api->%s(", methodName))

				// Add example parameters
				var params []string
				for _, param := range op.Parameters {
					if param.In == paramLocationPath {
						testValue := getPHPTestValue(param.Schema)
						params = append(params, testValue)
					}
				}
				if len(params) > 0 {
					examples.WriteString(strings.Join(params, ", "))
				}
				examples.WriteString(");\n")
				examples.WriteString("    print_r($response);\n")
				examples.WriteString("} catch (\\Exception $e) {\n")
				examples.WriteString("    echo \"Error: \" . $e->getMessage() . \"\\n\";\n")
				examples.WriteString("}\n\n")
				exampleCount++
			}
		}
	}

	// Add model usage example
	if ok && extractedData != nil && len(extractedData.Schemas) > 0 {
		examples.WriteString("// Example: Working with models\n")
		for name, schema := range extractedData.Schemas {
			className := toPascalCase(name)
			examples.WriteString("$data = [\n")
			if schema.Type == "object" && len(schema.Properties) > 0 {
				requiredSet := make(map[string]bool)
				for _, req := range schema.Required {
					requiredSet[req] = true
				}
				for propName, propSchema := range schema.Properties {
					if propSchema == nil {
						continue
					}
					if requiredSet[propName] {
						testValue := getPHPTestValue(propSchema)
						examples.WriteString(fmt.Sprintf("    '%s' => %s,\n", propName, testValue))
					}
				}
			}
			examples.WriteString("];\n")
			examples.WriteString(fmt.Sprintf("$model = %s::fromArray($data);\n", className))
			examples.WriteString("$json = json_encode($model->jsonSerialize());\n")
			examples.WriteString("echo $json . \"\\n\";\n")
			break // Just show first model
		}
	}

	return examples.String()
}

// generatePHPTests is now implemented in php_tests.go

// generatePHPQualityConfigs generates code quality configuration files
func generatePHPQualityConfigs(packageDir string) error {
	// Generate phpcs.xml (PHP_CodeSniffer configuration)
	phpcsContent := generatePHPCodeSnifferConfig()
	phpcsPath := filepath.Join(packageDir, "phpcs.xml")
	// #nosec G306 -- 0644 is appropriate for config files
	if err := os.WriteFile(phpcsPath, []byte(phpcsContent), 0644); err != nil {
		return fmt.Errorf("failed to write phpcs.xml: %w", err)
	}

	// Generate phpstan.neon (PHPStan configuration)
	phpstanContent := generatePHPPHPStanConfig()
	phpstanPath := filepath.Join(packageDir, "phpstan.neon")
	// #nosec G306 -- 0644 is appropriate for config files
	if err := os.WriteFile(phpstanPath, []byte(phpstanContent), 0644); err != nil {
		return fmt.Errorf("failed to write phpstan.neon: %w", err)
	}

	// Generate .php-cs-fixer.php (PHP-CS-Fixer configuration)
	csFixerContent := generatePHPCsFixerConfig()
	csFixerPath := filepath.Join(packageDir, ".php-cs-fixer.php")
	// #nosec G306 -- 0644 is appropriate for config files
	if err := os.WriteFile(csFixerPath, []byte(csFixerContent), 0644); err != nil {
		return fmt.Errorf("failed to write .php-cs-fixer.php: %w", err)
	}

	return nil
}

// generatePHPCodeSnifferConfig generates phpcs.xml configuration
func generatePHPCodeSnifferConfig() string {
	return `<?xml version="1.0"?>
<ruleset name="SDK CodeSniffer Rules">
    <description>PHP_CodeSniffer configuration for generated SDK</description>
    
    <!-- Include the whole project -->
    <file>src</file>
    <file>tests</file>
    
    <!-- Use PSR-12 standard -->
    <rule ref="PSR12"/>
    
    <!-- Exclude vendor directory -->
    <exclude-pattern>vendor/*</exclude-pattern>
    <exclude-pattern>*.cache</exclude-pattern>
</ruleset>
`
}

// generatePHPPHPStanConfig generates phpstan.neon configuration
func generatePHPPHPStanConfig() string {
	return `parameters:
    level: 5
    paths:
        - src
        - tests
    excludePaths:
        - vendor
    checkMissingIterableValueType: false
    checkGenericClassInNonGenericObjectType: false
`
}

// generatePHPCsFixerConfig generates .php-cs-fixer.php configuration
func generatePHPCsFixerConfig() string {
	return `<?php

$finder = PhpCsFixer\Finder::create()
    ->in(__DIR__)
    ->exclude('vendor')
    ->exclude('.phpunit.cache')
    ->name('*.php');

$config = new PhpCsFixer\Config();
return $config
    ->setRules([
        '@PSR12' => true,
        'array_syntax' => ['syntax' => 'short'],
        'ordered_imports' => ['sort_algorithm' => 'alpha'],
        'no_unused_imports' => true,
        'not_operator_with_successor_space' => true,
        'trailing_comma_in_multiline' => true,
        'phpdoc_scalar' => true,
        'unary_operator_spaces' => true,
        'binary_operator_spaces' => true,
        'blank_line_before_statement' => [
            'statements' => ['break', 'continue', 'declare', 'return', 'throw', 'try'],
        ],
        'phpdoc_single_line_var_spacing' => true,
        'phpdoc_var_without_name' => true,
    ])
    ->setFinder($finder);
`
}

// formatPHPFile formats a PHP source file using PHP-CS-Fixer (if available)
func formatPHPFile(filePath string) error {
	// Try PHP-CS-Fixer first
	cmd := exec.Command("php-cs-fixer", "fix", "--quiet", filePath)
	if err := cmd.Run(); err == nil {
		return nil
	}

	// If PHP-CS-Fixer is not available, that's okay - just skip formatting
	return fmt.Errorf("no PHP formatter available (php-cs-fixer)")
}
