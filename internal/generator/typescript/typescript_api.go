// Package typescript provides TypeScript API method generation functionality.
package typescript

import (
	"fmt"
	"strings"

	"github.com/vubon/sdk-forge/internal/generator/common"
)

// generateTypeScriptAPIIndex generates api/index.ts
func generateTypeScriptAPIIndex(operations []common.APIOperation, data common.TemplateData) string {
	var buf strings.Builder
	buf.WriteString("// Auto-generated API exports\n")
	buf.WriteString("// This file exports all API modules\n\n")

	// Group operations by tags
	operationsByTag := common.GroupOperationsByTag(operations)

	// Export all API modules
	for tag := range operationsByTag {
		moduleName := common.ToKebabCase(tag)
		apiClassName := common.ToPascalCase(tag) + "Api"
		buf.WriteString(fmt.Sprintf("export { %s } from './%s';\n", apiClassName, moduleName))
	}

	return buf.String()
}

// generateTypeScriptAPIModule generates an API module file for a specific tag
func generateTypeScriptAPIModule(tag string, operations []common.APIOperation, data common.TemplateData) string {
	var buf strings.Builder

	apiClassName := common.ToPascalCase(tag) + "Api"
	clientClassName := data.ClientClassName

	// Add JSDoc comment
	buf.WriteString("/**\n")
	buf.WriteString(fmt.Sprintf(" * %s API\n", tag))
	buf.WriteString(" * Auto-generated from OpenAPI schema\n")
	buf.WriteString(" */\n\n")

	// Imports
	buf.WriteString(fmt.Sprintf("import { %s } from '../client';\n", clientClassName))
	buf.WriteString("import { ApiException } from '../exceptions';\n\n")

	// API class
	buf.WriteString(fmt.Sprintf("export class %s {\n", apiClassName))
	buf.WriteString(fmt.Sprintf("  constructor(private client: %s) {}\n\n", clientClassName))

	// Generate methods for each operation
	for _, op := range operations {
		buf.WriteString(generateTypeScriptAPIMethod(op, data))
	}

	buf.WriteString("}\n")

	return buf.String()
}

// generateTypeScriptAPIMethod generates a single API method
func generateTypeScriptAPIMethod(op common.APIOperation, data common.TemplateData) string {
	var buf strings.Builder

	methodName := common.GetOperationMethodName(op)
	if methodName == "" {
		// Fallback naming
		pathPart := strings.ReplaceAll(strings.Trim(op.Path, "/"), "/", "-")
		methodName = strings.ToLower(op.Method) + common.ToPascalCase(pathPart)
	}
	methodName = common.ToCamelCase(methodName)

	// Add JSDoc comment
	buf.WriteString("  /**\n")
	if op.Summary != "" {
		buf.WriteString(fmt.Sprintf("   * %s\n", op.Summary))
	}
	if op.Description != "" {
		descLines := strings.Split(op.Description, "\n")
		for _, line := range descLines {
			line = strings.TrimSpace(line)
			if line != "" {
				buf.WriteString(fmt.Sprintf("   * %s\n", line))
			}
		}
	}

	// Add parameter documentation
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

	if len(pathParams) > 0 || len(queryParams) > 0 || op.RequestBody != nil {
		buf.WriteString("   *\n")
		buf.WriteString("   * @param params - Request parameters\n")
		if op.RequestBody != nil {
			buf.WriteString("   * @param body - Request body\n")
		}
	}

	// Determine return type from responses
	returnType := getTypeScriptReturnType(op.Responses)
	buf.WriteString(fmt.Sprintf("   * @returns Promise<%s>\n", returnType))
	buf.WriteString("   */\n")

	// Method signature
	buf.WriteString(fmt.Sprintf("  async %s(", methodName))

	// Build parameters
	var params []string
	if len(pathParams) > 0 || len(queryParams) > 0 || op.RequestBody != nil {
		// Use a params object for cleaner API
		paramsType := generateTypeScriptMethodParamsType(op, pathParams, queryParams)
		params = append(params, fmt.Sprintf("params: %s", paramsType))
	}

	if len(params) > 0 {
		buf.WriteString(strings.Join(params, ", "))
	}

	buf.WriteString(fmt.Sprintf("): Promise<%s> {\n", returnType))

	// Method body
	buf.WriteString(generateTypeScriptMethodBody(op, pathParams, queryParams, data))

	buf.WriteString("  }\n\n")

	return buf.String()
}

// generateTypeScriptMethodParamsType generates TypeScript type for method parameters
func generateTypeScriptMethodParamsType(op common.APIOperation, pathParams, queryParams []string) string {
	var buf strings.Builder
	buf.WriteString("{\n")

	// Path parameters (required)
	for _, paramName := range pathParams {
		// Find parameter schema
		var paramSchema *common.Schema
		for _, param := range op.Parameters {
			if param.Name == paramName && param.In == "path" {
				paramSchema = param.Schema
				break
			}
		}
		tsType := getTypeScriptType(paramSchema, nil)
		buf.WriteString(fmt.Sprintf("    %s: %s;\n", common.ToCamelCase(paramName), tsType))
	}

	// Query parameters (optional)
	for _, paramName := range queryParams {
		// Find parameter schema
		var paramSchema *common.Schema
		for _, param := range op.Parameters {
			if param.Name == paramName && param.In == "query" {
				paramSchema = param.Schema
				break
			}
		}
		tsType := getTypeScriptType(paramSchema, nil)
		buf.WriteString(fmt.Sprintf("    %s?: %s;\n", common.ToCamelCase(paramName), tsType))
	}

	// Request body (optional)
	if op.RequestBody != nil {
		buf.WriteString("    body?: any;\n")
	}

	buf.WriteString("  }")
	return buf.String()
}

// generateTypeScriptMethodBody generates the method body implementation
func generateTypeScriptMethodBody(op common.APIOperation, pathParams, queryParams []string, data common.TemplateData) string {
	var buf strings.Builder

	// Build path with parameters
	path := op.Path
	for _, param := range pathParams {
		path = strings.ReplaceAll(path, fmt.Sprintf("{%s}", param), fmt.Sprintf("${params.%s}", common.ToCamelCase(param)))
	}

	// Build query parameters object
	queryParamsCode := ""
	if len(queryParams) > 0 {
		buf.WriteString("    const queryParams: Record<string, any> = {};\n")
		for _, param := range queryParams {
			buf.WriteString(fmt.Sprintf("    if (params.%s !== undefined) {\n", common.ToCamelCase(param)))
			buf.WriteString(fmt.Sprintf("      queryParams.%s = params.%s;\n", common.ToCamelCase(param), common.ToCamelCase(param)))
			buf.WriteString("    }\n")
		}
		queryParamsCode = "queryParams"
	}

	// Make request
	buf.WriteString(fmt.Sprintf("    return this.client.request<%s>({\n", getTypeScriptReturnType(op.Responses)))
	buf.WriteString(fmt.Sprintf("      method: '%s',\n", strings.ToUpper(op.Method)))
	buf.WriteString(fmt.Sprintf("      url: `%s`,\n", path))
	if queryParamsCode != "" {
		buf.WriteString(fmt.Sprintf("      params: %s,\n", queryParamsCode))
	}
	if op.RequestBody != nil {
		buf.WriteString("      body: params.body,\n")
	}
	buf.WriteString("    });\n")

	return buf.String()
}

// getTypeScriptReturnType determines the return type from operation responses
func getTypeScriptReturnType(responses map[string]common.Response) string {
	// Look for 200 response first
	if response, ok := responses["200"]; ok {
		if jsonContent, ok := response.Content["application/json"]; ok {
			if jsonContent.Schema != nil {
				return getTypeScriptType(jsonContent.Schema, nil)
			}
		}
	}

	// Look for 201 response
	if response, ok := responses["201"]; ok {
		if jsonContent, ok := response.Content["application/json"]; ok {
			if jsonContent.Schema != nil {
				return getTypeScriptType(jsonContent.Schema, nil)
			}
		}
	}

	// Default to any
	return "any"
}
