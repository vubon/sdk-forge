// Package ruby provides Ruby SDK generation functionality.
package ruby

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/vubon/sdk-forge/internal/generator/common"
	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

//go:embed templates/gemspec.tmpl
var rubyGemspecTemplate string

//go:embed templates/README.md.tmpl
var rubyReadmeTemplate string

func getRubyGemspecTemplateContent() string {
	return rubyGemspecTemplate
}

func getRubyReadmeTemplateContent() string {
	return rubyReadmeTemplate
}

// GenerateRubySDK generates a Ruby SDK
// If version is nil, uses the default Ruby version
// If sdkVersion is empty, extracts from OpenAPI schema or defaults to "1.0.0"
// retryConfig specifies retry behavior for HTTP requests (can be disabled)
func GenerateRubySDK(
	outputPath, sdkName, httpLib string,
	openAPIDoc interface{},
	version *common.LanguageVersion,
	sdkVersion string,
	generateTests bool,
	retryConfig common.RetryConfig,
) error {
	// Use default version if not provided
	if version == nil {
		defaultVersion := common.GetRubyDefaultVersion()
		version = &defaultVersion
	}

	// Convert openAPIDoc to *openapi3.T
	doc, ok := openAPIDoc.(*openapi3.T)
	if !ok {
		// If not an openapi3.T, try to extract from ExtractedData
		if extractedData, ok := openAPIDoc.(*common.ExtractedData); ok {
			return generateRubySDKFromExtracted(outputPath, sdkName, httpLib, extractedData, *version, sdkVersion, generateTests, retryConfig)
		}
		return fmt.Errorf("invalid OpenAPI document type")
	}

	// Extract data from OpenAPI document
	extractedData, err := common.ExtractOpenAPIData(doc)
	if err != nil {
		return fmt.Errorf("failed to extract OpenAPI data: %w", err)
	}

	return generateRubySDKFromExtracted(outputPath, sdkName, httpLib, extractedData, *version, sdkVersion, generateTests, retryConfig)
}

// generateRubySDKFromExtracted generates SDK from extracted data
func generateRubySDKFromExtracted(
	outputPath, sdkName, httpLib string,
	extractedData *common.ExtractedData,
	version common.LanguageVersion,
	sdkVersion string,
	generateTests bool,
	retryConfig common.RetryConfig,
) error {
	// Get HTTP library config
	libConfig, err := httplib.GetLibraryConfig("ruby", httpLib)
	if err != nil {
		return fmt.Errorf("failed to get HTTP library config: %w", err)
	}

	// Sanitize SDK name for Ruby (snake_case for directory and module)
	sanitizedName := common.ToSnakeCase(sdkName)
	packageDir := filepath.Join(outputPath, sanitizedName)

	// Create package directory
	if err := os.MkdirAll(packageDir, 0750); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Template data
	data := common.TemplateData{
		SDKName:         sanitizedName,
		Language:        "ruby",
		HTTPLib:         httpLib,
		HTTPLibImport:   libConfig.Import,
		HTTPLibConfig:   libConfig,
		OpenAPIDoc:      extractedData,
		ClientClassName: common.GetClientClassName(sdkName),
		RetryConfig:     retryConfig,
	}

	// Determine SDK version using common utility
	finalSDKVersion := common.DetermineSDKVersion(extractedData, sdkVersion)

	// Create lib/ directory
	libDir := filepath.Join(packageDir, "lib")
	if err := os.MkdirAll(libDir, 0750); err != nil {
		return fmt.Errorf("failed to create lib directory: %w", err)
	}

	// Create lib/sdk_name/ directory
	sdkLibDir := filepath.Join(libDir, sanitizedName)
	if err := os.MkdirAll(sdkLibDir, 0750); err != nil {
		return fmt.Errorf("failed to create SDK lib directory: %w", err)
	}

	// Generate gemspec file
	gemspecContent, err := generateRubyGemspec(sdkName, finalSDKVersion, httpLib, version, libConfig, extractedData)
	if err != nil {
		return fmt.Errorf("failed to generate gemspec: %w", err)
	}
	gemspecPath := filepath.Join(packageDir, sanitizedName+".gemspec")
	// #nosec G306 -- 0644 is appropriate for gemspec file
	if err := os.WriteFile(gemspecPath, []byte(gemspecContent), 0644); err != nil {
		return fmt.Errorf("failed to write gemspec: %w", err)
	}

	// Generate Gemfile
	gemfileContent := generateRubyGemfile(sanitizedName)
	gemfilePath := filepath.Join(packageDir, "Gemfile")
	// #nosec G306 -- 0644 is appropriate for Gemfile
	if err := os.WriteFile(gemfilePath, []byte(gemfileContent), 0644); err != nil {
		return fmt.Errorf("failed to write Gemfile: %w", err)
	}

	// Generate main lib file (lib/sdk_name.rb)
	mainLibContent := generateRubyMainLib(sanitizedName, extractedData)
	mainLibPath := filepath.Join(libDir, sanitizedName+".rb")
	// #nosec G306 -- 0644 is appropriate for Ruby source files
	if err := os.WriteFile(mainLibPath, []byte(mainLibContent), 0644); err != nil {
		return fmt.Errorf("failed to write main lib file: %w", err)
	}

	// Generate client class
	clientContent, err := generateRubyClient(data, version)
	if err != nil {
		return fmt.Errorf("failed to generate client: %w", err)
	}
	clientPath := filepath.Join(sdkLibDir, "client.rb")
	// #nosec G306 -- 0644 is appropriate for Ruby source files
	if err := os.WriteFile(clientPath, []byte(clientContent), 0644); err != nil {
		return fmt.Errorf("failed to write client.rb: %w", err)
	}

	// Generate version file
	versionContent := generateRubyVersion(sanitizedName, finalSDKVersion)
	versionPath := filepath.Join(sdkLibDir, "version.rb")
	// #nosec G306 -- 0644 is appropriate for Ruby source files
	if err := os.WriteFile(versionPath, []byte(versionContent), 0644); err != nil {
		return fmt.Errorf("failed to write version.rb: %w", err)
	}

	// Generate models if schemas exist
	if len(extractedData.Schemas) > 0 {
		modelsDir := filepath.Join(sdkLibDir, "models")
		if err := os.MkdirAll(modelsDir, 0750); err != nil {
			return fmt.Errorf("failed to create models directory: %w", err)
		}

		for name, schema := range extractedData.Schemas {
			modelContent := generateRubyModel(name, schema, sanitizedName)
			modelFileName := common.ToSnakeCase(name) + ".rb"
			modelPath := filepath.Join(modelsDir, modelFileName)
			// #nosec G306 -- 0644 is appropriate for Ruby source files
			if err := os.WriteFile(modelPath, []byte(modelContent), 0644); err != nil {
				return fmt.Errorf("failed to write model %s: %w", name, err)
			}
		}
	}

	// Generate API methods if operations exist
	if len(extractedData.Operations) > 0 {
		apiDir := filepath.Join(sdkLibDir, "api")
		if err := os.MkdirAll(apiDir, 0750); err != nil {
			return fmt.Errorf("failed to create api directory: %w", err)
		}

		operationsByTag := common.GroupOperationsByTag(extractedData.Operations)
		for tag, operations := range operationsByTag {
			apiContent := generateRubyAPIModule(tag, operations, data, sanitizedName)
			apiFileName := common.ToSnakeCase(tag) + "_api.rb"
			apiPath := filepath.Join(apiDir, apiFileName)
			// #nosec G306 -- 0644 is appropriate for Ruby source files
			if err := os.WriteFile(apiPath, []byte(apiContent), 0644); err != nil {
				return fmt.Errorf("failed to write API module %s: %w", tag, err)
			}
		}
	}

	// Generate exceptions
	exceptionsDir := filepath.Join(sdkLibDir, "exceptions")
	if err := os.MkdirAll(exceptionsDir, 0750); err != nil {
		return fmt.Errorf("failed to create exceptions directory: %w", err)
	}

	exceptionContent := generateRubyException(sanitizedName)
	exceptionPath := filepath.Join(exceptionsDir, "api_exception.rb")
	// #nosec G306 -- 0644 is appropriate for Ruby source files
	if err := os.WriteFile(exceptionPath, []byte(exceptionContent), 0644); err != nil {
		return fmt.Errorf("failed to write api_exception.rb: %w", err)
	}

	// Generate README.md
	readmeContent, err := generateRubyReadme(data, finalSDKVersion)
	if err != nil {
		return fmt.Errorf("failed to generate README: %w", err)
	}
	readmePath := filepath.Join(packageDir, "README.md")
	// #nosec G306 -- 0644 is appropriate for README file
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	// Generate examples
	examplesDir := filepath.Join(packageDir, "examples")
	if err := os.MkdirAll(examplesDir, 0750); err != nil {
		return fmt.Errorf("failed to create examples directory: %w", err)
	}

	exampleContent := generateRubyExample(data, sanitizedName)
	examplePath := filepath.Join(examplesDir, "basic_usage.rb")
	// #nosec G306 -- 0644 is appropriate for Ruby example files
	if err := os.WriteFile(examplePath, []byte(exampleContent), 0644); err != nil {
		return fmt.Errorf("failed to write example: %w", err)
	}

	// Generate tests if enabled
	if generateTests {
		if err := generateRubyTests(packageDir, sdkLibDir, data, extractedData, sanitizedName); err != nil {
			return fmt.Errorf("failed to generate tests: %w", err)
		}
	}

	// Generate quality configs
	if err := generateRubyQualityConfigs(packageDir); err != nil {
		return fmt.Errorf("failed to generate quality configs: %w", err)
	}

	return nil
}

// generateRubyGemspec generates the gemspec file
func generateRubyGemspec(sdkName, sdkVersion, httpLib string, version common.LanguageVersion, libConfig *httplib.LibraryConfig, extractedData *common.ExtractedData) (string, error) {
	moduleName := common.ToPascalCase(sdkName)
	sanitizedName := common.ToSnakeCase(sdkName)

	description := "Auto-generated Ruby SDK from OpenAPI schema"
	if extractedData != nil && extractedData.Description != "" {
		description = extractedData.Description
	}

	// Escape description for gemspec
	description = strings.ReplaceAll(description, `"`, `\"`)
	description = strings.ReplaceAll(description, "\n", " ")

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("# frozen_string_literal: true\n\n"))
	buf.WriteString(fmt.Sprintf("require_relative 'lib/%s/version'\n\n", sanitizedName))
	buf.WriteString(fmt.Sprintf("Gem::Specification.new do |spec|\n"))
	buf.WriteString(fmt.Sprintf("  spec.name          = %q\n", sanitizedName))
	buf.WriteString(fmt.Sprintf("  spec.version       = %s::VERSION\n", moduleName))
	buf.WriteString(fmt.Sprintf("  spec.authors       = ['Auto-generated']\n"))
	buf.WriteString(fmt.Sprintf("  spec.email         = ['noreply@example.com']\n\n"))
	buf.WriteString(fmt.Sprintf("  spec.summary       = %q\n", description))
	buf.WriteString(fmt.Sprintf("  spec.description   = %q\n", description))
	buf.WriteString(fmt.Sprintf("  spec.homepage      = 'https://github.com/example/%s'\n", sanitizedName))
	buf.WriteString(fmt.Sprintf("  spec.license       = 'MIT'\n\n"))
	buf.WriteString(fmt.Sprintf("  spec.required_ruby_version = '>= %s'\n\n", version.GetRubyVersionString()))
	buf.WriteString(fmt.Sprintf("  spec.files = Dir['lib/**/*.rb', 'README.md', 'LICENSE']\n"))
	buf.WriteString(fmt.Sprintf("  spec.require_paths = ['lib']\n\n"))

	// Add HTTP library dependency
	if libConfig.Dependency != "" {
		parts := strings.Split(libConfig.Dependency, ":")
		if len(parts) >= 2 {
			buf.WriteString(fmt.Sprintf("  spec.add_dependency '%s', '%s'\n", parts[0], parts[1]))
		} else {
			buf.WriteString(fmt.Sprintf("  spec.add_dependency '%s'\n", parts[0]))
		}
	}

	buf.WriteString(fmt.Sprintf("\n  spec.add_development_dependency 'rspec', '~> 3.0'\n"))
	buf.WriteString(fmt.Sprintf("  spec.add_development_dependency 'rubocop', '~> 1.0'\n"))
	buf.WriteString(fmt.Sprintf("  spec.add_development_dependency 'yard', '~> 0.9'\n"))
	buf.WriteString(fmt.Sprintf("end\n"))

	return buf.String(), nil
}

// generateRubyGemfile generates the Gemfile
func generateRubyGemfile(sanitizedName string) string {
	var buf strings.Builder
	buf.WriteString("# frozen_string_literal: true\n\n")
	buf.WriteString("source 'https://rubygems.org'\n\n")
	buf.WriteString("gemspec\n")
	return buf.String()
}

// generateRubyMainLib generates the main lib file
func generateRubyMainLib(sanitizedName string, extractedData *common.ExtractedData) string {
	moduleName := common.ToPascalCase(sanitizedName)

	var buf strings.Builder
	buf.WriteString("# frozen_string_literal: true\n\n")
	buf.WriteString(fmt.Sprintf("require_relative '%s/version'\n", sanitizedName))
	buf.WriteString(fmt.Sprintf("require_relative '%s/client'\n", sanitizedName))
	buf.WriteString(fmt.Sprintf("require_relative '%s/exceptions/api_exception'\n", sanitizedName))

	// Require models
	if extractedData != nil && len(extractedData.Schemas) > 0 {
		buf.WriteString("\n# Models\n")
		for name := range extractedData.Schemas {
			modelFile := common.ToSnakeCase(name)
			buf.WriteString(fmt.Sprintf("require_relative '%s/models/%s'\n", sanitizedName, modelFile))
		}
	}

	// Require API modules
	if extractedData != nil && len(extractedData.Operations) > 0 {
		buf.WriteString("\n# API Modules\n")
		operationsByTag := common.GroupOperationsByTag(extractedData.Operations)
		for tag := range operationsByTag {
			apiFile := common.ToSnakeCase(tag) + "_api"
			buf.WriteString(fmt.Sprintf("require_relative '%s/api/%s'\n", sanitizedName, apiFile))
		}
	}

	buf.WriteString(fmt.Sprintf("\nmodule %s\n", moduleName))
	buf.WriteString("  class Error < StandardError; end\n")
	buf.WriteString("end\n")

	return buf.String()
}

// generateRubyVersion generates the version file
func generateRubyVersion(sanitizedName, sdkVersion string) string {
	moduleName := common.ToPascalCase(sanitizedName)

	var buf strings.Builder
	buf.WriteString("# frozen_string_literal: true\n\n")
	buf.WriteString(fmt.Sprintf("module %s\n", moduleName))
	buf.WriteString(fmt.Sprintf("  VERSION = '%s'\n", sdkVersion))
	buf.WriteString("end\n")

	return buf.String()
}

// generateRubyException generates the API exception class
func generateRubyException(sanitizedName string) string {
	moduleName := common.ToPascalCase(sanitizedName)

	var buf strings.Builder
	buf.WriteString("# frozen_string_literal: true\n\n")
	buf.WriteString(fmt.Sprintf("module %s\n", moduleName))
	buf.WriteString("  module Exceptions\n")
	buf.WriteString("    # API Exception class for handling errors\n")
	buf.WriteString("    class ApiException < StandardError\n")
	buf.WriteString("      attr_reader :status_code, :response_body\n\n")
	buf.WriteString("      def initialize(message, status_code: nil, response_body: nil)\n")
	buf.WriteString("        super(message)\n")
	buf.WriteString("        @status_code = status_code\n")
	buf.WriteString("        @response_body = response_body\n")
	buf.WriteString("      end\n")
	buf.WriteString("    end\n")
	buf.WriteString("  end\n")
	buf.WriteString("end\n")

	return buf.String()
}

// generateRubyQualityConfigs generates code quality configuration files
func generateRubyQualityConfigs(packageDir string) error {
	// Generate .rubocop.yml
	rubocopContent := `# RuboCop configuration for generated SDK

AllCops:
  TargetRubyVersion: 3.0
  NewCops: enable
  Exclude:
    - 'vendor/**/*'
    - 'spec/**/*'

Style/Documentation:
  Enabled: false

Metrics/MethodLength:
  Max: 50

Metrics/ClassLength:
  Max: 300

Layout/LineLength:
  Max: 120
`
	rubocopPath := filepath.Join(packageDir, ".rubocop.yml")
	// #nosec G306 -- 0644 is appropriate for config files
	if err := os.WriteFile(rubocopPath, []byte(rubocopContent), 0644); err != nil {
		return fmt.Errorf("failed to write .rubocop.yml: %w", err)
	}

	// Generate .yardopts
	yardoptsContent := `--markup markdown
--no-private
lib/**/*.rb
-
README.md
CHANGELOG.md
`
	yardoptsPath := filepath.Join(packageDir, ".yardopts")
	// #nosec G306 -- 0644 is appropriate for config files
	if err := os.WriteFile(yardoptsPath, []byte(yardoptsContent), 0644); err != nil {
		return fmt.Errorf("failed to write .yardopts: %w", err)
	}

	return nil
}

// generateRubyReadme generates README.md (placeholder for now)
func generateRubyReadme(data common.TemplateData, sdkVersion string) (string, error) {
	// Placeholder implementation
	sanitizedName := data.SDKName
	moduleName := common.ToPascalCase(sanitizedName)

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("# %s\n\n", moduleName))
	buf.WriteString("Auto-generated Ruby SDK from OpenAPI schema.\n\n")
	buf.WriteString("## Installation\n\n")
	buf.WriteString("Add this line to your application's Gemfile:\n\n")
	buf.WriteString("```ruby\n")
	buf.WriteString(fmt.Sprintf("gem '%s'\n", sanitizedName))
	buf.WriteString("```\n\n")
	buf.WriteString("## Usage\n\n")
	buf.WriteString("```ruby\n")
	buf.WriteString(fmt.Sprintf("require '%s'\n\n", sanitizedName))
	buf.WriteString(fmt.Sprintf("client = %s::Client.new(base_url: 'https://api.example.com')\n", moduleName))
	buf.WriteString("```\n\n")
	buf.WriteString("## Version\n\n")
	buf.WriteString(fmt.Sprintf("Current version: %s\n", sdkVersion))

	return buf.String(), nil
}

// generateRubyExample generates basic usage example (placeholder for now)
func generateRubyExample(data common.TemplateData, sanitizedName string) string {
	moduleName := common.ToPascalCase(sanitizedName)

	var buf strings.Builder
	buf.WriteString("# frozen_string_literal: true\n\n")
	buf.WriteString(fmt.Sprintf("require '%s'\n\n", sanitizedName))
	buf.WriteString("# Initialize the client\n")
	buf.WriteString(fmt.Sprintf("client = %s::Client.new(base_url: 'https://api.example.com')\n\n", moduleName))
	buf.WriteString("# Example API call\n")
	buf.WriteString("# response = client.some_method\n")
	buf.WriteString("# puts response\n")

	return buf.String()
}
