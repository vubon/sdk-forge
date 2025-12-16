// Package generator provides template loading functionality.
package generator

import (
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed templates/go/client.go.tmpl
var goClientTemplate string

//go:embed templates/go/go.mod.tmpl
var goModTemplate string

//go:embed templates/go/README.md.tmpl
var goReadmeTemplate string

//go:embed templates/python/client.py.tmpl
var pythonClientTemplate string

//go:embed templates/python/__init__.py.tmpl
var pythonInitTemplate string

//go:embed templates/python/setup.py.tmpl
var pythonSetupTemplate string

//go:embed templates/python/README.md.tmpl
var pythonReadmeTemplate string

// LoadTemplate loads and parses a template string with custom functions
func LoadTemplate(tmplContent string) (*template.Template, error) {
	tmpl, err := template.New("sdk").Funcs(FuncMap()).Parse(tmplContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	return tmpl, nil
}

// GetGoClientTemplate returns the Go client template
func GetGoClientTemplate() string {
	return goClientTemplate
}

// GetGoModTemplate returns the Go mod template
func GetGoModTemplate() string {
	return goModTemplate
}

// GetGoReadmeTemplate returns the Go README template
func GetGoReadmeTemplate() string {
	return goReadmeTemplate
}

// GetPythonClientTemplate returns the Python client template
func GetPythonClientTemplate() string {
	return pythonClientTemplate
}

// GetPythonInitTemplate returns the Python __init__.py template
func GetPythonInitTemplate() string {
	return pythonInitTemplate
}

// GetPythonSetupTemplate returns the Python setup.py template
func GetPythonSetupTemplate() string {
	return pythonSetupTemplate
}

// GetPythonReadmeTemplate returns the Python README template
func GetPythonReadmeTemplate() string {
	return pythonReadmeTemplate
}
