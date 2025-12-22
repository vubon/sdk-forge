// Package typescript provides TypeScript helper functions for generating configuration files.
package typescript

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/vubon/sdk-forge/internal/generator/common"
	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

// generateTypeScriptPackageJSON generates package.json file
func generateTypeScriptPackageJSON(sdkName, sdkVersion, httpLib string, libConfig *httplib.LibraryConfig, tsVersion common.LanguageVersion) (string, error) {
	// Prepare dependencies
	dependencies := make(map[string]string)
	devDependencies := make(map[string]string)

	// Add HTTP library dependency
	if libConfig.Dependency != "" {
		// Parse dependency (format: "package@version" or "package:version")
		dep := libConfig.Dependency
		if strings.Contains(dep, ":") {
			parts := strings.Split(dep, ":")
			if len(parts) == 2 {
				dependencies[parts[0]] = parts[1]
			} else {
				dependencies[parts[0]] = "latest"
			}
		} else if strings.Contains(dep, "@") {
			parts := strings.Split(dep, "@")
			if len(parts) >= 2 {
				version := strings.Join(parts[1:], "@")
				dependencies[parts[0]] = version
			} else {
				dependencies[dep] = "latest"
			}
		} else {
			dependencies[dep] = "latest"
		}
	}

	// Add dev dependencies
	devDependencies["typescript"] = tsVersion.GetTypeScriptVersionString()
	devDependencies["@types/node"] = "^20.0.0"
	devDependencies["jest"] = "^29.0.0"
	devDependencies["@types/jest"] = "^29.0.0"
	devDependencies["ts-jest"] = "^29.0.0"
	devDependencies["eslint"] = "^8.0.0"
	devDependencies["@typescript-eslint/parser"] = "^6.0.0"
	devDependencies["@typescript-eslint/eslint-plugin"] = "^6.0.0"
	devDependencies["prettier"] = "^3.0.0"

	// Add type definitions for HTTP libraries if needed
	switch httpLib {
	case "axios":
		devDependencies["@types/node"] = "^20.0.0"
	case "node-fetch":
		devDependencies["@types/node-fetch"] = "^2.6.0"
	}

	// Build package.json structure
	packageJSON := map[string]interface{}{
		"name":        sdkName,
		"version":     sdkVersion,
		"type":        "module", // ESM module type for Node.js
		"description": "Auto-generated TypeScript SDK from OpenAPI schema",
		"main":        "dist/index.js",
		"types":       "dist/index.d.ts",
		"scripts": map[string]string{
			"build":   "tsc",
			"test":    "jest",
			"lint":    "eslint src --ext .ts",
			"prepare": "npm run build",
		},
		"keywords": []string{
			"sdk",
			"api",
			"typescript",
			"openapi",
		},
		"author":          "SDK Forge",
		"license":         "MIT",
		"dependencies":    dependencies,
		"devDependencies": devDependencies,
		"files": []string{
			"dist",
			"src",
			"README.md",
		},
	}

	// Add exports for ESM and CommonJS
	exports := map[string]interface{}{
		".": map[string]interface{}{
			"import":  "./dist/index.js",
			"require": "./dist/index.cjs.js",
			"types":   "./dist/index.d.ts",
		},
	}
	packageJSON["exports"] = exports

	// Marshal to JSON with indentation
	jsonBytes, err := json.MarshalIndent(packageJSON, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal package.json: %w", err)
	}

	return string(jsonBytes) + "\n", nil
}

// generateTypeScriptTSConfig generates tsconfig.json file
func generateTypeScriptTSConfig() string {
	return `{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "lib": ["ES2020"],
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true,
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "moduleResolution": "node",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": false,
    "incremental": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist", "**/*.test.ts", "**/*.spec.ts"]
}
`
}

// generateTypeScriptIndex generates src/index.ts (main entry point)
func generateTypeScriptIndex(data common.TemplateData) string {
	var buf strings.Builder

	buf.WriteString("// Auto-generated SDK entry point\n")
	buf.WriteString("// This file exports all public APIs\n\n")

	// Export client
	// Note: Use .js extensions for ESM compatibility (even in .ts files)
	buf.WriteString(fmt.Sprintf("export { %s, type ClientConfig", data.ClientClassName))
	if data.RetryConfig.Enabled {
		buf.WriteString(", type RetryConfig")
	}
	buf.WriteString(" } from './client.js';\n")
	buf.WriteString(fmt.Sprintf("export { %s as default } from './client.js';\n\n", data.ClientClassName))

	// Export exceptions
	buf.WriteString("export { ApiException, NetworkException, TimeoutException } from './exceptions.js';\n\n")

	// Export models if they exist
	extractedData, ok := data.OpenAPIDoc.(*common.ExtractedData)
	if ok && extractedData != nil && len(extractedData.Schemas) > 0 {
		buf.WriteString("// Export all models\n")
		buf.WriteString("export * from './models/index.js';\n\n")
	}

	// Export API modules if they exist
	if ok && extractedData != nil && len(extractedData.Operations) > 0 {
		buf.WriteString("// Export all API modules\n")
		buf.WriteString("export * from './api/index.js';\n\n")
	}

	return buf.String()
}

// generateTypeScriptReadme generates README.md
func generateTypeScriptReadme(data common.TemplateData, sdkVersion string) (string, error) {
	extractedData, ok := data.OpenAPIDoc.(*common.ExtractedData)
	displayName := strings.ReplaceAll(data.SDKName, "-", " ")
	c := cases.Title(language.English)
	displayName = c.String(strings.ToLower(displayName))

	// Get base URL
	baseURL := "https://api.example.com/v1"
	if ok && extractedData != nil && extractedData.BaseURL != "" {
		baseURL = extractedData.BaseURL
	}

	// Get description
	description := "Auto-generated TypeScript SDK from OpenAPI schema"
	if ok && extractedData != nil && extractedData.Description != "" {
		description = extractedData.Description
	}

	// Load and render template
	tmpl, err := common.LoadTemplate(getTypeScriptReadmeTemplateContent())
	if err != nil {
		// Fallback to direct generation
		return generateTypeScriptReadmeFallback(data, sdkVersion, displayName, description, baseURL, extractedData), nil
	}

	// Prepare template data
	type TypeScriptReadmeData struct {
		DisplayName     string
		Description     string
		SDKVersion      string
		SDKName         string
		ClientClassName string
		BaseURL         string
		HTTPLib         string
		HasAuth         bool
		HasOperations   bool
		RetryEnabled    bool
	}
	templateData := TypeScriptReadmeData{
		DisplayName:     displayName,
		Description:     description,
		SDKVersion:      sdkVersion,
		SDKName:         data.SDKName,
		ClientClassName: data.ClientClassName,
		BaseURL:         baseURL,
		HTTPLib:         data.HTTPLib,
		HasAuth:         ok && extractedData != nil && len(extractedData.SecuritySchemes) > 0,
		HasOperations:   ok && extractedData != nil && len(extractedData.Operations) > 0,
		RetryEnabled:    data.RetryConfig.Enabled,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, templateData); err != nil {
		// Fallback to direct generation
		return generateTypeScriptReadmeFallback(data, sdkVersion, displayName, description, baseURL, extractedData), nil
	}

	return buf.String(), nil
}

// generateTypeScriptReadmeFallback generates README directly (fallback)
func generateTypeScriptReadmeFallback(
	data common.TemplateData,
	sdkVersion, displayName, description, baseURL string,
	extractedData *common.ExtractedData,
) string {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("# %s SDK\n\n", displayName))
	buf.WriteString(fmt.Sprintf("%s\n\n", description))
	buf.WriteString(fmt.Sprintf("Version: %s\n\n", sdkVersion))
	buf.WriteString("## Installation\n\n")
	buf.WriteString("```bash\n")
	buf.WriteString(fmt.Sprintf("npm install %s\n", data.SDKName))
	buf.WriteString("```\n\n")
	buf.WriteString("## Usage\n\n")
	buf.WriteString("```typescript\n")
	buf.WriteString(fmt.Sprintf("import { %s } from '%s';\n\n", data.ClientClassName, data.SDKName))
	buf.WriteString(fmt.Sprintf("const client = new %s({\n", data.ClientClassName))
	buf.WriteString(fmt.Sprintf("  baseUrl: '%s',\n", baseURL))
	buf.WriteString("});\n\n")
	buf.WriteString("// Use client methods...\n")
	buf.WriteString("```\n\n")
	buf.WriteString("## HTTP Library\n\n")
	buf.WriteString(fmt.Sprintf("This SDK uses **%s** for HTTP requests.\n\n", data.HTTPLib))
	buf.WriteString("## Authentication\n\n")
	buf.WriteString("Configure authentication when creating the client:\n\n")
	buf.WriteString("```typescript\n")
	buf.WriteString(fmt.Sprintf("const client = new %s({\n", data.ClientClassName))
	buf.WriteString(fmt.Sprintf("  baseUrl: '%s',\n", baseURL))
	if extractedData != nil && len(extractedData.SecuritySchemes) > 0 {
		for name, scheme := range extractedData.SecuritySchemes {
			switch scheme.Type {
			case "apiKey":
				buf.WriteString(fmt.Sprintf("  %s: 'your-api-key',\n", common.ToCamelCase(name)))
			case "http":
				switch scheme.Scheme {
				case "bearer":
					buf.WriteString("  bearerToken: 'your-token',\n")
				case "basic":
					buf.WriteString("  username: 'your-username',\n")
					buf.WriteString("  password: 'your-password',\n")
				}
			}
			break // Just show first one
		}
	}
	buf.WriteString("});\n")
	buf.WriteString("```\n")

	return buf.String()
}

// generateTypeScriptExamples generates examples/basic-usage.ts
func generateTypeScriptExamples(data common.TemplateData) string {
	var buf strings.Builder

	extractedData, ok := data.OpenAPIDoc.(*common.ExtractedData)
	baseURL := "https://api.example.com/v1"
	if ok && extractedData != nil && extractedData.BaseURL != "" {
		baseURL = extractedData.BaseURL
	}

	buf.WriteString("/**\n")
	buf.WriteString(" * Basic usage examples for the SDK\n")
	buf.WriteString(" * Auto-generated from OpenAPI schema\n")
	buf.WriteString(" */\n\n")
	buf.WriteString(fmt.Sprintf("import { %s } from '../src';\n\n", data.ClientClassName))
	buf.WriteString("// Initialize the client\n")
	buf.WriteString(fmt.Sprintf("const client = new %s({\n", data.ClientClassName))
	buf.WriteString(fmt.Sprintf("  baseUrl: '%s',\n", baseURL))

	// Add authentication examples if available
	if ok && extractedData != nil && len(extractedData.SecuritySchemes) > 0 {
		for name, scheme := range extractedData.SecuritySchemes {
			switch scheme.Type {
			case "apiKey":
				buf.WriteString(fmt.Sprintf("  %s: 'your-%s',\n", common.ToCamelCase(name), name))
			case "http":
				switch scheme.Scheme {
				case "bearer":
					buf.WriteString("  bearerToken: 'your-bearer-token',\n")
				case "basic":
					buf.WriteString("  username: 'your-username',\n")
					buf.WriteString("  password: 'your-password',\n")
				}
			}
			break // Just show first one
		}
	}

	buf.WriteString("});\n\n")
	buf.WriteString("// Example: Use API methods\n")
	buf.WriteString("// const response = await client.request({ method: 'GET', url: '/users' });\n")
	buf.WriteString("// console.log(response);\n")

	return buf.String()
}

// generateTypeScriptGitignore generates .gitignore file
func generateTypeScriptGitignore() string {
	return `# Dependencies
node_modules/
package-lock.json
yarn.lock
pnpm-lock.yaml

# Build output
dist/
build/
*.tsbuildinfo

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Logs
*.log
npm-debug.log*
yarn-debug.log*
yarn-error.log*

# Testing
coverage/
.nyc_output/

# Environment
.env
.env.local
.env.*.local
`
}

// generateTypeScriptQualityConfigs generates code quality configuration files
func generateTypeScriptQualityConfigs(packageDir string) error {
	// Generate .eslintrc.json (ESLint configuration)
	eslintContent := generateTypeScriptESLintConfig()
	eslintPath := filepath.Join(packageDir, ".eslintrc.json")
	// #nosec G306 -- 0644 is appropriate for config files
	if err := os.WriteFile(eslintPath, []byte(eslintContent), 0644); err != nil {
		return fmt.Errorf("failed to write .eslintrc.json: %w", err)
	}

	// Generate .prettierrc.json (Prettier configuration)
	prettierContent := generateTypeScriptPrettierConfig()
	prettierPath := filepath.Join(packageDir, ".prettierrc.json")
	// #nosec G306 -- 0644 is appropriate for config files
	if err := os.WriteFile(prettierPath, []byte(prettierContent), 0644); err != nil {
		return fmt.Errorf("failed to write .prettierrc.json: %w", err)
	}

	// Generate .prettierignore (Prettier ignore file)
	prettierIgnoreContent := generateTypeScriptPrettierIgnore()
	prettierIgnorePath := filepath.Join(packageDir, ".prettierignore")
	// #nosec G306 -- 0644 is appropriate for ignore files
	if err := os.WriteFile(prettierIgnorePath, []byte(prettierIgnoreContent), 0644); err != nil {
		return fmt.Errorf("failed to write .prettierignore: %w", err)
	}

	return nil
}

// generateTypeScriptESLintConfig generates .eslintrc.json configuration
func generateTypeScriptESLintConfig() string {
	return `{
  "env": {
    "browser": true,
    "es2021": true,
    "node": true,
    "jest": true
  },
  "extends": [
    "eslint:recommended",
    "plugin:@typescript-eslint/recommended"
  ],
  "parser": "@typescript-eslint/parser",
  "parserOptions": {
    "ecmaVersion": "latest",
    "sourceType": "module",
    "project": "./tsconfig.json"
  },
  "plugins": [
    "@typescript-eslint"
  ],
  "rules": {
    "@typescript-eslint/no-explicit-any": "warn",
    "@typescript-eslint/explicit-function-return-type": "off",
    "@typescript-eslint/explicit-module-boundary-types": "off",
    "@typescript-eslint/no-unused-vars": [
      "warn",
      {
        "argsIgnorePattern": "^_"
      }
    ],
    "no-console": "off"
  },
  "ignorePatterns": [
    "dist",
    "node_modules",
    "*.js",
    "coverage"
  ]
}
`
}

// generateTypeScriptPrettierConfig generates .prettierrc.json configuration
func generateTypeScriptPrettierConfig() string {
	return `{
  "semi": true,
  "trailingComma": "es5",
  "singleQuote": true,
  "printWidth": 100,
  "tabWidth": 2,
  "useTabs": false,
  "arrowParens": "always",
  "endOfLine": "lf"
}
`
}

// generateTypeScriptPrettierIgnore generates .prettierignore file
func generateTypeScriptPrettierIgnore() string {
	return `# Dependencies
node_modules/
package-lock.json
yarn.lock
pnpm-lock.yaml

# Build output
dist/
build/
*.tsbuildinfo

# Generated files
coverage/
.nyc_output/

# Logs
*.log
`
}
