package ruby

import (
	"fmt"
	"strings"

	"github.com/vubon/sdk-forge/internal/generator/common"
)

// generateRubyModel generates a Ruby model class from an OpenAPI schema
func generateRubyModel(name string, schema *common.Schema, sanitizedSDKName string) string {
	if schema == nil {
		return ""
	}

	moduleName := common.ToPascalCase(sanitizedSDKName)
	className := common.ToPascalCase(name)

	var buf strings.Builder

	// File header
	buf.WriteString("# frozen_string_literal: true\n\n")
	buf.WriteString("require 'json'\n\n")

	// Module and class definition
	buf.WriteString(fmt.Sprintf("module %s\n", moduleName))
	buf.WriteString("  module Models\n")
	buf.WriteString(fmt.Sprintf("    # %s model\n", className))

	// Add description if available
	// Add description if available
	if schema.Description != "" {
		for _, line := range strings.Split(schema.Description, "\n") {
			buf.WriteString(fmt.Sprintf("    # %s\n", line))
		}
	}

	buf.WriteString(fmt.Sprintf("    class %s\n", className))

	// Attributes
	if schema.Type == "object" && len(schema.Properties) > 0 {
		buf.WriteString("      attr_accessor ")

		var attrs []string
		for propName := range schema.Properties {
			attrs = append(attrs, fmt.Sprintf(":%s", common.ToSnakeCase(propName)))
		}
		buf.WriteString(strings.Join(attrs, ", "))
		buf.WriteString("\n\n")

		// Initialize method
		buf.WriteString("      # Initialize a new instance\n")
		buf.WriteString("      #\n")
		for propName, propSchema := range schema.Properties {
			snakeName := common.ToSnakeCase(propName)
			rubyType := getRubyType(propSchema)
			isRequired := contains(schema.Required, propName)

			if isRequired {
				buf.WriteString(fmt.Sprintf("      # @param %s [%s] (required)\n", snakeName, rubyType))
			} else {
				buf.WriteString(fmt.Sprintf("      # @param %s [%s, nil]\n", snakeName, rubyType))
			}
		}
		buf.WriteString("      def initialize(")

		// Constructor parameters
		var params []string
		var requiredParams []string
		var optionalParams []string

		for propName := range schema.Properties {
			snakeName := common.ToSnakeCase(propName)
			isRequired := contains(schema.Required, propName)

			if isRequired {
				requiredParams = append(requiredParams, fmt.Sprintf("%s:", snakeName))
			} else {
				optionalParams = append(optionalParams, fmt.Sprintf("%s: nil", snakeName))
			}
		}

		params = append(params, requiredParams...)
		params = append(params, optionalParams...)
		buf.WriteString(strings.Join(params, ", "))
		buf.WriteString(")\n")

		// Assign attributes
		for propName := range schema.Properties {
			snakeName := common.ToSnakeCase(propName)
			buf.WriteString(fmt.Sprintf("        @%s = %s\n", snakeName, snakeName))
		}

		buf.WriteString("      end\n\n")

		// to_h method
		buf.WriteString("      # Convert model to hash\n")
		buf.WriteString("      # @return [Hash]\n")
		buf.WriteString("      def to_h\n")
		buf.WriteString("        {\n")
		for propName := range schema.Properties {
			snakeName := common.ToSnakeCase(propName)
			buf.WriteString(fmt.Sprintf("          %s: @%s,\n", propName, snakeName))
		}
		buf.WriteString("        }.compact\n")
		buf.WriteString("      end\n\n")

		// to_json method
		buf.WriteString("      # Convert model to JSON\n")
		buf.WriteString("      # @return [String]\n")
		buf.WriteString("      def to_json(*_args)\n")
		buf.WriteString("        to_h.to_json\n")
		buf.WriteString("      end\n\n")

		// from_hash class method
		buf.WriteString("      # Create instance from hash\n")
		buf.WriteString("      # @param hash [Hash]\n")
		buf.WriteString(fmt.Sprintf("      # @return [%s]\n", className))
		buf.WriteString("      def self.from_hash(hash)\n")
		buf.WriteString("        return nil if hash.nil?\n\n")
		buf.WriteString("        new(\n")

		var fromHashParams []string
		for propName := range schema.Properties {
			snakeName := common.ToSnakeCase(propName)
			fromHashParams = append(fromHashParams, fmt.Sprintf("          %s: hash['%s'] || hash[:%s]", snakeName, propName, snakeName))
		}
		buf.WriteString(strings.Join(fromHashParams, ",\n"))
		buf.WriteString("\n        )\n")
		buf.WriteString("      end\n\n")

		// from_json class method
		buf.WriteString("      # Create instance from JSON string\n")
		buf.WriteString("      # @param json [String]\n")
		buf.WriteString(fmt.Sprintf("      # @return [%s]\n", className))
		buf.WriteString("      def self.from_json(json)\n")
		buf.WriteString("        from_hash(JSON.parse(json))\n")
		buf.WriteString("      end\n")
	} else {
		// Simple type or no properties
		buf.WriteString("      attr_accessor :value\n\n")
		buf.WriteString("      def initialize(value: nil)\n")
		buf.WriteString("        @value = value\n")
		buf.WriteString("      end\n\n")
		buf.WriteString("      def to_json(*_args)\n")
		buf.WriteString("        @value.to_json\n")
		buf.WriteString("      end\n")
	}

	buf.WriteString("    end\n")
	buf.WriteString("  end\n")
	buf.WriteString("end\n")

	return buf.String()
}

// getRubyType converts OpenAPI type to Ruby type string
func getRubyType(schema *common.Schema) string {
	if schema == nil {
		return "Object"
	}

	switch schema.Type {
	case "string":
		return "String"
	case "integer":
		return "Integer"
	case "number":
		return "Float"
	case "boolean":
		return "Boolean"
	case "array":
		if schema.Items != nil {
			itemType := getRubyType(schema.Items)
			return fmt.Sprintf("Array<%s>", itemType)
		}
		return "Array"
	case "object":
		return "Hash"
	default:
		return "Object"
	}
}

// contains checks if a string slice contains a value
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
