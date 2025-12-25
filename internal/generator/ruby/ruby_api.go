package ruby

import (
	"fmt"
	"strings"

	"github.com/vubon/sdk-forge/internal/generator/common"
)

// generateRubyAPIModule generates Ruby API module for a tag
func generateRubyAPIModule(tag string, operations []common.APIOperation, data common.TemplateData, sanitizedSDKName string) string {
	moduleName := common.ToPascalCase(sanitizedSDKName)
	apiModuleName := common.ToPascalCase(tag) + "Api"

	var buf strings.Builder

	// File header
	buf.WriteString("# frozen_string_literal: true\n\n")

	// Module definition
	buf.WriteString(fmt.Sprintf("module %s\n", moduleName))
	buf.WriteString("  module API\n")
	buf.WriteString(fmt.Sprintf("    # %s API methods\n", apiModuleName))
	buf.WriteString(fmt.Sprintf("    module %s\n", apiModuleName))

	// Generate methods for each operation
	for _, op := range operations {
		buf.WriteString(generateRubyAPIMethod(op, data))
		buf.WriteString("\n")
	}

	buf.WriteString("    end\n")
	buf.WriteString("  end\n")
	buf.WriteString("end\n")

	return buf.String()
}

// generateRubyAPIMethod generates a single API method
func generateRubyAPIMethod(op common.APIOperation, _ /* data */ common.TemplateData) string {
	methodName := getMethodName(op)

	var buf strings.Builder

	// Method documentation
	// Method documentation
	if op.Summary != "" {
		for _, line := range strings.Split(op.Summary, "\n") {
			buf.WriteString(fmt.Sprintf("      # %s\n", line))
		}
	}
	if op.Description != "" {
		for _, line := range strings.Split(op.Description, "\n") {
			buf.WriteString(fmt.Sprintf("      # %s\n", line))
		}
	}
	buf.WriteString("      #\n")
	buf.WriteString("      # @param client [Client] The API client instance\n")

	// Document parameters
	for _, param := range op.Parameters {
		if param.In == "path" || param.In == "query" {
			rubyType := "String"
			if param.Schema != nil {
				rubyType = getRubyTypeFromSchema(param.Schema.Type)
			}
			required := ""
			if param.Required {
				required = " (required)"
			}
			buf.WriteString(fmt.Sprintf("      # @param %s [%s]%s %s\n",
				common.ToSnakeCase(param.Name), rubyType, required, param.Description))
		}
	}

	// Request body parameter
	if op.RequestBody != nil {
		buf.WriteString("      # @param body [Hash, nil] Request body\n")
	}

	buf.WriteString("      # @return [Hash] Response body\n")
	buf.WriteString("      # @raise [Exceptions::ApiException] If request fails\n")

	// Method signature
	buf.WriteString(fmt.Sprintf("      def self.%s(client", methodName))

	// Add parameters
	var requiredParams []string
	var optionalParams []string

	for _, param := range op.Parameters {
		switch param.In {
		case "path":
			requiredParams = append(requiredParams, fmt.Sprintf("%s:", common.ToSnakeCase(param.Name)))
		case "query":
			if param.Required {
				requiredParams = append(requiredParams, fmt.Sprintf("%s:", common.ToSnakeCase(param.Name)))
			} else {
				optionalParams = append(optionalParams, fmt.Sprintf("%s: nil", common.ToSnakeCase(param.Name)))
			}
		}
	}

	// Add body parameter if present
	if op.RequestBody != nil {
		optionalParams = append(optionalParams, "body: nil")
	}

	if len(requiredParams) > 0 || len(optionalParams) > 0 {
		buf.WriteString(", ")
		allParams := append(requiredParams, optionalParams...)
		buf.WriteString(strings.Join(allParams, ", "))
	}

	buf.WriteString(")\n")

	// Method body
	// Build path
	path := op.Path
	for _, param := range op.Parameters {
		if param.In == "path" {
			paramName := common.ToSnakeCase(param.Name)
			placeholder := fmt.Sprintf("{%s}", param.Name)
			replacement := fmt.Sprintf(`#{%s}`, paramName)
			path = strings.ReplaceAll(path, placeholder, replacement)
		}
	}
	buf.WriteString(fmt.Sprintf("        path = \"%s\"\n", path))

	// Build query parameters
	hasQueryParams := false
	for _, param := range op.Parameters {
		if param.In == "query" {
			if !hasQueryParams {
				buf.WriteString("        params = {}\n")
				hasQueryParams = true
			}
			paramName := common.ToSnakeCase(param.Name)
			buf.WriteString(fmt.Sprintf("        params[:%s] = %s unless %s.nil?\n", param.Name, paramName, paramName))
		}
	}

	if !hasQueryParams {
		buf.WriteString("        params = {}\n")
	}

	// Make request
	method := strings.ToLower(op.Method)
	buf.WriteString(fmt.Sprintf("\n        client.request(:%s, path, params: params", method))

	if op.RequestBody != nil {
		buf.WriteString(", body: body")
	}

	buf.WriteString(")\n")
	buf.WriteString("      end\n")

	return buf.String()
}

// getMethodName generates a Ruby method name from an operation
func getMethodName(op common.APIOperation) string {
	methodName := common.GetOperationMethodName(op)
	if methodName == "" {
		// Fallback: construct from method and path
		pathPart := strings.ReplaceAll(strings.Trim(op.Path, "/"), "/", "_")
		pathPart = strings.ReplaceAll(pathPart, "{", "")
		pathPart = strings.ReplaceAll(pathPart, "}", "")
		methodName = strings.ToLower(op.Method) + "_" + pathPart
	}
	return common.ToSnakeCase(methodName)
}

// getRubyTypeFromSchema returns Ruby type string from OpenAPI type
func getRubyTypeFromSchema(schemaType string) string {
	switch schemaType {
	case "string":
		return "String"
	case "integer":
		return "Integer"
	case "number":
		return "Float"
	case "boolean":
		return "Boolean"
	case "array":
		return "Array"
	case "object":
		return "Hash"
	default:
		return "Object"
	}
}
