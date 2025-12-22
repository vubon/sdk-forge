// Package php provides PHP API module generation functionality.
package php

import (
	"fmt"
	"strings"

	"github.com/vubon/sdk-forge/internal/generator/common"
)

// generatePHPAPIModule generates PHP API module class organized by tag
func generatePHPAPIModule(tag string, operations []common.APIOperation, data common.TemplateData, version common.LanguageVersion) string {
	if len(operations) == 0 {
		return "<?php\n\n// No API methods defined in OpenAPI schema\n"
	}

	var module strings.Builder
	namespace := fmt.Sprintf("Vendor\\%s", common.ToPascalCase(data.SDKName))
	clientClassName := common.GetClientClassName(data.SDKName)
	apiClassName := common.ToPascalCase(tag) + "Api"

	// Generate class header
	module.WriteString("<?php\n\n")
	module.WriteString(fmt.Sprintf("namespace %s\\Api;\n\n", namespace))
	module.WriteString(fmt.Sprintf("use %s\\%s;\n", namespace, clientClassName))
	module.WriteString("use " + namespace + "\\Exceptions\\ApiException;\n\n")

	// Generate PHPDoc for class
	module.WriteString("/**\n")
	module.WriteString(fmt.Sprintf(" * %s API endpoints\n", tag))
	module.WriteString(" * Auto-generated from OpenAPI schema\n")
	module.WriteString(" */\n")
	module.WriteString(fmt.Sprintf("class %s\n", apiClassName))
	module.WriteString("{\n")
	module.WriteString(fmt.Sprintf("    private %s $client;\n\n", clientClassName))

	// Constructor
	module.WriteString("    /**\n")
	module.WriteString("     * Create a new API instance\n")
	module.WriteString(fmt.Sprintf("     *\n     * @param %s $client\n", clientClassName))
	module.WriteString("     */\n")
	module.WriteString(fmt.Sprintf("    public function __construct(%s $client)\n", clientClassName))
	module.WriteString("    {\n")
	module.WriteString("        $this->client = $client;\n")
	module.WriteString("    }\n\n")

	// Generate methods for each operation
	for _, op := range operations {
		methodCode := generatePHPAPIMethod(op, clientClassName, version)
		module.WriteString(methodCode)
		module.WriteString("\n")
	}

	module.WriteString("}\n")

	return module.String()
}

// generatePHPAPIMethod generates a single PHP API method
func generatePHPAPIMethod(op common.APIOperation, clientClassName string, version common.LanguageVersion) string {
	var method strings.Builder

	// Get method name
	methodName := common.GetOperationMethodName(op)
	if methodName == "" {
		// Fallback naming
		pathPart := strings.ReplaceAll(strings.Trim(op.Path, "/"), "/", "_")
		methodName = strings.ToLower(op.Method) + common.ToPascalCase(pathPart)
	} else {
		// Convert to camelCase for PHP
		methodName = common.ToCamelCase(methodName)
	}

	// Generate PHPDoc
	method.WriteString("    /**\n")
	if op.Summary != "" {
		method.WriteString(fmt.Sprintf("     * %s\n", op.Summary))
		if op.Description != "" {
			method.WriteString("     *\n")
		}
	}
	if op.Description != "" {
		descLines := strings.Split(op.Description, "\n")
		for _, line := range descLines {
			line = strings.TrimSpace(line)
			if line != "" {
				method.WriteString(fmt.Sprintf("     * %s\n", line))
			}
		}
	}

	// Collect parameters for PHPDoc
	var pathParams []string
	var queryParams []string
	for _, param := range op.Parameters {
		switch param.In {
		case "path":
			pathParams = append(pathParams, param.Name)
		case "query":
			queryParams = append(queryParams, param.Name)
		}
	}

	// Add @param tags
	for _, param := range op.Parameters {
		paramType := getPHPType(param.Schema, version)
		if param.In == "query" {
			paramType = "?" + paramType
		}
		desc := param.Description
		if desc == "" {
			desc = param.Name
		}
		method.WriteString(fmt.Sprintf("     * @param %s $%s %s\n", paramType, param.Name, desc))
	}

	if op.RequestBody != nil {
		method.WriteString("     * @param array<string, mixed>|null $body Request body data\n")
	}

	method.WriteString("     * @return array<string, mixed> Response data\n")
	method.WriteString("     * @throws ApiException\n")
	method.WriteString("     */\n")

	// Generate method signature
	method.WriteString(fmt.Sprintf("    public function %s(", methodName))

	// Add path parameters (required)
	for _, param := range op.Parameters {
		if param.In == "path" {
			paramType := getPHPType(param.Schema, version)
			method.WriteString(fmt.Sprintf("%s $%s, ", paramType, param.Name))
		}
	}

	// Add query parameters (optional)
	for _, param := range op.Parameters {
		if param.In == "query" {
			paramType := getPHPType(param.Schema, version)
			method.WriteString(fmt.Sprintf("?%s $%s = null, ", paramType, param.Name))
		}
	}

	// Add request body parameter (optional)
	if op.RequestBody != nil {
		method.WriteString("?array $body = null, ")
	}

	// Remove trailing comma and space
	sig := method.String()
	sig = strings.TrimSuffix(sig, ", ")
	method.Reset()
	method.WriteString(sig)

	method.WriteString("): array\n")
	method.WriteString("    {\n")

	// Build path with parameters
	path := op.Path
	if len(pathParams) > 0 {
		// Replace {param} with PHP string interpolation
		for _, param := range pathParams {
			// Escape any existing $ signs in the path
			path = strings.ReplaceAll(path, "$", "\\$")
			// Replace {param} with {$param}
			path = strings.ReplaceAll(path, fmt.Sprintf("{%s}", param), fmt.Sprintf("{$%s}", param))
		}
		method.WriteString(fmt.Sprintf("        $path = \"%s\";\n", path))
	} else {
		method.WriteString(fmt.Sprintf("        $path = \"%s\";\n", path))
	}

	// Build query parameters array
	if len(queryParams) > 0 {
		method.WriteString("        $queryParams = [];\n")
		for _, param := range queryParams {
			method.WriteString(fmt.Sprintf("        if ($%s !== null) {\n", param))
			method.WriteString(fmt.Sprintf("            $queryParams['%s'] = $%s;\n", param, param))
			method.WriteString("        }\n")
		}
	}

	// Build request options
	method.WriteString("        $options = [];\n")

	// Add query parameters to options
	if len(queryParams) > 0 {
		method.WriteString("        if (!empty($queryParams)) {\n")
		method.WriteString("            $options['query'] = $queryParams;\n")
		method.WriteString("        }\n")
	}

	// Add request body to options
	if op.RequestBody != nil {
		method.WriteString("        if ($body !== null) {\n")
		method.WriteString("            $options['json'] = $body;\n")
		method.WriteString("        }\n")
	}

	// Call client's private request method
	// Note: We need to make request() public or use a public wrapper
	// For now, we'll assume there's a public method or we'll need to adjust
	method.WriteString(fmt.Sprintf("        return $this->client->request('%s', $path, $options);\n", strings.ToUpper(op.Method)))
	method.WriteString("    }\n")

	return method.String()
}
