// Package generator provides code generation functionality for SDKs.
package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

// generatePHPAPIModule generates PHP API module
func generatePHPAPIModule(tag string, operations []APIOperation, data TemplateData, version LanguageVersion) string {
	// TODO: Implement PHP API module generation
	return fmt.Sprintf("<?php\n\n// PHP API Module %s - TODO: Implement\n", tag)
}

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
	// TODO: Implement PHP README generation
	return fmt.Sprintf("# %s PHP SDK\n\nAuto-generated from OpenAPI schema\n\nVersion: %s\n", data.SDKName, sdkVersion)
}

// generatePHPExamples generates PHP usage examples
func generatePHPExamples(data TemplateData) string {
	// TODO: Implement PHP examples generation
	return "<?php\n\n// PHP Examples - TODO: Implement\n"
}

// generatePHPTests generates PHPUnit tests
func generatePHPTests(packageDir, srcDir string, data TemplateData, extractedData *ExtractedData) error {
	// TODO: Implement PHP test generation
	return nil
}

// generatePHPQualityConfigs generates code quality configuration files
func generatePHPQualityConfigs(packageDir string) error {
	// TODO: Implement PHP quality config generation (phpcs.xml, phpstan.neon, .php-cs-fixer.php)
	return nil
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
