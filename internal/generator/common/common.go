// Package generator provides common utilities shared across all language generators.
package generator

import (
	"strings"
)

// Common utilities shared by all language generators

// getClientClassName removes common suffixes (sdk, api, client) from SDK name and converts to PascalCase
// This ensures class names like "PetstoreClient" instead of "PetstoreSdkClient"
// Used by all language generators
func getClientClassName(sdkName string) string {
	// Remove common suffixes (case-insensitive)
	name := strings.ToLower(sdkName)
	name = strings.TrimSuffix(name, "_sdk")
	name = strings.TrimSuffix(name, "-sdk")
	name = strings.TrimSuffix(name, "_api")
	name = strings.TrimSuffix(name, "-api")
	name = strings.TrimSuffix(name, "_client")
	name = strings.TrimSuffix(name, "-client")

	// Convert to PascalCase
	return toPascalCase(name)
}

// groupOperationsByTag groups operations by their tags
// Used by all language generators for organizing API methods
func groupOperationsByTag(operations []APIOperation) map[string][]APIOperation {
	tagMap := make(map[string][]APIOperation)
	for _, op := range operations {
		if len(op.Tags) == 0 {
			// If no tags, use "default"
			tagMap["default"] = append(tagMap["default"], op)
		} else {
			// Add to first tag (primary tag)
			tag := op.Tags[0]
			tagMap[tag] = append(tagMap[tag], op)
		}
	}
	return tagMap
}

// determineSDKVersion determines the final SDK version using priority:
// 1. OpenAPI schema version (from extractedData.Version)
// 2. User-provided version (sdkVersion parameter)
// 3. Default version ("1.0.0")
// Used by all language generators
func determineSDKVersion(extractedData *ExtractedData, sdkVersion string) string {
	if extractedData != nil && extractedData.Version != "" {
		return extractedData.Version
	}
	if sdkVersion != "" {
		return sdkVersion
	}
	return "1.0.0"
}
