// Package parser provides OpenAPI schema parsing and validation functionality.
package parser

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// ParseResult contains the parsed OpenAPI document and validation errors
type ParseResult struct {
	Doc      *openapi3.T
	Errors   []ValidationError
	Warnings []ValidationError
}

// ValidationError represents a validation error or warning
type ValidationError struct {
	Severity string // "major" or "minor"
	Message  string
	Path     string // JSON path in schema
}

// ParseOpenAPI parses an OpenAPI schema from a file path or URL
func ParseOpenAPI(schemaPath string) (*openapi3.T, error) {
	var data []byte
	var err error

	// Check if it's a URL
	if strings.HasPrefix(schemaPath, "http://") || strings.HasPrefix(schemaPath, "https://") {
		// #nosec G107 -- URL is user-provided, which is expected for schema fetching
		resp, err := http.Get(schemaPath)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch schema from URL: %w", err)
		}
		defer func() {
			if closeErr := resp.Body.Close(); closeErr != nil {
				// Log but don't fail on close errors
				_ = closeErr
			}
		}()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch schema: HTTP %d", resp.StatusCode)
		}

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read schema from URL: %w", err)
		}
	} else {
		// Local file
		// #nosec G304 -- File path is user-provided, which is expected for schema loading
		data, err = os.ReadFile(schemaPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read schema file: %w", err)
		}
	}

	// Determine format (YAML or JSON)
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	// Try to load as OpenAPI 3.x
	doc, err := loader.LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI schema: %w", err)
	}

	return doc, nil
}

// ValidateOpenAPI validates an OpenAPI document and returns errors categorized by severity
func ValidateOpenAPI(doc *openapi3.T, ignoreMinor bool) (*ParseResult, error) {
	result := &ParseResult{
		Doc:      doc,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	// Validate the document
	loader := openapi3.NewLoader()
	if err := doc.Validate(loader.Context); err != nil {
		// Parse validation errors
		validationErrs := parseValidationErrors(err)
		for _, ve := range validationErrs {
			if ve.Severity == "major" {
				result.Errors = append(result.Errors, ve)
			} else {
				if !ignoreMinor {
					result.Errors = append(result.Errors, ve)
				} else {
					result.Warnings = append(result.Warnings, ve)
				}
			}
		}
	}

	// Additional custom validations
	customErrs := validateCustomRules(doc, ignoreMinor)
	result.Errors = append(result.Errors, customErrs.Errors...)
	result.Warnings = append(result.Warnings, customErrs.Warnings...)

	return result, nil
}

// validateCustomRules performs additional custom validations
func validateCustomRules(doc *openapi3.T, ignoreMinor bool) *ParseResult {
	result := &ParseResult{
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	// Major validations (always block)
	if doc.Info == nil {
		result.Errors = append(result.Errors, ValidationError{
			Severity: "major",
			Message:  "Missing required field 'info'",
			Path:     "/info",
		})
	} else {
		if doc.Info.Title == "" {
			result.Errors = append(result.Errors, ValidationError{
				Severity: "major",
				Message:  "Missing required field 'info.title'",
				Path:     "/info/title",
			})
		}
		if doc.Info.Version == "" {
			result.Errors = append(result.Errors, ValidationError{
				Severity: "major",
				Message:  "Missing required field 'info.version'",
				Path:     "/info/version",
			})
		}
	}

	if doc.Paths == nil || len(doc.Paths.Map()) == 0 {
		result.Errors = append(result.Errors, ValidationError{
			Severity: "major",
			Message:  "Missing required field 'paths' or paths is empty",
			Path:     "/paths",
		})
	}

	// Minor validations (can be ignored)
	if doc.Info != nil && doc.Info.Description == "" {
		if !ignoreMinor {
			result.Errors = append(result.Errors, ValidationError{
				Severity: "minor",
				Message:  "Missing optional field 'info.description'",
				Path:     "/info/description",
			})
		} else {
			result.Warnings = append(result.Warnings, ValidationError{
				Severity: "minor",
				Message:  "Missing optional field 'info.description'",
				Path:     "/info/description",
			})
		}
	}

	return result
}

// parseValidationErrors parses validation errors from kin-openapi
func parseValidationErrors(err error) []ValidationError {
	var errors []ValidationError

	// Simple error parsing - can be enhanced
	errStr := err.Error()
	if strings.Contains(errStr, "required") {
		errors = append(errors, ValidationError{
			Severity: "major",
			Message:  errStr,
			Path:     "",
		})
	} else {
		errors = append(errors, ValidationError{
			Severity: "minor",
			Message:  errStr,
			Path:     "",
		})
	}

	return errors
}

// FormatValidationErrors formats validation errors for display
func FormatValidationErrors(result *ParseResult) string {
	if len(result.Errors) == 0 && len(result.Warnings) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d validation errors", len(result.Errors)))
	if len(result.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf(" (%d major, %d minor)",
			countBySeverity(result.Errors, "major"),
			countBySeverity(result.Errors, "minor")+len(result.Warnings)))
	}
	sb.WriteString("\n\n")

	// Major errors
	majorErrs := filterBySeverity(result.Errors, "major")
	if len(majorErrs) > 0 {
		sb.WriteString("Major Issues:\n")
		for _, err := range majorErrs {
			sb.WriteString(fmt.Sprintf("  ✗ %s", err.Message))
			if err.Path != "" {
				sb.WriteString(fmt.Sprintf(" (at %s)", err.Path))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Minor errors (if not ignored)
	minorErrs := filterBySeverity(result.Errors, "minor")
	if len(minorErrs) > 0 {
		sb.WriteString("Minor Issues:\n")
		for _, err := range minorErrs {
			sb.WriteString(fmt.Sprintf("  ⚠ %s", err.Message))
			if err.Path != "" {
				sb.WriteString(fmt.Sprintf(" (at %s)", err.Path))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Warnings (if minor issues were ignored)
	if len(result.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf("Warnings (ignored): %d minor issues\n", len(result.Warnings)))
	}

	if len(result.Errors) > 0 {
		sb.WriteString("\nPlease fix these issues or use --ignore-minor-issues to ignore minor issues.\n")
	}

	return sb.String()
}

func countBySeverity(errors []ValidationError, severity string) int {
	count := 0
	for _, err := range errors {
		if err.Severity == severity {
			count++
		}
	}
	return count
}

func filterBySeverity(errors []ValidationError, severity string) []ValidationError {
	var filtered []ValidationError
	for _, err := range errors {
		if err.Severity == severity {
			filtered = append(filtered, err)
		}
	}
	return filtered
}
