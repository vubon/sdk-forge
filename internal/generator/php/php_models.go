// Package php provides PHP model generation functionality.
package php

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vubon/sdk-forge/internal/generator/common"
)

// generatePHPModelWithNamespace generates a PHP model class with namespace
func generatePHPModelWithNamespace(name string, schema *common.Schema, allSchemas map[string]*common.Schema, sdkName string, version common.LanguageVersion) string {
	var code strings.Builder

	// Convert schema name to PascalCase for class name
	className := common.ToPascalCase(name)
	namespace := fmt.Sprintf("Vendor\\%s", common.ToPascalCase(sdkName))

	// Generate class header
	code.WriteString("<?php\n\n")
	code.WriteString(fmt.Sprintf("namespace %s\\Models;\n\n", namespace))
	code.WriteString("use JsonSerializable;\n\n")

	// Generate PHPDoc
	if schema.Description != "" {
		code.WriteString("/**\n")
		descLines := strings.Split(schema.Description, "\n")
		for _, line := range descLines {
			line = strings.TrimSpace(line)
			if line != "" {
				code.WriteString(fmt.Sprintf(" * %s\n", line))
			}
		}
		code.WriteString(" */\n")
	} else {
		code.WriteString(fmt.Sprintf("/**\n * %s model\n */\n", className))
	}

	code.WriteString(fmt.Sprintf("class %s implements JsonSerializable\n", className))
	code.WriteString("{\n")

	// Handle different schema types
	switch schema.Type {
	case "object":
		code.WriteString(generatePHPObjectFields(schema, allSchemas, version))
		code.WriteString(generatePHPObjectConstructor(className, schema, allSchemas, version))
		code.WriteString(generatePHPFromArray(className, schema, allSchemas, version))
		code.WriteString(generatePHPJsonSerialize(schema, allSchemas, version))
	case "array":
		code.WriteString(generatePHPArrayFields(schema, allSchemas, version))
		code.WriteString(generatePHPArrayConstructor(className, schema, allSchemas, version))
		code.WriteString(generatePHPArrayFromArray(className, schema, allSchemas, version))
		code.WriteString(generatePHPArrayJsonSerialize(schema, allSchemas, version))
	default:
		code.WriteString(generatePHPPrimitiveFields(schema, version))
		code.WriteString(generatePHPPrimitiveConstructor(className, schema, version))
		code.WriteString(generatePHPPrimitiveFromArray(className, schema, version))
		code.WriteString(generatePHPPrimitiveJsonSerialize(schema, version))
	}

	code.WriteString("}\n")

	return code.String()
}

// generatePHPObjectFields generates typed properties for an object schema
func generatePHPObjectFields(schema *common.Schema, allSchemas map[string]*common.Schema, version common.LanguageVersion) string {
	var fields strings.Builder

	if len(schema.Properties) == 0 {
		fields.WriteString("    // No properties defined\n")
		return fields.String()
	}

	// Track required fields
	requiredSet := make(map[string]bool)
	for _, req := range schema.Required {
		requiredSet[req] = true
	}

	// Separate required and optional fields for consistent ordering
	type fieldInfo struct {
		name        string
		schema      *common.Schema
		isRequired  bool
		description string
	}

	var requiredFields []fieldInfo
	var optionalFields []fieldInfo

	// Collect and categorize fields
	for propName, propSchema := range schema.Properties {
		if propSchema == nil {
			continue
		}

		isRequired := requiredSet[propName]
		info := fieldInfo{
			name:        propName,
			schema:      propSchema,
			isRequired:  isRequired,
			description: propSchema.Description,
		}

		if isRequired {
			requiredFields = append(requiredFields, info)
		} else {
			optionalFields = append(optionalFields, info)
		}
	}

	// Sort fields by name for consistent output
	sort.Slice(requiredFields, func(i, j int) bool {
		return requiredFields[i].name < requiredFields[j].name
	})
	sort.Slice(optionalFields, func(i, j int) bool {
		return optionalFields[i].name < optionalFields[j].name
	})

	// Generate required fields first
	for _, info := range requiredFields {
		fieldType := getPHPTypeFromSchema(info.schema, allSchemas, version)
		if info.description != "" {
			fields.WriteString(fmt.Sprintf("    /**\n     * %s\n     */\n", info.description))
		}
		fields.WriteString(fmt.Sprintf("    public %s $%s;\n\n", fieldType, common.ToCamelCase(info.name)))
	}

	// Generate optional fields second (nullable)
	for _, info := range optionalFields {
		fieldType := getPHPTypeFromSchema(info.schema, allSchemas, version)
		fieldName := common.ToCamelCase(info.name)
		if info.description != "" {
			fields.WriteString(fmt.Sprintf("    /**\n     * %s\n     */\n", info.description))
		}
		// Make nullable for optional fields
		if !strings.HasPrefix(fieldType, "?") {
			fieldType = "?" + fieldType
		}
		fields.WriteString(fmt.Sprintf("    public %s $%s;\n\n", fieldType, fieldName))
	}

	return fields.String()
}

// generatePHPObjectConstructor generates constructor for object model
func generatePHPObjectConstructor(className string, schema *common.Schema, allSchemas map[string]*common.Schema, version common.LanguageVersion) string {
	var constructor strings.Builder

	requiredSet := make(map[string]bool)
	for _, req := range schema.Required {
		requiredSet[req] = true
	}

	constructor.WriteString("    /**\n")
	constructor.WriteString("     * Create a new instance\n")
	constructor.WriteString("     */\n")
	constructor.WriteString("    public function __construct(\n")

	// Add required parameters first
	var requiredParams []string
	var optionalParams []string

	for propName, propSchema := range schema.Properties {
		if propSchema == nil {
			continue
		}
		fieldType := getPHPTypeFromSchema(propSchema, allSchemas, version)
		fieldName := common.ToCamelCase(propName)
		param := fmt.Sprintf("        %s $%s", fieldType, fieldName)
		if requiredSet[propName] {
			requiredParams = append(requiredParams, param)
		} else {
			// Make nullable for optional
			if !strings.HasPrefix(fieldType, "?") {
				fieldType = "?" + fieldType
			}
			param = fmt.Sprintf("        %s $%s = null", fieldType, fieldName)
			optionalParams = append(optionalParams, param)
		}
	}

	// Sort for consistent output
	sort.Strings(requiredParams)
	sort.Strings(optionalParams)

	allParams := append(requiredParams, optionalParams...)
	constructor.WriteString(strings.Join(allParams, ",\n"))
	constructor.WriteString("\n    ) {\n")

	// Assign properties
	for propName := range schema.Properties {
		fieldName := common.ToCamelCase(propName)
		constructor.WriteString(fmt.Sprintf("        $this->%s = $%s;\n", fieldName, fieldName))
	}

	constructor.WriteString("    }\n\n")

	return constructor.String()
}

// generatePHPFromArray generates fromArray static method for object model
func generatePHPFromArray(className string, schema *common.Schema, allSchemas map[string]*common.Schema, version common.LanguageVersion) string {
	var method strings.Builder

	method.WriteString("    /**\n")
	method.WriteString("     * Create instance from array\n")
	method.WriteString("     *\n")
	method.WriteString("     * @param array<string, mixed> $data\n")
	method.WriteString(fmt.Sprintf("     * @return %s\n", className))
	method.WriteString("     */\n")
	method.WriteString(fmt.Sprintf("    public static function fromArray(array $data): %s\n", className))
	method.WriteString("    {\n")

	requiredSet := make(map[string]bool)
	for _, req := range schema.Required {
		requiredSet[req] = true
	}

	// Build constructor call
	method.WriteString("        return new self(\n")

	var params []string
	for propName, propSchema := range schema.Properties {
		if propSchema == nil {
			continue
		}
		phpType := getPHPTypeFromSchema(propSchema, allSchemas, version)

		// Handle conversion based on type
		phpType = strings.TrimPrefix(phpType, "?")

		paramValue := fmt.Sprintf("$data['%s']", propName)
		if !requiredSet[propName] {
			paramValue = fmt.Sprintf("$data['%s'] ?? null", propName)
		}

		// Handle nested objects and arrays
		if strings.Contains(phpType, "[]") || (propSchema.Type == "object" && propSchema.Ref == "") {
			// Array or inline object - pass as-is
			params = append(params, fmt.Sprintf("            %s", paramValue))
		} else if propSchema.Ref != "" {
			// Reference to another model - need to call fromArray
			refName := extractRefName(propSchema.Ref)
			refClassName := common.ToPascalCase(refName)
			if requiredSet[propName] {
				params = append(params, fmt.Sprintf("            %s::fromArray($data['%s']),", refClassName, propName))
			} else {
				params = append(params, fmt.Sprintf("            isset($data['%s']) ? %s::fromArray($data['%s']) : null,", propName, refClassName, propName))
			}
		} else {
			params = append(params, fmt.Sprintf("            %s", paramValue))
		}
	}

	method.WriteString(strings.Join(params, ",\n"))
	method.WriteString("\n        );\n")
	method.WriteString("    }\n\n")

	return method.String()
}

// generatePHPJsonSerialize generates jsonSerialize method for object model
func generatePHPJsonSerialize(schema *common.Schema, allSchemas map[string]*common.Schema, version common.LanguageVersion) string {
	var method strings.Builder

	method.WriteString("    /**\n")
	method.WriteString("     * Serialize to JSON\n")
	method.WriteString("     *\n")
	method.WriteString("     * @return array<string, mixed>\n")
	method.WriteString("     */\n")
	method.WriteString("    public function jsonSerialize(): array\n")
	method.WriteString("    {\n")
	method.WriteString("        return array_filter([\n")

	for propName := range schema.Properties {
		fieldName := common.ToCamelCase(propName)
		method.WriteString(fmt.Sprintf("            '%s' => $this->%s,\n", propName, fieldName))
	}

	method.WriteString("        ], fn($value) => $value !== null);\n")
	method.WriteString("    }\n\n")

	return method.String()
}

// generatePHPArrayFields generates fields for array schema
func generatePHPArrayFields(schema *common.Schema, allSchemas map[string]*common.Schema, version common.LanguageVersion) string {
	var fields strings.Builder

	fields.WriteString("    /**\n     * Array items\n     */\n")
	fields.WriteString("    public array $items;\n\n")

	return fields.String()
}

// generatePHPArrayConstructor generates constructor for array model
func generatePHPArrayConstructor(className string, schema *common.Schema, allSchemas map[string]*common.Schema, version common.LanguageVersion) string {
	var constructor strings.Builder

	constructor.WriteString("    /**\n")
	constructor.WriteString("     * Create a new instance\n")
	constructor.WriteString("     */\n")
	constructor.WriteString("    public function __construct(array $items = [])\n")
	constructor.WriteString("    {\n")
	constructor.WriteString("        $this->items = $items;\n")
	constructor.WriteString("    }\n\n")

	return constructor.String()
}

// generatePHPArrayFromArray generates fromArray for array model
func generatePHPArrayFromArray(className string, schema *common.Schema, allSchemas map[string]*common.Schema, version common.LanguageVersion) string {
	var method strings.Builder

	method.WriteString("    /**\n")
	method.WriteString("     * Create instance from array\n")
	method.WriteString("     *\n")
	method.WriteString("     * @param array<int, mixed> $data\n")
	method.WriteString(fmt.Sprintf("     * @return %s\n", className))
	method.WriteString("     */\n")
	method.WriteString(fmt.Sprintf("    public static function fromArray(array $data): %s\n", className))
	method.WriteString("    {\n")
	method.WriteString("        return new self($data);\n")
	method.WriteString("    }\n\n")

	return method.String()
}

// generatePHPArrayJsonSerialize generates jsonSerialize for array model
func generatePHPArrayJsonSerialize(schema *common.Schema, allSchemas map[string]*common.Schema, version common.LanguageVersion) string {
	var method strings.Builder

	method.WriteString("    /**\n")
	method.WriteString("     * Serialize to JSON\n")
	method.WriteString("     *\n")
	method.WriteString("     * @return array<int, mixed>\n")
	method.WriteString("     */\n")
	method.WriteString("    public function jsonSerialize(): array\n")
	method.WriteString("    {\n")
	method.WriteString("        return $this->items;\n")
	method.WriteString("    }\n\n")

	return method.String()
}

// generatePHPPrimitiveFields generates fields for primitive schema
func generatePHPPrimitiveFields(schema *common.Schema, version common.LanguageVersion) string {
	var fields strings.Builder

	fieldType := getPHPType(schema, version)
	fields.WriteString("    /**\n     * Value\n     */\n")
	fields.WriteString(fmt.Sprintf("    public %s $value;\n\n", fieldType))

	return fields.String()
}

// generatePHPPrimitiveConstructor generates constructor for primitive model
func generatePHPPrimitiveConstructor(className string, schema *common.Schema, version common.LanguageVersion) string {
	var constructor strings.Builder

	fieldType := getPHPType(schema, version)
	constructor.WriteString("    /**\n")
	constructor.WriteString("     * Create a new instance\n")
	constructor.WriteString("     */\n")
	constructor.WriteString(fmt.Sprintf("    public function __construct(%s $value)\n", fieldType))
	constructor.WriteString("    {\n")
	constructor.WriteString("        $this->value = $value;\n")
	constructor.WriteString("    }\n\n")

	return constructor.String()
}

// generatePHPPrimitiveFromArray generates fromArray for primitive model
func generatePHPPrimitiveFromArray(className string, schema *common.Schema, version common.LanguageVersion) string {
	var method strings.Builder

	method.WriteString("    /**\n")
	method.WriteString("     * Create instance from array\n")
	method.WriteString("     *\n")
	method.WriteString("     * @param mixed $data\n")
	method.WriteString(fmt.Sprintf("     * @return %s\n", className))
	method.WriteString("     */\n")
	method.WriteString(fmt.Sprintf("    public static function fromArray($data): %s\n", className))
	method.WriteString("    {\n")
	method.WriteString("        return new self($data);\n")
	method.WriteString("    }\n\n")

	return method.String()
}

// generatePHPPrimitiveJsonSerialize generates jsonSerialize for primitive model
func generatePHPPrimitiveJsonSerialize(schema *common.Schema, version common.LanguageVersion) string {
	var method strings.Builder

	method.WriteString("    /**\n")
	method.WriteString("     * Serialize to JSON\n")
	method.WriteString("     *\n")
	method.WriteString("     * @return mixed\n")
	method.WriteString("     */\n")
	method.WriteString("    public function jsonSerialize()\n")
	method.WriteString("    {\n")
	method.WriteString("        return $this->value;\n")
	method.WriteString("    }\n\n")

	return method.String()
}

// getPHPTypeFromSchema converts a schema to PHP type, handling refs
func getPHPTypeFromSchema(schema *common.Schema, allSchemas map[string]*common.Schema, version common.LanguageVersion) string {
	if schema == nil {
		return "mixed"
	}

	// Handle $ref
	if schema.Ref != "" {
		refName := extractRefName(schema.Ref)
		return common.ToPascalCase(refName)
	}

	// Handle array type
	if schema.Type == "array" {
		itemType := "mixed"
		if schema.Items != nil {
			itemType = getPHPTypeFromSchema(schema.Items, allSchemas, version)
		}
		return itemType + "[]"
	}

	// Handle object type
	if schema.Type == "object" {
		// If it has properties, it's an inline object - use array
		if len(schema.Properties) > 0 {
			return "array"
		}
		// Otherwise might be a reference or generic object
		return "array"
	}

	// Handle primitive types
	return getPHPType(schema, version)
}

// getPHPType converts a schema to PHP primitive type
func getPHPType(schema *common.Schema, version common.LanguageVersion) string {
	if schema == nil {
		return "mixed"
	}

	switch schema.Type {
	case "string":
		return "string"
	case "integer", "int32", "int64":
		return "int"
	case "number", "float", "double":
		return "float"
	case "boolean":
		return "bool"
	case "array":
		return "array"
	case "object":
		return "array"
	default:
		return "mixed"
	}
}

// extractRefName extracts model name from $ref (e.g., "#/components/schemas/Pet" -> "Pet")
func extractRefName(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ref
}
