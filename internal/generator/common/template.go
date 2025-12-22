// Package common provides template rendering and string transformation utilities.
package common

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

// TemplateData holds data for template rendering
type TemplateData struct {
	SDKName         string
	Language        string
	HTTPLib         string
	HTTPLibImport   string
	HTTPLibConfig   *httplib.LibraryConfig
	OpenAPIDoc      interface{} // Will be properly typed later
	ClientClassName string      // Client class name without "Sdk" suffix
	RetryConfig     RetryConfig // Retry configuration for HTTP requests
}

// FuncMap returns custom template functions
func FuncMap() template.FuncMap {
	return template.FuncMap{
		"camelCase":  ToCamelCase,
		"snakeCase":  ToSnakeCase,
		"pascalCase": ToPascalCase,
		"kebabCase":  ToKebabCase,
		"lower":      strings.ToLower,
		"upper":      strings.ToUpper,
		"title":      toTitleCase,
		"capitalize": capitalize,
		"plural":     pluralize,
		"singular":   singularize,
		"trim":       strings.TrimSpace,
		"replace":    strings.ReplaceAll,
	}
}

// LoadTemplate loads and parses a template string with custom functions
func LoadTemplate(tmplContent string) (*template.Template, error) {
	tmpl, err := template.New("sdk").Funcs(FuncMap()).Parse(tmplContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	return tmpl, nil
}

// RenderTemplate renders a template with the given data
func RenderTemplate(tmplContent string, data TemplateData) (string, error) {
	tmpl, err := LoadTemplate(tmplContent)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// String transformation functions
func ToCamelCase(s string) string {
	if s == "" {
		return s
	}

	parts := splitWords(s)
	if len(parts) == 0 {
		return s
	}

	result := strings.ToLower(parts[0])
	for i := 1; i < len(parts); i++ {
		result += capitalize(parts[i])
	}
	return result
}

func ToPascalCase(s string) string {
	if s == "" {
		return s
	}

	parts := splitWords(s)
	result := ""
	for _, part := range parts {
		result += capitalize(part)
	}
	return result
}

func ToSnakeCase(s string) string {
	if s == "" {
		return s
	}

	parts := splitWords(s)
	return strings.ToLower(strings.Join(parts, "_"))
}

func ToKebabCase(s string) string {
	if s == "" {
		return s
	}

	parts := splitWords(s)
	return strings.ToLower(strings.Join(parts, "-"))
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	c := cases.Title(language.English)
	return c.String(strings.ToLower(s))
}

func toTitleCase(s string) string {
	if s == "" {
		return s
	}
	c := cases.Title(language.English)
	return c.String(strings.ToLower(s))
}

func splitWords(s string) []string {
	// Split on various delimiters and camelCase boundaries
	var words []string
	var current strings.Builder

	for i, r := range s {
		switch {
		case (r >= 'A' && r <= 'Z') && i > 0:
			// Uppercase letter after lowercase/number = new word
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			current.WriteRune(r)
		case r == '_' || r == '-' || r == ' ' || r == '.':
			// Delimiter
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

func pluralize(s string) string {
	// Simple pluralization rules
	if strings.HasSuffix(s, "y") {
		return strings.TrimSuffix(s, "y") + "ies"
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") || strings.HasSuffix(s, "z") {
		return s + "es"
	}
	return s + "s"
}

func singularize(s string) string {
	// Simple singularization rules
	if strings.HasSuffix(s, "ies") {
		return strings.TrimSuffix(s, "ies") + "y"
	}
	if strings.HasSuffix(s, "es") {
		return strings.TrimSuffix(s, "es")
	}
	if strings.HasSuffix(s, "s") {
		return strings.TrimSuffix(s, "s")
	}
	return s
}
