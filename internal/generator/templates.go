// Package generator provides template loading functionality.
package generator

import (
	_ "embed"
	"fmt"
	"text/template"

	"github.com/vubon/sdk-forge/internal/generator/common"
)

//go:embed go/templates/client.go.tmpl
var goClientTemplate string

//go:embed go/templates/go.mod.tmpl
var goModTemplate string

//go:embed go/templates/README.md.tmpl
var goReadmeTemplate string

//go:embed python/templates/client.py.tmpl
var pythonClientTemplate string

//go:embed python/templates/__init__.py.tmpl
var pythonInitTemplate string

//go:embed python/templates/setup.py.tmpl
var pythonSetupTemplate string

//go:embed python/templates/README.md.tmpl
var pythonReadmeTemplate string

//go:embed php/templates/client.php.tmpl
var phpClientTemplate string

//go:embed php/templates/composer.json.tmpl
var phpComposerTemplate string

//go:embed php/templates/README.md.tmpl
var phpReadmeTemplate string

// LoadTemplate loads and parses a template string with custom functions
func LoadTemplate(tmplContent string) (*template.Template, error) {
	tmpl, err := template.New("sdk").Funcs(common.FuncMap()).Parse(tmplContent)
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

// GetPHPClientTemplate returns the PHP client template
func GetPHPClientTemplate() string {
	return phpClientTemplate
}

// GetPHPComposerTemplate returns the PHP composer.json template
func GetPHPComposerTemplate() string {
	return phpComposerTemplate
}

// GetPHPReadmeTemplate returns the PHP README template
func GetPHPReadmeTemplate() string {
	return phpReadmeTemplate
}
