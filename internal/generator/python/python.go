// Package python provides Python SDK generation functionality.
package python

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/vubon/sdk-forge/internal/generator/common"
	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

//go:embed templates/__init__.py.tmpl
var pythonInitTemplate string

//go:embed templates/client.py.tmpl
var pythonClientTemplate string

//go:embed templates/setup.py.tmpl
var pythonSetupTemplate string

//go:embed templates/README.md.tmpl
var pythonReadmeTemplate string

func getPythonInitTemplateContent() string {
	return pythonInitTemplate
}

func getPythonClientTemplateContent() string {
	return pythonClientTemplate
}

func getPythonSetupTemplateContent() string {
	return pythonSetupTemplate
}

func getPythonReadmeTemplateContent() string {
	return pythonReadmeTemplate
}

// GeneratePythonSDK generates a Python SDK
// If version is nil, uses the default Python version
// If sdkVersion is empty, extracts from OpenAPI schema or defaults to "1.0.0"
// retryConfig specifies retry behavior for HTTP requests (can be disabled)
func GeneratePythonSDK(
	outputPath, sdkName, httpLib string,
	openAPIDoc interface{},
	version *common.LanguageVersion,
	sdkVersion string,
	generateTests bool,
	retryConfig common.RetryConfig,
) error {
	// Use default version if not provided
	if version == nil {
		defaultVersion := common.GetPythonDefaultVersion()
		version = &defaultVersion
	}

	// Convert openAPIDoc to *openapi3.T
	doc, ok := openAPIDoc.(*openapi3.T)
	if !ok {
		// If not an openapi3.T, try to extract from ExtractedData
		if extractedData, ok := openAPIDoc.(*common.ExtractedData); ok {
			return generatePythonSDKFromExtracted(outputPath, sdkName, httpLib, extractedData, *version, sdkVersion, generateTests, retryConfig)
		}
		return fmt.Errorf("invalid OpenAPI document type")
	}

	// Extract data from OpenAPI document
	extractedData, err := common.ExtractOpenAPIData(doc)
	if err != nil {
		return fmt.Errorf("failed to extract OpenAPI data: %w", err)
	}

	return generatePythonSDKFromExtracted(outputPath, sdkName, httpLib, extractedData, *version, sdkVersion, generateTests, retryConfig)
}

// generatePythonSDKFromExtracted generates SDK from extracted data
func generatePythonSDKFromExtracted(
	outputPath, sdkName, httpLib string,
	extractedData *common.ExtractedData,
	version common.LanguageVersion,
	sdkVersion string,
	generateTests bool,
	retryConfig common.RetryConfig,
) error {
	// Get HTTP library config
	libConfig, err := httplib.GetLibraryConfig("python", httpLib)
	if err != nil {
		return fmt.Errorf("failed to get HTTP library config: %w", err)
	}

	// Sanitize SDK name for Python (snake_case)
	sanitizedName := common.ToSnakeCase(sdkName)
	packageDir := filepath.Join(outputPath, sanitizedName)

	// Create package directory
	if err := os.MkdirAll(packageDir, 0750); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Template data
	data := common.TemplateData{
		SDKName:         sanitizedName,
		Language:        "python",
		HTTPLib:         httpLib,
		HTTPLibImport:   libConfig.Import,
		HTTPLibConfig:   libConfig,
		OpenAPIDoc:      extractedData,
		ClientClassName: common.GetClientClassName(sanitizedName),
		RetryConfig:     retryConfig,
	}

	// Determine SDK version using common utility
	finalSDKVersion := common.DetermineSDKVersion(extractedData, sdkVersion)

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
		operationsByTag := common.GroupOperationsByTag(extractedData.Operations)
		for tag, operations := range operationsByTag {
			// Generate api/{tag}.py
			tagFileName := common.ToSnakeCase(tag) + ".py"
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

	// Generate tests/ directory if test generation is enabled
	if generateTests {
		if err := generatePythonTests(outputPath, packageDir, data, extractedData); err != nil {
			return fmt.Errorf("failed to generate tests: %w", err)
		}
	}

	return nil
}

// formatPythonFile formats a Python source file using black (if available)
func formatPythonFile(filePath string) error {
	// Skip formatting in tests to improve performance
	if os.Getenv("SKIP_FORMATTING") == "true" || os.Getenv("TESTING") == "true" {
		return nil
	}

	// Try black first (most common Python formatter)
	cmd := exec.Command("black", "--quiet", filePath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Fallback to autopep8 if black is not available
	cmd = exec.Command("autopep8", "--in-place", "--aggressive", "--aggressive", filePath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err == nil {
		return nil
	}

	// If neither formatter is available, that's okay - just skip formatting
	return fmt.Errorf("no Python formatter available (black or autopep8)")
}

func generatePythonInit(data common.TemplateData, sdkVersion string) string {
	// Check if we have models
	extractedData, ok := data.OpenAPIDoc.(*common.ExtractedData)
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
		ClientClassName: common.GetClientClassName(data.SDKName),
		HasModels:       hasModels,
		Version:         sdkVersion,
	}
	if hasModels {
		for name := range extractedData.Schemas {
			templateData.ModelNames = append(templateData.ModelNames, common.ToPascalCase(name))
		}
	}

	// Load and render template
	tmpl, err := common.LoadTemplate(getPythonInitTemplateContent())
	if err != nil {
		// Fallback to old method
		var initContent strings.Builder
		initContent.WriteString(fmt.Sprintf("\"\"\"%s SDK - Auto-generated from OpenAPI schema\"\"\"\n\n", displayName))
		initContent.WriteString(fmt.Sprintf("from .client import %s\n", common.GetClientClassName(data.SDKName)))
		if hasModels {
			initContent.WriteString("from .models import *\n")
		}
		initContent.WriteString(fmt.Sprintf("\n__all__ = [\"%s\"", common.GetClientClassName(data.SDKName)))
		if hasModels {
			for name := range extractedData.Schemas {
				initContent.WriteString(fmt.Sprintf(", \"%s\"", common.ToPascalCase(name)))
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
		initContent.WriteString(fmt.Sprintf("from .client import %s\n", common.GetClientClassName(data.SDKName)))
		if hasModels {
			initContent.WriteString("from .models import *\n")
		}
		initContent.WriteString(fmt.Sprintf("\n__all__ = [\"%s\"", common.GetClientClassName(data.SDKName)))
		if hasModels {
			for name := range extractedData.Schemas {
				initContent.WriteString(fmt.Sprintf(", \"%s\"", common.ToPascalCase(name)))
			}
		}
		initContent.WriteString(fmt.Sprintf("]\n__version__ = \"%s\"\n", sdkVersion))
		return initContent.String()
	}

	return buf.String()
}

func generatePythonClient(data common.TemplateData) string {
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
	extractedData, ok := data.OpenAPIDoc.(*common.ExtractedData)
	if !ok {
		extractedData = &common.ExtractedData{BaseURL: "https://api.example.com/v1"}
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

	// Generate retry setup if enabled
	retrySetup := ""
	retryHelper := ""
	if data.RetryConfig.Enabled {
		retrySetup = generatePythonRetrySetup(data.RetryConfig)
		retryHelper = generatePythonRetryHelper(data.HTTPLib, data.RetryConfig)
	}

	// Prepare template data
	type PythonClientData struct {
		DisplayName     string
		HTTPLibImport   string
		ClientClassName string
		BaseURLDefault  string
		ClientClass     string
		AuthSetup       string
		APIMethods      string
		RetryEnabled    bool
		RetrySetup      string
		RetryHelper     string
	}
	templateData := PythonClientData{
		DisplayName:     displayName,
		HTTPLibImport:   data.HTTPLibImport,
		ClientClassName: common.GetClientClassName(data.SDKName),
		BaseURLDefault:  baseURLDefault,
		ClientClass:     clientClass,
		AuthSetup:       authSetup,
		APIMethods:      apiMethods,
		RetryEnabled:    data.RetryConfig.Enabled,
		RetrySetup:      retrySetup,
		RetryHelper:     retryHelper,
	}

	// Load and render template
	tmpl, err := common.LoadTemplate(getPythonClientTemplateContent())
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
			common.GetClientClassName(data.SDKName),
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
			common.GetClientClassName(data.SDKName),
			displayName,
			baseURLDefault,
			clientClass,
			authSetup,
			apiMethods)
	}

	return buf.String()
}

// generatePythonRetrySetup generates retry configuration setup code for __init__
func generatePythonRetrySetup(config common.RetryConfig) string {
	if !config.Enabled {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("        # Retry configuration\n")
	buf.WriteString("        self.retry_enabled = True\n")
	buf.WriteString(fmt.Sprintf("        self.retry_max_attempts = %d\n", config.MaxAttempts))
	buf.WriteString(fmt.Sprintf("        self.retry_initial_delay = %.1f\n", config.InitialDelay.Seconds()))
	buf.WriteString(fmt.Sprintf("        self.retry_max_delay = %.1f\n", config.MaxDelay.Seconds()))
	buf.WriteString(fmt.Sprintf("        self.retry_backoff_multiplier = %.1f\n", config.BackoffMultiplier))
	buf.WriteString(fmt.Sprintf("        self.retry_strategy = %q\n", config.Strategy))
	buf.WriteString("        self.retryable_status_codes = [")
	for i, code := range config.RetryableStatusCodes {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%d", code))
	}
	buf.WriteString("]\n")
	// Use Python boolean values (True/False) instead of Go boolean values (true/false)
	retryOnNetworkErrors := "False"
	if config.RetryOnNetworkErrors {
		retryOnNetworkErrors = "True"
	}
	buf.WriteString(fmt.Sprintf("        self.retry_on_network_errors = %s\n", retryOnNetworkErrors))

	return buf.String()
}

// generatePythonRetryHelper generates retry helper function based on HTTP library
func generatePythonRetryHelper(httpLib string, config common.RetryConfig) string {
	if !config.Enabled {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("\n    def _calculate_retry_delay(self, attempt: int) -> float:\n")
	buf.WriteString("        \"\"\"Calculate delay for retry attempt based on strategy\"\"\"\n")
	buf.WriteString(fmt.Sprintf("        if self.retry_strategy == %q:\n", common.RetryStrategyExponential))
	buf.WriteString("            # Exponential backoff: initial_delay * (multiplier ^ attempt)\n")
	buf.WriteString("            delay = self.retry_initial_delay * (self.retry_backoff_multiplier ** attempt)\n")
	buf.WriteString(fmt.Sprintf("        elif self.retry_strategy == %q:\n", common.RetryStrategyLinear))
	buf.WriteString("            # Linear backoff: initial_delay * (attempt + 1)\n")
	buf.WriteString("            delay = self.retry_initial_delay * (attempt + 1)\n")
	buf.WriteString(fmt.Sprintf("        elif self.retry_strategy == %q:\n", common.RetryStrategyFixed))
	buf.WriteString("            # Fixed delay: always use initial_delay\n")
	buf.WriteString("            delay = self.retry_initial_delay\n")
	buf.WriteString("        else:\n")
	buf.WriteString("            # Default to exponential\n")
	buf.WriteString("            delay = self.retry_initial_delay * (self.retry_backoff_multiplier ** attempt)\n")
	buf.WriteString("        return min(delay, self.retry_max_delay)\n")

	// Generate retry logic based on HTTP library
	switch httpLib {
	case "requests":
		buf.WriteString("\n    def _request_with_retry(self, method: str, url: str, **kwargs):\n")
		buf.WriteString("        \"\"\"Make HTTP request with retry logic\"\"\"\n")
		buf.WriteString("        import time\n")
		buf.WriteString("        last_exception = None\n\n")
		buf.WriteString("        for attempt in range(self.retry_max_attempts):\n")
		buf.WriteString("            try:\n")
		buf.WriteString("                response = self.session.request(method, url, **kwargs)\n")
		buf.WriteString("                # Check if status code is retryable\n")
		buf.WriteString("                if response.status_code not in self.retryable_status_codes:\n")
		buf.WriteString("                    return response\n")
		buf.WriteString("                # Retryable status code - will retry below\n")
		buf.WriteString("            except (requests.exceptions.RequestException,\n")
		buf.WriteString("                    requests.exceptions.Timeout,\n")
		buf.WriteString("                    requests.exceptions.ConnectionError) as e:\n")
		buf.WriteString("                if not self.retry_on_network_errors:\n")
		buf.WriteString("                    raise\n")
		buf.WriteString("                last_exception = e\n")
		buf.WriteString("                # Network error - will retry below\n\n")
		buf.WriteString("            # If we get here, we need to retry\n")
		buf.WriteString("            if attempt < self.retry_max_attempts - 1:\n")
		buf.WriteString("                delay = self._calculate_retry_delay(attempt)\n")
		buf.WriteString("                time.sleep(delay)\n")
		buf.WriteString("            else:\n")
		buf.WriteString("                # Max attempts exceeded\n")
		buf.WriteString("                if last_exception:\n")
		buf.WriteString("                    raise last_exception\n")
		buf.WriteString("                response.raise_for_status()\n")
		buf.WriteString("                raise Exception(f\"Request failed after {self.retry_max_attempts} attempts\")\n\n")
		buf.WriteString("        return response\n")

	case "httpx":
		buf.WriteString("\n    def _request_with_retry(self, method: str, url: str, **kwargs):\n")
		buf.WriteString("        \"\"\"Make HTTP request with retry logic\"\"\"\n")
		buf.WriteString("        import time\n")
		buf.WriteString("        last_exception = None\n\n")
		buf.WriteString("        for attempt in range(self.retry_max_attempts):\n")
		buf.WriteString("            try:\n")
		buf.WriteString("                response = self.session.request(method, url, **kwargs)\n")
		buf.WriteString("                # Check if status code is retryable\n")
		buf.WriteString("                if response.status_code not in self.retryable_status_codes:\n")
		buf.WriteString("                    return response\n")
		buf.WriteString("                # Retryable status code - will retry below\n")
		buf.WriteString("            except (httpx.RequestError, httpx.TimeoutException,\n")
		buf.WriteString("                    httpx.ConnectError, httpx.NetworkError) as e:\n")
		buf.WriteString("                if not self.retry_on_network_errors:\n")
		buf.WriteString("                    raise\n")
		buf.WriteString("                last_exception = e\n")
		buf.WriteString("                # Network error - will retry below\n\n")
		buf.WriteString("            # If we get here, we need to retry\n")
		buf.WriteString("            if attempt < self.retry_max_attempts - 1:\n")
		buf.WriteString("                delay = self._calculate_retry_delay(attempt)\n")
		buf.WriteString("                time.sleep(delay)\n")
		buf.WriteString("            else:\n")
		buf.WriteString("                # Max attempts exceeded\n")
		buf.WriteString("                if last_exception:\n")
		buf.WriteString("                    raise last_exception\n")
		buf.WriteString("                response.raise_for_status()\n")
		buf.WriteString("                raise Exception(f\"Request failed after {self.retry_max_attempts} attempts\")\n\n")
		buf.WriteString("        return response\n")

	case "aiohttp":
		// Note: aiohttp requires async/await, but the current _request method is synchronous
		// For now, we'll generate a synchronous wrapper that won't work perfectly with aiohttp
		// Full aiohttp support would require making _request async, which is a larger change
		buf.WriteString("\n    async def _request_with_retry_async(self, method: str, url: str, **kwargs):\n")
		buf.WriteString("        \"\"\"Make async HTTP request with retry logic (for aiohttp)\"\"\"\n")
		buf.WriteString("        import asyncio\n")
		buf.WriteString("        last_exception = None\n")
		buf.WriteString("        response_data = None\n\n")
		buf.WriteString("        for attempt in range(self.retry_max_attempts):\n")
		buf.WriteString("            try:\n")
		buf.WriteString("                async with self.session.request(method, url, **kwargs) as response:\n")
		buf.WriteString("                    # Check if status code is retryable\n")
		buf.WriteString("                    if response.status not in self.retryable_status_codes:\n")
		buf.WriteString("                        response_data = await response.json()\n")
		buf.WriteString("                        return response_data\n")
		buf.WriteString("                    # Retryable status code - will retry below\n")
		buf.WriteString("                    response_data = await response.text()\n")
		buf.WriteString("            except (aiohttp.ClientError, asyncio.TimeoutError) as e:\n")
		buf.WriteString("                if not self.retry_on_network_errors:\n")
		buf.WriteString("                    raise\n")
		buf.WriteString("                last_exception = e\n")
		buf.WriteString("                # Network error - will retry below\n\n")
		buf.WriteString("            # If we get here, we need to retry\n")
		buf.WriteString("            if attempt < self.retry_max_attempts - 1:\n")
		buf.WriteString("                delay = self._calculate_retry_delay(attempt)\n")
		buf.WriteString("                await asyncio.sleep(delay)\n")
		buf.WriteString("            else:\n")
		buf.WriteString("                # Max attempts exceeded\n")
		buf.WriteString("                if last_exception:\n")
		buf.WriteString("                    raise last_exception\n")
		buf.WriteString("                raise Exception(f\"Request failed after {self.retry_max_attempts} attempts\")\n\n")
		buf.WriteString("        return response_data\n")

	case "urllib3":
		buf.WriteString("\n    def _request_with_retry(self, method: str, url: str, **kwargs):\n")
		buf.WriteString("        \"\"\"Make HTTP request with retry logic\"\"\"\n")
		buf.WriteString("        import time\n")
		buf.WriteString("        from urllib3.exceptions import HTTPError, MaxRetryError, TimeoutError\n")
		buf.WriteString("        last_exception = None\n\n")
		buf.WriteString("        for attempt in range(self.retry_max_attempts):\n")
		buf.WriteString("            try:\n")
		buf.WriteString("                response = self.session.request(method, url, **kwargs)\n")
		buf.WriteString("                # Check if status code is retryable\n")
		buf.WriteString("                if response.status not in self.retryable_status_codes:\n")
		buf.WriteString("                    return response\n")
		buf.WriteString("                # Retryable status code - will retry below\n")
		buf.WriteString("            except (HTTPError, MaxRetryError, TimeoutError) as e:\n")
		buf.WriteString("                if not self.retry_on_network_errors:\n")
		buf.WriteString("                    raise\n")
		buf.WriteString("                last_exception = e\n")
		buf.WriteString("                # Network error - will retry below\n\n")
		buf.WriteString("            # If we get here, we need to retry\n")
		buf.WriteString("            if attempt < self.retry_max_attempts - 1:\n")
		buf.WriteString("                delay = self._calculate_retry_delay(attempt)\n")
		buf.WriteString("                time.sleep(delay)\n")
		buf.WriteString("            else:\n")
		buf.WriteString("                # Max attempts exceeded\n")
		buf.WriteString("                if last_exception:\n")
		buf.WriteString("                    raise last_exception\n")
		buf.WriteString("                raise Exception(f\"Request failed after {self.retry_max_attempts} attempts\")\n\n")
		buf.WriteString("        return response\n")

	default:
		// Default to requests-like behavior
		buf.WriteString("\n    def _request_with_retry(self, method: str, url: str, **kwargs):\n")
		buf.WriteString("        \"\"\"Make HTTP request with retry logic\"\"\"\n")
		buf.WriteString("        import time\n")
		buf.WriteString("        last_exception = None\n\n")
		buf.WriteString("        for attempt in range(self.retry_max_attempts):\n")
		buf.WriteString("            try:\n")
		buf.WriteString("                response = self.session.request(method, url, **kwargs)\n")
		buf.WriteString("                # Check if status code is retryable\n")
		buf.WriteString("                if response.status_code not in self.retryable_status_codes:\n")
		buf.WriteString("                    return response\n")
		buf.WriteString("                # Retryable status code - will retry below\n")
		buf.WriteString("            except Exception as e:\n")
		buf.WriteString("                if not self.retry_on_network_errors:\n")
		buf.WriteString("                    raise\n")
		buf.WriteString("                last_exception = e\n")
		buf.WriteString("                # Network error - will retry below\n\n")
		buf.WriteString("            # If we get here, we need to retry\n")
		buf.WriteString("            if attempt < self.retry_max_attempts - 1:\n")
		buf.WriteString("                delay = self._calculate_retry_delay(attempt)\n")
		buf.WriteString("                time.sleep(delay)\n")
		buf.WriteString("            else:\n")
		buf.WriteString("                # Max attempts exceeded\n")
		buf.WriteString("                if last_exception:\n")
		buf.WriteString("                    raise last_exception\n")
		buf.WriteString("                response.raise_for_status()\n")
		buf.WriteString("                raise Exception(f\"Request failed after {self.retry_max_attempts} attempts\")\n\n")
		buf.WriteString("        return response\n")
	}

	return buf.String()
}

// generatePythonAuthSetup generates authentication setup code
func generatePythonAuthSetup(securitySchemes map[string]common.SecurityScheme) string {
	if len(securitySchemes) == 0 {
		return "        # No authentication required"
	}

	var setupCode strings.Builder
	if len(securitySchemes) > 0 {
		setupCode.WriteString("        # Set up authentication\n")
	}

	for name, scheme := range securitySchemes {
		switch scheme.Type {
		case "apiKey":
			setupCode.WriteString(fmt.Sprintf("        self.%s = auth_params.get('%s')\n", name, name))
			setupCode.WriteString(fmt.Sprintf("        if self.%s:\n", name))
			switch scheme.In {
			case "header":
				setupCode.WriteString(fmt.Sprintf("            self.session.headers['%s'] = self.%s\n", scheme.Name, name))
			case "query":
				setupCode.WriteString(fmt.Sprintf("            # Query parameter '%s' will be added per request\n", scheme.Name))
			}
		case "http":
			switch scheme.Scheme {
			case "bearer":
				setupCode.WriteString("        self.bearer_token = auth_params.get('bearer_token')\n")
				setupCode.WriteString("        if self.bearer_token:\n")
				setupCode.WriteString("            self.session.headers['Authorization'] = f'Bearer {self.bearer_token}'\n")
			case "basic":
				setupCode.WriteString("        self.username = auth_params.get('username')\n")
				setupCode.WriteString("        self.password = auth_params.get('password')\n")
				setupCode.WriteString("        if self.username and self.password:\n")
				setupCode.WriteString("            from requests.auth import HTTPBasicAuth\n")
				setupCode.WriteString("            self.session.auth = HTTPBasicAuth(self.username, self.password)\n")
			case "digest":
				setupCode.WriteString("        self.username = auth_params.get('username')\n")
				setupCode.WriteString("        self.password = auth_params.get('password')\n")
				setupCode.WriteString("        if self.username and self.password:\n")
				setupCode.WriteString("            from requests.auth import HTTPDigestAuth\n")
				setupCode.WriteString("            self.session.auth = HTTPDigestAuth(self.username, self.password)\n")
			}
		case "oauth2":
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
		case "openIdConnect":
			setupCode.WriteString(fmt.Sprintf("        self.%s_token = auth_params.get('%s_token')\n", name, name))
			setupCode.WriteString(fmt.Sprintf("        if self.%s_token:\n", name))
			bearerLine := fmt.Sprintf("            self.session.headers['Authorization'] = f'Bearer {self.%s_token}'\n", name)
			setupCode.WriteString(bearerLine)
			if scheme.OpenIDConnectURL != "" {
				setupCode.WriteString(fmt.Sprintf("        # OpenID Connect discovery URL: %s\n", scheme.OpenIDConnectURL))
			}
		case "mutualTLS":
			setupCode.WriteString(fmt.Sprintf("        self.%s_cert = auth_params.get('%s_cert')\n", name, name))
			setupCode.WriteString(fmt.Sprintf("        self.%s_key = auth_params.get('%s_key')\n", name, name))
			setupCode.WriteString(fmt.Sprintf("        if self.%s_cert and self.%s_key:\n", name, name))
			setupCode.WriteString(fmt.Sprintf("            self.session.cert = (self.%s_cert, self.%s_key)\n", name, name))
		}
	}

	return setupCode.String()
}

// generatePythonAPIMethods generates API methods from operations
func generatePythonAPIMethods(operations []common.APIOperation) string {
	if len(operations) == 0 {
		return "    # No API methods defined in OpenAPI schema"
	}

	var methods strings.Builder

	for _, op := range operations {
		methodName := common.GetOperationMethodName(op)
		if methodName == "" {
			// Fallback naming
			methodName = strings.ToLower(op.Method) + "_" + strings.ReplaceAll(strings.Trim(op.Path, "/"), "/", "_")
			methodName = common.ToSnakeCase(methodName)
		}

		// Generate method signature
		methods.WriteString(fmt.Sprintf("    def %s(self", methodName))

		// Add path parameters
		var pathParams []string
		var queryParams []string
		for _, param := range op.Parameters {
			switch param.In {
			case "path":
				pathParams = append(pathParams, param.Name)
				methods.WriteString(fmt.Sprintf(", %s: %s", param.Name, getPythonType(param.Schema)))
			case "query":
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
func getPythonType(schema *common.Schema) string {
	if schema == nil {
		return "any"
	}

	switch schema.Type {
	case "string":
		return "str"
	case "integer":
		return "int"
	case "number":
		return "float"
	case "boolean":
		return "bool"
	case "array":
		return "list"
	case "object":
		return "dict"
	default:
		return "any"
	}
}

func generatePythonRequirements(data common.TemplateData) string {
	if data.HTTPLibConfig == nil {
		return "# Dependencies\n"
	}

	requirements := []string{
		data.HTTPLibConfig.Dependency,
		"# Add other dependencies as needed",
	}

	return strings.Join(requirements, "\n") + "\n"
}

func generatePythonREADME(data common.TemplateData) string {
	// Load and render template
	tmpl, err := common.LoadTemplate(getPythonReadmeTemplateContent())
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
// escapePythonString escapes a string for use in a Python string literal
func escapePythonString(s string) string {
	// Replace backslashes first (to avoid double-escaping)
	s = strings.ReplaceAll(s, "\\", "\\\\")
	// Escape double quotes
	s = strings.ReplaceAll(s, "\"", "\\\"")
	// Replace newlines with \n
	s = strings.ReplaceAll(s, "\n", "\\n")
	// Replace carriage returns
	s = strings.ReplaceAll(s, "\r", "\\r")
	// Replace tabs
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

func generatePythonSetup(data common.TemplateData, pythonVersion common.LanguageVersion, sdkVersion string) string {
	extractedData, ok := data.OpenAPIDoc.(*common.ExtractedData)
	description := "Auto-generated Python SDK"
	if ok && extractedData != nil {
		if extractedData.Description != "" {
			description = extractedData.Description
		}
	}

	// Escape description for Python string
	escapedDescription := escapePythonString(description)

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
	availableVersions := common.GetPythonAvailableVersions()
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
		Description:    escapedDescription,
		PythonVersion:  fmt.Sprintf("%d.%d", pythonVersion.Major, pythonVersion.Minor),
		PythonRequires: fmt.Sprintf(">=%d.%d", pythonVersion.Major, pythonVersion.Minor),
		Classifiers:    classifiers,
	}

	// Load and render template
	tmpl, err := common.LoadTemplate(getPythonSetupTemplateContent())
	if err != nil {
		// Fallback to old method
		var setup strings.Builder
		setup.WriteString("from setuptools import setup, find_packages\n\n")
		setup.WriteString("setup(\n")
		setup.WriteString(fmt.Sprintf("    name=\"%s\",\n", data.SDKName))
		setup.WriteString(fmt.Sprintf("    version=\"%s\",\n", sdkVersion))
		// Use escaped description
		setup.WriteString(fmt.Sprintf("    description=\"%s\",\n", escapedDescription))
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
		availableVersions := common.GetPythonAvailableVersions()
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
		// Use escaped description
		setup.WriteString(fmt.Sprintf("    description=\"%s\",\n", escapedDescription))
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
		availableVersions := common.GetPythonAvailableVersions()
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
func generatePythonAPIInit(data common.TemplateData) string {
	extractedData, ok := data.OpenAPIDoc.(*common.ExtractedData)
	if !ok || extractedData == nil || len(extractedData.Operations) == 0 {
		return "\"\"\"API endpoint modules\"\"\"\n"
	}

	operationsByTag := common.GroupOperationsByTag(extractedData.Operations)
	var init strings.Builder
	init.WriteString("\"\"\"API endpoint modules\"\"\"\n\n")

	// Import all tag modules
	for tag := range operationsByTag {
		tagModule := common.ToSnakeCase(tag)
		init.WriteString(fmt.Sprintf("from . import %s\n", tagModule))
	}

	init.WriteString("\n__all__ = [\n")
	first := true
	for tag := range operationsByTag {
		if !first {
			init.WriteString(",\n")
		}
		tagModule := common.ToSnakeCase(tag)
		init.WriteString(fmt.Sprintf("    \"%s\"", tagModule))
		first = false
	}
	init.WriteString("\n]\n")

	return init.String()
}

// generatePythonAPIModule generates api/{tag}.py with operations for that tag
func generatePythonAPIModule(tag string, operations []common.APIOperation, data common.TemplateData) string {
	clientClassName := common.GetClientClassName(data.SDKName)
	var module strings.Builder
	module.WriteString(fmt.Sprintf("\"\"\"%s API endpoints\"\"\"\n\n", tag))
	module.WriteString("from typing import Optional, Dict, Any\n")
	module.WriteString(fmt.Sprintf("from ..client import %s\n\n\n", clientClassName))

	// Generate functions for each operation
	for _, op := range operations {
		methodName := common.GetOperationMethodName(op)
		if methodName == "" {
			// Fallback naming
			pathPart := strings.ReplaceAll(strings.Trim(op.Path, "/"), "/", "_")
			methodName = strings.ToLower(op.Method) + "_" + pathPart
			methodName = common.ToSnakeCase(methodName)
		}

		// Function signature - takes client as first parameter
		module.WriteString(fmt.Sprintf("def %s(client: %s", methodName, clientClassName))

		// Add parameters
		var pathParams []string
		var queryParams []string
		for _, param := range op.Parameters {
			switch param.In {
			case "path":
				pathParams = append(pathParams, param.Name)
				paramType := getPythonType(param.Schema)
				module.WriteString(fmt.Sprintf(", %s: %s", param.Name, paramType))
			case "query":
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
func generatePythonExamples(data common.TemplateData) string {
	extractedData, ok := data.OpenAPIDoc.(*common.ExtractedData)
	clientClassName := common.GetClientClassName(data.SDKName)

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
			case "apiKey":
				examples.WriteString(fmt.Sprintf("    %s=\"your-%s\",\n", name, name))
			case "http":
				switch scheme.Scheme {
				case "bearer":
					examples.WriteString("    bearer_token=\"your-bearer-token\",\n")
				case "basic":
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

// generatePythonTests generates test files for Python SDK
func generatePythonTests(outputPath, packageDir string, data common.TemplateData, extractedData *common.ExtractedData) error {
	// Create tests/ directory
	testsDir := filepath.Join(outputPath, "tests")
	if err := os.MkdirAll(testsDir, 0750); err != nil {
		return fmt.Errorf("failed to create tests directory: %w", err)
	}

	// Generate tests/__init__.py
	initContent := "# Test package\n"
	initPath := filepath.Join(testsDir, "__init__.py")
	// #nosec G306 -- 0644 is appropriate for Python package files
	if err := os.WriteFile(initPath, []byte(initContent), 0644); err != nil {
		return fmt.Errorf("failed to write tests/__init__.py: %w", err)
	}

	// Generate tests/conftest.py (pytest fixtures)
	conftestContent := generatePythonConftest(data, extractedData)
	conftestPath := filepath.Join(testsDir, "conftest.py")
	// #nosec G306 -- 0644 is appropriate for Python test files
	if err := os.WriteFile(conftestPath, []byte(conftestContent), 0644); err != nil {
		return fmt.Errorf("failed to write tests/conftest.py: %w", err)
	}
	// Format with black (if available)
	if err := formatPythonFile(conftestPath); err != nil {
		_ = err
	}

	// Generate tests/test_client.py
	clientTestContent := generatePythonClientTest(data, extractedData)
	clientTestPath := filepath.Join(testsDir, "test_client.py")
	// #nosec G306 -- 0644 is appropriate for Python test files
	if err := os.WriteFile(clientTestPath, []byte(clientTestContent), 0644); err != nil {
		return fmt.Errorf("failed to write tests/test_client.py: %w", err)
	}
	// Format with black (if available)
	if err := formatPythonFile(clientTestPath); err != nil {
		_ = err
	}

	// Generate tests/test_models.py if schemas exist
	if len(extractedData.Schemas) > 0 {
		modelsTestContent := generatePythonModelsTest(data, extractedData.Schemas)
		modelsTestPath := filepath.Join(testsDir, "test_models.py")
		// #nosec G306 -- 0644 is appropriate for Python test files
		if err := os.WriteFile(modelsTestPath, []byte(modelsTestContent), 0644); err != nil {
			return fmt.Errorf("failed to write tests/test_models.py: %w", err)
		}
		// Format with black (if available)
		if err := formatPythonFile(modelsTestPath); err != nil {
			_ = err
		}
	}

	// Generate tests/test_api_methods.py if operations exist
	if len(extractedData.Operations) > 0 {
		apiTestContent := generatePythonAPITest(data, extractedData.Operations, extractedData)
		apiTestPath := filepath.Join(testsDir, "test_api_methods.py")
		// #nosec G306 -- 0644 is appropriate for Python test files
		if err := os.WriteFile(apiTestPath, []byte(apiTestContent), 0644); err != nil {
			return fmt.Errorf("failed to write tests/test_api_methods.py: %w", err)
		}
		// Format with black (if available)
		if err := formatPythonFile(apiTestPath); err != nil {
			_ = err
		}
	}

	// Generate tests/test_auth.py if security schemes exist
	if len(extractedData.SecuritySchemes) > 0 {
		authTestContent := generatePythonAuthTest(data, extractedData.SecuritySchemes, extractedData)
		authTestPath := filepath.Join(testsDir, "test_auth.py")
		// #nosec G306 -- 0644 is appropriate for Python test files
		if err := os.WriteFile(authTestPath, []byte(authTestContent), 0644); err != nil {
			return fmt.Errorf("failed to write tests/test_auth.py: %w", err)
		}
		// Format with black (if available)
		if err := formatPythonFile(authTestPath); err != nil {
			_ = err
		}
	}

	// Generate test fixtures from OpenAPI examples
	if err := generatePythonTestFixtures(testsDir, extractedData); err != nil {
		return fmt.Errorf("failed to generate test fixtures: %w", err)
	}

	return nil
}

// generatePythonTestFixtures generates test fixture files from OpenAPI examples
func generatePythonTestFixtures(testsDir string, extractedData *common.ExtractedData) error {
	fixturesDir := filepath.Join(testsDir, "fixtures")
	if err := os.MkdirAll(fixturesDir, 0750); err != nil {
		return fmt.Errorf("failed to create fixtures directory: %w", err)
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
		// Generate fixtures.py file
		fixturesContent := generatePythonFixturesFile(fixtures)
		fixturesPath := filepath.Join(fixturesDir, "fixtures.py")
		// #nosec G306 -- 0644 is appropriate for Python fixture files
		if err := os.WriteFile(fixturesPath, []byte(fixturesContent), 0644); err != nil {
			return fmt.Errorf("failed to write fixtures/fixtures.py: %w", err)
		}
	}

	return nil
}

// generatePythonFixturesFile generates a Python file with test fixtures
func generatePythonFixturesFile(fixtures map[string]interface{}) string {
	var buf bytes.Buffer
	buf.WriteString("\"\"\"Test fixtures extracted from OpenAPI examples\"\"\"\n\n")

	for key, example := range fixtures {
		// Convert key to valid Python variable name
		varName := common.ToSnakeCase(key)
		exampleJSON := formatExampleForPython(example)
		fmt.Fprintf(&buf, "%s = %s\n\n", varName, exampleJSON)
	}

	return buf.String()
}

// generatePythonConftest generates pytest conftest.py with fixtures
func generatePythonConftest(data common.TemplateData, extractedData *common.ExtractedData) string {
	var conftest bytes.Buffer
	conftest.WriteString("import pytest\n")
	conftest.WriteString(fmt.Sprintf("from %s import %s\n\n", data.SDKName, data.ClientClassName))
	conftest.WriteString("@pytest.fixture\n")
	conftest.WriteString("def client():\n")
	conftest.WriteString("    \"\"\"Create a test client instance.\"\"\"\n")

	// Use base URL from OpenAPI spec, fallback to default
	baseURL := extractedData.BaseURL
	if baseURL == "" {
		baseURL = "https://api.example.com/v1"
	}
	conftest.WriteString(fmt.Sprintf("    return %s(base_url=%q)\n", data.ClientClassName, baseURL))
	return conftest.String()
}

// generatePythonClientTest generates test_client.py
func generatePythonClientTest(data common.TemplateData, extractedData *common.ExtractedData) string {
	var test bytes.Buffer
	test.WriteString("import pytest\n")
	test.WriteString(fmt.Sprintf("from %s import %s\n\n\n", data.SDKName, data.ClientClassName))
	test.WriteString("def test_client_initialization():\n")
	test.WriteString("    \"\"\"Test client can be instantiated.\"\"\"\n")

	// Use base URL from OpenAPI spec, fallback to default
	baseURL := extractedData.BaseURL
	if baseURL == "" {
		baseURL = "https://api.example.com/v1"
	}
	test.WriteString(fmt.Sprintf("    client = %s(base_url=%q)\n", data.ClientClassName, baseURL))
	test.WriteString(fmt.Sprintf("    assert client.base_url == %q\n", baseURL))
	test.WriteString("    assert client.session is not None\n")
	return test.String()
}

// generatePythonModelsTest generates test_models.py with schema-based tests
func generatePythonModelsTest(data common.TemplateData, schemas map[string]*common.Schema) string {
	var test bytes.Buffer
	test.WriteString("import pytest\n")
	test.WriteString("from dataclasses import asdict\n")
	test.WriteString(fmt.Sprintf("from %s import models\n\n\n", data.SDKName))

	// Generate tests for each schema
	for name, schema := range schemas {
		className := common.ToPascalCase(name)
		test.WriteString(fmt.Sprintf("class Test%s:\n", className))
		test.WriteString(fmt.Sprintf("    \"\"\"Tests for %s model\"\"\"\n\n", className))

		// Test model instantiation
		test.WriteString(fmt.Sprintf("    def test_%s_creation(self):\n", common.ToSnakeCase(name)))
		test.WriteString(fmt.Sprintf("        \"\"\"Test %s can be instantiated.\"\"\"\n", className))

		// Generate test data based on schema properties
		if schema.Type == "object" && len(schema.Properties) > 0 {
			test.WriteString(fmt.Sprintf("        model = models.%s(\n", className))

			// Track required fields
			requiredSet := make(map[string]bool)
			for _, req := range schema.Required {
				requiredSet[req] = true
			}

			// Generate test values for each property
			first := true
			for propName, propSchema := range schema.Properties {
				if propSchema == nil {
					continue
				}

				if !first {
					test.WriteString(",\n")
				}
				first = false

				propSnakeName := common.ToSnakeCase(propName)
				testValue := generatePythonTestValue(propSchema, propName)

				if requiredSet[propName] {
					test.WriteString(fmt.Sprintf("            %s=%s", propSnakeName, testValue))
				} else {
					// Optional fields can be None or have a value
					test.WriteString(fmt.Sprintf("            %s=%s", propSnakeName, testValue))
				}
			}
			test.WriteString("\n        )\n")
			test.WriteString("        assert model is not None\n")

			// Test property access
			for propName := range schema.Properties {
				propSnakeName := common.ToSnakeCase(propName)
				test.WriteString(fmt.Sprintf("        assert hasattr(model, '%s')\n", propSnakeName))
			}
		} else {
			// Primitive/enum model (has 'value' field) or model without properties
			// Primitive models (string with enum, etc.) have a 'value' field
			if schema.Type != "object" && schema.Type != "array" {
				// Primitive type model - has required 'value' field
				testValue := generatePythonTestValue(schema, "value")
				test.WriteString(fmt.Sprintf("        model = models.%s(value=%s)\n", className, testValue))
			} else if len(schema.Required) > 0 {
				// Object model with required fields but no properties (edge case)
				test.WriteString(fmt.Sprintf("        model = models.%s(\n", className))
				first := true
				for _, reqField := range schema.Required {
					if !first {
						test.WriteString(",\n")
					}
					first = false
					fieldSnakeName := common.ToSnakeCase(reqField)
					test.WriteString(fmt.Sprintf("            %s=\"test_value\"", fieldSnakeName))
				}
				test.WriteString("\n        )\n")
			} else {
				// No required fields, can instantiate without arguments
				test.WriteString(fmt.Sprintf("        model = models.%s()\n", className))
			}
			test.WriteString("        assert model is not None\n")
		}
		test.WriteString("\n")

		// Test serialization (if dataclass)
		test.WriteString(fmt.Sprintf("    def test_%s_serialization(self):\n", common.ToSnakeCase(name)))
		test.WriteString(fmt.Sprintf("        \"\"\"Test %s can be serialized to dict.\"\"\"\n", className))
		if schema.Type == "object" && len(schema.Properties) > 0 {
			test.WriteString(fmt.Sprintf("        model = models.%s(\n", className))

			requiredSet := make(map[string]bool)
			for _, req := range schema.Required {
				requiredSet[req] = true
			}

			first := true
			for propName, propSchema := range schema.Properties {
				if propSchema == nil {
					continue
				}
				if !first {
					test.WriteString(",\n")
				}
				first = false
				propSnakeName := common.ToSnakeCase(propName)
				testValue := generatePythonTestValue(propSchema, propName)
				test.WriteString(fmt.Sprintf("            %s=%s", propSnakeName, testValue))
			}
			test.WriteString("\n        )\n")
			test.WriteString("        data = asdict(model)\n")
			test.WriteString("        assert isinstance(data, dict)\n")
		} else {
			// Primitive/enum model (has 'value' field) or model without properties
			if schema.Type != "object" && schema.Type != "array" {
				// Primitive type model - has required 'value' field
				testValue := generatePythonTestValue(schema, "value")
				test.WriteString(fmt.Sprintf("        model = models.%s(value=%s)\n", className, testValue))
			} else if len(schema.Required) > 0 {
				// Object model with required fields but no properties (edge case)
				test.WriteString(fmt.Sprintf("        model = models.%s(\n", className))
				first := true
				for _, reqField := range schema.Required {
					if !first {
						test.WriteString(",\n")
					}
					first = false
					fieldSnakeName := common.ToSnakeCase(reqField)
					test.WriteString(fmt.Sprintf("            %s=\"test_value\"", fieldSnakeName))
				}
				test.WriteString("\n        )\n")
			} else {
				// No required fields
				test.WriteString(fmt.Sprintf("        model = models.%s()\n", className))
			}
			test.WriteString("        data = asdict(model)\n")
			test.WriteString("        assert isinstance(data, dict)\n")
		}
		test.WriteString("\n\n")

		// Test required fields validation (if any)
		if len(schema.Required) > 0 {
			test.WriteString(fmt.Sprintf("    def test_%s_required_fields(self):\n", common.ToSnakeCase(name)))
			test.WriteString(fmt.Sprintf("        \"\"\"Test %s required fields are enforced.\"\"\"\n", className))
			test.WriteString("        # Note: dataclasses don't enforce required fields at runtime\n")
			test.WriteString("        # This test serves as documentation of required fields\n")
			test.WriteString("        required_fields = [")
			for i, req := range schema.Required {
				if i > 0 {
					test.WriteString(", ")
				}
				test.WriteString(fmt.Sprintf("\"%s\"", common.ToSnakeCase(req)))
			}
			test.WriteString("]\n")
			test.WriteString("        assert len(required_fields) > 0\n")
			test.WriteString("\n\n")
		}
	}

	return test.String()
}

// generatePythonTestValue generates a test value for a schema property
func generatePythonTestValue(schema *common.Schema, propName string) string {
	if schema == nil {
		return "\"test_value\""
	}

	switch schema.Type {
	case "string":
		if schema.Format == "date" {
			return "\"2024-01-01\""
		}
		if schema.Format == "date-time" {
			return "\"2024-01-01T00:00:00Z\""
		}
		if schema.Format == "email" {
			return "\"test@example.com\""
		}
		return fmt.Sprintf("\"test_%s\"", common.ToSnakeCase(propName))
	case "integer", "number":
		return "42"
	case "boolean":
		return "True"
	case "array":
		if schema.Items != nil {
			itemValue := generatePythonTestValue(schema.Items, "item")
			return fmt.Sprintf("[%s]", itemValue)
		}
		return "[]"
	case "object":
		return "{}"
	default:
		return "\"test_value\""
	}
}

// generatePythonAPITest generates test_api_methods.py with operation-based tests
func generatePythonAPITest(data common.TemplateData, operations []common.APIOperation, extractedData *common.ExtractedData) string {
	var test bytes.Buffer
	test.WriteString("import pytest\n")
	test.WriteString("from unittest.mock import Mock, patch\n")
	test.WriteString(fmt.Sprintf("from %s import %s\n\n\n", data.SDKName, data.ClientClassName))

	// Group operations by tag for better organization
	operationsByTag := common.GroupOperationsByTag(operations)

	// Generate tests for each tag/group
	for tag, tagOperations := range operationsByTag {
		test.WriteString(fmt.Sprintf("class Test%sAPI:\n", common.ToPascalCase(tag)))
		test.WriteString(fmt.Sprintf("    \"\"\"Tests for %s API methods\"\"\"\n\n", tag))

		// Generate test for each operation
		for _, op := range tagOperations {
			methodName := common.GetOperationMethodName(op)
			testMethodName := fmt.Sprintf("test_%s", methodName)

			// Patch requests.Session.request since client uses session.request()
			test.WriteString("    @patch('requests.Session.request')\n")
			test.WriteString(fmt.Sprintf("    def %s(self, mock_request):\n", testMethodName))
			test.WriteString(fmt.Sprintf("        \"\"\"Test %s %s operation.\"\"\"\n", op.Method, op.Path))

			// Setup mock response
			test.WriteString("        # Setup mock response\n")
			test.WriteString("        mock_response = Mock()\n")

			// Determine expected status code from responses
			expectedStatus := "200"
			if _, ok := op.Responses["200"]; !ok {
				// Find first available status code
				for statusCode := range op.Responses {
					expectedStatus = statusCode
					break
				}
			}

			test.WriteString(fmt.Sprintf("        mock_response.status_code = %s\n", expectedStatus))

			// Use example from OpenAPI response if available
			exampleJSON := getPythonExampleFromResponse(op.Responses[expectedStatus])
			test.WriteString(fmt.Sprintf("        mock_response.json.return_value = %s\n", exampleJSON))
			// Convert Python dict to JSON string for text response
			jsonText := strings.ReplaceAll(exampleJSON, "True", "true")
			jsonText = strings.ReplaceAll(jsonText, "False", "false")
			jsonText = strings.ReplaceAll(jsonText, "None", "null")
			test.WriteString(fmt.Sprintf("        mock_response.text = %q\n", jsonText))
			test.WriteString("        mock_request.return_value = mock_response\n\n")

			// Create client
			test.WriteString("        # Create client\n")
			// Use base URL from OpenAPI spec, fallback to default
			baseURL := extractedData.BaseURL
			if baseURL == "" {
				baseURL = "https://api.example.com/v1"
			}
			test.WriteString(fmt.Sprintf("        client = %s(base_url=%q)\n\n", data.ClientClassName, baseURL))

			// Generate method call with parameters
			test.WriteString("        # Call API method\n")
			test.WriteString(fmt.Sprintf("        result = client.%s(", methodName))

			// Add path parameters
			hasParams := false
			for _, param := range op.Parameters {
				if param.In == "path" {
					if hasParams {
						test.WriteString(", ")
					}
					paramName := common.ToSnakeCase(param.Name)
					testValue := generatePythonTestValueFromParam(param)
					test.WriteString(fmt.Sprintf("%s=%s", paramName, testValue))
					hasParams = true
				}
			}

			// Add query parameters
			for _, param := range op.Parameters {
				if param.In == "query" {
					if hasParams {
						test.WriteString(", ")
					}
					paramName := common.ToSnakeCase(param.Name)
					testValue := generatePythonTestValueFromParam(param)
					test.WriteString(fmt.Sprintf("%s=%s", paramName, testValue))
					hasParams = true
				}
			}

			// Add request body if present
			if op.RequestBody != nil {
				if hasParams {
					test.WriteString(", ")
				}
				test.WriteString("body={\"test\": \"data\"}")
			}

			test.WriteString(")\n\n")

			// Assertions
			test.WriteString("        # Assertions\n")
			test.WriteString("        assert result is not None\n")
			test.WriteString("        mock_request.assert_called_once()\n")
			test.WriteString("\n")

			// Generate error handling tests for 4xx/5xx responses
			generatePythonErrorTests(&test, op, data, extractedData)
		}
		test.WriteString("\n")
	}

	return test.String()
}

// generatePythonTestValueFromParam generates a test value from a parameter
func generatePythonTestValueFromParam(param common.Parameter) string {
	if param.Schema == nil {
		return "\"test_value\""
	}
	return generatePythonTestValue(param.Schema, param.Name)
}

// generatePythonAuthTest generates test_auth.py with authentication tests
func generatePythonAuthTest(data common.TemplateData, securitySchemes map[string]common.SecurityScheme, extractedData *common.ExtractedData) string {
	var test bytes.Buffer
	test.WriteString("import pytest\n")
	test.WriteString(fmt.Sprintf("from %s import %s\n\n\n", data.SDKName, data.ClientClassName))

	test.WriteString("class TestAuthentication:\n")
	test.WriteString("    \"\"\"Tests for authentication methods\"\"\"\n\n")

	// Use base URL from OpenAPI spec, fallback to default
	baseURL := extractedData.BaseURL
	if baseURL == "" {
		baseURL = "https://api.example.com/v1"
	}

	// Generate tests for each security scheme
	for name, scheme := range securitySchemes {
		schemeName := common.ToSnakeCase(name)

		switch scheme.Type {
		case "apiKey":
			test.WriteString(fmt.Sprintf("    def test_%s_api_key_auth(self):\n", schemeName))
			test.WriteString(fmt.Sprintf("        \"\"\"Test %s API key authentication.\"\"\"\n", name))
			test.WriteString(fmt.Sprintf("        client = %s(\n", data.ClientClassName))
			test.WriteString("            base_url=\"https://api.example.com\",\n")
			apiKeyValue := "test-api-key" //nolint:goconst // Test value for generated code
			// Use camelCase for attribute name (matches client code)
			attrName := common.ToCamelCase(name)
			test.WriteString(fmt.Sprintf("            %s=%q\n", attrName, apiKeyValue))
			test.WriteString("        )\n")
			test.WriteString(fmt.Sprintf("        assert client.%s == %q\n", attrName, apiKeyValue))
			test.WriteString("\n")

		case "http":
			switch scheme.Scheme {
			case "bearer":
				test.WriteString(fmt.Sprintf("    def test_%s_bearer_auth(self):\n", schemeName))
				test.WriteString(fmt.Sprintf("        \"\"\"Test %s Bearer token authentication.\"\"\"\n", name))
				test.WriteString(fmt.Sprintf("        client = %s(\n", data.ClientClassName))
				test.WriteString(fmt.Sprintf("            base_url=%q,\n", baseURL))
				test.WriteString("            bearer_token=\"test-bearer-token\"\n")
				test.WriteString("        )\n")
				test.WriteString("        assert client.bearer_token == \"test-bearer-token\"\n")
				test.WriteString("\n")
			case "basic":
				test.WriteString(fmt.Sprintf("    def test_%s_basic_auth(self):\n", schemeName))
				test.WriteString(fmt.Sprintf("        \"\"\"Test %s Basic authentication.\"\"\"\n", name))
				test.WriteString(fmt.Sprintf("        client = %s(\n", data.ClientClassName))
				test.WriteString(fmt.Sprintf("            base_url=%q,\n", baseURL))
				test.WriteString("            username=\"test-user\",\n")
				test.WriteString("            password=\"test-password\"\n")
				test.WriteString("        )\n")
				test.WriteString("        assert client.username == \"test-user\"\n")
				test.WriteString("        assert client.password == \"test-password\"\n")
				test.WriteString("\n")
			case "digest":
				test.WriteString(fmt.Sprintf("    def test_%s_digest_auth(self):\n", schemeName))
				test.WriteString(fmt.Sprintf("        \"\"\"Test %s Digest authentication.\"\"\"\n", name))
				test.WriteString(fmt.Sprintf("        client = %s(\n", data.ClientClassName))
				test.WriteString(fmt.Sprintf("            base_url=%q,\n", baseURL))
				test.WriteString("            username=\"test-user\",\n")
				test.WriteString("            password=\"test-password\"\n")
				test.WriteString("        )\n")
				test.WriteString("        assert client.username == \"test-user\"\n")
				test.WriteString("        assert client.password == \"test-password\"\n")
				test.WriteString("\n")
			}

		case "oauth2":
			test.WriteString(fmt.Sprintf("    def test_%s_oauth2_auth(self):\n", schemeName))
			test.WriteString(fmt.Sprintf("        \"\"\"Test %s OAuth2 authentication.\"\"\"\n", name))
			test.WriteString(fmt.Sprintf("        client = %s(\n", data.ClientClassName))
			test.WriteString(fmt.Sprintf("            base_url=%q,\n", baseURL))
			test.WriteString("            oauth2_token=\"test-oauth2-token\"\n")
			test.WriteString("        )\n")
			test.WriteString("        assert client.oauth2_token == \"test-oauth2-token\"\n")
			test.WriteString("\n")

		case "openIdConnect":
			test.WriteString(fmt.Sprintf("    def test_%s_openid_connect_auth(self):\n", schemeName))
			test.WriteString(fmt.Sprintf("        \"\"\"Test %s OpenID Connect authentication.\"\"\"\n", name))
			test.WriteString(fmt.Sprintf("        client = %s(\n", data.ClientClassName))
			test.WriteString(fmt.Sprintf("            base_url=%q,\n", baseURL))
			// Use camelCase for parameter name (matches client code)
			paramName := common.ToCamelCase(name) + "_token"
			test.WriteString(fmt.Sprintf("            %s=\"test-openid-token\"\n", paramName))
			test.WriteString("        )\n")
			test.WriteString(fmt.Sprintf("        assert client.%s == \"test-openid-token\"\n", paramName))
			test.WriteString("\n")

		case "mutualTLS":
			test.WriteString(fmt.Sprintf("    def test_%s_mutual_tls_auth(self):\n", schemeName))
			test.WriteString(fmt.Sprintf("        \"\"\"Test %s Mutual TLS authentication.\"\"\"\n", name))
			test.WriteString(fmt.Sprintf("        client = %s(\n", data.ClientClassName))
			test.WriteString(fmt.Sprintf("            base_url=%q,\n", baseURL))
			// Use camelCase for parameter name (matches client code)
			certParamName := common.ToCamelCase(name) + "_cert"
			keyParamName := common.ToCamelCase(name) + "_key"
			test.WriteString(fmt.Sprintf("            %s=\"test-cert.pem\",\n", certParamName))
			test.WriteString(fmt.Sprintf("            %s=\"test-key.pem\"\n", keyParamName))
			test.WriteString("        )\n")
			test.WriteString(fmt.Sprintf("        assert client.%s == \"test-cert.pem\"\n", certParamName))
			test.WriteString(fmt.Sprintf("        assert client.%s == \"test-key.pem\"\n", keyParamName))
			test.WriteString("\n")
		}
	}

	return test.String()
}

// getPythonExampleFromResponse extracts example from response for Python test
func getPythonExampleFromResponse(response common.Response) string {
	// Look for JSON content type first
	if jsonContent, ok := response.Content["application/json"]; ok {
		if len(jsonContent.Examples) > 0 {
			// Use first example
			for _, example := range jsonContent.Examples {
				return formatExampleForPython(example)
			}
		}
		// Fallback: generate example from schema if no examples
		if jsonContent.Schema != nil {
			return generatePythonExampleFromSchema(jsonContent.Schema)
		}
	}
	// Default fallback
	return "{\"success\": True}"
}

// formatExampleForPython converts an example value to Python code string
func formatExampleForPython(example interface{}) string {
	if example == nil {
		return "None"
	}

	// Use JSON encoding to properly format the example
	jsonBytes, err := json.Marshal(example)
	if err != nil {
		// Fallback to string representation
		return fmt.Sprintf("%#v", example)
	}

	// Convert JSON to Python dict/list format
	jsonStr := string(jsonBytes)
	// Replace JSON null with Python None
	jsonStr = strings.ReplaceAll(jsonStr, "null", "None")
	// Replace JSON true/false with Python True/False
	jsonStr = strings.ReplaceAll(jsonStr, "true", "True")
	jsonStr = strings.ReplaceAll(jsonStr, "false", "False")

	return jsonStr
}

// generatePythonExampleFromSchema generates a Python example from schema
func generatePythonExampleFromSchema(schema *common.Schema) string {
	if schema == nil {
		return "{}"
	}

	switch schema.Type {
	case "object":
		if len(schema.Properties) == 0 {
			return "{}"
		}
		var parts []string
		for propName, propSchema := range schema.Properties {
			value := generatePythonTestValue(propSchema, propName)
			parts = append(parts, fmt.Sprintf("%q: %s", propName, value))
		}
		return fmt.Sprintf("{%s}", strings.Join(parts, ", "))
	case "array":
		if schema.Items != nil {
			itemValue := generatePythonTestValue(schema.Items, "item")
			return fmt.Sprintf("[%s]", itemValue)
		}
		return "[]"
	default:
		return generatePythonTestValue(schema, "value")
	}
}

// generatePythonErrorTests generates error handling tests for 4xx/5xx responses
func generatePythonErrorTests(test *bytes.Buffer, op common.APIOperation, data common.TemplateData, extractedData *common.ExtractedData) {
	// Find error responses (4xx, 5xx)
	errorStatuses := []string{}
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

	methodName := common.GetOperationMethodName(op)

	// Generate test for each error status
	for _, statusCode := range errorStatuses {
		errorTestName := fmt.Sprintf("test_%s_%s_error", methodName, statusCode)
		// Patch requests.Session.request since client uses session.request()
		test.WriteString("    @patch('requests.Session.request')\n")
		fmt.Fprintf(test, "    def %s(self, mock_request):\n", errorTestName)
		fmt.Fprintf(test, "        \"\"\"Test %s %s operation returns %s error.\"\"\"\n", op.Method, op.Path, statusCode)

		// Setup mock error response
		test.WriteString("        # Setup mock error response\n")
		test.WriteString("        mock_response = Mock()\n")
		fmt.Fprintf(test, "        mock_response.status_code = %s\n", statusCode)

		// Get error example if available
		errorResponse := op.Responses[statusCode]
		errorExample := getPythonExampleFromResponse(errorResponse)
		fmt.Fprintf(test, "        mock_response.json.return_value = %s\n", errorExample)
		jsonText := strings.ReplaceAll(errorExample, "True", "true")
		jsonText = strings.ReplaceAll(jsonText, "False", "false")
		jsonText = strings.ReplaceAll(jsonText, "None", "null")
		fmt.Fprintf(test, "        mock_response.text = %q\n", jsonText)
		test.WriteString("        mock_request.return_value = mock_response\n\n")

		// Create client
		baseURL := extractedData.BaseURL
		if baseURL == "" {
			baseURL = "https://api.example.com/v1"
		}
		fmt.Fprintf(test, "        client = %s(base_url=%q)\n\n", data.ClientClassName, baseURL)

		// Call API method and expect error
		test.WriteString("        # Call API method and expect error\n")
		fmt.Fprintf(test, "        # result = client.%s(...)\n", methodName)
		test.WriteString("        # assert result is None or raises exception\n")
		test.WriteString("        # mock_request.assert_called_once()\n")
		test.WriteString("\n")
	}
}
