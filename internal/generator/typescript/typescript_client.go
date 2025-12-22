// Package typescript provides TypeScript client generation functionality.
package typescript

import (
	"fmt"
	"strings"

	"github.com/vubon/sdk-forge/internal/generator/common"
	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

// generateTypeScriptClient generates the main client class
func generateTypeScriptClient(data common.TemplateData) (string, error) {
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
	authFields := generateTypeScriptAuthFields(extractedData.SecuritySchemes)
	authSetup := generateTypeScriptAuthSetup(extractedData.SecuritySchemes)

	// Generate retry setup if enabled
	retryFields := ""
	retryHelper := ""
	retryInit := ""
	if data.RetryConfig.Enabled {
		retryFields = generateTypeScriptRetryFields(data.RetryConfig)
		retryHelper = generateTypeScriptRetryHelper(data.HTTPLib, data.RetryConfig)
		retryInit = generateTypeScriptRetryInit(data.RetryConfig)
	}

	// Generate HTTP client initialization based on library
	httpClientInit := generateTypeScriptHTTPClientInit(data.HTTPLib, data.HTTPLibConfig)

	// Load and render template
	tmpl, err := common.LoadTemplate(getTypeScriptClientTemplateContent())
	if err != nil {
		// Fallback to direct generation
		return generateTypeScriptClientFallback(data, extractedData, baseURLDefault, authFields, authSetup, httpClientInit, retryFields, retryHelper, retryInit), nil
	}

	// Prepare template data
	type TypeScriptClientData struct {
		ClientClassName string
		HTTPLibImport   string
		BaseURLDefault  string
		AuthFields      string
		AuthSetup       string
		HTTPClientInit  string
		RetryEnabled    bool
		RetryFields     string
		RetryHelper     string
		RetryInit       string
	}
	templateData := TypeScriptClientData{
		ClientClassName: data.ClientClassName,
		HTTPLibImport:   data.HTTPLibImport,
		BaseURLDefault:  baseURLDefault,
		AuthFields:      authFields,
		AuthSetup:       authSetup,
		HTTPClientInit:  httpClientInit,
		RetryEnabled:    data.RetryConfig.Enabled,
		RetryFields:     retryFields,
		RetryHelper:     retryHelper,
		RetryInit:       retryInit,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, templateData); err != nil {
		// Fallback to direct generation
		return generateTypeScriptClientFallback(data, extractedData, baseURLDefault, authFields, authSetup, httpClientInit, retryFields, retryHelper, retryInit), nil
	}

	return buf.String(), nil
}

// generateTypeScriptClientFallback generates client code directly (fallback when template fails)
func generateTypeScriptClientFallback(
	data common.TemplateData,
	extractedData *common.ExtractedData,
	baseURLDefault, authFields, authSetup, httpClientInit, retryFields, retryHelper, retryInit string,
) string {
	var buf strings.Builder

	// Imports
	buf.WriteString("import { ApiException } from './exceptions';\n")
	if data.HTTPLibImport != "" {
		buf.WriteString(fmt.Sprintf("import %s from '%s';\n", getHTTPLibImportName(data.HTTPLib), data.HTTPLibImport))
	}
	buf.WriteString("\n")

	// Client configuration interface
	buf.WriteString("export interface ClientConfig {\n")
	buf.WriteString("  baseUrl?: string;\n")
	buf.WriteString(authFields)
	if data.RetryConfig.Enabled {
		buf.WriteString("  retryConfig?: RetryConfig;\n")
	}
	buf.WriteString("}\n\n")

	// Retry config interface
	if data.RetryConfig.Enabled {
		buf.WriteString("export interface RetryConfig {\n")
		buf.WriteString("  enabled?: boolean;\n")
		buf.WriteString("  maxAttempts?: number;\n")
		buf.WriteString("  strategy?: 'exponential' | 'linear' | 'fixed';\n")
		buf.WriteString("  initialDelay?: number;\n")
		buf.WriteString("  maxDelay?: number;\n")
		buf.WriteString("  retryableStatusCodes?: number[];\n")
		buf.WriteString("  retryOnNetworkErrors?: boolean;\n")
		buf.WriteString("}\n\n")
	}

	// Client class
	buf.WriteString(fmt.Sprintf("export class %s {\n", data.ClientClassName))
	buf.WriteString("  private baseUrl: string;\n")
	buf.WriteString(fmt.Sprintf("  private httpClient: %s;\n", getHTTPClientType(data.HTTPLib)))
	if data.RetryConfig.Enabled {
		buf.WriteString("  private retryConfig: RetryConfig;\n")
	}
	buf.WriteString(authFields)
	buf.WriteString("\n")

	// Constructor
	buf.WriteString("  constructor(config: ClientConfig = {}) {\n")
	buf.WriteString(fmt.Sprintf("    this.baseUrl = (config.baseUrl || '%s').replace(/\\/$/, '');\n", baseURLDefault))
	buf.WriteString(httpClientInit)
	buf.WriteString(authSetup)
	if data.RetryConfig.Enabled {
		buf.WriteString(retryInit)
	}
	buf.WriteString("  }\n\n")

	// Request method
	buf.WriteString("  async request<T>(options: {\n")
	buf.WriteString("    method: string;\n")
	buf.WriteString("    url: string;\n")
	buf.WriteString("    params?: Record<string, any>;\n")
	buf.WriteString("    headers?: Record<string, string>;\n")
	buf.WriteString("    body?: any;\n")
	buf.WriteString("  }): Promise<T> {\n")
	buf.WriteString("    const url = `${this.baseUrl}${options.url}`;\n")
	buf.WriteString("    \n")
	if data.RetryConfig.Enabled {
		buf.WriteString("    return this.requestWithRetry<T>(url, options);\n")
	} else {
		buf.WriteString(generateTypeScriptRequestMethod(data.HTTPLib))
	}
	buf.WriteString("  }\n\n")

	// Retry helper methods
	if data.RetryConfig.Enabled {
		buf.WriteString(retryHelper)
	}

	// Authentication setter methods
	buf.WriteString(generateTypeScriptAuthSetters(extractedData.SecuritySchemes))

	buf.WriteString("}\n")

	return buf.String()
}

// generateTypeScriptHTTPClientInit generates HTTP client initialization code
func generateTypeScriptHTTPClientInit(httpLib string, libConfig *httplib.LibraryConfig) string {
	switch httpLib {
	case "axios":
		return "    this.httpClient = axios.create({ baseURL: this.baseUrl });\n"
	case "fetch", "node-fetch":
		return "    // fetch is a global function, no initialization needed\n"
	case "ky":
		return "    this.httpClient = ky.create({ prefixUrl: this.baseUrl });\n"
	default:
		return "    // HTTP client initialization\n"
	}
}

// getHTTPClientType returns the TypeScript type for the HTTP client
func getHTTPClientType(httpLib string) string {
	switch httpLib {
	case "axios":
		return "import('axios').AxiosInstance"
	case "fetch", "node-fetch":
		return "typeof fetch"
	case "ky":
		return "import('ky').Ky"
	default:
		return "any"
	}
}

// getHTTPLibImportName returns the import name for the HTTP library
func getHTTPLibImportName(httpLib string) string {
	switch httpLib {
	case "axios":
		return "axios"
	case "fetch":
		return "" // fetch is global
	case "node-fetch":
		return "fetch"
	case "ky":
		return "ky"
	default:
		return "httpClient"
	}
}

// generateTypeScriptRequestMethod generates the request method implementation
func generateTypeScriptRequestMethod(httpLib string) string {
	switch httpLib {
	case "axios":
		return `    try {
      const response = await this.httpClient.request({
        method: options.method,
        url: options.url,
        params: options.params,
        headers: {
          ...this.getAuthHeaders(),
          ...options.headers,
        },
        data: options.body,
      });
      return response.data;
    } catch (error: any) {
      if (error.response) {
        throw new ApiException(
          error.response.data?.message || error.message,
          error.response.status,
          error.response.data
        );
      }
      throw error;
    }
`
	case "fetch", "node-fetch":
		return `    try {
      const url = new URL(options.url, this.baseUrl);
      if (options.params) {
        Object.entries(options.params).forEach(([key, value]) => {
          url.searchParams.append(key, String(value));
        });
      }

      const response = await fetch(url.toString(), {
        method: options.method,
        headers: {
          ...this.getAuthHeaders(),
          ...options.headers,
        },
        body: options.body ? JSON.stringify(options.body) : undefined,
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new ApiException(
          errorData.message || response.statusText,
          response.status,
          errorData
        );
      }

      return await response.json();
    } catch (error: any) {
      if (error instanceof ApiException) {
        throw error;
      }
      throw new ApiException(error.message || 'Request failed', 0, error);
    }
`
	case "ky":
		return `    try {
      const response = await this.httpClient(options.url, {
        method: options.method as any,
        searchParams: options.params,
        headers: {
          ...this.getAuthHeaders(),
          ...options.headers,
        },
        json: options.body,
      });
      return await response.json<T>();
    } catch (error: any) {
      if (error.response) {
        const errorData = await error.response.json().catch(() => ({}));
        throw new ApiException(
          errorData.message || error.message,
          error.response.status,
          errorData
        );
      }
      throw error;
    }
`
	default:
		return `    throw new Error('Unsupported HTTP library');
`
	}
}

// generateTypeScriptAuthFields generates authentication fields
func generateTypeScriptAuthFields(securitySchemes map[string]common.SecurityScheme) string {
	if len(securitySchemes) == 0 {
		return ""
	}

	var buf strings.Builder
	for name, scheme := range securitySchemes {
		switch scheme.Type {
		case "apiKey":
			buf.WriteString(fmt.Sprintf("  private %s?: string;\n", common.ToCamelCase(name)))
		case "http":
			switch scheme.Scheme {
			case "bearer", "Bearer":
				buf.WriteString("  private bearerToken?: string;\n")
			case "basic", "Basic":
				buf.WriteString("  private username?: string;\n")
				buf.WriteString("  private password?: string;\n")
			}
		case "oauth2", "OAuth2":
			buf.WriteString(fmt.Sprintf("  private %sToken?: string;\n", common.ToCamelCase(name)))
			buf.WriteString(fmt.Sprintf("  private %sTokenType?: string;\n", common.ToCamelCase(name)))
		case "openIdConnect", "OpenIDConnect":
			buf.WriteString(fmt.Sprintf("  private %sToken?: string;\n", common.ToCamelCase(name)))
		}
	}
	return buf.String()
}

// generateTypeScriptAuthSetup generates authentication setup code
func generateTypeScriptAuthSetup(securitySchemes map[string]common.SecurityScheme) string {
	if len(securitySchemes) == 0 {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("    // Set up authentication\n")
	for name, scheme := range securitySchemes {
		switch scheme.Type {
		case "apiKey":
			buf.WriteString(fmt.Sprintf("    this.%s = config.%s;\n", common.ToCamelCase(name), common.ToCamelCase(name)))
		case "http":
			switch scheme.Scheme {
			case "bearer", "Bearer":
				buf.WriteString("    this.bearerToken = config.bearerToken;\n")
			case "basic", "Basic":
				buf.WriteString("    this.username = config.username;\n")
				buf.WriteString("    this.password = config.password;\n")
			}
		case "oauth2", "OAuth2":
			buf.WriteString(fmt.Sprintf("    this.%sToken = config.%sToken;\n", common.ToCamelCase(name), common.ToCamelCase(name)))
			tokenName := common.ToCamelCase(name)
			buf.WriteString(fmt.Sprintf("    this.%sTokenType = config.%sTokenType || 'Bearer';\n", tokenName, tokenName))
		case "openIdConnect", "OpenIDConnect":
			buf.WriteString(fmt.Sprintf("    this.%sToken = config.%sToken;\n", common.ToCamelCase(name), common.ToCamelCase(name)))
		}
	}
	return buf.String()
}

// generateTypeScriptAuthSetters generates authentication setter methods
func generateTypeScriptAuthSetters(securitySchemes map[string]common.SecurityScheme) string {
	if len(securitySchemes) == 0 {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("  // Authentication setter methods\n")
	for name, scheme := range securitySchemes {
		switch scheme.Type {
		case "apiKey":
			buf.WriteString(fmt.Sprintf("  set%s(value: string): void {\n", common.ToPascalCase(name)))
			buf.WriteString(fmt.Sprintf("    this.%s = value;\n", common.ToCamelCase(name)))
			buf.WriteString("  }\n\n")
		case "http":
			switch scheme.Scheme {
			case "bearer", "Bearer":
				buf.WriteString("  setBearerToken(token: string): void {\n")
				buf.WriteString("    this.bearerToken = token;\n")
				buf.WriteString("  }\n\n")
			case "basic", "Basic":
				buf.WriteString("  setBasicAuth(username: string, password: string): void {\n")
				buf.WriteString("    this.username = username;\n")
				buf.WriteString("    this.password = password;\n")
				buf.WriteString("  }\n\n")
			}
		case "oauth2", "OAuth2":
			buf.WriteString(fmt.Sprintf("  set%sToken(token: string, tokenType: string = 'Bearer'): void {\n", common.ToPascalCase(name)))
			buf.WriteString(fmt.Sprintf("    this.%sToken = token;\n", common.ToCamelCase(name)))
			buf.WriteString(fmt.Sprintf("    this.%sTokenType = tokenType;\n", common.ToCamelCase(name)))
			buf.WriteString("  }\n\n")
		case "openIdConnect", "OpenIDConnect":
			buf.WriteString(fmt.Sprintf("  set%sToken(token: string): void {\n", common.ToPascalCase(name)))
			buf.WriteString(fmt.Sprintf("    this.%sToken = token;\n", common.ToCamelCase(name)))
			buf.WriteString("  }\n\n")
		}
	}

	// Add getAuthHeaders helper method
	buf.WriteString("  private getAuthHeaders(): Record<string, string> {\n")
	buf.WriteString("    const headers: Record<string, string> = {};\n")
	for name, scheme := range securitySchemes {
		switch scheme.Type {
		case "apiKey":
			buf.WriteString(fmt.Sprintf("    if (this.%s) {\n", common.ToCamelCase(name)))
			switch scheme.In {
			case "header":
				buf.WriteString(fmt.Sprintf("      headers['%s'] = this.%s;\n", scheme.Name, common.ToCamelCase(name)))
			case "query":
				buf.WriteString(fmt.Sprintf("      // Query parameter '%s' will be added per request\n", scheme.Name))
			}
			buf.WriteString("    }\n")
		case "http":
			switch scheme.Scheme {
			case "bearer", "Bearer":
				buf.WriteString("    if (this.bearerToken) {\n")
				buf.WriteString("      headers['Authorization'] = `Bearer ${this.bearerToken}`;\n")
				buf.WriteString("    }\n")
			case "basic", "Basic":
				buf.WriteString("    if (this.username && this.password) {\n")
				buf.WriteString("      const credentials = btoa(`${this.username}:${this.password}`);\n")
				buf.WriteString("      headers['Authorization'] = `Basic ${credentials}`;\n")
				buf.WriteString("    }\n")
			}
		case "oauth2", "OAuth2":
			buf.WriteString(fmt.Sprintf("    if (this.%sToken) {\n", common.ToCamelCase(name)))
			buf.WriteString(fmt.Sprintf("      const tokenType = this.%sTokenType || 'Bearer';\n", common.ToCamelCase(name)))
			buf.WriteString(fmt.Sprintf("      headers['Authorization'] = `${tokenType} ${this.%sToken}`;\n", common.ToCamelCase(name)))
			buf.WriteString("    }\n")
		case "openIdConnect", "OpenIDConnect":
			buf.WriteString(fmt.Sprintf("    if (this.%sToken) {\n", common.ToCamelCase(name)))
			buf.WriteString(fmt.Sprintf("      headers['Authorization'] = `Bearer ${this.%sToken}`;\n", common.ToCamelCase(name)))
			buf.WriteString("    }\n")
		}
	}
	buf.WriteString("    return headers;\n")
	buf.WriteString("  }\n\n")

	return buf.String()
}

// generateTypeScriptRetryFields generates retry configuration fields
func generateTypeScriptRetryFields(config common.RetryConfig) string {
	if !config.Enabled {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("  private retryMaxAttempts: number;\n")
	buf.WriteString("  private retryInitialDelay: number;\n")
	buf.WriteString("  private retryMaxDelay: number;\n")
	buf.WriteString("  private retryBackoffMultiplier: number;\n")
	buf.WriteString("  private retryStrategy: 'exponential' | 'linear' | 'fixed';\n")
	buf.WriteString("  private retryableStatusCodes: number[];\n")
	buf.WriteString("  private retryOnNetworkErrors: boolean;\n")

	return buf.String()
}

// generateTypeScriptRetryInit generates retry configuration initialization
func generateTypeScriptRetryInit(config common.RetryConfig) string {
	if !config.Enabled {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("    // Initialize retry configuration\n")
	buf.WriteString("    const retryConfig = config.retryConfig || {};\n")
	buf.WriteString(fmt.Sprintf("    this.retryMaxAttempts = retryConfig.maxAttempts ?? %d;\n", config.MaxAttempts))
	buf.WriteString(fmt.Sprintf("    this.retryInitialDelay = retryConfig.initialDelay ?? %.0f;\n", config.InitialDelay.Seconds()*1000))
	buf.WriteString(fmt.Sprintf("    this.retryMaxDelay = retryConfig.maxDelay ?? %.0f;\n", config.MaxDelay.Seconds()*1000))
	buf.WriteString(fmt.Sprintf("    this.retryBackoffMultiplier = retryConfig.backoffMultiplier ?? %.1f;\n", config.BackoffMultiplier))
	buf.WriteString(fmt.Sprintf("    this.retryStrategy = retryConfig.strategy ?? '%s';\n", config.Strategy))
	buf.WriteString("    this.retryableStatusCodes = retryConfig.retryableStatusCodes ?? [")
	for i, code := range config.RetryableStatusCodes {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%d", code))
	}
	buf.WriteString("];\n")
	buf.WriteString(fmt.Sprintf("    this.retryOnNetworkErrors = retryConfig.retryOnNetworkErrors ?? %v;\n", config.RetryOnNetworkErrors))

	return buf.String()
}

// generateTypeScriptRetryHelper generates retry helper methods
func generateTypeScriptRetryHelper(httpLib string, config common.RetryConfig) string {
	if !config.Enabled {
		return ""
	}

	var buf strings.Builder

	// Calculate delay helper
	buf.WriteString("  private calculateRetryDelay(attempt: number): number {\n")
	buf.WriteString("    switch (this.retryStrategy) {\n")
	buf.WriteString(fmt.Sprintf("      case '%s':\n", common.RetryStrategyExponential))
	buf.WriteString("        return Math.min(\n")
	buf.WriteString("          this.retryInitialDelay * Math.pow(this.retryBackoffMultiplier, attempt),\n")
	buf.WriteString("          this.retryMaxDelay\n")
	buf.WriteString("        );\n")
	buf.WriteString(fmt.Sprintf("      case '%s':\n", common.RetryStrategyLinear))
	buf.WriteString("        return Math.min(\n")
	buf.WriteString("          this.retryInitialDelay * (attempt + 1),\n")
	buf.WriteString("          this.retryMaxDelay\n")
	buf.WriteString("        );\n")
	buf.WriteString(fmt.Sprintf("      case '%s':\n", common.RetryStrategyFixed))
	buf.WriteString("        return this.retryInitialDelay;\n")
	buf.WriteString("      default:\n")
	buf.WriteString("        return this.retryInitialDelay;\n")
	buf.WriteString("    }\n")
	buf.WriteString("  }\n\n")

	// Request with retry method
	buf.WriteString("  private async requestWithRetry<T>(url: string, options: {\n")
	buf.WriteString("    method: string;\n")
	buf.WriteString("    url: string;\n")
	buf.WriteString("    params?: Record<string, any>;\n")
	buf.WriteString("    headers?: Record<string, string>;\n")
	buf.WriteString("    body?: any;\n")
	buf.WriteString("  }): Promise<T> {\n")
	buf.WriteString("    let lastError: any;\n")
	buf.WriteString("    \n")
	buf.WriteString("    for (let attempt = 0; attempt < this.retryMaxAttempts; attempt++) {\n")
	buf.WriteString("      try {\n")
	buf.WriteString("        return await this.request<T>(options);\n")
	buf.WriteString("      } catch (error: any) {\n")
	buf.WriteString("        lastError = error;\n")
	buf.WriteString("        \n")
	buf.WriteString("        // Check if error is retryable\n")
	buf.WriteString("        if (error instanceof ApiException) {\n")
	buf.WriteString("          if (!this.retryableStatusCodes.includes(error.statusCode)) {\n")
	buf.WriteString("            throw error;\n")
	buf.WriteString("          }\n")
	buf.WriteString("        } else if (!this.retryOnNetworkErrors) {\n")
	buf.WriteString("          throw error;\n")
	buf.WriteString("        }\n")
	buf.WriteString("        \n")
	buf.WriteString("        // If not last attempt, wait and retry\n")
	buf.WriteString("        if (attempt < this.retryMaxAttempts - 1) {\n")
	buf.WriteString("          const delay = this.calculateRetryDelay(attempt);\n")
	buf.WriteString("          await new Promise(resolve => setTimeout(resolve, delay));\n")
	buf.WriteString("        }\n")
	buf.WriteString("      }\n")
	buf.WriteString("    }\n")
	buf.WriteString("    \n")
	buf.WriteString("    // Max attempts exceeded\n")
	buf.WriteString("    throw lastError;\n")
	buf.WriteString("  }\n\n")

	return buf.String()
}
