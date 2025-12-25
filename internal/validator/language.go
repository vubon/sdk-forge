// Package validator provides validation and normalization functions for SDK generation.
package validator

import (
	"fmt"
	"strings"
)

// NormalizeLanguage normalizes language input (handles aliases)
func NormalizeLanguage(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))

	// Handle aliases
	aliases := map[string]string{
		"py":         "python",
		"golang":     "go",
		"js":         "javascript",
		"ts":         "typescript",
		"javascript": "javascript",
		"typescript": "typescript",
		"rb":         "ruby",
	}

	if normalized, exists := aliases[lang]; exists {
		return normalized
	}

	return lang
}

// ValidateLanguage checks if the language is supported
func ValidateLanguage(language string) error {
	normalized := NormalizeLanguage(language)

	// "all" is a special case that means generate for all languages
	if normalized == "all" {
		return nil
	}

	supported := []string{"python", "go", "php", "javascript", "typescript", "ruby"}
	for _, lang := range supported {
		if normalized == lang {
			return nil
		}
	}

	return fmt.Errorf("unsupported language: %s. Supported languages: %v, or 'all'", language, supported)
}

// GetImplementedLanguages returns a list of languages that are currently implemented
func GetImplementedLanguages() []string {
	return []string{"python", "go", "php", "typescript", "javascript", "ruby"}
}

// ValidateSDKName validates and sanitizes SDK name based on language
func ValidateSDKName(name, language string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("SDK name cannot be empty")
	}

	// Normalize language for future use (more specific validation per language will be done during generation)
	_ = NormalizeLanguage(language)

	// Basic validation - alphanumeric, hyphens, underscores
	for _, c := range name {
		valid := (c == '-') || (c == '_') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !valid {
			return "", fmt.Errorf("SDK name contains invalid character: %q", c)
		}
	}

	return name, nil
}
