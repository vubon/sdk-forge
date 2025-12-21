// Package generator provides PHP client generation functionality.
package generator

import (
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// generatePHPClient generates PHP client class using template
func generatePHPClient(data TemplateData, version LanguageVersion) (string, error) {
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

	// Format SDK name for display
	displayName := strings.ReplaceAll(data.SDKName, "_", " ")
	c := cases.Title(language.English)
	displayName = c.String(strings.ToLower(displayName))

	// Generate namespace (using Vendor\SDKName pattern)
	namespace := fmt.Sprintf("Vendor\\%s", data.SDKName)
	clientClassName := getClientClassName(data.SDKName)

	// Generate authentication setup
	authSetup := generatePHPAuthSetup(extractedData.SecuritySchemes, clientClassName)

	// Generate retry setup if enabled
	retryFields := ""
	retryInit := ""
	retryHelper := ""
	if data.RetryConfig.Enabled {
		retryFields = generatePHPRetryFields(data.RetryConfig)
		retryInit = generatePHPRetryInit(data.RetryConfig)
		retryHelper = generatePHPRetryHelper(data.HTTPLib, data.RetryConfig, clientClassName)
	}

	// Build imports
	imports := buildPHPImports(data, version, data.RetryConfig.Enabled)

	// Generate authentication fields
	authFields := generatePHPAuthFields(extractedData.SecuritySchemes)

	// Generate auth method body
	authMethodBody := generatePHPAuthMethodBody(extractedData.SecuritySchemes)

	// Prepare template data
	type PHPClientData struct {
		Namespace        string
		Imports          string
		DisplayName      string
		ClientClassName  string
		HTTPClientType   string
		BaseURLDefault   string
		HTTPClientMethod string
		AuthFields       string
		RetryEnabled     bool
		RetryFields      string
		RetryInit        string
		RetryHelper      string
		AuthSetup        string
		AuthMethodBody   string
	}
	templateData := PHPClientData{
		Namespace:        namespace,
		Imports:          imports,
		DisplayName:      displayName,
		ClientClassName:  clientClassName,
		HTTPClientType:   getPHPHTTPClientType(data.HTTPLib),
		BaseURLDefault:   baseURLDefault,
		HTTPClientMethod: getPHPHTTPClientMethod(data.HTTPLib),
		AuthFields:       authFields,
		RetryEnabled:     data.RetryConfig.Enabled,
		RetryFields:      retryFields,
		RetryInit:        retryInit,
		RetryHelper:      retryHelper,
		AuthSetup:        authSetup,
		AuthMethodBody:   authMethodBody,
	}

	// Load and render template
	tmpl, err := LoadTemplate(GetPHPClientTemplate())
	if err != nil {
		return "", fmt.Errorf("failed to load PHP client template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", fmt.Errorf("failed to execute PHP client template: %w", err)
	}

	return buf.String(), nil
}

// generatePHPAuthFields generates authentication field declarations
func generatePHPAuthFields(securitySchemes map[string]SecurityScheme) string {
	if len(securitySchemes) == 0 {
		return ""
	}

	var fields strings.Builder
	fields.WriteString("    // Authentication fields\n")
	for name, scheme := range securitySchemes {
		switch scheme.Type {
		case securitySchemeAPIKey:
			fields.WriteString(fmt.Sprintf("    private ?string $%s = null;\n", toCamelCase(name)))
		case securitySchemeHTTP:
			switch scheme.Scheme {
			case securitySchemeBearer:
				fields.WriteString("    private ?string $bearerToken = null;\n")
			case securitySchemeBasic, securitySchemeDigest:
				fields.WriteString("    private ?string $username = null;\n")
				fields.WriteString("    private ?string $password = null;\n")
			}
		case securitySchemeOAuth2, securitySchemeOpenIDConnect:
			fields.WriteString(fmt.Sprintf("    private ?string $%sToken = null;\n", toCamelCase(name)))
		case securitySchemeMutualTLS:
			fields.WriteString(fmt.Sprintf("    private ?string $%sCert = null;\n", toCamelCase(name)))
			fields.WriteString(fmt.Sprintf("    private ?string $%sKey = null;\n", toCamelCase(name)))
		}
	}
	fields.WriteString("\n")
	return fields.String()
}

// generatePHPAuthMethodBody generates the body of the applyAuth method
func generatePHPAuthMethodBody(securitySchemes map[string]SecurityScheme) string {
	if len(securitySchemes) == 0 {
		return "        // No authentication required\n"
	}

	var body strings.Builder
	body.WriteString("        if (!isset($options['headers'])) {\n")
	body.WriteString("            $options['headers'] = [];\n")
	body.WriteString("        }\n\n")

	for name, scheme := range securitySchemes {
		switch scheme.Type {
		case securitySchemeAPIKey:
			fieldName := toCamelCase(name)
			body.WriteString(fmt.Sprintf("        if ($this->%s !== null) {\n", fieldName))
			switch scheme.In {
			case paramLocationHeader:
				body.WriteString(fmt.Sprintf("            $options['headers']['%s'] = $this->%s;\n", scheme.Name, fieldName))
			case paramLocationQuery:
				body.WriteString("            if (!isset($options['query'])) {\n")
				body.WriteString("                $options['query'] = [];\n")
				body.WriteString("            }\n")
				body.WriteString(fmt.Sprintf("            $options['query']['%s'] = $this->%s;\n", scheme.Name, fieldName))
			}
			body.WriteString("        }\n\n")
		case securitySchemeHTTP:
			switch scheme.Scheme {
			case securitySchemeBearer:
				body.WriteString("        if ($this->bearerToken !== null) {\n")
				body.WriteString("            $options['headers']['Authorization'] = 'Bearer ' . $this->bearerToken;\n")
				body.WriteString("        }\n\n")
			case securitySchemeBasic:
				body.WriteString("        if ($this->username !== null && $this->password !== null) {\n")
				body.WriteString("            $options['auth'] = [$this->username, $this->password];\n")
				body.WriteString("        }\n\n")
			}
		case securitySchemeOAuth2, securitySchemeOpenIDConnect:
			fieldName := toCamelCase(name)
			body.WriteString(fmt.Sprintf("        if ($this->%sToken !== null) {\n", fieldName))
			body.WriteString(fmt.Sprintf("            $options['headers']['Authorization'] = 'Bearer ' . $this->%sToken;\n", fieldName))
			body.WriteString("        }\n\n")
		}
	}
	return body.String()
}

// buildPHPImports builds the import/use statements for PHP
func buildPHPImports(data TemplateData, version LanguageVersion, retryEnabled bool) string {
	var imports strings.Builder
	imports.WriteString("use " + data.HTTPLibImport + ";\n")
	imports.WriteString(fmt.Sprintf("use %s\\Exceptions\\ApiException;\n", getPHPNamespace(data.SDKName)))

	if retryEnabled {
		imports.WriteString("use Psr\\Http\\Message\\ResponseInterface;\n")
	}

	return imports.String()
}

// getPHPNamespace returns the PHP namespace for the SDK
func getPHPNamespace(sdkName string) string {
	return fmt.Sprintf("Vendor\\%s", sdkName)
}

// getPHPHTTPClientType returns the PHP type for the HTTP client based on library
func getPHPHTTPClientType(httpLib string) string {
	switch httpLib {
	case "guzzle":
		return "\\GuzzleHttp\\Client"
	case "symfony":
		return "\\Symfony\\Component\\HttpClient\\HttpClient"
	case "curl":
		return "\\Curl\\Curl"
	default:
		return "\\GuzzleHttp\\Client"
	}
}

// getPHPHTTPClientMethod returns the method name for making requests based on HTTP library
func getPHPHTTPClientMethod(httpLib string) string {
	switch httpLib {
	case "guzzle":
		return "request"
	case "symfony":
		return "request"
	case "curl":
		return "request"
	default:
		return "request"
	}
}

// generatePHPAuthSetup generates PHP authentication setup code
func generatePHPAuthSetup(securitySchemes map[string]SecurityScheme, clientClassName string) string {
	if len(securitySchemes) == 0 {
		return "        // No authentication required\n"
	}

	var setupCode strings.Builder
	setupCode.WriteString("        // Set up authentication\n")

	for name, scheme := range securitySchemes {
		switch scheme.Type {
		case securitySchemeAPIKey:
			fieldName := toCamelCase(name)
			setupCode.WriteString(fmt.Sprintf("        $this->%s = $options['%s'] ?? null;\n", fieldName, name))
		case securitySchemeHTTP:
			switch scheme.Scheme {
			case securitySchemeBearer:
				setupCode.WriteString("        $this->bearerToken = $options['bearer_token'] ?? null;\n")
			case securitySchemeBasic, securitySchemeDigest:
				setupCode.WriteString("        $this->username = $options['username'] ?? null;\n")
				setupCode.WriteString("        $this->password = $options['password'] ?? null;\n")
			}
		case securitySchemeOAuth2:
			fieldName := toCamelCase(name)
			setupCode.WriteString(fmt.Sprintf("        $this->%sToken = $options['%s_token'] ?? null;\n", fieldName, name))
		case securitySchemeOpenIDConnect:
			fieldName := toCamelCase(name)
			setupCode.WriteString(fmt.Sprintf("        $this->%sToken = $options['%s_token'] ?? null;\n", fieldName, name))
		case securitySchemeMutualTLS:
			fieldName := toCamelCase(name)
			setupCode.WriteString(fmt.Sprintf("        $this->%sCert = $options['%s_cert'] ?? null;\n", fieldName, name))
			setupCode.WriteString(fmt.Sprintf("        $this->%sKey = $options['%s_key'] ?? null;\n", fieldName, name))
		}
	}

	return setupCode.String()
}

// generatePHPRetryFields generates PHP retry configuration fields
func generatePHPRetryFields(config RetryConfig) string {
	if !config.Enabled {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("    // Retry configuration\n")
	buf.WriteString("    private bool $retryEnabled = true;\n")
	buf.WriteString(fmt.Sprintf("    private int $retryMaxAttempts = %d;\n", config.MaxAttempts))
	buf.WriteString(fmt.Sprintf("    private float $retryInitialDelay = %.1f;\n", config.InitialDelay.Seconds()))
	buf.WriteString(fmt.Sprintf("    private float $retryMaxDelay = %.1f;\n", config.MaxDelay.Seconds()))
	buf.WriteString(fmt.Sprintf("    private float $retryBackoffMultiplier = %.1f;\n", config.BackoffMultiplier))
	buf.WriteString(fmt.Sprintf("    private string $retryStrategy = '%s';\n", config.Strategy))
	buf.WriteString("    private array $retryableStatusCodes = [")
	for i, code := range config.RetryableStatusCodes {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%d", code))
	}
	buf.WriteString("];\n")
	retryOnNetworkErrors := "false"
	if config.RetryOnNetworkErrors {
		retryOnNetworkErrors = "true"
	}
	buf.WriteString(fmt.Sprintf("    private bool $retryOnNetworkErrors = %s;\n", retryOnNetworkErrors))

	return buf.String()
}

// generatePHPRetryInit generates PHP retry configuration initialization
func generatePHPRetryInit(config RetryConfig) string {
	if !config.Enabled {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("\n        // Initialize retry configuration\n")
	if config.MaxAttempts > 0 {
		buf.WriteString(fmt.Sprintf("        $this->retryMaxAttempts = $options['retry_max_attempts'] ?? %d;\n", config.MaxAttempts))
	}
	if config.InitialDelay.Seconds() > 0 {
		buf.WriteString(fmt.Sprintf("        $this->retryInitialDelay = $options['retry_initial_delay'] ?? %.1f;\n", config.InitialDelay.Seconds()))
	}
	if config.MaxDelay.Seconds() > 0 {
		buf.WriteString(fmt.Sprintf("        $this->retryMaxDelay = $options['retry_max_delay'] ?? %.1f;\n", config.MaxDelay.Seconds()))
	}
	if config.BackoffMultiplier > 0 {
		buf.WriteString(fmt.Sprintf("        $this->retryBackoffMultiplier = $options['retry_backoff_multiplier'] ?? %.1f;\n", config.BackoffMultiplier))
	}
	buf.WriteString(fmt.Sprintf("        $this->retryStrategy = $options['retry_strategy'] ?? '%s';\n", config.Strategy))
	buf.WriteString("        $this->retryableStatusCodes = $options['retry_status_codes'] ?? [")
	for i, code := range config.RetryableStatusCodes {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%d", code))
	}
	buf.WriteString("];\n")
	retryOnNetworkErrors := "false"
	if config.RetryOnNetworkErrors {
		retryOnNetworkErrors = "true"
	}
	buf.WriteString(fmt.Sprintf("        $this->retryOnNetworkErrors = $options['retry_on_network_errors'] ?? %s;\n", retryOnNetworkErrors))

	return buf.String()
}

// generatePHPRetryHelper generates PHP retry helper methods
func generatePHPRetryHelper(httpLib string, config RetryConfig, clientClassName string) string {
	if !config.Enabled {
		return ""
	}

	var buf strings.Builder

	// Calculate retry delay helper
	buf.WriteString("    /**\n")
	buf.WriteString("     * Calculate delay for retry attempt based on strategy\n")
	buf.WriteString("     *\n")
	buf.WriteString("     * @param int $attempt Retry attempt number (0-based)\n")
	buf.WriteString("     * @return float Delay in seconds\n")
	buf.WriteString("     */\n")
	buf.WriteString("    private function calculateRetryDelay(int $attempt): float\n")
	buf.WriteString("    {\n")
	buf.WriteString(fmt.Sprintf("        if ($this->retryStrategy === '%s') {\n", RetryStrategyExponential))
	buf.WriteString("            // Exponential backoff: initialDelay * (multiplier ^ attempt)\n")
	buf.WriteString("            $delay = $this->retryInitialDelay * pow($this->retryBackoffMultiplier, $attempt);\n")
	buf.WriteString(fmt.Sprintf("        } elseif ($this->retryStrategy === '%s') {\n", RetryStrategyLinear))
	buf.WriteString("            // Linear backoff: initialDelay * (attempt + 1)\n")
	buf.WriteString("            $delay = $this->retryInitialDelay * ($attempt + 1);\n")
	buf.WriteString(fmt.Sprintf("        } elseif ($this->retryStrategy === '%s') {\n", RetryStrategyFixed))
	buf.WriteString("            // Fixed delay: always use initialDelay\n")
	buf.WriteString("            $delay = $this->retryInitialDelay;\n")
	buf.WriteString("        } else {\n")
	buf.WriteString("            // Default to exponential\n")
	buf.WriteString("            $delay = $this->retryInitialDelay * pow($this->retryBackoffMultiplier, $attempt);\n")
	buf.WriteString("        }\n\n")
	buf.WriteString("        return min($delay, $this->retryMaxDelay);\n")
	buf.WriteString("    }\n\n")

	// Is retryable status code helper
	buf.WriteString("    /**\n")
	buf.WriteString("     * Check if a status code should trigger a retry\n")
	buf.WriteString("     *\n")
	buf.WriteString("     * @param int $statusCode HTTP status code\n")
	buf.WriteString("     * @return bool True if retryable\n")
	buf.WriteString("     */\n")
	buf.WriteString("    private function isRetryableStatusCode(int $statusCode): bool\n")
	buf.WriteString("    {\n")
	buf.WriteString("        return in_array($statusCode, $this->retryableStatusCodes, true);\n")
	buf.WriteString("    }\n\n")

	// Request with retry method (for Guzzle)
	if httpLib == "guzzle" {
		buf.WriteString("    /**\n")
		buf.WriteString("     * Make HTTP request with retry logic\n")
		buf.WriteString("     *\n")
		buf.WriteString("     * @param string $method HTTP method\n")
		buf.WriteString("     * @param string $url Request URL\n")
		buf.WriteString("     * @param array<string, mixed> $options Request options\n")
		buf.WriteString("     * @return array<string, mixed> Response data\n")
		buf.WriteString("     * @throws ApiException\n")
		buf.WriteString("     */\n")
		buf.WriteString("    private function requestWithRetry(string $method, string $url, array $options = []): array\n")
		buf.WriteString("    {\n")
		buf.WriteString("        $lastException = null;\n\n")
		buf.WriteString("        for ($attempt = 0; $attempt < $this->retryMaxAttempts; $attempt++) {\n")
		buf.WriteString("            try {\n")
		buf.WriteString("                $response = $this->httpClient->request($method, $url, $options);\n")
		buf.WriteString("                $statusCode = $response->getStatusCode();\n\n")
		buf.WriteString("                // Check if status code is retryable\n")
		buf.WriteString("                if (!$this->isRetryableStatusCode($statusCode)) {\n")
		buf.WriteString("                    return json_decode($response->getBody()->getContents(), true);\n")
		buf.WriteString("                }\n")
		buf.WriteString("                // Retryable status code - will retry below\n")
		buf.WriteString("            } catch (\\GuzzleHttp\\Exception\\RequestException $e) {\n")
		buf.WriteString("                if (!$this->retryOnNetworkErrors) {\n")
		buf.WriteString("                    throw new ApiException($e->getMessage(), $e->getCode(), $e);\n")
		buf.WriteString("                }\n")
		buf.WriteString("                $lastException = $e;\n")
		buf.WriteString("                // Network error - will retry below\n")
		buf.WriteString("            }\n\n")
		buf.WriteString("            // If we get here, we need to retry\n")
		buf.WriteString("            if ($attempt < $this->retryMaxAttempts - 1) {\n")
		buf.WriteString("                $delay = $this->calculateRetryDelay($attempt);\n")
		buf.WriteString("                usleep((int)($delay * 1000000)); // Convert seconds to microseconds\n")
		buf.WriteString("            } else {\n")
		buf.WriteString("                // Max attempts exceeded\n")
		buf.WriteString("                if ($lastException !== null) {\n")
		buf.WriteString("                    throw new ApiException($lastException->getMessage(), $lastException->getCode(), $lastException);\n")
		buf.WriteString("                }\n")
		buf.WriteString("                // If we got a response but it's a retryable status code, throw error\n")
		buf.WriteString("                if (isset($response)) {\n")
		buf.WriteString("                    $body = $response->getBody()->getContents();\n")
		buf.WriteString("                    throw new ApiException(\n")
		buf.WriteString("                        \"Request failed after {$this->retryMaxAttempts} attempts\",\n")
		buf.WriteString("                        $statusCode,\n")
		buf.WriteString("                        null,\n")
		buf.WriteString("                        json_decode($body, true)\n")
		buf.WriteString("                    );\n")
		buf.WriteString("                }\n")
		buf.WriteString("            }\n")
		buf.WriteString("        }\n\n")
		buf.WriteString("        // Should never reach here, but just in case\n")
		buf.WriteString("        throw new ApiException(\"Request failed after {$this->retryMaxAttempts} attempts\");\n")
		buf.WriteString("    }\n")
	}

	return buf.String()
}
