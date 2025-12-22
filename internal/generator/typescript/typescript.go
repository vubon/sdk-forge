// Package typescript provides TypeScript/JavaScript SDK generation functionality.
package typescript

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/vubon/sdk-forge/internal/generator/common"
	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

//go:embed templates/README.md.tmpl
var typescriptReadmeTemplate string

//go:embed templates/client.ts.tmpl
var typescriptClientTemplate string

func getTypeScriptReadmeTemplateContent() string {
	return typescriptReadmeTemplate
}

func getTypeScriptClientTemplateContent() string {
	return typescriptClientTemplate
}

// GenerateTypeScriptSDK generates a TypeScript SDK
// If sdkVersion is empty, extracts from OpenAPI schema or defaults to "1.0.0"
// retryConfig specifies retry behavior for HTTP requests (can be disabled)
func GenerateTypeScriptSDK(
	outputPath, sdkName, httpLib string,
	openAPIDoc interface{},
	version *common.LanguageVersion, // Reserved for future TypeScript version support
	sdkVersion string,
	generateTests bool,
	retryConfig common.RetryConfig,
) error {
	// Convert openAPIDoc to *openapi3.T
	doc, ok := openAPIDoc.(*openapi3.T)
	if !ok {
		// If not an openapi3.T, try to extract from ExtractedData
		if extractedData, ok := openAPIDoc.(*common.ExtractedData); ok {
			return generateTypeScriptSDKFromExtracted(outputPath, sdkName, httpLib, extractedData, sdkVersion, generateTests, retryConfig)
		}
		return fmt.Errorf("invalid OpenAPI document type")
	}

	// Extract data from OpenAPI document
	extractedData, err := common.ExtractOpenAPIData(doc)
	if err != nil {
		return fmt.Errorf("failed to extract OpenAPI data: %w", err)
	}

	return generateTypeScriptSDKFromExtracted(outputPath, sdkName, httpLib, extractedData, sdkVersion, generateTests, retryConfig)
}

// generateTypeScriptSDKFromExtracted generates SDK from extracted data
func generateTypeScriptSDKFromExtracted(
	outputPath, sdkName, httpLib string,
	extractedData *common.ExtractedData,
	sdkVersion string,
	generateTests bool,
	retryConfig common.RetryConfig,
) error {
	// Get HTTP library config
	libConfig, err := httplib.GetLibraryConfig("typescript", httpLib)
	if err != nil {
		return fmt.Errorf("failed to get HTTP library config: %w", err)
	}

	// Sanitize SDK name for npm (kebab-case)
	sanitizedName := common.ToKebabCase(sdkName)
	packageDir := filepath.Join(outputPath, sanitizedName)

	// Create package directory
	if err := os.MkdirAll(packageDir, 0750); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Template data
	data := common.TemplateData{
		SDKName:         sanitizedName,
		Language:        "typescript",
		HTTPLib:         httpLib,
		HTTPLibImport:   libConfig.Import,
		HTTPLibConfig:   libConfig,
		OpenAPIDoc:      extractedData,
		ClientClassName: common.GetClientClassName(sanitizedName),
		RetryConfig:     retryConfig,
	}

	// Determine SDK version using common utility
	finalSDKVersion := common.DetermineSDKVersion(extractedData, sdkVersion)

	// Generate package.json
	packageContent, err := generateTypeScriptPackageJSON(sanitizedName, finalSDKVersion, httpLib, libConfig)
	if err != nil {
		return fmt.Errorf("failed to generate package.json: %w", err)
	}
	packagePath := filepath.Join(packageDir, "package.json")
	// #nosec G306 -- 0644 is appropriate for package.json file
	if err := os.WriteFile(packagePath, []byte(packageContent), 0644); err != nil {
		return fmt.Errorf("failed to write package.json: %w", err)
	}

	// Generate tsconfig.json
	tsconfigContent := generateTypeScriptTSConfig()
	tsconfigPath := filepath.Join(packageDir, "tsconfig.json")
	// #nosec G306 -- 0644 is appropriate for tsconfig.json file
	if err := os.WriteFile(tsconfigPath, []byte(tsconfigContent), 0644); err != nil {
		return fmt.Errorf("failed to write tsconfig.json: %w", err)
	}

	// Create src/ directory
	srcDir := filepath.Join(packageDir, "src")
	if err := os.MkdirAll(srcDir, 0750); err != nil {
		return fmt.Errorf("failed to create src directory: %w", err)
	}

	// Generate src/index.ts (main entry point)
	indexContent := generateTypeScriptIndex(data)
	indexPath := filepath.Join(srcDir, "index.ts")
	// #nosec G306 -- 0644 is appropriate for TypeScript source files
	if err := os.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
		return fmt.Errorf("failed to write index.ts: %w", err)
	}

	// Generate src/client.ts
	clientContent, err := generateTypeScriptClient(data)
	if err != nil {
		return fmt.Errorf("failed to generate client: %w", err)
	}
	clientPath := filepath.Join(srcDir, "client.ts")
	// #nosec G306 -- 0644 is appropriate for TypeScript source files
	if err := os.WriteFile(clientPath, []byte(clientContent), 0644); err != nil {
		return fmt.Errorf("failed to write client.ts: %w", err)
	}

	// Generate src/exceptions.ts
	exceptionsContent := generateTypeScriptExceptions(data)
	exceptionsPath := filepath.Join(srcDir, "exceptions.ts")
	// #nosec G306 -- 0644 is appropriate for TypeScript source files
	if err := os.WriteFile(exceptionsPath, []byte(exceptionsContent), 0644); err != nil {
		return fmt.Errorf("failed to write exceptions.ts: %w", err)
	}

	// Generate src/models/ directory if schemas exist
	if len(extractedData.Schemas) > 0 {
		modelsDir := filepath.Join(srcDir, "models")
		if err := os.MkdirAll(modelsDir, 0750); err != nil {
			return fmt.Errorf("failed to create models directory: %w", err)
		}

		// Generate models/index.ts
		modelsIndexContent := generateTypeScriptModelsIndex(extractedData.Schemas, data.SDKName)
		modelsIndexPath := filepath.Join(modelsDir, "index.ts")
		// #nosec G306 -- 0644 is appropriate for TypeScript source files
		if err := os.WriteFile(modelsIndexPath, []byte(modelsIndexContent), 0644); err != nil {
			return fmt.Errorf("failed to write models/index.ts: %w", err)
		}

		// Generate individual model files
		for name, schema := range extractedData.Schemas {
			modelContent := generateTypeScriptModel(name, schema, extractedData.Schemas)
			modelFileName := fmt.Sprintf("%s.ts", common.ToKebabCase(name))
			modelPath := filepath.Join(modelsDir, modelFileName)
			// #nosec G306 -- 0644 is appropriate for TypeScript source files
			if err := os.WriteFile(modelPath, []byte(modelContent), 0644); err != nil {
				return fmt.Errorf("failed to write model %s: %w", name, err)
			}
		}
	}

	// Generate src/api/ directory if operations exist
	if len(extractedData.Operations) > 0 {
		apiDir := filepath.Join(srcDir, "api")
		if err := os.MkdirAll(apiDir, 0750); err != nil {
			return fmt.Errorf("failed to create api directory: %w", err)
		}

		// Generate api/index.ts
		apiIndexContent := generateTypeScriptAPIIndex(extractedData.Operations, data)
		apiIndexPath := filepath.Join(apiDir, "index.ts")
		// #nosec G306 -- 0644 is appropriate for TypeScript source files
		if err := os.WriteFile(apiIndexPath, []byte(apiIndexContent), 0644); err != nil {
			return fmt.Errorf("failed to write api/index.ts: %w", err)
		}

		// Group operations by tags
		operationsByTag := common.GroupOperationsByTag(extractedData.Operations)
		for tag, operations := range operationsByTag {
			apiContent := generateTypeScriptAPIModule(tag, operations, data)
			apiFileName := fmt.Sprintf("%s.ts", common.ToKebabCase(tag))
			apiPath := filepath.Join(apiDir, apiFileName)
			// #nosec G306 -- 0644 is appropriate for TypeScript source files
			if err := os.WriteFile(apiPath, []byte(apiContent), 0644); err != nil {
				return fmt.Errorf("failed to write API module %s: %w", tag, err)
			}
		}
	}

	// Generate README.md
	readmeContent, err := generateTypeScriptReadme(data, finalSDKVersion)
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

	// Generate examples/basic-usage.ts
	examplesContent := generateTypeScriptExamples(data)
	examplesPath := filepath.Join(examplesDir, "basic-usage.ts")
	// #nosec G306 -- 0644 is appropriate for TypeScript example files
	if err := os.WriteFile(examplesPath, []byte(examplesContent), 0644); err != nil {
		return fmt.Errorf("failed to write examples: %w", err)
	}

	// Generate .gitignore
	gitignoreContent := generateTypeScriptGitignore()
	gitignorePath := filepath.Join(packageDir, ".gitignore")
	// #nosec G306 -- 0644 is appropriate for .gitignore file
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	// Generate tests/ directory if test generation is enabled
	if generateTests {
		if err := generateTypeScriptTests(packageDir, srcDir, data, extractedData); err != nil {
			return fmt.Errorf("failed to generate tests: %w", err)
		}
	}

	return nil
}
