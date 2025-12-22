// Package typescript provides TypeScript exception generation functionality.
package typescript

import (
	"strings"

	"github.com/vubon/sdk-forge/internal/generator/common"
)

// generateTypeScriptExceptions generates exceptions.ts file
func generateTypeScriptExceptions(data common.TemplateData) string {
	var buf strings.Builder

	buf.WriteString("/**\n")
	buf.WriteString(" * API Exception\n")
	buf.WriteString(" * Custom exception class for API errors\n")
	buf.WriteString(" */\n")
	buf.WriteString("export class ApiException extends Error {\n")
	buf.WriteString("  public readonly statusCode: number;\n")
	buf.WriteString("  public readonly responseBody?: any;\n")
	buf.WriteString("\n")
	buf.WriteString("  constructor(\n")
	buf.WriteString("    message: string,\n")
	buf.WriteString("    statusCode: number = 0,\n")
	buf.WriteString("    responseBody?: any\n")
	buf.WriteString("  ) {\n")
	buf.WriteString("    super(message);\n")
	buf.WriteString("    this.name = 'ApiException';\n")
	buf.WriteString("    this.statusCode = statusCode;\n")
	buf.WriteString("    this.responseBody = responseBody;\n")
	buf.WriteString("    \n")
	buf.WriteString("    // Maintains proper stack trace for where our error was thrown (only available on V8)\n")
	buf.WriteString("    if (Error.captureStackTrace) {\n")
	buf.WriteString("      Error.captureStackTrace(this, ApiException);\n")
	buf.WriteString("    }\n")
	buf.WriteString("  }\n")
	buf.WriteString("}\n\n")

	buf.WriteString("/**\n")
	buf.WriteString(" * Network Exception\n")
	buf.WriteString(" * Exception for network-related errors\n")
	buf.WriteString(" */\n")
	buf.WriteString("export class NetworkException extends ApiException {\n")
	buf.WriteString("  constructor(message: string = 'Network error occurred', originalError?: Error) {\n")
	buf.WriteString("    super(message, 0);\n")
	buf.WriteString("    this.name = 'NetworkException';\n")
	buf.WriteString("    if (originalError) {\n")
	buf.WriteString("      this.cause = originalError;\n")
	buf.WriteString("    }\n")
	buf.WriteString("  }\n")
	buf.WriteString("}\n\n")

	buf.WriteString("/**\n")
	buf.WriteString(" * Timeout Exception\n")
	buf.WriteString(" * Exception for request timeout errors\n")
	buf.WriteString(" */\n")
	buf.WriteString("export class TimeoutException extends ApiException {\n")
	buf.WriteString("  constructor(message: string = 'Request timeout', timeoutMs?: number) {\n")
	buf.WriteString("    super(message, 408);\n")
	buf.WriteString("    this.name = 'TimeoutException';\n")
	buf.WriteString("  }\n")
	buf.WriteString("}\n")

	return buf.String()
}
