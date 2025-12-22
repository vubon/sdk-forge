// Package typescript provides TypeScript model generation functionality.
package typescript

import (
	"fmt"
	"strings"

	"github.com/vubon/sdk-forge/internal/generator/common"
)

// generateTypeScriptModelsIndex generates models/index.ts
func generateTypeScriptModelsIndex(schemas map[string]*common.Schema, sdkName string) string {
	var buf strings.Builder
	buf.WriteString("// Auto-generated model exports\n")
	buf.WriteString("// This file exports all model types\n\n")

	// Export all models
	for name := range schemas {
		modelName := common.ToPascalCase(name)
		fileName := common.ToKebabCase(name)
		buf.WriteString(fmt.Sprintf("export * from './%s';\n", fileName))
		buf.WriteString(fmt.Sprintf("export type { %s } from './%s';\n", modelName, fileName))
	}

	return buf.String()
}

// generateTypeScriptModel generates a TypeScript model file
func generateTypeScriptModel(name string, schema *common.Schema, allSchemas map[string]*common.Schema) string {
	var buf strings.Builder

	modelName := common.ToPascalCase(name)

	// Add JSDoc comment
	if schema.Description != "" {
		buf.WriteString("/**\n")
		// Split description by newlines and add proper JSDoc formatting
		descLines := strings.Split(schema.Description, "\n")
		for _, line := range descLines {
			line = strings.TrimSpace(line)
			if line != "" {
				buf.WriteString(fmt.Sprintf(" * %s\n", line))
			}
		}
		buf.WriteString(" */\n")
	}

	// Generate interface or type based on schema type
	switch schema.Type {
	case "object":
		buf.WriteString(fmt.Sprintf("export interface %s {\n", modelName))
		buf.WriteString(generateTypeScriptObjectFields(schema, allSchemas))
		buf.WriteString("}\n")
	case "array":
		if schema.Items != nil {
			itemType := getTypeScriptType(schema.Items, allSchemas)
			buf.WriteString(fmt.Sprintf("export type %s = %s[];\n", modelName, itemType))
		} else {
			buf.WriteString(fmt.Sprintf("export type %s = any[];\n", modelName))
		}
	default:
		// Primitive type
		tsType := getTypeScriptType(schema, allSchemas)
		buf.WriteString(fmt.Sprintf("export type %s = %s;\n", modelName, tsType))
	}

	return buf.String()
}

// generateTypeScriptObjectFields generates fields for an object interface
func generateTypeScriptObjectFields(schema *common.Schema, allSchemas map[string]*common.Schema) string {
	var buf strings.Builder

	if len(schema.Properties) == 0 {
		buf.WriteString("  [key: string]: any;\n")
		return buf.String()
	}

	// Track required fields
	requiredSet := make(map[string]bool)
	for _, req := range schema.Required {
		requiredSet[req] = true
	}

	// Generate fields
	for propName, propSchema := range schema.Properties {
		if propSchema == nil {
			continue
		}

		// Add JSDoc comment if description exists
		if propSchema.Description != "" {
			buf.WriteString("  /**\n")
			descLines := strings.Split(propSchema.Description, "\n")
			for _, line := range descLines {
				line = strings.TrimSpace(line)
				if line != "" {
					buf.WriteString(fmt.Sprintf("   * %s\n", line))
				}
			}
			buf.WriteString("   */\n")
		}

		// Field name (camelCase for TypeScript)
		fieldName := common.ToCamelCase(propName)
		tsType := getTypeScriptType(propSchema, allSchemas)

		// Optional field marker
		optional := ""
		if !requiredSet[propName] {
			optional = "?"
		}

		buf.WriteString(fmt.Sprintf("  %s%s: %s;\n", fieldName, optional, tsType))
	}

	return buf.String()
}

// getTypeScriptType converts OpenAPI schema type to TypeScript type
func getTypeScriptType(schema *common.Schema, allSchemas map[string]*common.Schema) string {
	if schema == nil {
		return "any"
	}

	// Handle $ref
	if schema.Ref != "" {
		// Extract model name from ref (e.g., "#/components/schemas/Pet" -> "Pet")
		refParts := strings.Split(schema.Ref, "/")
		if len(refParts) > 0 {
			refName := refParts[len(refParts)-1]
			return common.ToPascalCase(refName)
		}
		return "any"
	}

	// Note: Enum support would require adding Enum field to common.Schema
	// For now, enums are treated as strings

	// Handle array
	if schema.Type == "array" {
		if schema.Items != nil {
			itemType := getTypeScriptType(schema.Items, allSchemas)
			return fmt.Sprintf("%s[]", itemType)
		}
		return "any[]"
	}

	// Handle object
	if schema.Type == "object" {
		// Note: AdditionalProperties support would require adding field to common.Schema
		// For now, objects are treated as Record<string, any>
		return "Record<string, any>"
	}

	// Handle primitive types
	switch schema.Type {
	case "string":
		// Handle string formats
		switch schema.Format {
		case "date":
			return "string" // ISO date string
		case "date-time":
			return "string" // ISO datetime string
		case "email":
			return "string"
		case "uri":
			return "string"
		default:
			return "string"
		}
	case "integer", "int32", "int64":
		return "number"
	case "number", "float", "double":
		return "number"
	case "boolean":
		return "boolean"
	case "null":
		return "null"
	default:
		return "any"
	}
}
