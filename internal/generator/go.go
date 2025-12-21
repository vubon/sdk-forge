// Package generator provides code generation functionality for SDKs.
package generator

import (
	"bytes"
	"encoding/json"
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
// retryConfig specifies retry behavior for HTTP requests (can be disabled)
func GenerateGoSDK(
	outputPath, sdkName, httpLib string,
	openAPIDoc interface{},
	version *LanguageVersion,
	sdkVersion string,
	generateTests bool,
	retryConfig RetryConfig,
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
			return generateGoSDKFromExtracted(outputPath, sdkName, httpLib, extractedData, *version, sdkVersion, generateTests, retryConfig)
		}
		return fmt.Errorf("invalid OpenAPI document type")
	}

	// Extract data from OpenAPI document
	extractedData, err := ExtractOpenAPIData(doc)
	if err != nil {
		return fmt.Errorf("failed to extract OpenAPI data: %w", err)
	}

	return generateGoSDKFromExtracted(outputPath, sdkName, httpLib, extractedData, *version, sdkVersion, generateTests, retryConfig)
}

// generateGoSDKFromExtracted generates SDK from extracted data
func generateGoSDKFromExtracted(
	outputPath, sdkName, httpLib string,
	extractedData *ExtractedData,
	version LanguageVersion,
	sdkVersion string,
	generateTests bool,
	retryConfig RetryConfig,
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
		RetryConfig:     retryConfig,
	}

	// Determine SDK version using common utility
	finalSDKVersion := determineSDKVersion(extractedData, sdkVersion)

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

	// Generate test files if test generation is enabled
	if generateTests {
		if err := generateGoTests(packageDir, data, extractedData, version); err != nil {
			return fmt.Errorf("failed to generate tests: %w", err)
		}
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

	// Generate retry setup if enabled
	retryFields := ""
	retryHelper := ""
	retryInit := ""
	if data.RetryConfig.Enabled {
		retryFields = generateGoRetryFields(data.RetryConfig)
		retryHelper = generateGoRetryHelper(data.HTTPLib, data.RetryConfig)
		retryInit = generateGoRetryInit(data.RetryConfig)
	}

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
		RetryEnabled       bool
		RetryFields        string
		RetryHelper        string
		RetryInit          string
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
		RetryEnabled:       data.RetryConfig.Enabled,
		RetryFields:        retryFields,
		RetryHelper:        retryHelper,
		RetryInit:          retryInit,
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
func buildGoImports(data TemplateData) string {
	imports := []string{
		"bytes",
		"encoding/json",
		"fmt",
		"io",
		"net/http",
		"strings",
	}

	// Add time import if retry is enabled
	if data.RetryConfig.Enabled {
		imports = append(imports, "time")
	}

	var importList strings.Builder
	for _, imp := range imports {
		importList.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
	}

	return fmt.Sprintf("import (\n%s)", importList.String())
}

// generateGoRetryFields generates retry configuration fields for client struct
func generateGoRetryFields(config RetryConfig) string {
	if !config.Enabled {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("\n\t// Retry configuration\n")
	buf.WriteString("\tretryMaxAttempts          int\n")
	buf.WriteString("\tretryInitialDelay         time.Duration\n")
	buf.WriteString("\tretryMaxDelay             time.Duration\n")
	buf.WriteString("\tretryBackoffMultiplier    float64\n")
	buf.WriteString("\tretryStrategy             string\n")
	buf.WriteString("\tretryableStatusCodes      []int\n")
	buf.WriteString("\tretryOnNetworkErrors      bool\n")

	return buf.String()
}

// generateGoRetryHelper generates retry helper functions based on HTTP library
func generateGoRetryHelper(httpLib string, config RetryConfig) string {
	if !config.Enabled {
		return ""
	}

	var buf strings.Builder

	// Calculate delay helper function - note: template will replace {{.ClientClassName}}
	buf.WriteString("\n// calculateRetryDelay calculates delay for retry attempt based on strategy\n")
	buf.WriteString("func (c *{{.ClientClassName}}) calculateRetryDelay(attempt int) time.Duration {\n")
	buf.WriteString("\tswitch c.retryStrategy {\n")
	buf.WriteString(fmt.Sprintf("\tcase %q:\n", RetryStrategyExponential))
	buf.WriteString("\t\t// Exponential backoff: initialDelay * (multiplier ^ attempt)\n")
	buf.WriteString("\t\tmultiplier := c.retryBackoffMultiplier\n")
	buf.WriteString("\t\tresult := float64(c.retryInitialDelay)\n")
	buf.WriteString("\t\tfor i := 0; i < attempt; i++ {\n")
	buf.WriteString("\t\t\tresult *= multiplier\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t\tdelay := time.Duration(result)\n")
	buf.WriteString("\t\tif delay > c.retryMaxDelay {\n")
	buf.WriteString("\t\t\treturn c.retryMaxDelay\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t\treturn delay\n")
	buf.WriteString(fmt.Sprintf("\tcase %q:\n", RetryStrategyLinear))
	buf.WriteString("\t\t// Linear backoff: initialDelay * (attempt + 1)\n")
	buf.WriteString("\t\tdelay := c.retryInitialDelay * time.Duration(attempt+1)\n")
	buf.WriteString("\t\tif delay > c.retryMaxDelay {\n")
	buf.WriteString("\t\t\treturn c.retryMaxDelay\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t\treturn delay\n")
	buf.WriteString(fmt.Sprintf("\tcase %q:\n", RetryStrategyFixed))
	buf.WriteString("\t\t// Fixed delay: always use initialDelay\n")
	buf.WriteString("\t\treturn c.retryInitialDelay\n")
	buf.WriteString("\tdefault:\n")
	buf.WriteString("\t\t// Default to exponential\n")
	buf.WriteString("\t\tmultiplier := c.retryBackoffMultiplier\n")
	buf.WriteString("\t\tresult := float64(c.retryInitialDelay)\n")
	buf.WriteString("\t\tfor i := 0; i < attempt; i++ {\n")
	buf.WriteString("\t\t\tresult *= multiplier\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t\tdelay := time.Duration(result)\n")
	buf.WriteString("\t\tif delay > c.retryMaxDelay {\n")
	buf.WriteString("\t\t\treturn c.retryMaxDelay\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t\treturn delay\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n")

	// isRetryableStatusCode helper
	buf.WriteString("\n// isRetryableStatusCode checks if a status code should trigger a retry\n")
	buf.WriteString("func (c *{{.ClientClassName}}) isRetryableStatusCode(statusCode int) bool {\n")
	buf.WriteString("\tfor _, code := range c.retryableStatusCodes {\n")
	buf.WriteString("\t\tif statusCode == code {\n")
	buf.WriteString("\t\t\treturn true\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\treturn false\n")
	buf.WriteString("}\n")

	// Generate retry logic based on HTTP library - for nethttp (default)
	buf.WriteString("\n// requestWithRetry makes an HTTP request with retry logic\n")
	buf.WriteString("func (c *{{.ClientClassName}}) requestWithRetry(req *http.Request) (*http.Response, error) {\n")
	buf.WriteString("\tvar lastErr error\n")
	buf.WriteString("\tvar resp *http.Response\n\n")
	buf.WriteString("\tfor attempt := 0; attempt < c.retryMaxAttempts; attempt++ {\n")
	buf.WriteString("\t\tresp, err := c.HTTPClient.Do(req)\n")
	buf.WriteString("\t\tif err != nil {\n")
	buf.WriteString("\t\t\tif !c.retryOnNetworkErrors {\n")
	buf.WriteString("\t\t\t\treturn nil, fmt.Errorf(\"failed to execute request: %w\", err)\n")
	buf.WriteString("\t\t\t}\n")
	buf.WriteString("\t\t\tlastErr = err\n")
	buf.WriteString("\t\t\t// Network error - will retry below\n")
	buf.WriteString("\t\t} else {\n")
	buf.WriteString("\t\t\t// Check if status code is retryable\n")
	buf.WriteString("\t\t\tif !c.isRetryableStatusCode(resp.StatusCode) {\n")
	buf.WriteString("\t\t\t\treturn resp, nil\n")
	buf.WriteString("\t\t\t}\n")
	buf.WriteString("\t\t\t// Retryable status code - close response and retry\n")
	buf.WriteString("\t\t\tresp.Body.Close()\n")
	buf.WriteString("\t\t}\n\n")
	buf.WriteString("\t\t// If we get here, we need to retry\n")
	buf.WriteString("\t\tif attempt < c.retryMaxAttempts-1 {\n")
	buf.WriteString("\t\t\tdelay := c.calculateRetryDelay(attempt)\n")
	buf.WriteString("\t\t\ttime.Sleep(delay)\n")
	buf.WriteString("\t\t} else {\n")
	buf.WriteString("\t\t\t// Max attempts exceeded\n")
	buf.WriteString("\t\t\tif lastErr != nil {\n")
	buf.WriteString("\t\t\t\treturn nil, fmt.Errorf(\"failed after %d attempts: %w\", c.retryMaxAttempts, lastErr)\n")
	buf.WriteString("\t\t\t}\n")
	buf.WriteString("\t\t\treturn nil, fmt.Errorf(\"request failed after %d attempts\", c.retryMaxAttempts)\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t}\n\n")
	buf.WriteString("\treturn resp, nil\n")
	buf.WriteString("}\n")

	return buf.String()
}

// generateGoRetryInit generates retry configuration initialization code for NewClient
func generateGoRetryInit(config RetryConfig) string {
	if !config.Enabled {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("\n\t// Initialize retry configuration\n")
	buf.WriteString(fmt.Sprintf("\tclient.retryMaxAttempts = %d\n", config.MaxAttempts))
	buf.WriteString(fmt.Sprintf("\tclient.retryInitialDelay = %d * time.Second\n", int(config.InitialDelay.Seconds())))
	buf.WriteString(fmt.Sprintf("\tclient.retryMaxDelay = %d * time.Second\n", int(config.MaxDelay.Seconds())))
	buf.WriteString(fmt.Sprintf("\tclient.retryBackoffMultiplier = %.1f\n", config.BackoffMultiplier))
	buf.WriteString(fmt.Sprintf("\tclient.retryStrategy = %q\n", config.Strategy))
	buf.WriteString("\tclient.retryableStatusCodes = []int{")
	for i, code := range config.RetryableStatusCodes {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%d", code))
	}
	buf.WriteString("}\n")
	buf.WriteString(fmt.Sprintf("\tclient.retryOnNetworkErrors = %v\n", config.RetryOnNetworkErrors))

	return buf.String()
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
		if len(pathParams) > 0 {
			// First escape any existing % signs
			path = strings.ReplaceAll(path, "%", "%%")
			// Replace {param} with appropriate format specifier based on parameter type
			for _, param := range pathParams {
				// Find the parameter schema to determine type
				var paramSchema *Schema
				for _, p := range op.Parameters {
					if p.Name == param && p.In == paramLocationPath {
						paramSchema = p.Schema
						break
					}
				}
				// Determine format specifier
				formatSpec := "%s" // default to string
				if paramSchema != nil {
					switch paramSchema.Type {
					case "integer", "int32", "int64":
						formatSpec = "%d"
					case "number", "float", "double":
						formatSpec = "%f"
					case "boolean":
						formatSpec = "%t"
					}
				}
				// Replace {param} with format specifier
				path = strings.ReplaceAll(path, "{%%"+param+"}", formatSpec)
				path = strings.ReplaceAll(path, "{"+param+"}", formatSpec)
			}
			methods.WriteString(fmt.Sprintf("\tpath := fmt.Sprintf(\"%s\"", path))
			for _, param := range pathParams {
				methods.WriteString(fmt.Sprintf(", %s", param))
			}
			methods.WriteString(")\n")
		} else {
			// No path parameters, just use the path as-is
			methods.WriteString(fmt.Sprintf("\tpath := \"%s\"\n", path))
		}

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
		if op.RequestBody != nil {
			methods.WriteString(fmt.Sprintf("\treturn c.Request(\"%s\", path, body)\n", op.Method))
		} else {
			methods.WriteString(fmt.Sprintf("\treturn c.Request(\"%s\", path, nil)\n", op.Method))
		}
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

	// Check if any operation has path parameters (needs fmt.Sprintf)
	needsFmt := false
	for _, op := range operations {
		for _, param := range op.Parameters {
			if param.In == paramLocationPath {
				needsFmt = true
				break
			}
		}
		if needsFmt {
			break
		}
	}

	module.WriteString("import (\n")
	if needsFmt {
		module.WriteString("\t\"fmt\"\n")
	}
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
			// Split description by newlines and add "// " prefix to each line
			descLines := strings.Split(op.Description, "\n")
			for _, line := range descLines {
				line = strings.TrimSpace(line)
				if line != "" {
					module.WriteString(fmt.Sprintf("// %s\n", line))
				}
			}
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
		if len(pathParams) > 0 {
			// First escape any existing % signs
			path = strings.ReplaceAll(path, "%", "%%")
			// Replace {param} with appropriate format specifier based on parameter type
			for _, param := range pathParams {
				// Find the parameter schema to determine type
				var paramSchema *Schema
				for _, p := range op.Parameters {
					if p.Name == param && p.In == paramLocationPath {
						paramSchema = p.Schema
						break
					}
				}
				// Determine format specifier
				formatSpec := "%s" // default to string
				if paramSchema != nil {
					switch paramSchema.Type {
					case "integer", "int32", "int64":
						formatSpec = "%d"
					case "number", "float", "double":
						formatSpec = "%f"
					case "boolean":
						formatSpec = "%t"
					}
				}
				// Replace {param} with format specifier
				path = strings.ReplaceAll(path, "{%%"+param+"}", formatSpec)
				path = strings.ReplaceAll(path, "{"+param+"}", formatSpec)
			}
			module.WriteString(fmt.Sprintf("\tpath := fmt.Sprintf(\"%s\"", path))
			for _, param := range pathParams {
				module.WriteString(fmt.Sprintf(", %s", param))
			}
			module.WriteString(")\n")
		} else {
			// No path parameters, just use the path as-is
			module.WriteString(fmt.Sprintf("\tpath := \"%s\"\n", path))
		}

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
	// Only add fmt if we're actually using it in the examples
	// For now, examples are commented out, so we don't need fmt
	// examples.WriteString("\t\"fmt\"\n")
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

// generateGoTests generates test files for Go SDK
func generateGoTests(packageDir string, data TemplateData, extractedData *ExtractedData, version LanguageVersion) error {
	// Generate client_test.go
	clientTestContent := generateGoClientTest(data, extractedData, version)
	clientTestPath := filepath.Join(packageDir, "client_test.go")
	// #nosec G306 -- 0644 is appropriate for Go test files
	if err := os.WriteFile(clientTestPath, []byte(clientTestContent), 0644); err != nil {
		return fmt.Errorf("failed to write client_test.go: %w", err)
	}
	// Format with gofmt
	if err := formatGoFile(clientTestPath); err != nil {
		_ = err
	}

	// Generate models_test.go if schemas exist
	if len(extractedData.Schemas) > 0 {
		modelsTestContent := generateGoModelsTest(data, extractedData.Schemas, version)
		modelsTestPath := filepath.Join(packageDir, "models_test.go")
		// #nosec G306 -- 0644 is appropriate for Go test files
		if err := os.WriteFile(modelsTestPath, []byte(modelsTestContent), 0644); err != nil {
			return fmt.Errorf("failed to write models_test.go: %w", err)
		}
		// Format with gofmt
		if err := formatGoFile(modelsTestPath); err != nil {
			_ = err
		}
	}

	// Generate api_test.go if operations exist
	if len(extractedData.Operations) > 0 {
		apiTestContent := generateGoAPITest(data, extractedData.Operations, extractedData, version)
		apiTestPath := filepath.Join(packageDir, "api_test.go")
		// #nosec G306 -- 0644 is appropriate for Go test files
		if err := os.WriteFile(apiTestPath, []byte(apiTestContent), 0644); err != nil {
			return fmt.Errorf("failed to write api_test.go: %w", err)
		}
		// Format with gofmt
		if err := formatGoFile(apiTestPath); err != nil {
			_ = err
		}
	}

	// Generate auth_test.go if security schemes exist
	if len(extractedData.SecuritySchemes) > 0 {
		authTestContent := generateGoAuthTest(data, extractedData.SecuritySchemes, extractedData, version)
		authTestPath := filepath.Join(packageDir, "auth_test.go")
		// #nosec G306 -- 0644 is appropriate for Go test files
		if err := os.WriteFile(authTestPath, []byte(authTestContent), 0644); err != nil {
			return fmt.Errorf("failed to write auth_test.go: %w", err)
		}
		// Format with gofmt
		if err := formatGoFile(authTestPath); err != nil {
			_ = err
		}
	}

	// Generate test fixtures from OpenAPI examples
	if err := generateGoTestFixtures(packageDir, extractedData); err != nil {
		return fmt.Errorf("failed to generate test fixtures: %w", err)
	}

	return nil
}

// generateGoTestFixtures generates test fixture files from OpenAPI examples
func generateGoTestFixtures(packageDir string, extractedData *ExtractedData) error {
	fixturesDir := filepath.Join(packageDir, "testdata")
	if err := os.MkdirAll(fixturesDir, 0750); err != nil {
		return fmt.Errorf("failed to create testdata directory: %w", err)
	}

	// Generate fixtures from response examples
	fixtures := make(map[string]interface{})

	for _, op := range extractedData.Operations {
		for statusCode, response := range op.Responses {
			if jsonContent, ok := response.Content["application/json"]; ok {
				if len(jsonContent.Examples) > 0 {
					// Use first example for each status code
					for name, example := range jsonContent.Examples {
						key := fmt.Sprintf("%s_%s_%s", op.OperationID, statusCode, name)
						fixtures[key] = example
					}
				}
			}
		}
	}

	if len(fixtures) > 0 {
		// Generate fixtures.go file
		fixturesContent := generateGoFixturesFile(fixtures)
		fixturesPath := filepath.Join(fixturesDir, "fixtures.go")
		// #nosec G306 -- 0644 is appropriate for Go fixture files
		if err := os.WriteFile(fixturesPath, []byte(fixturesContent), 0644); err != nil {
			return fmt.Errorf("failed to write testdata/fixtures.go: %w", err)
		}
		// Format with gofmt
		if err := formatGoFile(fixturesPath); err != nil {
			_ = err
		}
	}

	return nil
}

// generateGoFixturesFile generates a Go file with test fixtures
func generateGoFixturesFile(fixtures map[string]interface{}) string {
	var buf bytes.Buffer
	buf.WriteString("package testdata\n\n")
	buf.WriteString("// Test fixtures extracted from OpenAPI examples\n\n")

	for key, example := range fixtures {
		// Convert key to valid Go variable name
		varName := toPascalCase(key)
		exampleJSON := formatExampleForGo(example)
		fmt.Fprintf(&buf, "var %s = %s\n\n", varName, exampleJSON)
	}

	return buf.String()
}

// generateGoClientTest generates client_test.go
func generateGoClientTest(data TemplateData, extractedData *ExtractedData, version LanguageVersion) string {
	var test bytes.Buffer
	test.WriteString(fmt.Sprintf("package %s\n\n", data.SDKName))
	test.WriteString("import (\n")
	test.WriteString("\t\"testing\"\n")
	test.WriteString(")\n\n")

	// Use base URL from OpenAPI spec, fallback to default
	baseURL := extractedData.BaseURL
	if baseURL == "" {
		baseURL = "https://api.example.com/v1"
	}

	test.WriteString(fmt.Sprintf("func TestNew%s(t *testing.T) {\n", data.ClientClassName))
	test.WriteString(fmt.Sprintf("\tclient := New%s(%q)\n", data.ClientClassName, baseURL))
	test.WriteString("\tif client == nil {\n")
	test.WriteString(fmt.Sprintf("\t\tt.Fatal(\"New%s() returned nil\")\n", data.ClientClassName))
	test.WriteString("\t}\n")
	test.WriteString(fmt.Sprintf("\tif client.BaseURL != %q {\n", baseURL))
	test.WriteString(fmt.Sprintf("\t\tt.Errorf(\"BaseURL = %%q, want %%q\", client.BaseURL, %q)\n", baseURL))
	test.WriteString("\t}\n")
	test.WriteString("}\n")
	return test.String()
}

// generateGoModelsTest generates models_test.go with schema-based tests
func generateGoModelsTest(data TemplateData, schemas map[string]*Schema, version LanguageVersion) string {
	var test bytes.Buffer
	test.WriteString(fmt.Sprintf("package %s\n\n", data.SDKName))
	test.WriteString("import (\n")
	test.WriteString("\t\"encoding/json\"\n")
	test.WriteString("\t\"testing\"\n")
	test.WriteString(")\n\n")

	// Generate tests for each schema
	for name, schema := range schemas {
		structName := toPascalCase(name)

		// Test struct creation
		test.WriteString(fmt.Sprintf("func Test%s_Creation(t *testing.T) {\n", structName))
		test.WriteString(fmt.Sprintf("\t// Test %s can be instantiated\n", structName))

		if schema.Type == "object" && len(schema.Properties) > 0 {
			// Track required fields
			requiredSet := make(map[string]bool)
			for _, req := range schema.Required {
				requiredSet[req] = true
			}

			// Generate struct literal with test values
			test.WriteString(fmt.Sprintf("\tmodel := %s{\n", structName))

			for propName, propSchema := range schema.Properties {
				if propSchema == nil {
					continue
				}

				fieldName := toPascalCase(propName)
				isRequired := requiredSet[propName]
				isPointer := !isRequired // Optional fields are pointers

				testValue := generateGoTestValue(propSchema, propName, version, isPointer)
				test.WriteString(fmt.Sprintf("\t\t%s: %s,\n", fieldName, testValue))
			}
			test.WriteString("\t}\n")
			test.WriteString("\t// Verify model can be instantiated\n")
			test.WriteString("\t_ = model\n")
		} else {
			// Simple model without properties
			test.WriteString(fmt.Sprintf("\tmodel := %s{}\n", structName))
			test.WriteString("\t// Verify model can be instantiated\n")
			test.WriteString("\t_ = model\n")
		}
		test.WriteString("}\n\n")

		// Test JSON serialization/deserialization
		if schema.Type == "object" && len(schema.Properties) > 0 {
			test.WriteString(fmt.Sprintf("func Test%s_Serialization(t *testing.T) {\n", structName))
			test.WriteString(fmt.Sprintf("\t// Test %s JSON serialization\n", structName))

			requiredSet := make(map[string]bool)
			for _, req := range schema.Required {
				requiredSet[req] = true
			}

			test.WriteString(fmt.Sprintf("\tmodel := %s{\n", structName))
			requiredSet2 := make(map[string]bool)
			for _, req := range schema.Required {
				requiredSet2[req] = true
			}
			for propName, propSchema := range schema.Properties {
				if propSchema == nil {
					continue
				}
				fieldName := toPascalCase(propName)
				isRequired := requiredSet2[propName]
				isPointer := !isRequired
				testValue := generateGoTestValue(propSchema, propName, version, isPointer)
				test.WriteString(fmt.Sprintf("\t\t%s: %s,\n", fieldName, testValue))
			}
			test.WriteString("\t}\n\n")

			test.WriteString("\t// Marshal to JSON\n")
			test.WriteString("\tdata, err := json.Marshal(model)\n")
			test.WriteString("\tif err != nil {\n")
			test.WriteString("\t\tt.Fatalf(\"Marshal() error = %v\", err)\n")
			test.WriteString("\t}\n")
			test.WriteString("\tif len(data) == 0 {\n")
			test.WriteString("\t\tt.Error(\"Marshal() should return non-empty data\")\n")
			test.WriteString("\t}\n\n")

			test.WriteString("\t// Unmarshal from JSON\n")
			test.WriteString(fmt.Sprintf("\tvar unmarshaled %s\n", structName))
			test.WriteString("\tif err := json.Unmarshal(data, &unmarshaled); err != nil {\n")
			test.WriteString("\t\tt.Fatalf(\"Unmarshal() error = %v\", err)\n")
			test.WriteString("\t}\n")
			test.WriteString("}\n\n")
		}
	}

	return test.String()
}

// generateGoTestValue generates a test value for a schema property in Go
// isPointer indicates if the field is a pointer type (for optional fields)
func generateGoTestValue(schema *Schema, propName string, version LanguageVersion, isPointer bool) string {
	if schema == nil {
		if isPointer {
			return "nil"
		}
		return "\"test_value\""
	}

	emptyInterface := version.GetGoEmptyInterface()
	var value string

	switch schema.Type {
	case "string":
		switch schema.Format {
		case "date":
			value = "\"2024-01-01\""
		case "date-time":
			value = "\"2024-01-01T00:00:00Z\""
		case "email":
			value = "\"test@example.com\""
		default:
			value = fmt.Sprintf("%q", "test_"+toSnakeCase(propName))
		}
	case "integer", "number":
		value = "42"
	case "boolean":
		value = "true"
	case "array":
		if schema.Items != nil {
			itemType := getGoType(schema.Items, version)
			// For arrays, generate a simple item value
			itemValue := generateGoTestValue(schema.Items, "item", version, false)
			arrayValue := fmt.Sprintf("[]%s{%s}", itemType, itemValue)
			if isPointer {
				// For pointer to array, we need to create the array first
				value = fmt.Sprintf("&%s", arrayValue)
			} else {
				value = arrayValue
			}
		} else {
			value = "nil"
		}
	case "object":
		// Use map[string]interface{} instead of any{}
		mapValue := fmt.Sprintf("map[string]%s{}", emptyInterface)
		if isPointer {
			value = fmt.Sprintf("&%s", mapValue)
		} else {
			value = mapValue
		}
	default:
		value = "\"test_value\""
	}

	// If field is a pointer, wrap with address-of operator
	if isPointer {
		// For pointer types, we need to take address of the value
		if value == "nil" {
			return "nil"
		}
		// Arrays and maps are already handled above with pointer logic
		// For primitives (string, int, bool), we need to create a variable first
		// In Go, we can't use &"string" directly, we need to use a helper variable
		// For struct literals, we can use: &value where value is a variable
		// But for string literals, we need: strPtr := "value"; &strPtr
		// Actually, in struct literals we can use: &[]string{...} or &map[string]interface{}{}
		// For string/int/bool, we need to create a temporary variable
		// However, Go allows: &"string" in some contexts but not in struct literals
		// The safest approach is to use a helper function or create variables
		// For now, let's use a pattern that works: create a variable name
		// Actually, we can use: &[]type{value} for arrays, &map[...]{...} for maps
		// For strings/ints/bools, we need to avoid &"literal" - use a temp variable pattern
		// But that's complex. Let's check if the value already has & prefix
		if strings.HasPrefix(value, "&") {
			return value // Already has address-of
		}
		// For string literals, we can't use &"string" in struct literals
		// We need to use a workaround: create a helper variable
		// But for now, let's use nil for optional string fields in tests
		// Or we can use: stringPtr := "value"; &stringPtr
		// Actually, the simplest is to use nil for optional fields in tests
		// Or create a helper: func stringPtr(s string) *string { return &s }
		// For now, let's just use nil for optional primitive fields
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			// It's a string literal - can't use &"string" in struct literal
			// Use nil instead for optional string fields
			return "nil"
		}
		// For other types (int, bool), same issue - use nil
		if value == "42" || value == "true" || value == "false" {
			return "nil"
		}
		// For arrays and maps, we already handled them above
		return fmt.Sprintf("&%s", value)
	}

	return value
}

// generateGoAPITest generates api_test.go with operation-based tests
func generateGoAPITest(data TemplateData, operations []APIOperation, extractedData *ExtractedData, version LanguageVersion) string {
	var test bytes.Buffer
	test.WriteString(fmt.Sprintf("package %s\n\n", data.SDKName))
	test.WriteString("import (\n")
	test.WriteString("\t\"net/http\"\n")
	test.WriteString("\t\"net/http/httptest\"\n")
	test.WriteString("\t\"testing\"\n")
	test.WriteString(")\n\n")

	// Group operations by tag for better organization
	operationsByTag := groupOperationsByTag(operations)

	// Generate tests for each tag/group
	for tag, tagOperations := range operationsByTag {
		test.WriteString(fmt.Sprintf("// Test%sAPI tests for %s API methods\n", toPascalCase(tag), tag))
		test.WriteString(fmt.Sprintf("func Test%sAPI(t *testing.T) {\n", toPascalCase(tag)))
		test.WriteString("\tt.Skip(\"TODO: Implement tests for this API group\")\n")
		test.WriteString("}\n\n")

		// Generate test for each operation
		for _, op := range tagOperations {
			methodName := GetOperationMethodName(op)
			testMethodName := fmt.Sprintf("Test%s_%s", toPascalCase(tag), toPascalCase(methodName))

			test.WriteString(fmt.Sprintf("func %s(t *testing.T) {\n", testMethodName))
			test.WriteString(fmt.Sprintf("\t// Test %s %s operation\n", op.Method, op.Path))
			test.WriteString("\tserver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {\n")

			// Determine expected status code from responses
			expectedStatus := 200 // http.StatusOK
			if _, ok := op.Responses["200"]; !ok {
				// Find first available status code
				for statusCode := range op.Responses {
					switch statusCode {
					case "201":
						expectedStatus = 201 // http.StatusCreated
					case "204":
						expectedStatus = 204 // http.StatusNoContent
					}
					break
				}
			}

			test.WriteString(fmt.Sprintf("\t\tw.WriteHeader(%d)\n", expectedStatus))

			// Use example from OpenAPI response if available
			statusCodeStr := fmt.Sprintf("%d", expectedStatus)
			exampleJSON := getGoExampleFromResponse(op.Responses[statusCodeStr])
			test.WriteString(fmt.Sprintf("\t\tw.Write([]byte(%s))\n", exampleJSON))
			test.WriteString("\t}))\n")
			test.WriteString("\tdefer server.Close()\n\n")

			test.WriteString(fmt.Sprintf("\tclient := New%s(server.URL)\n", data.ClientClassName))
			test.WriteString("\tif client == nil {\n")
			test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
			test.WriteString("\t}\n\n")

			// Generate method call with parameters
			test.WriteString("\t// Call API method\n")
			test.WriteString("\t// TODO: Uncomment and implement based on your API method signature\n")
			test.WriteString(fmt.Sprintf("\t// result, err := client.%s(", toPascalCase(methodName)))

			// Add path parameters
			hasParams := false
			for _, param := range op.Parameters {
				if param.In == "path" {
					if hasParams {
						test.WriteString(", ")
					}
					paramName := toPascalCase(param.Name)
					testValue := generateGoTestValueFromParam(param, version)
					test.WriteString(fmt.Sprintf("%s: %s", paramName, testValue))
					hasParams = true
				}
			}

			// Add query parameters
			for _, param := range op.Parameters {
				if param.In == "query" {
					if hasParams {
						test.WriteString(", ")
					}
					paramName := toPascalCase(param.Name)
					testValue := generateGoTestValueFromParam(param, version)
					test.WriteString(fmt.Sprintf("%s: %s", paramName, testValue))
					hasParams = true
				}
			}

			test.WriteString(")\n")
			test.WriteString("\t// if err != nil {\n")
			test.WriteString("\t//     t.Fatalf(\"Method call error = %v\", err)\n")
			test.WriteString("\t// }\n")
			test.WriteString("\t// if result == nil {\n")
			test.WriteString("\t//     t.Error(\"Result should not be nil\")\n")
			test.WriteString("\t// }\n")
			test.WriteString("}\n\n")

			// Generate error handling tests for 4xx/5xx responses
			generateGoErrorTests(&test, op, data, extractedData)
		}
	}

	return test.String()
}

// generateGoErrorTests generates error handling tests for 4xx/5xx responses
func generateGoErrorTests(test *bytes.Buffer, op APIOperation, data TemplateData, extractedData *ExtractedData) {
	// Find error responses (4xx, 5xx)
	var errorStatuses []string
	for statusCode := range op.Responses {
		if len(statusCode) >= 3 {
			firstDigit := statusCode[0]
			if firstDigit == '4' || firstDigit == '5' {
				errorStatuses = append(errorStatuses, statusCode)
			}
		}
	}

	if len(errorStatuses) == 0 {
		return // No error responses to test
	}

	methodName := GetOperationMethodName(op)

	// Generate test for each error status
	for _, statusCode := range errorStatuses {
		statusCodeInt := 0
		if _, err := fmt.Sscanf(statusCode, "%d", &statusCodeInt); err != nil || statusCodeInt == 0 {
			continue
		}

		testMethodName := fmt.Sprintf("Test%s_%s_%sError", toPascalCase(op.Tags[0]), toPascalCase(methodName), statusCode)
		fmt.Fprintf(test, "func %s(t *testing.T) {\n", testMethodName)
		fmt.Fprintf(test, "\t// Test %s %s operation returns %s error\n", op.Method, op.Path, statusCode)
		fmt.Fprintf(test, "\tserver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {\n")
		fmt.Fprintf(test, "\t\tw.WriteHeader(%d)\n", statusCodeInt)

		// Get error example if available
		errorResponse := op.Responses[statusCode]
		exampleJSON := getGoExampleFromResponse(errorResponse)
		fmt.Fprintf(test, "\t\tw.Write([]byte(%s))\n", exampleJSON)
		test.WriteString("\t}))\n")
		test.WriteString("\tdefer server.Close()\n\n")

		fmt.Fprintf(test, "\tclient := New%s(server.URL)\n", data.ClientClassName)
		test.WriteString("\tif client == nil {\n")
		test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
		test.WriteString("\t}\n\n")

		test.WriteString("\t// Call API method and expect error\n")
		fmt.Fprintf(test, "\t// result, err := client.%s(...)\n", toPascalCase(methodName))
		test.WriteString("\t// if err == nil {\n")
		test.WriteString("\t//     t.Error(\"Expected error but got nil\")\n")
		test.WriteString("\t// }\n")
		test.WriteString("}\n\n")
	}
}

// generateGoTestValueFromParam generates a test value from a parameter in Go
func generateGoTestValueFromParam(param Parameter, version LanguageVersion) string {
	if param.Schema == nil {
		return "\"test_value\""
	}
	// Parameters are typically not pointers (they're function parameters)
	return generateGoTestValue(param.Schema, param.Name, version, false)
}

// generateGoAuthTest generates auth_test.go with authentication tests
func generateGoAuthTest(data TemplateData, securitySchemes map[string]SecurityScheme, extractedData *ExtractedData, version LanguageVersion) string {
	var test bytes.Buffer
	test.WriteString(fmt.Sprintf("package %s\n\n", data.SDKName))
	test.WriteString("import (\n")
	test.WriteString("\t\"testing\"\n")
	test.WriteString(")\n\n")

	// Use base URL from OpenAPI spec, fallback to default
	baseURL := extractedData.BaseURL
	if baseURL == "" {
		baseURL = "https://api.example.com/v1"
	}

	test.WriteString("// TestAuthentication tests for authentication methods\n")
	test.WriteString("func TestAuthentication(t *testing.T) {\n")
	test.WriteString("\tt.Skip(\"TODO: Implement authentication tests\")\n")
	test.WriteString("}\n\n")

	// Generate tests for each security scheme
	for name, scheme := range securitySchemes {
		schemeName := toPascalCase(name)

		switch scheme.Type {
		case "apiKey":
			test.WriteString(fmt.Sprintf("func Test%s_APIKeyAuth(t *testing.T) {\n", schemeName))
			test.WriteString(fmt.Sprintf("\t// Test %s API key authentication\n", name))
			test.WriteString(fmt.Sprintf("\tclient := New%s(%q)\n", data.ClientClassName, baseURL))
			test.WriteString("\tif client == nil {\n")
			test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
			test.WriteString("\t}\n")
			// Use the security scheme name (not the header name) to match client field
			apiKeyField := toPascalCase(name)
			test.WriteString(fmt.Sprintf("\tclient.%s = \"test-api-key\"\n", apiKeyField))
			test.WriteString(fmt.Sprintf("\tif client.%s != \"test-api-key\" {\n", apiKeyField))
			test.WriteString("\t\tt.Error(\"API key should be set\")\n")
			test.WriteString("\t}\n")
			test.WriteString("}\n\n")

		case "http":
			switch scheme.Scheme {
			case "bearer":
				test.WriteString(fmt.Sprintf("func Test%s_BearerAuth(t *testing.T) {\n", schemeName))
				test.WriteString(fmt.Sprintf("\t// Test %s Bearer token authentication\n", name))
				test.WriteString(fmt.Sprintf("\tclient := New%s(%q)\n", data.ClientClassName, baseURL))
				test.WriteString("\tif client == nil {\n")
				test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
				test.WriteString("\t}\n")
				test.WriteString("\tclient.BearerToken = \"test-bearer-token\"\n")
				test.WriteString("\tif client.BearerToken != \"test-bearer-token\" {\n")
				test.WriteString("\t\tt.Error(\"Bearer token should be set\")\n")
				test.WriteString("\t}\n")
				test.WriteString("}\n\n")
			case "basic":
				test.WriteString(fmt.Sprintf("func Test%s_BasicAuth(t *testing.T) {\n", schemeName))
				test.WriteString(fmt.Sprintf("\t// Test %s Basic authentication\n", name))
				test.WriteString(fmt.Sprintf("\tclient := New%s(%q)\n", data.ClientClassName, baseURL))
				test.WriteString("\tif client == nil {\n")
				test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
				test.WriteString("\t}\n")
				test.WriteString("\tclient.Username = \"test-user\"\n")
				test.WriteString("\tclient.Password = \"test-password\"\n")
				test.WriteString("\tif client.Username != \"test-user\" {\n")
				test.WriteString("\t\tt.Error(\"Username should be set\")\n")
				test.WriteString("\t}\n")
				test.WriteString("\tif client.Password != \"test-password\" {\n")
				test.WriteString("\t\tt.Error(\"Password should be set\")\n")
				test.WriteString("\t}\n")
				test.WriteString("}\n\n")
			case "digest":
				test.WriteString(fmt.Sprintf("func Test%s_DigestAuth(t *testing.T) {\n", schemeName))
				test.WriteString(fmt.Sprintf("\t// Test %s Digest authentication\n", name))
				test.WriteString(fmt.Sprintf("\tclient := New%s(%q)\n", data.ClientClassName, baseURL))
				test.WriteString("\tif client == nil {\n")
				test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
				test.WriteString("\t}\n")
				test.WriteString("\tclient.Username = \"test-user\"\n")
				test.WriteString("\tclient.Password = \"test-password\"\n")
				test.WriteString("\tif client.Username != \"test-user\" {\n")
				test.WriteString("\t\tt.Error(\"Username should be set\")\n")
				test.WriteString("\t}\n")
				test.WriteString("\tif client.Password != \"test-password\" {\n")
				test.WriteString("\t\tt.Error(\"Password should be set\")\n")
				test.WriteString("\t}\n")
				test.WriteString("}\n\n")
			}

		case "oauth2":
			test.WriteString(fmt.Sprintf("func Test%s_OAuth2Auth(t *testing.T) {\n", schemeName))
			test.WriteString(fmt.Sprintf("\t// Test %s OAuth2 authentication\n", name))
			test.WriteString(fmt.Sprintf("\tclient := New%s(%q)\n", data.ClientClassName, baseURL))
			test.WriteString("\tif client == nil {\n")
			test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
			test.WriteString("\t}\n")
			test.WriteString("\tclient.Oauth2Token = \"test-oauth2-token\"\n")
			test.WriteString("\tif client.Oauth2Token != \"test-oauth2-token\" {\n")
			test.WriteString("\t\tt.Error(\"OAuth2 token should be set\")\n")
			test.WriteString("\t}\n")
			test.WriteString("}\n\n")

		case "openIdConnect":
			test.WriteString(fmt.Sprintf("func Test%s_OpenIDConnectAuth(t *testing.T) {\n", schemeName))
			test.WriteString(fmt.Sprintf("\t// Test %s OpenID Connect authentication\n", name))
			test.WriteString(fmt.Sprintf("\tclient := New%s(%q)\n", data.ClientClassName, baseURL))
			test.WriteString("\tif client == nil {\n")
			test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
			test.WriteString("\t}\n")
			test.WriteString("\tclient.OpenIdConnectToken = \"test-openid-token\"\n")
			test.WriteString("\tif client.OpenIdConnectToken != \"test-openid-token\" {\n")
			test.WriteString("\t\tt.Error(\"OpenID Connect token should be set\")\n")
			test.WriteString("\t}\n")
			test.WriteString("}\n\n")

		case "mutualTLS":
			test.WriteString(fmt.Sprintf("func Test%s_MutualTLSAuth(t *testing.T) {\n", schemeName))
			test.WriteString(fmt.Sprintf("\t// Test %s Mutual TLS authentication\n", name))
			test.WriteString(fmt.Sprintf("\tclient := New%s(%q)\n", data.ClientClassName, baseURL))
			test.WriteString("\tif client == nil {\n")
			test.WriteString("\t\tt.Fatal(\"Client is nil\")\n")
			test.WriteString("\t}\n")
			test.WriteString("\t// Note: Mutual TLS requires certificate configuration\n")
			test.WriteString("\t// This test verifies the client can be instantiated\n")
			test.WriteString("\t_ = client\n")
			test.WriteString("}\n\n")
		}
	}

	return test.String()
}

// getGoExampleFromResponse extracts example from response for Go test
func getGoExampleFromResponse(response Response) string {
	// Look for JSON content type first
	if jsonContent, ok := response.Content["application/json"]; ok {
		if len(jsonContent.Examples) > 0 {
			// Use first example
			for _, example := range jsonContent.Examples {
				return formatExampleForGo(example)
			}
		}
		// Fallback: generate example from schema if no examples
		if jsonContent.Schema != nil {
			return generateGoExampleFromSchema(jsonContent.Schema)
		}
	}
	// Default fallback
	return "`{\"success\": true}`"
}

// formatExampleForGo converts an example value to Go code string
func formatExampleForGo(example interface{}) string {
	if example == nil {
		return "`null`"
	}

	// Use JSON encoding to properly format the example
	jsonBytes, err := json.Marshal(example)
	if err != nil {
		// Fallback: escape and quote
		return fmt.Sprintf("%q", fmt.Sprintf("%v", example))
	}

	// Return as Go raw string literal
	return fmt.Sprintf("`%s`", string(jsonBytes))
}

// generateGoExampleFromSchema generates a Go example from schema
func generateGoExampleFromSchema(schema *Schema) string {
	if schema == nil {
		return "`{}`"
	}

	// For now, return a simple example based on type
	// This can be enhanced to generate more complex examples
	switch schema.Type {
	case "object":
		return "`{}`"
	case "array":
		return "`[]`"
	default:
		return "`{\"value\": \"test\"}`"
	}
}
