// Package generator provides code generation functionality for SDKs.
package generator

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/getkin/kin-openapi/openapi3"

	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

// getClientClassName removes common suffixes (sdk, api, client) from SDK name and converts to PascalCase
// This ensures class names like "PetstoreClient" instead of "PetstoreSdkClient"
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

// GeneratePythonSDK generates a Python SDK
// If version is nil, uses the default Python version
// If sdkVersion is empty, extracts from OpenAPI schema or defaults to "1.0.0"
func GeneratePythonSDK(
	outputPath, sdkName, httpLib string,
	openAPIDoc interface{},
	version *LanguageVersion,
	sdkVersion string,
) error {
	// Use default version if not provided
	if version == nil {
		defaultVersion := GetPythonDefaultVersion()
		version = &defaultVersion
	}

	// Convert openAPIDoc to *openapi3.T
	doc, ok := openAPIDoc.(*openapi3.T)
	if !ok {
		// If not an openapi3.T, try to extract from ExtractedData
		if extractedData, ok := openAPIDoc.(*ExtractedData); ok {
			return generatePythonSDKFromExtracted(outputPath, sdkName, httpLib, extractedData, *version, sdkVersion)
		}
		return fmt.Errorf("invalid OpenAPI document type")
	}

	// Extract data from OpenAPI document
	extractedData, err := ExtractOpenAPIData(doc)
	if err != nil {
		return fmt.Errorf("failed to extract OpenAPI data: %w", err)
	}

	return generatePythonSDKFromExtracted(outputPath, sdkName, httpLib, extractedData, *version, sdkVersion)
}

// generatePythonSDKFromExtracted generates SDK from extracted data
func generatePythonSDKFromExtracted(
	outputPath, sdkName, httpLib string,
	extractedData *ExtractedData,
	version LanguageVersion,
	sdkVersion string,
) error {
	// Get HTTP library config
	libConfig, err := httplib.GetLibraryConfig("python", httpLib)
	if err != nil {
		return fmt.Errorf("failed to get HTTP library config: %w", err)
	}

	// Sanitize SDK name for Python (snake_case)
	sanitizedName := toSnakeCase(sdkName)
	packageDir := filepath.Join(outputPath, sanitizedName)

	// Create package directory
	if err := os.MkdirAll(packageDir, 0750); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Template data
	data := TemplateData{
		SDKName:         sanitizedName,
		Language:        "python",
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

	// Generate __init__.py
	initContent := generatePythonInit(data, finalSDKVersion)
	initPath := filepath.Join(packageDir, "__init__.py")
	// #nosec G306 -- 0644 is appropriate for Python package files
	if err := os.WriteFile(initPath, []byte(initContent), 0644); err != nil {
		return fmt.Errorf("failed to write __init__.py: %w", err)
	}
	// Format with black (if available)
	if err := formatPythonFile(initPath); err != nil {
		// Log but don't fail - formatting is nice-to-have
		_ = err
	}

	// Generate models.py if schemas exist
	if len(extractedData.Schemas) > 0 {
		modelsContent := generatePythonModels(extractedData.Schemas)
		modelsPath := filepath.Join(packageDir, "models.py")
		// #nosec G306 -- 0644 is appropriate for Python package files
		if err := os.WriteFile(modelsPath, []byte(modelsContent), 0644); err != nil {
			return fmt.Errorf("failed to write models.py: %w", err)
		}
		// Format with black (if available)
		if err := formatPythonFile(modelsPath); err != nil {
			// Log but don't fail - formatting is nice-to-have
			_ = err
		}
	}

	// Generate client.py
	clientContent := generatePythonClient(data)
	clientPath := filepath.Join(packageDir, "client.py")
	// #nosec G306 -- 0644 is appropriate for Python source files
	if err := os.WriteFile(clientPath, []byte(clientContent), 0644); err != nil {
		return fmt.Errorf("failed to write client.py: %w", err)
	}
	// Format with black (if available)
	if err := formatPythonFile(clientPath); err != nil {
		// Log but don't fail - formatting is nice-to-have
		_ = err
	}

	// Generate requirements.txt
	requirementsContent := generatePythonRequirements(data)
	requirementsPath := filepath.Join(outputPath, "requirements.txt")
	// #nosec G306 -- 0644 is appropriate for requirements.txt
	if err := os.WriteFile(requirementsPath, []byte(requirementsContent), 0644); err != nil {
		return fmt.Errorf("failed to write requirements.txt: %w", err)
	}

	// Generate README.md
	readmeContent := generatePythonREADME(data)
	readmePath := filepath.Join(outputPath, "README.md")
	// #nosec G306 -- 0644 is appropriate for README files
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	// Generate setup.py
	setupContent := generatePythonSetup(data, version, finalSDKVersion)
	setupPath := filepath.Join(outputPath, "setup.py")
	// #nosec G306 -- 0644 is appropriate for setup.py
	if err := os.WriteFile(setupPath, []byte(setupContent), 0644); err != nil {
		return fmt.Errorf("failed to write setup.py: %w", err)
	}

	// Generate api/ directory with endpoint modules organized by tags
	if len(extractedData.Operations) > 0 {
		apiDir := filepath.Join(packageDir, "api")
		if err := os.MkdirAll(apiDir, 0750); err != nil {
			return fmt.Errorf("failed to create api directory: %w", err)
		}

		// Generate api/__init__.py
		apiInitContent := generatePythonAPIInit(data)
		apiInitPath := filepath.Join(apiDir, "__init__.py")
		// #nosec G306 -- 0644 is appropriate for Python package files
		if err := os.WriteFile(apiInitPath, []byte(apiInitContent), 0644); err != nil {
			return fmt.Errorf("failed to write api/__init__.py: %w", err)
		}
		// Format with black (if available)
		if err := formatPythonFile(apiInitPath); err != nil {
			// Log but don't fail - formatting is nice-to-have
			_ = err
		}

		// Group operations by tags
		operationsByTag := groupOperationsByTag(extractedData.Operations)
		for tag, operations := range operationsByTag {
			// Generate api/{tag}.py
			tagFileName := toSnakeCase(tag) + ".py"
			tagContent := generatePythonAPIModule(tag, operations, data)
			tagPath := filepath.Join(apiDir, tagFileName)
			// #nosec G306 -- 0644 is appropriate for Python source files
			if err := os.WriteFile(tagPath, []byte(tagContent), 0644); err != nil {
				return fmt.Errorf("failed to write api/%s: %w", tagFileName, err)
			}
			// Format with black (if available)
			if err := formatPythonFile(tagPath); err != nil {
				// Log but don't fail - formatting is nice-to-have
				_ = err
			}
		}
	}

	// Generate examples/ directory
	examplesDir := filepath.Join(outputPath, "examples")
	if err := os.MkdirAll(examplesDir, 0750); err != nil {
		return fmt.Errorf("failed to create examples directory: %w", err)
	}

	// Generate examples/basic_usage.py
	examplesContent := generatePythonExamples(data)
	examplesPath := filepath.Join(examplesDir, "basic_usage.py")
	// #nosec G306 -- 0644 is appropriate for Python example files
	if err := os.WriteFile(examplesPath, []byte(examplesContent), 0644); err != nil {
		return fmt.Errorf("failed to write examples/basic_usage.py: %w", err)
	}
	// Format with black (if available)
	if err := formatPythonFile(examplesPath); err != nil {
		// Log but don't fail - formatting is nice-to-have
		_ = err
	}

	return nil
}

// formatPythonFile formats a Python source file using black (if available)
func formatPythonFile(filePath string) error {
	// Try black first (most common Python formatter)
	cmd := exec.Command("black", "--quiet", filePath)
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Fallback to autopep8 if black is not available
	cmd = exec.Command("autopep8", "--in-place", "--aggressive", "--aggressive", filePath)
	if err := cmd.Run(); err == nil {
		return nil
	}

	// If neither formatter is available, that's okay - just skip formatting
	return fmt.Errorf("no Python formatter available (black or autopep8)")
}

func generatePythonInit(data TemplateData, sdkVersion string) string {
	// Check if we have models
	extractedData, ok := data.OpenAPIDoc.(*ExtractedData)
	hasModels := ok && extractedData != nil && len(extractedData.Schemas) > 0

	// Format SDK name for display
	displayName := strings.ReplaceAll(data.SDKName, "_", " ")
	c := cases.Title(language.English)
	displayName = c.String(strings.ToLower(displayName))

	// Prepare template data
	type PythonInitData struct {
		DisplayName     string
		ClientClassName string
		HasModels       bool
		ModelNames      []string
		Version         string
	}
	templateData := PythonInitData{
		DisplayName:     displayName,
		ClientClassName: getClientClassName(data.SDKName),
		HasModels:       hasModels,
		Version:         sdkVersion,
	}
	if hasModels {
		for name := range extractedData.Schemas {
			templateData.ModelNames = append(templateData.ModelNames, toPascalCase(name))
		}
	}

	// Load and render template
	tmpl, err := LoadTemplate(GetPythonInitTemplate())
	if err != nil {
		// Fallback to old method
		var initContent strings.Builder
		initContent.WriteString(fmt.Sprintf("\"\"\"%s SDK - Auto-generated from OpenAPI schema\"\"\"\n\n", displayName))
		initContent.WriteString(fmt.Sprintf("from .client import %s\n", getClientClassName(data.SDKName)))
		if hasModels {
			initContent.WriteString("from .models import *\n")
		}
		initContent.WriteString(fmt.Sprintf("\n__all__ = [\"%s\"", getClientClassName(data.SDKName)))
		if hasModels {
			for name := range extractedData.Schemas {
				initContent.WriteString(fmt.Sprintf(", \"%s\"", toPascalCase(name)))
			}
		}
		initContent.WriteString(fmt.Sprintf("]\n__version__ = \"%s\"\n", sdkVersion))
		return initContent.String()
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		// Fallback to old method
		var initContent strings.Builder
		initContent.WriteString(fmt.Sprintf("\"\"\"%s SDK - Auto-generated from OpenAPI schema\"\"\"\n\n", displayName))
		initContent.WriteString(fmt.Sprintf("from .client import %s\n", getClientClassName(data.SDKName)))
		if hasModels {
			initContent.WriteString("from .models import *\n")
		}
		initContent.WriteString(fmt.Sprintf("\n__all__ = [\"%s\"", getClientClassName(data.SDKName)))
		if hasModels {
			for name := range extractedData.Schemas {
				initContent.WriteString(fmt.Sprintf(", \"%s\"", toPascalCase(name)))
			}
		}
		initContent.WriteString(fmt.Sprintf("]\n__version__ = \"%s\"\n", sdkVersion))
		return initContent.String()
	}

	return buf.String()
}

func generatePythonClient(data TemplateData) string {
	// Build template with proper replacements
	clientClass := "requests.Session"
	if data.HTTPLibConfig != nil {
		clientClass = data.HTTPLibConfig.ClientClass
	}

	// Format SDK name for display (replace underscores with spaces, then title case)
	displayName := strings.ReplaceAll(data.SDKName, "_", " ")
	c := cases.Title(language.English)
	displayName = c.String(strings.ToLower(displayName))

	// Extract OpenAPI data
	extractedData, ok := data.OpenAPIDoc.(*ExtractedData)
	if !ok {
		extractedData = &ExtractedData{BaseURL: "https://api.example.com/v1"}
	}

	// Generate base URL default
	baseURLDefault := extractedData.BaseURL
	if baseURLDefault == "" {
		baseURLDefault = "https://api.example.com/v1"
	}

	// Generate authentication setup
	authSetup := generatePythonAuthSetup(extractedData.SecuritySchemes)

	// Generate API methods
	apiMethods := generatePythonAPIMethods(extractedData.Operations)

	// Prepare template data
	type PythonClientData struct {
		DisplayName     string
		HTTPLibImport   string
		ClientClassName string
		BaseURLDefault  string
		ClientClass     string
		AuthSetup       string
		APIMethods      string
	}
	templateData := PythonClientData{
		DisplayName:     displayName,
		HTTPLibImport:   data.HTTPLibImport,
		ClientClassName: getClientClassName(data.SDKName),
		BaseURLDefault:  baseURLDefault,
		ClientClass:     clientClass,
		AuthSetup:       authSetup,
		APIMethods:      apiMethods,
	}

	// Load and render template
	tmpl, err := LoadTemplate(GetPythonClientTemplate())
	if err != nil {
		// Fallback to old method
		return fmt.Sprintf(`"""%s API Client - Auto-generated from OpenAPI schema"""

import %s
from typing import Optional, Dict, Any


class %s:
    """Client for %s API"""
    
    def __init__(self, base_url: str = "%s", **auth_params):
        """
        Initialize the client
        
        Args:
            base_url: Base URL for the API
            **auth_params: Authentication parameters (api_key, bearer_token, etc.)
        """
        self.base_url = base_url.rstrip('/')
        self.session = %s()
        
%s
    
    def _request(self, method: str, path: str, **kwargs):
        """Make an HTTP request"""
        url = f"{self.base_url}{path}"
        response = self.session.request(method, url, **kwargs)
        response.raise_for_status()
        return response.json()
    
%s
`,
			displayName,
			data.HTTPLibImport,
			getClientClassName(data.SDKName),
			displayName,
			baseURLDefault,
			clientClass,
			authSetup,
			apiMethods)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		// Fallback to old method
		return fmt.Sprintf(`"""%s API Client - Auto-generated from OpenAPI schema"""

import %s
from typing import Optional, Dict, Any


class %s:
    """Client for %s API"""
    
    def __init__(self, base_url: str = "%s", **auth_params):
        """
        Initialize the client
        
        Args:
            base_url: Base URL for the API
            **auth_params: Authentication parameters (api_key, bearer_token, etc.)
        """
        self.base_url = base_url.rstrip('/')
        self.session = %s()
        
%s
    
    def _request(self, method: str, path: str, **kwargs):
        """Make an HTTP request"""
        url = f"{self.base_url}{path}"
        response = self.session.request(method, url, **kwargs)
        response.raise_for_status()
        return response.json()
    
%s
`,
			displayName,
			data.HTTPLibImport,
			getClientClassName(data.SDKName),
			displayName,
			baseURLDefault,
			clientClass,
			authSetup,
			apiMethods)
	}

	return buf.String()
}

// generatePythonAuthSetup generates authentication setup code
func generatePythonAuthSetup(securitySchemes map[string]SecurityScheme) string {
	if len(securitySchemes) == 0 {
		return "        # No authentication required"
	}

	var setupCode strings.Builder
	if len(securitySchemes) > 0 {
		setupCode.WriteString("        # Set up authentication\n")
	}

	for name, scheme := range securitySchemes {
		switch scheme.Type {
		case securitySchemeAPIKey:
			setupCode.WriteString(fmt.Sprintf("        self.%s = auth_params.get('%s')\n", name, name))
			setupCode.WriteString(fmt.Sprintf("        if self.%s:\n", name))
			switch scheme.In {
			case paramLocationHeader:
				setupCode.WriteString(fmt.Sprintf("            self.session.headers['%s'] = self.%s\n", scheme.Name, name))
			case paramLocationQuery:
				setupCode.WriteString(fmt.Sprintf("            # Query parameter '%s' will be added per request\n", scheme.Name))
			}
		case securitySchemeHTTP:
			switch scheme.Scheme {
			case securitySchemeBearer:
				setupCode.WriteString("        self.bearer_token = auth_params.get('bearer_token')\n")
				setupCode.WriteString("        if self.bearer_token:\n")
				setupCode.WriteString("            self.session.headers['Authorization'] = f'Bearer {self.bearer_token}'\n")
			case securitySchemeBasic:
				setupCode.WriteString("        self.username = auth_params.get('username')\n")
				setupCode.WriteString("        self.password = auth_params.get('password')\n")
				setupCode.WriteString("        if self.username and self.password:\n")
				setupCode.WriteString("            from requests.auth import HTTPBasicAuth\n")
				setupCode.WriteString("            self.session.auth = HTTPBasicAuth(self.username, self.password)\n")
			case securitySchemeDigest:
				setupCode.WriteString("        self.username = auth_params.get('username')\n")
				setupCode.WriteString("        self.password = auth_params.get('password')\n")
				setupCode.WriteString("        if self.username and self.password:\n")
				setupCode.WriteString("            from requests.auth import HTTPDigestAuth\n")
				setupCode.WriteString("            self.session.auth = HTTPDigestAuth(self.username, self.password)\n")
			}
		case securitySchemeOAuth2:
			setupCode.WriteString(fmt.Sprintf("        self.%s_token = auth_params.get('%s_token')\n", name, name))
			tokenTypeLine := fmt.Sprintf("        self.%s_token_type = auth_params.get('%s_token_type', 'Bearer')\n", name, name)
			setupCode.WriteString(tokenTypeLine)
			setupCode.WriteString(fmt.Sprintf("        if self.%s_token:\n", name))
			authFmt := "            self.session.headers['Authorization'] = " +
				"f'{self.%s_token_type} {self.%s_token}'\n"
			authLine := fmt.Sprintf(authFmt, name, name)
			setupCode.WriteString(authLine)
			// Add OAuth2 flow information as comments
			if scheme.OAuth2Flows != nil {
				setupCode.WriteString("        # OAuth2 flows available: ")
				var flows []string
				if scheme.OAuth2Flows.AuthorizationCode != nil {
					flows = append(flows, "authorizationCode")
				}
				if scheme.OAuth2Flows.ClientCredentials != nil {
					flows = append(flows, "clientCredentials")
				}
				if scheme.OAuth2Flows.Implicit != nil {
					flows = append(flows, "implicit")
				}
				if scheme.OAuth2Flows.Password != nil {
					flows = append(flows, "password")
				}
				if len(flows) > 0 {
					setupCode.WriteString(strings.Join(flows, ", "))
				} else {
					setupCode.WriteString("(configured)")
				}
				setupCode.WriteString("\n")
			}
		case securitySchemeOpenIDConnect:
			setupCode.WriteString(fmt.Sprintf("        self.%s_token = auth_params.get('%s_token')\n", name, name))
			setupCode.WriteString(fmt.Sprintf("        if self.%s_token:\n", name))
			bearerLine := fmt.Sprintf("            self.session.headers['Authorization'] = f'Bearer {self.%s_token}'\n", name)
			setupCode.WriteString(bearerLine)
			if scheme.OpenIDConnectURL != "" {
				setupCode.WriteString(fmt.Sprintf("        # OpenID Connect discovery URL: %s\n", scheme.OpenIDConnectURL))
			}
		case securitySchemeMutualTLS:
			setupCode.WriteString(fmt.Sprintf("        self.%s_cert = auth_params.get('%s_cert')\n", name, name))
			setupCode.WriteString(fmt.Sprintf("        self.%s_key = auth_params.get('%s_key')\n", name, name))
			setupCode.WriteString(fmt.Sprintf("        if self.%s_cert and self.%s_key:\n", name, name))
			setupCode.WriteString(fmt.Sprintf("            self.session.cert = (self.%s_cert, self.%s_key)\n", name, name))
		}
	}

	return setupCode.String()
}

// generatePythonAPIMethods generates API methods from operations
func generatePythonAPIMethods(operations []APIOperation) string {
	if len(operations) == 0 {
		return "    # No API methods defined in OpenAPI schema"
	}

	var methods strings.Builder

	for _, op := range operations {
		methodName := GetOperationMethodName(op)
		if methodName == "" {
			// Fallback naming
			methodName = strings.ToLower(op.Method) + "_" + strings.ReplaceAll(strings.Trim(op.Path, "/"), "/", "_")
			methodName = toSnakeCase(methodName)
		}

		// Generate method signature
		methods.WriteString(fmt.Sprintf("    def %s(self", methodName))

		// Add path parameters
		var pathParams []string
		var queryParams []string
		for _, param := range op.Parameters {
			switch param.In {
			case paramLocationPath:
				pathParams = append(pathParams, param.Name)
				methods.WriteString(fmt.Sprintf(", %s: %s", param.Name, getPythonType(param.Schema)))
			case paramLocationQuery:
				queryParams = append(queryParams, param.Name)
				// Add query parameters to function signature
				methods.WriteString(fmt.Sprintf(", %s: Optional[%s] = None", param.Name, getPythonType(param.Schema)))
			}
		}

		// Add request body parameter
		if op.RequestBody != nil {
			methods.WriteString(", body: Optional[Dict[str, Any]] = None")
		}

		methods.WriteString(") -> Dict[str, Any]:\n")

		// Generate docstring
		if op.Summary != "" || op.Description != "" {
			methods.WriteString("        \"\"\"\n")
			if op.Summary != "" {
				methods.WriteString(fmt.Sprintf("        %s\n\n", op.Summary))
			}
			if op.Description != "" {
				methods.WriteString(fmt.Sprintf("        %s\n\n", op.Description))
			}
			methods.WriteString("        Args:\n")
			for _, param := range op.Parameters {
				if param.In == "path" || param.In == "query" {
					desc := param.Description
					if desc == "" {
						desc = param.Name
					}
					methods.WriteString(fmt.Sprintf("            %s: %s\n", param.Name, desc))
				}
			}
			if op.RequestBody != nil {
				methods.WriteString("            body: Request body data\n")
			}
			methods.WriteString("\n        Returns:\n")
			methods.WriteString("            Response data as dictionary\n")
			methods.WriteString("        \"\"\"\n")
		}

		// Build path with parameters
		path := op.Path
		for _, param := range pathParams {
			path = strings.ReplaceAll(path, fmt.Sprintf("{%s}", param), fmt.Sprintf("{%s}", param))
		}

		// Generate method body
		methods.WriteString(fmt.Sprintf("        path = \"%s\"\n", path))
		for _, param := range pathParams {
			methods.WriteString(fmt.Sprintf("        path = path.replace('{%s}', str(%s))\n", param, param))
		}

		// Add query parameters
		if len(queryParams) > 0 {
			methods.WriteString("        params = {}\n")
			for _, param := range queryParams {
				methods.WriteString(fmt.Sprintf("        if %s is not None:\n", param))
				methods.WriteString(fmt.Sprintf("            params['%s'] = %s\n", param, param))
			}
			methods.WriteString(fmt.Sprintf("        return self._request('%s', path, params=params", op.Method))
		} else {
			methods.WriteString(fmt.Sprintf("        return self._request('%s', path", op.Method))
		}

		// Add request body
		if op.RequestBody != nil {
			methods.WriteString(", json=body")
		}

		methods.WriteString(")\n\n")
	}

	return methods.String()
}

// getPythonType converts schema type to Python type hint
func getPythonType(schema *Schema) string {
	if schema == nil {
		return pythonTypeAny
	}

	switch schema.Type {
	case pythonTypeString:
		return "str"
	case pythonTypeInteger:
		return "int"
	case pythonTypeNumber:
		return "float"
	case pythonTypeBoolean:
		return "bool"
	case pythonTypeArray:
		return "list"
	case pythonTypeObject:
		return pythonTypeDict
	default:
		return pythonTypeAny
	}
}

func generatePythonRequirements(data TemplateData) string {
	if data.HTTPLibConfig == nil {
		return "# Dependencies\n"
	}

	requirements := []string{
		data.HTTPLibConfig.Dependency,
		"# Add other dependencies as needed",
	}

	return strings.Join(requirements, "\n") + "\n"
}

func generatePythonREADME(data TemplateData) string {
	// Load and render template
	tmpl, err := LoadTemplate(GetPythonReadmeTemplate())
	if err != nil {
		// Fallback
		c := cases.Title(language.English)
		titleName := c.String(strings.ToLower(data.SDKName))
		return fmt.Sprintf("# %s SDK\n\nAuto-generated Python SDK\n", titleName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		// Fallback
		c := cases.Title(language.English)
		titleName := c.String(strings.ToLower(data.SDKName))
		return fmt.Sprintf("# %s SDK\n\nAuto-generated Python SDK\n", titleName)
	}

	return buf.String()
}

// generatePythonSetup generates setup.py for Python package
func generatePythonSetup(data TemplateData, pythonVersion LanguageVersion, sdkVersion string) string {
	extractedData, ok := data.OpenAPIDoc.(*ExtractedData)
	description := "Auto-generated Python SDK"
	if ok && extractedData != nil {
		if extractedData.Description != "" {
			description = extractedData.Description
		}
	}

	// Prepare template data
	type PythonSetupData struct {
		SDKName        string
		Version        string
		Description    string
		PythonVersion  string
		PythonRequires string
		Classifiers    []string
	}

	// Generate Python version classifiers
	classifiers := []string{
		"Development Status :: 4 - Beta",
		"Intended Audience :: Developers",
		"Programming Language :: Python :: 3",
	}
	// Add version-specific classifiers (from the selected minimum version up to available versions)
	// Only include versions that meet the minimum requirement
	availableVersions := GetPythonAvailableVersions()
	for _, availableVersion := range availableVersions {
		// Only include versions >= the selected minimum version
		if availableVersion.Major == pythonVersion.Major &&
			availableVersion.Minor >= pythonVersion.Minor {
			classifiers = append(classifiers, fmt.Sprintf("Programming Language :: Python :: 3.%d", availableVersion.Minor))
		}
	}

	templateData := PythonSetupData{
		SDKName:        data.SDKName,
		Version:        sdkVersion,
		Description:    description,
		PythonVersion:  fmt.Sprintf("%d.%d", pythonVersion.Major, pythonVersion.Minor),
		PythonRequires: fmt.Sprintf(">=%d.%d", pythonVersion.Major, pythonVersion.Minor),
		Classifiers:    classifiers,
	}

	// Load and render template
	tmpl, err := LoadTemplate(GetPythonSetupTemplate())
	if err != nil {
		// Fallback to old method
		var setup strings.Builder
		setup.WriteString("from setuptools import setup, find_packages\n\n")
		setup.WriteString("setup(\n")
		setup.WriteString(fmt.Sprintf("    name=\"%s\",\n", data.SDKName))
		setup.WriteString(fmt.Sprintf("    version=\"%s\",\n", sdkVersion))
		setup.WriteString(fmt.Sprintf("    description=\"%s\",\n", description))
		setup.WriteString("    long_description=open(\"README.md\").read(),\n")
		setup.WriteString("    long_description_content_type=\"text/markdown\",\n")
		setup.WriteString("    author=\"SDK Forge\",\n")
		setup.WriteString("    author_email=\"support@example.com\",\n")
		setup.WriteString("    url=\"https://github.com/example/sdk\",\n")
		setup.WriteString("    packages=find_packages(),\n")
		setup.WriteString("    install_requires=open(\"requirements.txt\").read().splitlines(),\n")
		setup.WriteString(fmt.Sprintf("    python_requires=\">=%d.%d\",\n", pythonVersion.Major, pythonVersion.Minor))
		setup.WriteString("    classifiers=[\n")
		setup.WriteString("        \"Development Status :: 4 - Beta\",\n")
		setup.WriteString("        \"Intended Audience :: Developers\",\n")
		setup.WriteString("        \"Programming Language :: Python :: 3\",\n")
		// Only include versions >= the selected minimum version
		availableVersions := GetPythonAvailableVersions()
		for _, availableVersion := range availableVersions {
			if availableVersion.Major == pythonVersion.Major &&
				availableVersion.Minor >= pythonVersion.Minor {
				setup.WriteString(fmt.Sprintf("        \"Programming Language :: Python :: 3.%d\",\n", availableVersion.Minor))
			}
		}
		setup.WriteString("    ],\n")
		setup.WriteString(")\n")
		return setup.String()
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		// Fallback to old method
		var setup strings.Builder
		setup.WriteString("from setuptools import setup, find_packages\n\n")
		setup.WriteString("setup(\n")
		setup.WriteString(fmt.Sprintf("    name=\"%s\",\n", data.SDKName))
		setup.WriteString(fmt.Sprintf("    version=\"%s\",\n", sdkVersion))
		setup.WriteString(fmt.Sprintf("    description=\"%s\",\n", description))
		setup.WriteString("    long_description=open(\"README.md\").read(),\n")
		setup.WriteString("    long_description_content_type=\"text/markdown\",\n")
		setup.WriteString("    author=\"SDK Forge\",\n")
		setup.WriteString("    author_email=\"support@example.com\",\n")
		setup.WriteString("    url=\"https://github.com/example/sdk\",\n")
		setup.WriteString("    packages=find_packages(),\n")
		setup.WriteString("    install_requires=open(\"requirements.txt\").read().splitlines(),\n")
		setup.WriteString(fmt.Sprintf("    python_requires=\">=%d.%d\",\n", pythonVersion.Major, pythonVersion.Minor))
		setup.WriteString("    classifiers=[\n")
		setup.WriteString("        \"Development Status :: 4 - Beta\",\n")
		setup.WriteString("        \"Intended Audience :: Developers\",\n")
		setup.WriteString("        \"Programming Language :: Python :: 3\",\n")
		// Only include versions >= the selected minimum version
		availableVersions := GetPythonAvailableVersions()
		for _, availableVersion := range availableVersions {
			if availableVersion.Major == pythonVersion.Major &&
				availableVersion.Minor >= pythonVersion.Minor {
				setup.WriteString(fmt.Sprintf("        \"Programming Language :: Python :: 3.%d\",\n", availableVersion.Minor))
			}
		}
		setup.WriteString("    ],\n")
		setup.WriteString(")\n")
		return setup.String()
	}

	return buf.String()
}

// generatePythonAPIInit generates api/__init__.py
func generatePythonAPIInit(data TemplateData) string {
	extractedData, ok := data.OpenAPIDoc.(*ExtractedData)
	if !ok || extractedData == nil || len(extractedData.Operations) == 0 {
		return "\"\"\"API endpoint modules\"\"\"\n"
	}

	operationsByTag := groupOperationsByTag(extractedData.Operations)
	var init strings.Builder
	init.WriteString("\"\"\"API endpoint modules\"\"\"\n\n")

	// Import all tag modules
	for tag := range operationsByTag {
		tagModule := toSnakeCase(tag)
		init.WriteString(fmt.Sprintf("from . import %s\n", tagModule))
	}

	init.WriteString("\n__all__ = [\n")
	first := true
	for tag := range operationsByTag {
		if !first {
			init.WriteString(",\n")
		}
		tagModule := toSnakeCase(tag)
		init.WriteString(fmt.Sprintf("    \"%s\"", tagModule))
		first = false
	}
	init.WriteString("\n]\n")

	return init.String()
}

// generatePythonAPIModule generates api/{tag}.py with operations for that tag
func generatePythonAPIModule(tag string, operations []APIOperation, data TemplateData) string {
	clientClassName := getClientClassName(data.SDKName)
	var module strings.Builder
	module.WriteString(fmt.Sprintf("\"\"\"%s API endpoints\"\"\"\n\n", tag))
	module.WriteString("from typing import Optional, Dict, Any\n")
	module.WriteString(fmt.Sprintf("from ..client import %s\n\n\n", clientClassName))

	// Generate functions for each operation
	for _, op := range operations {
		methodName := GetOperationMethodName(op)
		if methodName == "" {
			// Fallback naming
			pathPart := strings.ReplaceAll(strings.Trim(op.Path, "/"), "/", "_")
			methodName = strings.ToLower(op.Method) + "_" + pathPart
			methodName = toSnakeCase(methodName)
		}

		// Function signature - takes client as first parameter
		module.WriteString(fmt.Sprintf("def %s(client: %s", methodName, clientClassName))

		// Add parameters
		var pathParams []string
		var queryParams []string
		for _, param := range op.Parameters {
			switch param.In {
			case paramLocationPath:
				pathParams = append(pathParams, param.Name)
				paramType := getPythonType(param.Schema)
				module.WriteString(fmt.Sprintf(", %s: %s", param.Name, paramType))
			case paramLocationQuery:
				queryParams = append(queryParams, param.Name)
				paramType := getPythonType(param.Schema)
				module.WriteString(fmt.Sprintf(", %s: Optional[%s] = None", param.Name, paramType))
			}
		}

		// Add request body
		if op.RequestBody != nil {
			module.WriteString(", body: Optional[Dict[str, Any]] = None")
		}

		module.WriteString(") -> Dict[str, Any]:\n")
		module.WriteString("    \"\"\"\n")
		if op.Summary != "" {
			module.WriteString(fmt.Sprintf("    %s\n\n", op.Summary))
		}
		if op.Description != "" {
			module.WriteString(fmt.Sprintf("    %s\n\n", op.Description))
		}
		module.WriteString("    \"\"\"\n")

		// Build path
		path := op.Path
		for _, param := range pathParams {
			path = strings.ReplaceAll(path, fmt.Sprintf("{%s}", param), fmt.Sprintf("{%s}", param))
		}

		// Function body
		module.WriteString(fmt.Sprintf("    path = \"%s\"\n", path))
		for _, param := range pathParams {
			module.WriteString(fmt.Sprintf("    path = path.replace('{%s}', str(%s))\n", param, param))
		}

		// Query parameters
		if len(queryParams) > 0 {
			module.WriteString("    params = {}\n")
			for _, param := range queryParams {
				module.WriteString(fmt.Sprintf("    if %s is not None:\n", param))
				module.WriteString(fmt.Sprintf("        params['%s'] = %s\n", param, param))
			}
			module.WriteString(fmt.Sprintf("    return client._request('%s', path, params=params", op.Method))
		} else {
			module.WriteString(fmt.Sprintf("    return client._request('%s', path", op.Method))
		}

		// Request body
		if op.RequestBody != nil {
			module.WriteString(", json=body")
		}

		module.WriteString(")\n\n")
	}

	return module.String()
}

// generatePythonExamples generates examples/basic_usage.py
func generatePythonExamples(data TemplateData) string {
	extractedData, ok := data.OpenAPIDoc.(*ExtractedData)
	clientClassName := getClientClassName(data.SDKName)

	var examples strings.Builder
	examples.WriteString("\"\"\"Basic usage examples for the SDK\"\"\"\n\n")
	examples.WriteString(fmt.Sprintf("from %s import %s\n", data.SDKName, clientClassName))
	examples.WriteString(fmt.Sprintf("from %s.models import *\n\n\n", data.SDKName))
	examples.WriteString("# Initialize the client\n")
	examples.WriteString(fmt.Sprintf("client = %s(\n", clientClassName))

	// Add base URL
	if ok && extractedData != nil && extractedData.BaseURL != "" {
		examples.WriteString(fmt.Sprintf("    base_url=\"%s\",\n", extractedData.BaseURL))
	} else {
		examples.WriteString("    base_url=\"https://api.example.com/v1\",\n")
	}

	// Add authentication examples if available
	if ok && extractedData != nil && len(extractedData.SecuritySchemes) > 0 {
		for name, scheme := range extractedData.SecuritySchemes {
			switch scheme.Type {
			case securitySchemeAPIKey:
				examples.WriteString(fmt.Sprintf("    %s=\"your-%s\",\n", name, name))
			case securitySchemeHTTP:
				switch scheme.Scheme {
				case securitySchemeBearer:
					examples.WriteString("    bearer_token=\"your-bearer-token\",\n")
				case securitySchemeBasic:
					examples.WriteString("    username=\"your-username\",\n")
					examples.WriteString("    password=\"your-password\",\n")
				}
			}
		}
	}

	examples.WriteString(")\n\n\n")
	examples.WriteString("# Example: List resources\n")
	examples.WriteString("# response = client.list_items()\n")
	examples.WriteString("# print(response)\n\n")
	examples.WriteString("# Example: Get a resource by ID\n")
	examples.WriteString("# response = client.get_item(id=123)\n")
	examples.WriteString("# print(response)\n\n")
	examples.WriteString("# Example: Create a resource\n")
	examples.WriteString("# data = {\"name\": \"Example\", \"value\": 42}\n")
	examples.WriteString("# response = client.create_item(body=data)\n")
	examples.WriteString("# print(response)\n\n")

	return examples.String()
}
