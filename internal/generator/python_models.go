package generator

import (
	"fmt"
	"strings"
)

// generatePythonModels generates Python data models from OpenAPI schemas
func generatePythonModels(schemas map[string]*Schema) string {
	if len(schemas) == 0 {
		return "# No data models defined in OpenAPI schema\n"
	}

	var models strings.Builder
	models.WriteString("# Data Models - Auto-generated from OpenAPI schema\n")
	models.WriteString("from typing import Optional, List, Dict, Any\n")
	models.WriteString("from dataclasses import dataclass, field\n\n")

	for name, schema := range schemas {
		modelCode := generatePythonModel(name, schema, schemas)
		models.WriteString(modelCode)
		models.WriteString("\n")
	}

	return models.String()
}

// generatePythonModel generates a single Python model class
func generatePythonModel(name string, schema *Schema, allSchemas map[string]*Schema) string {
	var code strings.Builder

	// Convert schema name to PascalCase for class name
	className := toPascalCase(name)

	// Generate docstring
	code.WriteString(fmt.Sprintf("@dataclass\nclass %s:\n", className))
	if schema.Description != "" {
		code.WriteString(fmt.Sprintf("    \"\"\"%s\"\"\"\n\n", schema.Description))
	} else {
		code.WriteString(fmt.Sprintf("    \"\"\"%s model\"\"\"\n\n", className))
	}

	// Handle different schema types
	switch schema.Type {
	case "object":
		code.WriteString(generatePythonObjectFields(schema, allSchemas))
	case "array":
		code.WriteString(generatePythonArrayModel(name, schema, allSchemas))
	default:
		code.WriteString(generatePythonPrimitiveModel(name, schema))
	}

	return code.String()
}

// generatePythonObjectFields generates fields for an object schema
func generatePythonObjectFields(schema *Schema, allSchemas map[string]*Schema) string {
	var fields strings.Builder

	if len(schema.Properties) == 0 {
		fields.WriteString("    pass  # No properties defined\n")
		return fields.String()
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

		fieldType := getPythonTypeFromSchema(propSchema, allSchemas)
		isRequired := requiredSet[propName]

		// Generate field with type hint
		if isRequired {
			fields.WriteString(fmt.Sprintf("    %s: %s\n", toSnakeCase(propName), fieldType))
		} else {
			fields.WriteString(fmt.Sprintf("    %s: Optional[%s] = None\n", toSnakeCase(propName), fieldType))
		}

		// Add field description if available
		if propSchema.Description != "" {
			fields.WriteString(fmt.Sprintf("    # %s\n", propSchema.Description))
		}
	}

	return fields.String()
}

// generatePythonArrayModel generates a model for array schemas
func generatePythonArrayModel(_ string, schema *Schema, allSchemas map[string]*Schema) string {
	var code strings.Builder

	itemType := "Any"
	if schema.Items != nil {
		itemType = getPythonTypeFromSchema(schema.Items, allSchemas)
	}

	code.WriteString(fmt.Sprintf("    items: List[%s] = field(default_factory=list)\n", itemType))
	return code.String()
}

// generatePythonPrimitiveModel generates a model for primitive types
func generatePythonPrimitiveModel(_ string, schema *Schema) string {
	var code strings.Builder
	fieldType := getPythonType(schema)
	code.WriteString(fmt.Sprintf("    value: %s\n", fieldType))
	return code.String()
}

// getPythonTypeFromSchema converts a schema to Python type hint, handling refs
func getPythonTypeFromSchema(schema *Schema, _ map[string]*Schema) string {
	if schema == nil {
		return pythonTypeAny
	}

	// Handle $ref
	if schema.Ref != "" {
		// Extract model name from ref (e.g., "#/components/schemas/Pet" -> "Pet")
		refParts := strings.Split(schema.Ref, "/")
		if len(refParts) > 0 {
			refName := refParts[len(refParts)-1]
			return toPascalCase(refName)
		}
	}

	// Handle object type (might be a nested object or reference)
	if schema.Type == pythonTypeObject {
		// If it has properties, it's an inline object - use Dict
		if len(schema.Properties) > 0 {
			return pythonTypeDict
		}
		// Otherwise might be a reference
		return pythonTypeDict
	}

	// Handle array type
	if schema.Type == pythonTypeArray {
		itemType := pythonTypeAny
		if schema.Items != nil {
			itemType = getPythonTypeFromSchema(schema.Items, nil)
		}
		return fmt.Sprintf("List[%s]", itemType)
	}

	// Handle primitive types
	return getPythonType(schema)
}
