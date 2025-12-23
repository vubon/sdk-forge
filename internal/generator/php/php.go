// Package php provides PHP SDK generation functionality.
package php

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/vubon/sdk-forge/internal/generator/common"
	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

//go:embed templates/composer.json.tmpl
var phpComposerTemplate string

//go:embed templates/README.md.tmpl
var phpReadmeTemplate string

func getPHPComposerTemplateContent() string {
	return phpComposerTemplate
}

func getPHPReadmeTemplateContent() string {
	return phpReadmeTemplate
}

// GeneratePHPSDK generates a PHP SDK
// If version is nil, uses the default PHP version
// If sdkVersion is empty, extracts from OpenAPI schema or defaults to "1.0.0"
// retryConfig specifies retry behavior for HTTP requests (can be disabled)
func GeneratePHPSDK(
	outputPath, sdkName, httpLib string,
	openAPIDoc interface{},
	version *common.LanguageVersion,
	sdkVersion string,
	generateTests bool,
	retryConfig common.RetryConfig,
) error {
	// Use default version if not provided
	if version == nil {
		defaultVersion := common.GetPHPDefaultVersion()
		version = &defaultVersion
	}

	// Convert openAPIDoc to *openapi3.T
	doc, ok := openAPIDoc.(*openapi3.T)
	if !ok {
		// If not an openapi3.T, try to extract from ExtractedData
		if extractedData, ok := openAPIDoc.(*common.ExtractedData); ok {
			return generatePHPSDKFromExtracted(outputPath, sdkName, httpLib, extractedData, *version, sdkVersion, generateTests, retryConfig)
		}
		return fmt.Errorf("invalid OpenAPI document type")
	}

	// Extract data from OpenAPI document
	extractedData, err := common.ExtractOpenAPIData(doc)
	if err != nil {
		return fmt.Errorf("failed to extract OpenAPI data: %w", err)
	}

	return generatePHPSDKFromExtracted(outputPath, sdkName, httpLib, extractedData, *version, sdkVersion, generateTests, retryConfig)
}

// generatePHPSDKFromExtracted generates SDK from extracted data
func generatePHPSDKFromExtracted(
	outputPath, sdkName, httpLib string,
	extractedData *common.ExtractedData,
	version common.LanguageVersion,
	sdkVersion string,
	generateTests bool,
	retryConfig common.RetryConfig,
) error {
	// Get HTTP library config
	libConfig, err := httplib.GetLibraryConfig("php", httpLib)
	if err != nil {
		return fmt.Errorf("failed to get HTTP library config: %w", err)
	}

	// Sanitize SDK name for PHP (PascalCase for namespace, but lowercase for directory)
	sanitizedName := common.ToPascalCase(sdkName)
	packageDir := filepath.Join(outputPath, sanitizedName)

	// Create package directory
	if err := os.MkdirAll(packageDir, 0750); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Template data
	data := common.TemplateData{
		SDKName:         sanitizedName,
		Language:        "php",
		HTTPLib:         httpLib,
		HTTPLibImport:   libConfig.Import,
		HTTPLibConfig:   libConfig,
		OpenAPIDoc:      extractedData,
		ClientClassName: common.GetClientClassName(sanitizedName),
		RetryConfig:     retryConfig,
	}

	// Determine SDK version using common utility
	finalSDKVersion := common.DetermineSDKVersion(extractedData, sdkVersion)

	// Generate composer.json using template
	composerContent, err := generatePHPComposerJSON(sdkName, finalSDKVersion, httpLib, version, libConfig)
	if err != nil {
		return fmt.Errorf("failed to generate composer.json: %w", err)
	}
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

	// Generate client class using template
	clientContent, err := generatePHPClient(data, version)
	if err != nil {
		return fmt.Errorf("failed to generate client: %w", err)
	}
	// Use client class name for filename to match PSR-4 autoloading requirements
	// The class name is already determined by GetClientClassName (e.g., "Petstore")
	clientFileName := fmt.Sprintf("%s.php", data.ClientClassName)
	clientPath := filepath.Join(srcDir, clientFileName)
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
			modelFileName := fmt.Sprintf("%s.php", common.ToPascalCase(name))
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
		operationsByTag := common.GroupOperationsByTag(extractedData.Operations)
		for tag, operations := range operationsByTag {
			apiContent := generatePHPAPIModule(tag, operations, data, version)
			apiFileName := fmt.Sprintf("%sApi.php", common.ToPascalCase(tag))
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
	exceptionContent := generatePHPException(data.SDKName)
	exceptionPath := filepath.Join(exceptionsDir, "ApiException.php")
	// #nosec G306 -- 0644 is appropriate for PHP source files
	if err := os.WriteFile(exceptionPath, []byte(exceptionContent), 0644); err != nil {
		return fmt.Errorf("failed to write ApiException: %w", err)
	}
	// Format with PHP-CS-Fixer (if available)
	if err := formatPHPFile(exceptionPath); err != nil {
		_ = err
	}

	// Generate README.md using template
	readmeContent, err := generatePHPReadme(data, finalSDKVersion)
	if err != nil {
		return fmt.Errorf("failed to generate README: %w", err)
	}
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

// generatePHPComposerJSON generates composer.json file using template
func generatePHPComposerJSON(sdkName, sdkVersion, httpLib string, version common.LanguageVersion, libConfig *httplib.LibraryConfig) (string, error) {
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

	// Build dependencies string
	var dependencies strings.Builder
	dependencies.WriteString(fmt.Sprintf(`        "php": "%s"`, version.GetPHPVersionString()))
	if packageName != "" {
		if packageVersion != "" {
			dependencies.WriteString(fmt.Sprintf(",\n        \"%s\": \"%s\"", packageName, packageVersion))
		} else {
			dependencies.WriteString(fmt.Sprintf(",\n        \"%s\"", packageName))
		}
	}

	namespace := fmt.Sprintf("Vendor\\%s\\", common.ToPascalCase(sdkName))
	// Escape namespace for JSON: each backslash must be doubled for JSON
	// Namespace includes trailing backslash for PSR-4 autoloading
	// "Vendor\Petstore\" becomes "Vendor\\Petstore\\" in JSON
	namespaceEscaped := strings.ReplaceAll(namespace, `\`, `\\`)

	// Prepare template data
	type ComposerData struct {
		SDKName          string
		SDKVersion       string
		PHPVersion       string
		Dependencies     string
		Namespace        string
		NamespaceEscaped string
	}
	templateData := ComposerData{
		SDKName:          strings.ToLower(sdkName),
		SDKVersion:       sdkVersion,
		PHPVersion:       version.GetPHPVersionString(),
		Dependencies:     dependencies.String(),
		Namespace:        namespace,
		NamespaceEscaped: namespaceEscaped,
	}

	// Load and render template
	tmpl, err := common.LoadTemplate(getPHPComposerTemplateContent())
	if err != nil {
		return "", fmt.Errorf("failed to load composer template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", fmt.Errorf("failed to execute composer template: %w", err)
	}

	return buf.String(), nil
}

// generatePHPClient generates PHP client class
// Implementation moved to php_client.go

// generatePHPModel is now generatePHPModelWithNamespace in php_models.go

// generatePHPAPIModule is now implemented in php_api.go

// generatePHPException generates PHP exception class
func generatePHPException(sdkName string) string {
	namespace := fmt.Sprintf("Vendor\\%s", common.ToPascalCase(sdkName))
	return fmt.Sprintf(`<?php

namespace %s\Exceptions;

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
`, namespace)
}

// generatePHPReadme generates README.md for PHP SDK using template
func generatePHPReadme(data common.TemplateData, sdkVersion string) (string, error) {
	extractedData, ok := data.OpenAPIDoc.(*common.ExtractedData)
	namespace := fmt.Sprintf("Vendor\\%s", common.ToPascalCase(data.SDKName))
	clientClassName := common.GetClientClassName(data.SDKName)

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

	// Generate auth example for quick start
	authExample := ""
	hasAuth := ok && extractedData != nil && len(extractedData.SecuritySchemes) > 0
	if hasAuth {
		for name, scheme := range extractedData.SecuritySchemes {
			switch scheme.Type {
			case "apiKey":
				authExample = fmt.Sprintf("        '%s' => 'your-api-key'", name)
			case "http":
				switch scheme.Scheme {
				case "bearer", "Bearer":
					authExample = "        'bearer_token' => 'your-token'"
				case "basic", "Basic":
					authExample = "        'username' => 'your-username',\n        'password' => 'your-password'"
				}
			case "oauth2", "OAuth2", "openIdConnect", "OpenIDConnect":
				authExample = fmt.Sprintf("        '%s_token' => 'your-token'", name)
			}
			break // Just show first one
		}
	}

	// Generate API imports
	apiImports := ""
	hasAPI := ok && extractedData != nil && len(extractedData.Operations) > 0
	if hasAPI {
		operationsByTag := common.GroupOperationsByTag(extractedData.Operations)
		for tag := range operationsByTag {
			apiClassName := common.ToPascalCase(tag) + "Api"
			apiImports = fmt.Sprintf("use %s\\Api\\%s;", namespace, apiClassName)
			break // Just show first one
		}
	}

	// Generate usage example
	usageExample := ""
	hasOperations := ok && extractedData != nil && len(extractedData.Operations) > 0
	if hasOperations {
		operationsByTag := common.GroupOperationsByTag(extractedData.Operations)
		for tag, operations := range operationsByTag {
			if len(operations) > 0 {
				apiClassName := common.ToPascalCase(tag) + "Api"
				op := operations[0]
				methodName := common.GetOperationMethodName(op)
				if methodName == "" {
					pathPart := strings.ReplaceAll(strings.Trim(op.Path, "/"), "/", "_")
					methodName = strings.ToLower(op.Method) + common.ToPascalCase(pathPart)
				}
				methodName = common.ToCamelCase(methodName)

				var params []string
				for _, param := range op.Parameters {
					if param.In == "path" {
						params = append(params, fmt.Sprintf("\"example-%s\"", param.Name))
					}
				}
				paramStr := ""
				if len(params) > 0 {
					paramStr = strings.Join(params, ", ")
				}

				usageExample = fmt.Sprintf("$api = new %s($client);\n$response = $api->%s(%s);\nprint_r($response);", apiClassName, methodName, paramStr)
				break
			}
		}
	}

	// Generate auth example code for authentication section
	authExampleCode := ""
	if hasAuth {
		for name, scheme := range extractedData.SecuritySchemes {
			switch scheme.Type {
			case "apiKey":
				authExampleCode = fmt.Sprintf("$client = new %s(\n    baseUrl: \"%s\",\n    options: ['%s' => 'your-api-key']\n);", clientClassName, baseURL, name)
			case "http":
				switch scheme.Scheme {
				case "bearer":
					authExampleCode = fmt.Sprintf("$client = new %s(\n    baseUrl: \"%s\",\n    options: ['bearer_token' => 'your-token']\n);", clientClassName, baseURL)
				case "basic":
					authExampleCode = fmt.Sprintf("$client = new %s(\n    baseUrl: \"%s\",\n    options: [\n        'username' => 'your-username',\n        'password' => 'your-password'\n    ]\n);", clientClassName, baseURL)
				}
			case "oauth2", "openIdConnect":
				authExampleCode = fmt.Sprintf("$client = new %s(\n    baseUrl: \"%s\",\n    options: ['%s_token' => 'your-token']\n);", clientClassName, baseURL, name)
			}
			break
		}
	}

	// Prepare template data
	type PHPReadmeData struct {
		DisplayName            string
		Description            string
		SDKVersion             string
		SDKName                string
		Namespace              string
		ClientClassName        string
		BaseURL                string
		HTTPLib                string
		HasAuth                bool
		AuthExample            string
		AuthExampleCode        string
		HasAPI                 bool
		APIImports             string
		HasOperations          bool
		UsageExample           string
		RetryEnabled           bool
		RetryMaxAttempts       int
		RetryStrategy          string
		RetryInitialDelay      float64
		RetryMaxDelay          float64
		RetryBackoffMultiplier float64
	}
	templateData := PHPReadmeData{
		DisplayName:            displayName,
		Description:            description,
		SDKVersion:             sdkVersion,
		SDKName:                strings.ToLower(data.SDKName),
		Namespace:              namespace,
		ClientClassName:        clientClassName,
		BaseURL:                baseURL,
		HTTPLib:                data.HTTPLib,
		HasAuth:                hasAuth,
		AuthExample:            authExample,
		AuthExampleCode:        authExampleCode,
		HasAPI:                 hasAPI,
		APIImports:             apiImports,
		HasOperations:          hasOperations,
		UsageExample:           usageExample,
		RetryEnabled:           data.RetryConfig.Enabled,
		RetryMaxAttempts:       data.RetryConfig.MaxAttempts,
		RetryStrategy:          string(data.RetryConfig.Strategy),
		RetryInitialDelay:      data.RetryConfig.InitialDelay.Seconds(),
		RetryMaxDelay:          data.RetryConfig.MaxDelay.Seconds(),
		RetryBackoffMultiplier: data.RetryConfig.BackoffMultiplier,
	}

	// Load and render template
	tmpl, err := common.LoadTemplate(getPHPReadmeTemplateContent())
	if err != nil {
		return "", fmt.Errorf("failed to load PHP README template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", fmt.Errorf("failed to execute PHP README template: %w", err)
	}

	return buf.String(), nil
}

// generatePHPExamples generates PHP usage examples
func generatePHPExamples(data common.TemplateData) string {
	extractedData, ok := data.OpenAPIDoc.(*common.ExtractedData)
	namespace := fmt.Sprintf("Vendor\\%s", common.ToPascalCase(data.SDKName))
	clientClassName := common.GetClientClassName(data.SDKName)

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
			operationsByTag := common.GroupOperationsByTag(extractedData.Operations)
			for tag := range operationsByTag {
				apiClassName := common.ToPascalCase(tag) + "Api"
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
			case "apiKey":
				examples.WriteString(fmt.Sprintf("    options: [\n        '%s' => 'your-api-key'\n    ]\n", name))
			case "http":
				switch scheme.Scheme {
				case "bearer":
					examples.WriteString("    options: [\n        'bearer_token' => 'your-token'\n    ]\n")
				case "basic":
					examples.WriteString("    options: [\n        'username' => 'your-username',\n        'password' => 'your-password'\n    ]\n")
				}
			case "oauth2", "openIdConnect":
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
		operationsByTag := common.GroupOperationsByTag(extractedData.Operations)
		exampleCount := 0
		for tag, operations := range operationsByTag {
			if exampleCount >= 3 {
				break // Limit to 3 examples
			}
			if len(operations) > 0 {
				apiClassName := common.ToPascalCase(tag) + "Api"
				op := operations[0]
				methodName := common.GetOperationMethodName(op)
				if methodName == "" {
					pathPart := strings.ReplaceAll(strings.Trim(op.Path, "/"), "/", "_")
					methodName = strings.ToLower(op.Method) + common.ToPascalCase(pathPart)
				}
				methodName = common.ToCamelCase(methodName)

				examples.WriteString(fmt.Sprintf("// Example %d: Use %s API\n", exampleCount+2, tag))
				examples.WriteString(fmt.Sprintf("$api = new %s($client);\n", apiClassName))
				examples.WriteString("try {\n")
				examples.WriteString(fmt.Sprintf("    $response = $api->%s(", methodName))

				// Add example parameters
				var params []string
				for _, param := range op.Parameters {
					if param.In == "path" {
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
			className := common.ToPascalCase(name)
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

// generatePHPTests is now implemented in php_testgen.go

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
	// Skip formatting in tests to improve performance
	if os.Getenv("SKIP_FORMATTING") == "true" || os.Getenv("TESTING") == "true" {
		return nil
	}

	// Try PHP-CS-Fixer first
	cmd := exec.Command("php-cs-fixer", "fix", "--quiet", filePath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err == nil {
		return nil
	}

	// If PHP-CS-Fixer is not available, that's okay - just skip formatting
	return fmt.Errorf("no PHP formatter available (php-cs-fixer)")
}
