// Package generator provides code generation functionality for SDKs.
package generator

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"

	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

// GenerateGoSDK generates a Go SDK
// If version is nil, uses the default Go version
// If sdkVersion is empty, extracts from OpenAPI schema or defaults to "1.0.0"
func GenerateGoSDK(
	outputPath, sdkName, httpLib string,
	openAPIDoc interface{},
	version *LanguageVersion,
	sdkVersion string,
) error {
	// Use default version if not provided
	if version == nil {
		defaultVersion := GetGoDefaultVersion()
		version = &defaultVersion
	}

	// Convert openAPIDoc to *openapi3.T
	doc, ok := openAPIDoc.(*openapi3.T)
	if !ok {
		// If not an openapi3.T, try to extract from ExtractedData
		if extractedData, ok := openAPIDoc.(*ExtractedData); ok {
			return generateGoSDKFromExtracted(outputPath, sdkName, httpLib, extractedData, *version, sdkVersion)
		}
		return fmt.Errorf("invalid OpenAPI document type")
	}

	// Extract data from OpenAPI document
	extractedData, err := ExtractOpenAPIData(doc)
	if err != nil {
		return fmt.Errorf("failed to extract OpenAPI data: %w", err)
	}

	return generateGoSDKFromExtracted(outputPath, sdkName, httpLib, extractedData, *version, sdkVersion)
}

// generateGoSDKFromExtracted generates SDK from extracted data
func generateGoSDKFromExtracted(
	outputPath, sdkName, httpLib string,
	extractedData *ExtractedData,
	version LanguageVersion,
	sdkVersion string,
) error {
	// Get HTTP library config
	libConfig, err := httplib.GetLibraryConfig("go", httpLib)
	if err != nil {
		return fmt.Errorf("failed to get HTTP library config: %w", err)
	}

	// Sanitize SDK name for Go (PascalCase for package name, but lowercase for directory)
	// Note: outputPath already includes the SDK name (output/language/sdk-name)
	// So we use it directly as packageDir, similar to Python
	sanitizedName := strings.ToLower(toPascalCase(sdkName))
	packageDir := outputPath

	// Create package directory
	if err := os.MkdirAll(packageDir, 0750); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Template data
	data := TemplateData{
		SDKName:         sanitizedName,
		Language:        "go",
		HTTPLib:         httpLib,
		HTTPLibImport:   libConfig.Import,
		HTTPLibConfig:   libConfig,
		OpenAPIDoc:      extractedData,
		ClientClassName: getClientClassName(sanitizedName),
	}

	// Determine SDK version: OpenAPI schema > user-provided > default
	finalSDKVersion := ""
	if extractedData != nil && extractedData.Version != "" {
		finalSDKVersion = extractedData.Version
	}
	if finalSDKVersion == "" && sdkVersion != "" {
		finalSDKVersion = sdkVersion
	}
	if finalSDKVersion == "" {
		finalSDKVersion = "1.0.0"
	}

	// Generate go.mod
	goModContent := generateGoMod(sdkName, extractedData, version)
	goModPath := filepath.Join(packageDir, "go.mod")
	// #nosec G306 -- 0644 is appropriate for go.mod file
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		return fmt.Errorf("failed to write go.mod: %w", err)
	}

	// Get package name (used for version.go and models.go)
	packageName := strings.ToLower(data.SDKName)

	// Generate version.go
	versionContent := generateGoVersion(packageName, finalSDKVersion)
	versionPath := filepath.Join(packageDir, "version.go")
	// #nosec G306 -- 0644 is appropriate for Go source files
	if err := os.WriteFile(versionPath, []byte(versionContent), 0644); err != nil {
		return fmt.Errorf("failed to write version.go: %w", err)
	}
	// Format with gofmt
	if err := formatGoFile(versionPath); err != nil {
		// Log but don't fail - formatting is nice-to-have
		_ = err
	}

	// Generate models.go if schemas exist
	if len(extractedData.Schemas) > 0 {
		modelsContent := fmt.Sprintf("package %s\n\n%s", packageName, generateGoModels(extractedData.Schemas, version))
		modelsPath := filepath.Join(packageDir, "models.go")
		// #nosec G306 -- 0644 is appropriate for Go source files
		if err := os.WriteFile(modelsPath, []byte(modelsContent), 0644); err != nil {
			return fmt.Errorf("failed to write models.go: %w", err)
		}
		// Format with gofmt
		if err := formatGoFile(modelsPath); err != nil {
			// Log but don't fail - formatting is nice-to-have
			_ = err
		}
	}

	// Generate client.go
	clientContent := generateGoClient(data, version)
	clientPath := filepath.Join(packageDir, "client.go")
	// #nosec G306 -- 0644 is appropriate for Go source files
	if err := os.WriteFile(clientPath, []byte(clientContent), 0644); err != nil {
		return fmt.Errorf("failed to write client.go: %w", err)
	}
	// Format with gofmt - this will align struct fields automatically
	if err := formatGoFile(clientPath); err != nil {
		// Log but don't fail - formatting is nice-to-have
		// Note: If there are syntax errors, gofmt will fail, but that's okay
		_ = err
	}

	// Generate README.md
	readmeContent := generateGoREADME(data)
	readmePath := filepath.Join(packageDir, "README.md")
	// #nosec G306 -- 0644 is appropriate for README files
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	// Generate api/ directory with endpoint files organized by tags
	// Note: In Go, files in the same package must be in the same directory
	// So we create api/ as a subdirectory with a separate package
	if len(extractedData.Operations) > 0 {
		apiDir := filepath.Join(packageDir, "api")
		if err := os.MkdirAll(apiDir, 0750); err != nil {
			return fmt.Errorf("failed to create api directory: %w", err)
		}

		// Group operations by tags
		operationsByTag := groupOperationsByTag(extractedData.Operations)
		for tag, operations := range operationsByTag {
			// Generate api/{tag}.go
			tagFileName := toSnakeCase(tag) + ".go"
			tagContent := generateGoAPIModule(tag, operations, data, version)
			tagPath := filepath.Join(apiDir, tagFileName)
			// #nosec G306 -- 0644 is appropriate for Go source files
			if err := os.WriteFile(tagPath, []byte(tagContent), 0644); err != nil {
				return fmt.Errorf("failed to write api/%s: %w", tagFileName, err)
			}
			// Format with gofmt
			if err := formatGoFile(tagPath); err != nil {
				// Log but don't fail - formatting is nice-to-have
				_ = err
			}
		}
	}

	// Generate examples/ directory
	examplesDir := filepath.Join(packageDir, "examples")
	if err := os.MkdirAll(examplesDir, 0750); err != nil {
		return fmt.Errorf("failed to create examples directory: %w", err)
	}

	// Generate examples/basic_usage.go
	examplesContent := generateGoExamples(data)
	examplesPath := filepath.Join(examplesDir, "basic_usage.go")
	// #nosec G306 -- 0644 is appropriate for Go example files
	if err := os.WriteFile(examplesPath, []byte(examplesContent), 0644); err != nil {
		return fmt.Errorf("failed to write examples/basic_usage.go: %w", err)
	}
	// Format with gofmt
	if err := formatGoFile(examplesPath); err != nil {
		// Log but don't fail - formatting is nice-to-have
		_ = err
	}

	return nil
}

// formatGoFile formats a Go source file using gofmt
func formatGoFile(filePath string) error {
	cmd := exec.Command("gofmt", "-w", filePath)
	if err := cmd.Run(); err != nil {
		// If gofmt is not available, that's okay - just skip formatting
		return fmt.Errorf("gofmt failed (may not be installed): %w", err)
	}
	return nil
}

// generateGoVersion generates version.go file with SDK version constant
func generateGoVersion(packageName, sdkVersion string) string {
	return fmt.Sprintf(`// Package %s provides SDK version information
// Auto-generated from OpenAPI schema

package %s

// Version is the version of this SDK
const Version = "%s"
`, packageName, packageName, sdkVersion)
}

// generateGoMod generates go.mod file
func generateGoMod(sdkName string, extractedData *ExtractedData, version LanguageVersion) string {
	packageName := strings.ToLower(toPascalCase(sdkName))
	moduleName := fmt.Sprintf("github.com/example/%s", packageName)
	goVersion := version

	// Prepare template data
	type GoModData struct {
		ModuleName        string
		GoVersion         string
		HTTPLibDependency string
	}
	data := GoModData{
		ModuleName:        moduleName,
		GoVersion:         goVersion.GetGoVersionString(),
		HTTPLibDependency: getGoHTTPLibDependency(extractedData),
	}

	// Load and render template
	tmpl, err := LoadTemplate(GetGoModTemplate())
	if err != nil {
		// Fallback to old method
		return fmt.Sprintf("module %s\n\n%s\n\nrequire (\n\t%s\n)\n",
			moduleName, goVersion.GetGoVersionString(), getGoHTTPLibDependency(extractedData))
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		// Fallback to old method
		return fmt.Sprintf("module %s\n\n%s\n\nrequire (\n\t%s\n)\n",
			moduleName, goVersion.GetGoVersionString(), getGoHTTPLibDependency(extractedData))
	}

	return buf.String()
}

// getGoHTTPLibDependency returns the Go dependency for the HTTP library
func getGoHTTPLibDependency(_ *ExtractedData) string {
	// Default to net/http (standard library, no dependency needed)
	return "\t// HTTP client: net/http (standard library)"
}

// generateGoClient generates Go client code
func generateGoClient(data TemplateData, version LanguageVersion) string {
	// Extract OpenAPI data
	extractedData, ok := data.OpenAPIDoc.(*ExtractedData)
	if !ok {
		extractedData = &ExtractedData{BaseURL: "https://api.example.com/v1"}
	}

	// Generate base URL default
	const defaultBaseURL = "https://api.example.com/v1"
	baseURLDefault := extractedData.BaseURL
	if baseURLDefault == "" {
		baseURLDefault = defaultBaseURL
	}

	// Generate authentication setup
	authSetup := generateGoAuthSetup(extractedData.SecuritySchemes, data.ClientClassName)

	// Generate API methods
	apiMethods := generateGoAPIMethods(extractedData.Operations, data.ClientClassName, version)

	// Format SDK name for display
	displayName := toPascalCase(data.SDKName)
	packageName := strings.ToLower(data.SDKName)

	// Build imports
	imports := buildGoImports(data)

	// Use provided version for type generation
	goVersion := version

	// Prepare template data
	type GoClientData struct {
		PackageName        string
		DisplayName        string
		Imports            string
		ClientClassName    string
		AuthFields         string
		BaseURLDefault     string
		AuthSetup          string
		APIMethods         string
		EmptyInterfaceType string
	}
	templateData := GoClientData{
		PackageName:        packageName,
		DisplayName:        displayName,
		Imports:            imports,
		ClientClassName:    data.ClientClassName,
		AuthFields:         generateGoClientAuthFields(extractedData.SecuritySchemes),
		BaseURLDefault:     baseURLDefault,
		AuthSetup:          authSetup,
		APIMethods:         apiMethods,
		EmptyInterfaceType: goVersion.GetGoEmptyInterface(),
	}

	// Load and render template
	tmpl, err := LoadTemplate(GetGoClientTemplate())
	if err != nil {
		// Fallback to old method (keep original implementation as fallback)
		return generateGoClientFallback(
			data, extractedData, baseURLDefault, authSetup, apiMethods,
			displayName, packageName, imports)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		// Fallback to old method
		return generateGoClientFallback(
			data, extractedData, baseURLDefault, authSetup, apiMethods,
			displayName, packageName, imports)
	}

	return buf.String()
}

// generateGoClientFallback is the original implementation kept as fallback
func generateGoClientFallback(
	data TemplateData,
	extractedData *ExtractedData,
	baseURLDefault, authSetup, apiMethods, displayName, packageName, imports string,
) string {
	goVersion := GetGoDefaultVersion()
	emptyInterface := goVersion.GetGoEmptyInterface()

	return fmt.Sprintf(`// Package %s provides a client for %s API
// Auto-generated from OpenAPI schema

package %s

%s

// %s represents the API client
type %s struct {
	BaseURL    string
	HTTPClient *http.Client%s
}

// New%s creates a new API client
func New%s(baseURL string, options ...%sOption) *%s {
	if baseURL == "" {
		baseURL = "%s"
	}
	client := &%s{
		BaseURL:    strings.TrimSuffix(baseURL, "/"),
		HTTPClient: &http.Client{},
	}
	
	// Apply options
	for _, option := range options {
		option(client)
	}
	
	return client
}

// %sOption is a function that configures the client
type %sOption func(*%s)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) %sOption {
	return func(c *%s) {
		c.HTTPClient = httpClient
	}
}

%s

// Request makes an HTTP request (public method for use by api package)
func (c *%s) Request(method, path string, body %s) ([]byte, error) {
	url := c.BaseURL + path
	
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %%w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}
	
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %%w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	// Apply authentication headers
	c.applyAuth(req)
	
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %%w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %%d - %%s", resp.StatusCode, string(bodyBytes))
	}
	
	return io.ReadAll(resp.Body)
}

%s
`,
		packageName,
		displayName,
		packageName,
		imports,
		data.ClientClassName,
		data.ClientClassName,
		generateGoClientAuthFields(extractedData.SecuritySchemes),
		data.ClientClassName,
		data.ClientClassName,
		data.ClientClassName,
		data.ClientClassName,
		baseURLDefault,
		data.ClientClassName,
		data.ClientClassName,
		data.ClientClassName,
		data.ClientClassName,
		data.ClientClassName,
		data.ClientClassName,
		authSetup,
		data.ClientClassName,
		emptyInterface,
		apiMethods)
}

// buildGoImports builds the import statement for Go
func buildGoImports(_ TemplateData) string {
	imports := []string{
		"bytes",
		"encoding/json",
		"fmt",
		"io",
		"net/http",
		"strings",
	}

	var importList strings.Builder
	for _, imp := range imports {
		importList.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
	}

	return fmt.Sprintf("import (\n%s)", importList.String())
}

// generateGoAuthSetup generates authentication setup code
func generateGoAuthSetup(securitySchemes map[string]SecurityScheme, clientClassName string) string {
	if len(securitySchemes) == 0 {
		return "// No authentication configured"
	}

	var setupCode strings.Builder
	setupCode.WriteString("// applyAuth applies authentication to the request\n")
	setupCode.WriteString(fmt.Sprintf("func (c *%s) applyAuth(req *http.Request) {\n", clientClassName))

	for name, scheme := range securitySchemes {
		switch scheme.Type {
		case securitySchemeAPIKey:
			switch scheme.In {
			case paramLocationHeader:
				setupCode.WriteString(fmt.Sprintf("\tif c.%s != \"\" {\n", toPascalCase(name)))
				setupCode.WriteString(fmt.Sprintf("\t\treq.Header.Set(\"%s\", c.%s)\n", scheme.Name, toPascalCase(name)))
				setupCode.WriteString("\t}\n")
			case paramLocationQuery:
				setupCode.WriteString(fmt.Sprintf("\tif c.%s != \"\" {\n", toPascalCase(name)))
				setupCode.WriteString("\t\tq := req.URL.Query()\n")
				setupCode.WriteString(fmt.Sprintf("\t\tq.Set(\"%s\", c.%s)\n", scheme.Name, toPascalCase(name)))
				setupCode.WriteString("\t\treq.URL.RawQuery = q.Encode()\n")
				setupCode.WriteString("\t}\n")
			}
		case securitySchemeHTTP:
			switch scheme.Scheme {
			case securitySchemeBearer:
				setupCode.WriteString("\tif c.BearerToken != \"\" {\n")
				setupCode.WriteString("\t\treq.Header.Set(\"Authorization\", \"Bearer \"+c.BearerToken)\n")
				setupCode.WriteString("\t}\n")
			case securitySchemeBasic:
				setupCode.WriteString("\tif c.Username != \"\" && c.Password != \"\" {\n")
				setupCode.WriteString("\t\treq.SetBasicAuth(c.Username, c.Password)\n")
				setupCode.WriteString("\t}\n")
			}
		case securitySchemeOAuth2:
			tokenName := toPascalCase(name)
			setupCode.WriteString(fmt.Sprintf("\tif c.%sToken != \"\" {\n", tokenName))
			setupCode.WriteString(fmt.Sprintf("\t\ttokenType := c.%sTokenType\n", tokenName))
			setupCode.WriteString("\t\tif tokenType == \"\" {\n")
			setupCode.WriteString("\t\t\ttokenType = \"Bearer\"\n")
			setupCode.WriteString("\t\t}\n")
			authHeader := fmt.Sprintf("\t\treq.Header.Set(\"Authorization\", tokenType+\" \"+c.%sToken)\n", tokenName)
			setupCode.WriteString(authHeader)
			setupCode.WriteString("\t}\n")
		case securitySchemeOpenIDConnect:
			tokenName := toPascalCase(name)
			setupCode.WriteString(fmt.Sprintf("\tif c.%sToken != \"\" {\n", tokenName))
			bearerAuth := fmt.Sprintf("\t\treq.Header.Set(\"Authorization\", \"Bearer \"+c.%sToken)\n", tokenName)
			setupCode.WriteString(bearerAuth)
			setupCode.WriteString("\t}\n")
		}
	}

	setupCode.WriteString("}\n")
	return setupCode.String()
}

// generateGoClientAuthFields generates authentication fields for the Client struct
// Fields are generated with consistent formatting - gofmt will align them properly
func generateGoClientAuthFields(securitySchemes map[string]SecurityScheme) string {
	if len(securitySchemes) == 0 {
		return ""
	}

	var fields strings.Builder
	// Add newline before first field for proper formatting
	fields.WriteString("\n")
	for name, scheme := range securitySchemes {
		switch scheme.Type {
		case securitySchemeAPIKey:
			// Use consistent tab + field name + type format
			fields.WriteString(fmt.Sprintf("\t%s string\n", toPascalCase(name)))
		case securitySchemeHTTP:
			switch scheme.Scheme {
			case securitySchemeBearer:
				fields.WriteString("\tBearerToken string\n")
			case securitySchemeBasic:
				fields.WriteString("\tUsername string\n")
				fields.WriteString("\tPassword string\n")
			}
		case securitySchemeOAuth2:
			// Generate fields with consistent formatting
			fields.WriteString(fmt.Sprintf("\t%sToken string\n", toPascalCase(name)))
			fields.WriteString(fmt.Sprintf("\t%sTokenType string\n", toPascalCase(name)))
		case securitySchemeOpenIDConnect:
			fields.WriteString(fmt.Sprintf("\t%sToken string\n", toPascalCase(name)))
		}
	}

	return fields.String()
}

// generateGoAPIMethods generates API methods from operations
func generateGoAPIMethods(operations []APIOperation, clientClassName string, version LanguageVersion) string {
	if len(operations) == 0 {
		return "// No API methods defined in OpenAPI schema"
	}

	var methods strings.Builder

	for _, op := range operations {
		methodName := GetOperationMethodName(op)
		if methodName == "" {
			// Fallback naming
			pathPart := strings.ReplaceAll(strings.Trim(op.Path, "/"), "/", "_")
			methodName = toPascalCase(strings.ToLower(op.Method)) + toPascalCase(pathPart)
		} else {
			// Convert snake_case to PascalCase for Go
			methodName = toPascalCase(methodName)
		}

		// Generate method signature
		methods.WriteString(fmt.Sprintf("// %s %s\n", methodName, op.Summary))
		if op.Description != "" {
			// Split description by newlines and add "// " prefix to each line
			descLines := strings.Split(op.Description, "\n")
			for _, line := range descLines {
				line = strings.TrimSpace(line)
				if line != "" {
					methods.WriteString(fmt.Sprintf("// %s\n", line))
				}
			}
		}
		methods.WriteString(fmt.Sprintf("func (c *%s) %s(", clientClassName, methodName))

		// Add path parameters
		var pathParams []string
		var queryParams []string
		for _, param := range op.Parameters {
			switch param.In {
			case paramLocationPath:
				pathParams = append(pathParams, param.Name)
				methods.WriteString(fmt.Sprintf("%s %s, ", param.Name, getGoType(param.Schema, version)))
			case paramLocationQuery:
				queryParams = append(queryParams, param.Name)
				methods.WriteString(fmt.Sprintf("%s *%s, ", param.Name, getGoType(param.Schema, version)))
			}
		}

		// Add request body parameter
		if op.RequestBody != nil {
			methods.WriteString(fmt.Sprintf("body %s, ", version.GetGoEmptyInterface()))
		}

		// Remove trailing comma and space
		sig := methods.String()
		sig = strings.TrimSuffix(sig, ", ")
		methods.Reset()
		methods.WriteString(sig)

		methods.WriteString(") ([]byte, error) {\n")

		// Build path with parameters
		path := op.Path
		for _, param := range pathParams {
			path = strings.ReplaceAll(path, "{"+param+"}", fmt.Sprintf("${%s}", param))
		}
		methods.WriteString(fmt.Sprintf("\tpath := fmt.Sprintf(\"%s\"", path))
		for _, param := range pathParams {
			methods.WriteString(fmt.Sprintf(", %s", param))
		}
		methods.WriteString(")\n")

		// Add query parameters
		if len(queryParams) > 0 {
			methods.WriteString("\tif len(path) > 0 {\n")
			methods.WriteString("\t\tqueryParts := []string{}\n")
			for _, param := range queryParams {
				methods.WriteString(fmt.Sprintf("\t\tif %s != nil {\n", param))
				queryFmt := fmt.Sprintf("\t\t\tqueryParts = append(queryParts, fmt.Sprintf(\"%s=%%v\", *%s))\n", param, param)
				methods.WriteString(queryFmt)
				methods.WriteString("\t\t}\n")
			}
			methods.WriteString("\t\tif len(queryParts) > 0 {\n")
			methods.WriteString("\t\t\tpath += \"?\" + strings.Join(queryParts, \"&\")\n")
			methods.WriteString("\t\t}\n")
			methods.WriteString("\t}\n")
		}

		// Make request
		methods.WriteString(fmt.Sprintf("\treturn c.Request(\"%s\", path, body)\n", op.Method))
		methods.WriteString("}\n\n")
	}

	return methods.String()
}

// getGoType converts OpenAPI schema type to Go type
func getGoType(schema *Schema, version LanguageVersion) string {
	emptyInterface := version.GetGoEmptyInterface()

	if schema == nil {
		return emptyInterface
	}

	switch schema.Type {
	case pythonTypeString:
		return "string"
	case pythonTypeInteger:
		return "int"
	case pythonTypeNumber:
		return "float64"
	case pythonTypeBoolean:
		return "bool"
	case pythonTypeArray:
		if schema.Items != nil {
			return "[]" + getGoType(schema.Items, version)
		}
		return "[]" + emptyInterface
	case pythonTypeObject:
		return "map[string]" + emptyInterface
	default:
		return emptyInterface
	}
}

// generateGoModels generates Go data models from OpenAPI schemas
func generateGoModels(schemas map[string]*Schema, version LanguageVersion) string {
	if len(schemas) == 0 {
		return "// No data models defined in OpenAPI schema\n"
	}

	var models strings.Builder
	models.WriteString("// Data Models - Auto-generated from OpenAPI schema\n\n")
	// Note: Package declaration is added in generateGoClient, models.go is in the same package

	for name, schema := range schemas {
		modelCode := generateGoModel(name, schema, schemas, version)
		models.WriteString(modelCode)
		models.WriteString("\n")
	}

	return models.String()
}

// generateGoModel generates a single Go model struct
func generateGoModel(name string, schema *Schema, allSchemas map[string]*Schema, version LanguageVersion) string {
	var code strings.Builder

	// Convert schema name to PascalCase for struct name
	structName := toPascalCase(name)

	// Generate struct
	code.WriteString(fmt.Sprintf("// %s %s\n", structName, schema.Description))
	code.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	// Handle different schema types
	switch schema.Type {
	case pythonTypeObject:
		code.WriteString(generateGoObjectFields(schema, allSchemas, version))
	case pythonTypeArray:
		// Arrays are handled as slices in Go
		code.WriteString(fmt.Sprintf("\tItems []%s\n", version.GetGoEmptyInterface()))
	default:
		code.WriteString(fmt.Sprintf("\tValue %s\n", getGoType(schema, version)))
	}

	code.WriteString("}\n")
	return code.String()
}

// generateGoObjectFields generates fields for an object schema
func generateGoObjectFields(schema *Schema, _ map[string]*Schema, version LanguageVersion) string {
	var fields strings.Builder

	if schema.Properties == nil {
		return "\t// No properties defined\n"
	}

	// Track required fields
	requiredMap := make(map[string]bool)
	for _, req := range schema.Required {
		requiredMap[req] = true
	}

	for propName, propSchema := range schema.Properties {
		fieldName := toPascalCase(propName)
		goType := getGoType(propSchema, version)

		// Check if field is required
		if !requiredMap[propName] {
			goType = "*" + goType
		}

		// Add JSON tag
		jsonTag := propName
		if !requiredMap[propName] {
			jsonTag += ",omitempty"
		}

		fields.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", fieldName, goType, jsonTag))
	}

	return fields.String()
}

// generateGoREADME generates README.md for Go SDK
func generateGoREADME(data TemplateData) string {
	displayName := toPascalCase(data.SDKName)
	packageName := strings.ToLower(data.SDKName)

	// Prepare template data
	type GoReadmeData struct {
		DisplayName     string
		PackageName     string
		ClientClassName string
	}
	templateData := GoReadmeData{
		DisplayName:     displayName,
		PackageName:     packageName,
		ClientClassName: data.ClientClassName,
	}

	// Load and render template
	tmpl, err := LoadTemplate(GetGoReadmeTemplate())
	if err != nil {
		// Fallback to old method
		readme := fmt.Sprintf("# %s Go SDK\n\n", displayName)
		readme += "Auto-generated Go SDK from OpenAPI schema.\n\n"
		readme += "## Installation\n\n"
		readme += fmt.Sprintf("```bash\ngo get github.com/example/%s\n```\n\n", packageName)
		readme += "## Usage\n\n"
		readme += "```go\npackage main\n\n"
		readme += fmt.Sprintf("import (\n\t\"github.com/example/%s\"\n)\n\n", packageName)
		readme += "func main() {\n"
		readme += fmt.Sprintf("\tclient := %s.New%s(\"https://api.example.com/v1\")\n\n", packageName, data.ClientClassName)
		readme += "\t// Use client methods...\n}\n```\n\n"
		readme += "## HTTP Library\n\n"
		readme += "This SDK uses `net/http` (Go standard library).\n\n"
		readme += "## Authentication\n\n"
		readme += "Configure authentication when creating the client:\n\n"
		readme += "```go\n"
		readme += fmt.Sprintf("client := %s.New%s(\"https://api.example.com/v1\")\n", packageName, data.ClientClassName)
		readme += "client.BearerToken = \"your-token\"\n"
		readme += "```\n"
		return readme
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		// Fallback to old method
		readme := fmt.Sprintf("# %s Go SDK\n\n", displayName)
		readme += "Auto-generated Go SDK from OpenAPI schema.\n\n"
		readme += "## Installation\n\n"
		readme += fmt.Sprintf("```bash\ngo get github.com/example/%s\n```\n\n", packageName)
		readme += "## Usage\n\n"
		readme += "```go\npackage main\n\n"
		readme += fmt.Sprintf("import (\n\t\"github.com/example/%s\"\n)\n\n", packageName)
		readme += "func main() {\n"
		readme += fmt.Sprintf("\tclient := %s.New%s(\"https://api.example.com/v1\")\n\n", packageName, data.ClientClassName)
		readme += "\t// Use client methods...\n}\n```\n\n"
		readme += "## HTTP Library\n\n"
		readme += "This SDK uses `net/http` (Go standard library).\n\n"
		readme += "## Authentication\n\n"
		readme += "Configure authentication when creating the client:\n\n"
		readme += "```go\n"
		readme += fmt.Sprintf("client := %s.New%s(\"https://api.example.com/v1\")\n", packageName, data.ClientClassName)
		readme += "client.BearerToken = \"your-token\"\n"
		readme += "```\n"
		return readme
	}

	return buf.String()
}

// generateGoAPIModule generates api/{tag}.go with operations for that tag
// Note: This creates a separate package 'api' that contains methods on the client
func generateGoAPIModule(tag string, operations []APIOperation, data TemplateData, version LanguageVersion) string {
	packageName := strings.ToLower(data.SDKName)
	clientClassName := data.ClientClassName

	var module strings.Builder
	module.WriteString(fmt.Sprintf("// Package api provides %s API endpoints\n", tag))
	module.WriteString("// Auto-generated from OpenAPI schema\n\n")
	module.WriteString("package api\n\n")
	module.WriteString("import (\n")
	module.WriteString("\t\"fmt\"\n")
	module.WriteString(fmt.Sprintf("\t\"github.com/example/%s\"\n", packageName))
	module.WriteString(")\n\n")

	// Generate methods for each operation
	for _, op := range operations {
		methodName := GetOperationMethodName(op)
		if methodName == "" {
			// Fallback naming
			pathPart := strings.ReplaceAll(strings.Trim(op.Path, "/"), "/", "_")
			methodName = toPascalCase(strings.ToLower(op.Method)) + toPascalCase(pathPart)
		} else {
			// Convert snake_case to PascalCase for Go
			methodName = toPascalCase(methodName)
		}

		// Method signature - methods are on the client from the parent package
		module.WriteString(fmt.Sprintf("// %s %s\n", methodName, op.Summary))
		if op.Description != "" {
			module.WriteString(fmt.Sprintf("// %s\n", op.Description))
		}
		module.WriteString(fmt.Sprintf("func %s(c *%s.%s, ", methodName, packageName, clientClassName))

		// Add parameters
		var pathParams []string
		var queryParams []string
		for _, param := range op.Parameters {
			switch param.In {
			case paramLocationPath:
				pathParams = append(pathParams, param.Name)
				module.WriteString(fmt.Sprintf("%s %s, ", param.Name, getGoType(param.Schema, version)))
			case paramLocationQuery:
				queryParams = append(queryParams, param.Name)
				module.WriteString(fmt.Sprintf("%s *%s, ", param.Name, getGoType(param.Schema, version)))
			}
		}

		// Add request body parameter
		if op.RequestBody != nil {
			module.WriteString(fmt.Sprintf("body %s, ", version.GetGoEmptyInterface()))
		}

		// Remove trailing comma and space
		sig := module.String()
		sig = strings.TrimSuffix(sig, ", ")
		module.Reset()
		module.WriteString(sig)

		module.WriteString(") ([]byte, error) {\n")

		// Build path with parameters
		path := op.Path
		for _, param := range pathParams {
			path = strings.ReplaceAll(path, "{"+param+"}", fmt.Sprintf("${%s}", param))
		}
		module.WriteString(fmt.Sprintf("\tpath := fmt.Sprintf(\"%s\"", path))
		for _, param := range pathParams {
			module.WriteString(fmt.Sprintf(", %s", param))
		}
		module.WriteString(")\n")

		// Add query parameters if any
		if len(queryParams) > 0 {
			module.WriteString("\tif path[len(path)-1] != '?' {\n")
			module.WriteString("\t\tpath += \"?\"\n")
			module.WriteString("\t}\n")
			for i, param := range queryParams {
				if i > 0 {
					module.WriteString("\t\tpath += \"&\"\n")
				}
				module.WriteString(fmt.Sprintf("\tif %s != nil {\n", param))
				module.WriteString(fmt.Sprintf("\t\tpath += fmt.Sprintf(\"%s=%%v\", *%s)\n", param, param))
				module.WriteString("\t}\n")
			}
		}

		// Call request method on the client
		module.WriteString(fmt.Sprintf("\treturn c.Request(\"%s\", path", op.Method))
		if op.RequestBody != nil {
			module.WriteString(", body")
		} else {
			module.WriteString(", nil")
		}
		module.WriteString(")\n")
		module.WriteString("}\n\n")
	}

	return module.String()
}

// generateGoExamples generates examples/basic_usage.go
func generateGoExamples(data TemplateData) string {
	extractedData, ok := data.OpenAPIDoc.(*ExtractedData)
	packageName := strings.ToLower(data.SDKName)
	clientClassName := data.ClientClassName

	var examples strings.Builder
	examples.WriteString("// Package main demonstrates basic usage of the SDK\n")
	examples.WriteString("// Auto-generated from OpenAPI schema\n\n")
	examples.WriteString("package main\n\n")
	examples.WriteString("import (\n")
	examples.WriteString("\t\"fmt\"\n")
	examples.WriteString(fmt.Sprintf("\t\"github.com/example/%s\"\n", packageName))
	examples.WriteString(")\n\n")
	examples.WriteString("func main() {\n")
	examples.WriteString("\t// Initialize the client\n")
	examples.WriteString(fmt.Sprintf("\tclient := %s.New%s(\n", packageName, clientClassName))

	// Add base URL
	if ok && extractedData != nil && extractedData.BaseURL != "" {
		examples.WriteString(fmt.Sprintf("\t\t\"%s\",\n", extractedData.BaseURL))
	} else {
		examples.WriteString("\t\t\"https://api.example.com/v1\",\n")
	}
	examples.WriteString("\t)\n\n")

	// Add authentication examples if available
	if ok && extractedData != nil && len(extractedData.SecuritySchemes) > 0 {
		for name, scheme := range extractedData.SecuritySchemes {
			switch scheme.Type {
			case securitySchemeAPIKey:
				examples.WriteString("\t// Set API key authentication\n")
				examples.WriteString(fmt.Sprintf("\tclient.%s = \"your-%s\"\n\n", toPascalCase(name), name))
			case securitySchemeHTTP:
				switch scheme.Scheme {
				case securitySchemeBearer:
					examples.WriteString("\t// Set Bearer token authentication\n")
					examples.WriteString("\tclient.BearerToken = \"your-bearer-token\"\n\n")
				case securitySchemeBasic:
					examples.WriteString("\t// Set Basic authentication\n")
					examples.WriteString("\tclient.Username = \"your-username\"\n")
					examples.WriteString("\tclient.Password = \"your-password\"\n\n")
				}
			}
		}
	}

	examples.WriteString("\t// Example: List resources\n")
	examples.WriteString("\t// response, err := client.ListItems()\n")
	examples.WriteString("\t// if err != nil {\n")
	examples.WriteString("\t//     fmt.Printf(\"Error: %v\\n\", err)\n")
	examples.WriteString("\t//     return\n")
	examples.WriteString("\t// }\n")
	examples.WriteString("\t// fmt.Printf(\"Response: %s\\n\", string(response))\n\n")
	examples.WriteString("\t// Example: Get a resource by ID\n")
	examples.WriteString("\t// response, err := client.GetItem(123)\n")
	examples.WriteString("\t// if err != nil {\n")
	examples.WriteString("\t//     fmt.Printf(\"Error: %v\\n\", err)\n")
	examples.WriteString("\t//     return\n")
	examples.WriteString("\t// }\n")
	examples.WriteString("\t// fmt.Printf(\"Response: %s\\n\", string(response))\n\n")
	examples.WriteString("\t// Example: Create a resource\n")
	examples.WriteString("\t// data := map[string]interface{}{\n")
	examples.WriteString("\t//     \"name\": \"Example\",\n")
	examples.WriteString("\t//     \"value\": 42,\n")
	examples.WriteString("\t// }\n")
	examples.WriteString("\t// response, err := client.CreateItem(data)\n")
	examples.WriteString("\t// if err != nil {\n")
	examples.WriteString("\t//     fmt.Printf(\"Error: %v\\n\", err)\n")
	examples.WriteString("\t//     return\n")
	examples.WriteString("\t// }\n")
	examples.WriteString("\t// fmt.Printf(\"Response: %s\\n\", string(response))\n")
	examples.WriteString("}\n")

	return examples.String()
}
