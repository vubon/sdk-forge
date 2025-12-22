// Package typescript provides TypeScript test generation functionality.
package typescript

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vubon/sdk-forge/internal/generator/common"
)

// generateTypeScriptTests generates test files for TypeScript SDK
func generateTypeScriptTests(packageDir, srcDir string, data common.TemplateData, extractedData *common.ExtractedData) error {
	// Create tests/ directory
	testsDir := filepath.Join(packageDir, "tests")
	if err := os.MkdirAll(testsDir, 0750); err != nil {
		return fmt.Errorf("failed to create tests directory: %w", err)
	}

	// Generate jest.config.js
	jestConfigContent := generateTypeScriptJestConfig()
	jestConfigPath := filepath.Join(packageDir, "jest.config.js")
	// #nosec G306 -- 0644 is appropriate for config files
	if err := os.WriteFile(jestConfigPath, []byte(jestConfigContent), 0644); err != nil {
		return fmt.Errorf("failed to write jest.config.js: %w", err)
	}

	// Generate tests/client.test.ts
	clientTestContent := generateTypeScriptClientTest(data, extractedData)
	clientTestPath := filepath.Join(testsDir, "client.test.ts")
	// #nosec G306 -- 0644 is appropriate for TypeScript test files
	if err := os.WriteFile(clientTestPath, []byte(clientTestContent), 0644); err != nil {
		return fmt.Errorf("failed to write client.test.ts: %w", err)
	}

	// Generate tests/models.test.ts if schemas exist
	if len(extractedData.Schemas) > 0 {
		modelsTestContent := generateTypeScriptModelsTest(data, extractedData.Schemas)
		modelsTestPath := filepath.Join(testsDir, "models.test.ts")
		// #nosec G306 -- 0644 is appropriate for TypeScript test files
		if err := os.WriteFile(modelsTestPath, []byte(modelsTestContent), 0644); err != nil {
			return fmt.Errorf("failed to write models.test.ts: %w", err)
		}
	}

	// Generate tests/api.test.ts if operations exist
	if len(extractedData.Operations) > 0 {
		apiTestContent := generateTypeScriptAPITest(data, extractedData.Operations, extractedData)
		apiTestPath := filepath.Join(testsDir, "api.test.ts")
		// #nosec G306 -- 0644 is appropriate for TypeScript test files
		if err := os.WriteFile(apiTestPath, []byte(apiTestContent), 0644); err != nil {
			return fmt.Errorf("failed to write api.test.ts: %w", err)
		}
	}

	return nil
}

// generateTypeScriptJestConfig generates jest.config.js
func generateTypeScriptJestConfig() string {
	return `module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  roots: ['<rootDir>/tests'],
  testMatch: ['**/*.test.ts'],
  collectCoverageFrom: [
    'src/**/*.ts',
    '!src/**/*.d.ts',
  ],
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
  },
};
`
}

// generateTypeScriptClientTest generates client.test.ts
func generateTypeScriptClientTest(data common.TemplateData, extractedData *common.ExtractedData) string {
	var buf strings.Builder

	baseURL := extractedData.BaseURL
	if baseURL == "" {
		baseURL = "https://api.example.com/v1"
	}

	buf.WriteString(fmt.Sprintf("import { %s } from '../src/client';\n\n", data.ClientClassName))
	buf.WriteString(fmt.Sprintf("describe('%s', () => {\n", data.ClientClassName))
	buf.WriteString("  it('should be instantiated', () => {\n")
	buf.WriteString(fmt.Sprintf("    const client = new %s({\n", data.ClientClassName))
	buf.WriteString(fmt.Sprintf("      baseUrl: '%s',\n", baseURL))
	buf.WriteString("    });\n")
	buf.WriteString("    expect(client).toBeDefined();\n")
	buf.WriteString("  });\n")
	buf.WriteString("});\n")

	return buf.String()
}

// generateTypeScriptModelsTest generates models.test.ts
func generateTypeScriptModelsTest(data common.TemplateData, schemas map[string]*common.Schema) string {
	var buf strings.Builder

	buf.WriteString("import * as models from '../src/models';\n\n")
	buf.WriteString("describe('Models', () => {\n")

	for name := range schemas {
		modelName := common.ToPascalCase(name)
		buf.WriteString(fmt.Sprintf("  describe('%s', () => {\n", modelName))
		buf.WriteString("    it('should be defined', () => {\n")
		buf.WriteString(fmt.Sprintf("      expect(models.%s).toBeDefined();\n", modelName))
		buf.WriteString("    });\n")
		buf.WriteString("  });\n\n")
	}

	buf.WriteString("});\n")

	return buf.String()
}

// generateTypeScriptAPITest generates api.test.ts
func generateTypeScriptAPITest(data common.TemplateData, operations []common.APIOperation, extractedData *common.ExtractedData) string {
	var buf strings.Builder

	baseURL := extractedData.BaseURL
	if baseURL == "" {
		baseURL = "https://api.example.com/v1"
	}

	buf.WriteString(fmt.Sprintf("import { %s } from '../src/client';\n", data.ClientClassName))
	buf.WriteString("import * as api from '../src/api';\n\n")
	buf.WriteString("describe('API Methods', () => {\n")
	buf.WriteString(fmt.Sprintf("  let client: %s;\n\n", data.ClientClassName))
	buf.WriteString("  beforeEach(() => {\n")
	buf.WriteString(fmt.Sprintf("    client = new %s({\n", data.ClientClassName))
	buf.WriteString(fmt.Sprintf("      baseUrl: '%s',\n", baseURL))
	buf.WriteString("    });\n")
	buf.WriteString("  });\n\n")

	// Group operations by tag
	operationsByTag := common.GroupOperationsByTag(operations)
	for tag, tagOperations := range operationsByTag {
		apiClassName := common.ToPascalCase(tag) + "Api"
		buf.WriteString(fmt.Sprintf("  describe('%s', () => {\n", apiClassName))
		buf.WriteString(fmt.Sprintf("    let %s: api.%s;\n\n", common.ToCamelCase(apiClassName), apiClassName))
		buf.WriteString("    beforeEach(() => {\n")
		buf.WriteString(fmt.Sprintf("      %s = new api.%s(client);\n", common.ToCamelCase(apiClassName), apiClassName))
		buf.WriteString("    });\n\n")

		for _, op := range tagOperations {
			methodName := common.GetOperationMethodName(op)
			if methodName == "" {
				continue
			}
			methodName = common.ToCamelCase(methodName)
			buf.WriteString(fmt.Sprintf("    it('should have %s method', () => {\n", methodName))
			buf.WriteString(fmt.Sprintf("      expect(%s.%s).toBeDefined();\n", common.ToCamelCase(apiClassName), methodName))
			buf.WriteString("    });\n\n")
		}

		buf.WriteString("  });\n\n")
	}

	buf.WriteString("});\n")

	return buf.String()
}
