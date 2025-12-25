package ruby

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vubon/sdk-forge/internal/generator/common"
)

// generateRubyTests generates RSpec test files
func generateRubyTests(packageDir, sdkLibDir string, data common.TemplateData, extractedData *common.ExtractedData, sanitizedName string) error {
	specDir := filepath.Join(packageDir, "spec")
	if err := os.MkdirAll(specDir, 0750); err != nil {
		return fmt.Errorf("failed to create spec directory: %w", err)
	}

	// Generate spec_helper.rb
	specHelperContent := generateRubySpecHelper(sanitizedName)
	specHelperPath := filepath.Join(specDir, "spec_helper.rb")
	// #nosec G306 -- 0644 is appropriate for spec files
	if err := os.WriteFile(specHelperPath, []byte(specHelperContent), 0644); err != nil {
		return fmt.Errorf("failed to write spec_helper.rb: %w", err)
	}

	// Generate client spec
	clientSpecContent := generateRubyClientSpec(data, sanitizedName)
	clientSpecPath := filepath.Join(specDir, "client_spec.rb")
	// #nosec G306 -- 0644 is appropriate for spec files
	if err := os.WriteFile(clientSpecPath, []byte(clientSpecContent), 0644); err != nil {
		return fmt.Errorf("failed to write client_spec.rb: %w", err)
	}

	// Generate model specs
	if extractedData != nil && len(extractedData.Schemas) > 0 {
		modelsSpecDir := filepath.Join(specDir, "models")
		if err := os.MkdirAll(modelsSpecDir, 0750); err != nil {
			return fmt.Errorf("failed to create models spec directory: %w", err)
		}

		for name, schema := range extractedData.Schemas {
			modelSpecContent := generateRubyModelSpec(name, schema, sanitizedName)
			specFileName := common.ToSnakeCase(name) + "_spec.rb"
			specPath := filepath.Join(modelsSpecDir, specFileName)
			// #nosec G306 -- 0644 is appropriate for spec files
			if err := os.WriteFile(specPath, []byte(modelSpecContent), 0644); err != nil {
				return fmt.Errorf("failed to write model spec %s: %w", name, err)
			}
		}
	}

	// Generate API specs
	if extractedData != nil && len(extractedData.Operations) > 0 {
		apiSpecDir := filepath.Join(specDir, "api")
		if err := os.MkdirAll(apiSpecDir, 0750); err != nil {
			return fmt.Errorf("failed to create api spec directory: %w", err)
		}

		operationsByTag := common.GroupOperationsByTag(extractedData.Operations)
		for tag, operations := range operationsByTag {
			apiSpecContent := generateRubyAPISpec(tag, operations, sanitizedName)
			specFileName := common.ToSnakeCase(tag) + "_api_spec.rb"
			specPath := filepath.Join(apiSpecDir, specFileName)
			// #nosec G306 -- 0644 is appropriate for spec files
			if err := os.WriteFile(specPath, []byte(apiSpecContent), 0644); err != nil {
				return fmt.Errorf("failed to write API spec %s: %w", tag, err)
			}
		}
	}

	return nil
}

// generateRubySpecHelper generates spec_helper.rb
func generateRubySpecHelper(sanitizedName string) string {
	var buf strings.Builder

	buf.WriteString("# frozen_string_literal: true\n\n")
	buf.WriteString(fmt.Sprintf("require '%s'\n", sanitizedName))
	buf.WriteString("require 'webmock/rspec'\n\n")
	buf.WriteString("RSpec.configure do |config|\n")
	buf.WriteString("  config.expect_with :rspec do |expectations|\n")
	buf.WriteString("    expectations.include_chain_clauses_in_custom_matcher_descriptions = true\n")
	buf.WriteString("  end\n\n")
	buf.WriteString("  config.mock_with :rspec do |mocks|\n")
	buf.WriteString("    mocks.verify_partial_doubles = true\n")
	buf.WriteString("  end\n\n")
	buf.WriteString("  config.shared_context_metadata_behavior = :apply_to_host_groups\n")
	buf.WriteString("  config.disable_monkey_patching!\n")
	buf.WriteString("  config.warnings = true\n\n")
	buf.WriteString("  config.default_formatter = 'doc' if config.files_to_run.one?\n")
	buf.WriteString("  config.order = :random\n")
	buf.WriteString("  Kernel.srand config.seed\n")
	buf.WriteString("end\n")

	return buf.String()
}

// generateRubyClientSpec generates client_spec.rb
func generateRubyClientSpec(data common.TemplateData, sanitizedName string) string {
	moduleName := common.ToPascalCase(sanitizedName)
	clientClassName := "Client" // Ruby convention: use "Client" as the class name

	var buf strings.Builder

	buf.WriteString("# frozen_string_literal: true\n\n")
	buf.WriteString("require 'spec_helper'\n\n")
	buf.WriteString(fmt.Sprintf("RSpec.describe %s::%s do\n", moduleName, clientClassName))
	buf.WriteString("  let(:base_url) { 'https://api.example.com' }\n")
	buf.WriteString("  let(:client) { described_class.new(base_url: base_url) }\n\n")
	buf.WriteString("  describe '#initialize' do\n")
	buf.WriteString("    it 'initializes with base_url' do\n")
	buf.WriteString("      expect(client.base_url).to eq(base_url)\n")
	buf.WriteString("    end\n\n")
	buf.WriteString("    it 'strips trailing slash from base_url' do\n")
	buf.WriteString("      client_with_slash = described_class.new(base_url: 'https://api.example.com/')\n")
	buf.WriteString("      expect(client_with_slash.base_url).to eq('https://api.example.com')\n")
	buf.WriteString("    end\n")
	buf.WriteString("  end\n\n")

	// Add authentication tests if applicable
	extractedData, ok := data.OpenAPIDoc.(*common.ExtractedData)
	if ok && extractedData != nil && len(extractedData.SecuritySchemes) > 0 {
		buf.WriteString("  describe 'authentication' do\n")
		for _, scheme := range extractedData.SecuritySchemes {
			switch scheme.Type {
			case "apiKey":
				buf.WriteString("    it 'sets API key' do\n")
				buf.WriteString("      client_with_key = described_class.new(base_url: base_url, api_key: 'test-key')\n")
				buf.WriteString("      expect(client_with_key.api_key).to eq('test-key')\n")
				buf.WriteString("    end\n\n")
			case "http":
				if scheme.Scheme == "bearer" {
					buf.WriteString("    it 'sets bearer token' do\n")
					buf.WriteString("      client_with_token = described_class.new(base_url: base_url, bearer_token: 'test-token')\n")
					buf.WriteString("      expect(client_with_token.bearer_token).to eq('test-token')\n")
					buf.WriteString("    end\n\n")
				}
			}
			break // Only first scheme
		}
		buf.WriteString("  end\n\n")
	}

	buf.WriteString("  describe '#request' do\n")
	buf.WriteString("    it 'makes GET request' do\n")
	buf.WriteString("      stub_request(:get, \"#{base_url}/test\")\n")
	buf.WriteString("        .to_return(status: 200, body: '{\"status\":\"ok\"}', headers: { 'Content-Type' => 'application/json' })\n\n")
	buf.WriteString("      response = client.request(:get, '/test')\n")
	buf.WriteString("      expect(response).to eq({ 'status' => 'ok' })\n")
	buf.WriteString("    end\n\n")
	buf.WriteString("    it 'raises ApiException on error' do\n")
	buf.WriteString("      stub_request(:get, \"#{base_url}/error\")\n")
	buf.WriteString("        .to_return(status: 404, body: 'Not Found')\n\n")
	buf.WriteString(fmt.Sprintf("      expect { client.request(:get, '/error') }.to raise_error(%s::Exceptions::ApiException)\n", moduleName))
	buf.WriteString("    end\n")
	buf.WriteString("  end\n")
	buf.WriteString("end\n")

	return buf.String()
}

// generateRubyModelSpec generates model spec
func generateRubyModelSpec(name string, schema *common.Schema, sanitizedName string) string {
	moduleName := common.ToPascalCase(sanitizedName)
	className := common.ToPascalCase(name)

	var buf strings.Builder

	buf.WriteString("# frozen_string_literal: true\n\n")
	buf.WriteString("require 'spec_helper'\n\n")
	buf.WriteString(fmt.Sprintf("RSpec.describe %s::Models::%s do\n", moduleName, className))

	if schema != nil && schema.Type == "object" && len(schema.Properties) > 0 {
		buf.WriteString("  describe '#initialize' do\n")
		buf.WriteString("    it 'creates instance with attributes' do\n")
		buf.WriteString("      instance = described_class.new(\n")

		var params []string
		for propName := range schema.Properties {
			snakeName := common.ToSnakeCase(propName)
			params = append(params, fmt.Sprintf("        %s: 'test'", snakeName))
		}
		buf.WriteString(strings.Join(params, ",\n"))
		buf.WriteString("\n      )\n\n")

		for propName := range schema.Properties {
			snakeName := common.ToSnakeCase(propName)
			buf.WriteString(fmt.Sprintf("      expect(instance.%s).to eq('test')\n", snakeName))
		}
		buf.WriteString("    end\n")
		buf.WriteString("  end\n\n")

		buf.WriteString("  describe '#to_h' do\n")
		buf.WriteString("    it 'converts to hash' do\n")
		buf.WriteString("      instance = described_class.new(\n")
		buf.WriteString(strings.Join(params, ",\n"))
		buf.WriteString("\n      )\n\n")
		buf.WriteString("      hash = instance.to_h\n")
		buf.WriteString("      expect(hash).to be_a(Hash)\n")
		buf.WriteString("    end\n")
		buf.WriteString("  end\n\n")

		buf.WriteString("  describe '.from_hash' do\n")
		buf.WriteString("    it 'creates instance from hash' do\n")
		buf.WriteString("      hash = { ")
		var hashItems []string
		for propName := range schema.Properties {
			hashItems = append(hashItems, fmt.Sprintf("'%s' => 'test'", propName))
		}
		buf.WriteString(strings.Join(hashItems, ", "))
		buf.WriteString(" }\n\n")
		buf.WriteString("      instance = described_class.from_hash(hash)\n")
		buf.WriteString("      expect(instance).to be_a(described_class)\n")
		buf.WriteString("    end\n")
		buf.WriteString("  end\n")
	}

	buf.WriteString("end\n")

	return buf.String()
}

// generateRubyAPISpec generates API spec
func generateRubyAPISpec(tag string, operations []common.APIOperation, sanitizedName string) string {
	moduleName := common.ToPascalCase(sanitizedName)
	apiModuleName := common.ToPascalCase(tag) + "Api"
	clientClassName := "Client"

	var buf strings.Builder

	buf.WriteString("# frozen_string_literal: true\n\n")
	buf.WriteString("require 'spec_helper'\n\n")
	buf.WriteString(fmt.Sprintf("RSpec.describe %s::API::%s do\n", moduleName, apiModuleName))
	buf.WriteString("  let(:base_url) { 'https://api.example.com' }\n")
	buf.WriteString(fmt.Sprintf("  let(:client) { %s::%s.new(base_url: base_url) }\n\n", moduleName, clientClassName))

	// Generate test for first operation
	if len(operations) > 0 {
		op := operations[0]
		methodName := getMethodName(op)

		buf.WriteString(fmt.Sprintf("  describe '.%s' do\n", methodName))
		buf.WriteString("    it 'makes API call' do\n")
		buf.WriteString(fmt.Sprintf("      stub_request(:%s, /#{base_url}/).to_return(status: 200, body: '{}', headers: { 'Content-Type' => 'application/json' })\n\n", strings.ToLower(op.Method)))
		buf.WriteString(fmt.Sprintf("      response = described_class.%s(client", methodName))

		// Add test parameters
		for _, param := range op.Parameters {
			if param.In == "path" {
				buf.WriteString(fmt.Sprintf(", %s: '1'", common.ToSnakeCase(param.Name)))
			}
		}

		buf.WriteString(")\n")
		buf.WriteString("      expect(response).to be_a(Hash)\n")
		buf.WriteString("    end\n")
		buf.WriteString("  end\n")
	}

	buf.WriteString("end\n")

	return buf.String()
}
