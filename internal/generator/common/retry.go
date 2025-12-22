// Package common provides retry mechanism configuration and utilities.
//
// The retry mechanism allows generated SDKs to automatically retry failed HTTP requests
// with configurable strategies (exponential backoff, linear backoff, or fixed delay).
// Retry can be configured via CLI flags or OpenAPI extensions (x-sdk-forge-retry).
package common

import (
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// RetryStrategy represents different retry delay strategies.
//
// Strategies determine how the delay between retry attempts is calculated:
//   - Exponential: Delay increases exponentially (1s, 2s, 4s, 8s...)
//   - Linear: Delay increases linearly (1s, 2s, 3s, 4s...)
//   - Fixed: Constant delay between retries (1s, 1s, 1s...)
type RetryStrategy string

const (
	// RetryStrategyExponential uses exponential backoff: initialDelay * (multiplier ^ attempt)
	// Best for rate-limited APIs or high-load scenarios
	RetryStrategyExponential RetryStrategy = "exponential"

	// RetryStrategyLinear uses linear backoff: initialDelay * (attempt + 1)
	// Best for predictable retry patterns
	RetryStrategyLinear RetryStrategy = "linear"

	// RetryStrategyFixed uses constant delay: always initialDelay
	// Best for simple retry scenarios
	RetryStrategyFixed RetryStrategy = "fixed"
)

// RetryConfig holds configuration for retry mechanism in generated SDKs.
//
// This configuration is used to generate retry logic in Python and Go SDKs.
// When Enabled is false, no retry logic is generated (backward compatible).
//
// Example usage:
//
//	config := DefaultRetryConfig()
//	config.Enabled = true
//	config.MaxAttempts = 5
//	config.Strategy = RetryStrategyExponential
type RetryConfig struct {
	// Enabled controls whether retry logic is generated in SDKs.
	// Default: false (disabled for backward compatibility)
	Enabled bool

	// MaxAttempts is the maximum number of retry attempts (including initial request).
	// Default: 3 (1 initial + 2 retries)
	MaxAttempts int

	// InitialDelay is the delay before the first retry attempt.
	// Default: 1 second
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retry attempts (caps exponential/linear backoff).
	// Default: 60 seconds
	MaxDelay time.Duration

	// BackoffMultiplier is used for exponential backoff strategy.
	// Formula: delay = initialDelay * (multiplier ^ attempt)
	// Default: 2.0
	BackoffMultiplier float64

	// RetryableStatusCodes are HTTP status codes that trigger a retry.
	// Default: [429, 500, 502, 503, 504]
	// Common codes:
	//   - 429: Too Many Requests (rate limit)
	//   - 500: Internal Server Error
	//   - 502: Bad Gateway
	//   - 503: Service Unavailable
	//   - 504: Gateway Timeout
	RetryableStatusCodes []int

	// RetryOnNetworkErrors controls whether to retry on network errors
	// (connection refused, timeouts, DNS errors, etc.).
	// Default: true
	RetryOnNetworkErrors bool

	// Strategy determines how delay is calculated between retry attempts.
	// Default: RetryStrategyExponential
	Strategy RetryStrategy
}

// DefaultRetryConfig returns a retry configuration with sensible defaults.
//
// The default configuration has retry disabled (Enabled=false) for backward compatibility.
// When enabled, it uses exponential backoff with 3 max attempts and retries on
// common server errors (429, 5xx) and network errors.
//
// Returns:
//   - Enabled: false (must be explicitly enabled)
//   - MaxAttempts: 3
//   - InitialDelay: 1 second
//   - MaxDelay: 60 seconds
//   - BackoffMultiplier: 2.0
//   - Strategy: exponential
//   - RetryableStatusCodes: [429, 500, 502, 503, 504]
//   - RetryOnNetworkErrors: true
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		Enabled:              false, // Disabled by default for backward compatibility
		MaxAttempts:          3,
		InitialDelay:         time.Second,
		MaxDelay:             60 * time.Second,
		BackoffMultiplier:    2.0,
		RetryableStatusCodes: []int{429, 500, 502, 503, 504},
		RetryOnNetworkErrors: true,
		Strategy:             RetryStrategyExponential,
	}
}

// CalculateDelay calculates the delay for a given retry attempt based on the configured strategy.
//
// The delay is calculated as follows:
//   - Exponential: initialDelay * (backoffMultiplier ^ attempt)
//   - Linear: initialDelay * (attempt + 1)
//   - Fixed: always initialDelay
//
// The result is capped at MaxDelay to prevent excessively long waits.
//
// Parameters:
//   - attempt: The retry attempt number (0-based, where 0 is the first retry)
//
// Returns:
//   - The calculated delay duration, capped at MaxDelay
//
// Example:
//
//	config := RetryConfig{
//	    Strategy: RetryStrategyExponential,
//	    InitialDelay: time.Second,
//	    BackoffMultiplier: 2.0,
//	    MaxDelay: 60 * time.Second,
//	}
//	delay1 := config.CalculateDelay(0) // Returns 1s
//	delay2 := config.CalculateDelay(1) // Returns 2s
//	delay3 := config.CalculateDelay(2) // Returns 4s
func (rc RetryConfig) CalculateDelay(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	var delay time.Duration

	switch rc.Strategy {
	case RetryStrategyExponential:
		// Exponential backoff: initialDelay * (multiplier ^ attempt)
		delay = time.Duration(float64(rc.InitialDelay) * pow(rc.BackoffMultiplier, float64(attempt)))
	case RetryStrategyLinear:
		// Linear backoff: initialDelay * (attempt + 1)
		delay = rc.InitialDelay * time.Duration(attempt+1)
	case RetryStrategyFixed:
		// Fixed delay: always use initialDelay
		delay = rc.InitialDelay
	default:
		// Default to exponential if unknown strategy
		delay = time.Duration(float64(rc.InitialDelay) * pow(rc.BackoffMultiplier, float64(attempt)))
	}

	// Cap at max delay
	if delay > rc.MaxDelay {
		delay = rc.MaxDelay
	}

	return delay
}

// IsRetryableStatusCode checks if an HTTP status code should trigger a retry.
//
// This method checks if the given status code is in the RetryableStatusCodes list.
// Common retryable codes include:
//   - 429: Too Many Requests (rate limit - may succeed after delay)
//   - 5xx: Server errors (may be transient)
//
// Non-retryable codes typically include:
//   - 2xx: Success (no retry needed)
//   - 3xx: Redirects (no retry needed)
//   - 4xx (except 429): Client errors (won't succeed on retry)
//
// Parameters:
//   - statusCode: The HTTP status code to check
//
// Returns:
//   - true if the status code should trigger a retry, false otherwise
func (rc RetryConfig) IsRetryableStatusCode(statusCode int) bool {
	for _, code := range rc.RetryableStatusCodes {
		if statusCode == code {
			return true
		}
	}
	return false
}

// pow calculates x^y (simple implementation for small values)
func pow(x, y float64) float64 {
	result := 1.0
	for i := 0; i < int(y); i++ {
		result *= x
	}
	return result
}

// ParseRetryConfigFromOpenAPI extracts retry configuration from OpenAPI document extensions.
//
// This function looks for the `x-sdk-forge-retry` extension in the OpenAPI document.
// The extension should be a map with the following optional fields:
//   - enabled: bool
//   - maxAttempts: number
//   - initialDelay: number (seconds)
//   - maxDelay: number (seconds)
//   - backoffMultiplier: number
//   - strategy: string ("exponential", "linear", or "fixed")
//   - retryableStatusCodes: array of numbers
//   - retryOnNetworkErrors: bool
//
// Example OpenAPI extension:
//
//	x-sdk-forge-retry:
//	  enabled: true
//	  maxAttempts: 3
//	  initialDelay: 1
//	  maxDelay: 60
//	  strategy: exponential
//	  retryableStatusCodes: [429, 500, 502, 503, 504]
//
// Parameters:
//   - doc: The OpenAPI document (must be *openapi3.T)
//
// Returns:
//   - *RetryConfig if extension is found and valid, nil otherwise
//
// Note: CLI flags take priority over OpenAPI extension values. Use MergeRetryConfig
// to combine OpenAPI extension (base) with CLI flags (override).
func ParseRetryConfigFromOpenAPI(doc interface{}) *RetryConfig {
	// Try to access Extensions from openapi3.T
	var extensions map[string]interface{}

	// Use type assertion to get the document
	if openapiDoc, ok := doc.(*openapi3.T); ok {
		if openapiDoc.Extensions == nil {
			return nil
		}
		extensions = openapiDoc.Extensions
	} else {
		// If not openapi3.T, try to get from ExtractedData or return nil
		return nil
	}

	// Look for x-sdk-forge-retry extension
	retryExt, exists := extensions["x-sdk-forge-retry"]
	if !exists {
		return nil
	}

	// Parse the extension value
	retryMap, ok := retryExt.(map[string]interface{})
	if !ok {
		return nil
	}

	config := DefaultRetryConfig()

	// Parse enabled
	if enabled, ok := retryMap["enabled"].(bool); ok {
		config.Enabled = enabled
	}

	// Parse maxAttempts
	if maxAttempts, ok := retryMap["maxAttempts"].(float64); ok {
		config.MaxAttempts = int(maxAttempts)
	}

	// Parse initialDelay (in seconds, convert to Duration)
	if initialDelay, ok := retryMap["initialDelay"].(float64); ok {
		config.InitialDelay = time.Duration(initialDelay) * time.Second
	}

	// Parse maxDelay (in seconds, convert to Duration)
	if maxDelay, ok := retryMap["maxDelay"].(float64); ok {
		config.MaxDelay = time.Duration(maxDelay) * time.Second
	}

	// Parse backoffMultiplier
	if backoffMult, ok := retryMap["backoffMultiplier"].(float64); ok {
		config.BackoffMultiplier = backoffMult
	}

	// Parse strategy
	if strategy, ok := retryMap["strategy"].(string); ok {
		switch strings.ToLower(strategy) {
		case "exponential":
			config.Strategy = RetryStrategyExponential
		case "linear":
			config.Strategy = RetryStrategyLinear
		case "fixed":
			config.Strategy = RetryStrategyFixed
		default:
			config.Strategy = RetryStrategyExponential
		}
	}

	// Parse retryableStatusCodes
	if codes, ok := retryMap["retryableStatusCodes"].([]interface{}); ok {
		statusCodes := make([]int, 0, len(codes))
		for _, code := range codes {
			if codeFloat, ok := code.(float64); ok {
				statusCodes = append(statusCodes, int(codeFloat))
			}
		}
		if len(statusCodes) > 0 {
			config.RetryableStatusCodes = statusCodes
		}
	}

	// Parse retryOnNetworkErrors
	if retryOnNetwork, ok := retryMap["retryOnNetworkErrors"].(bool); ok {
		config.RetryOnNetworkErrors = retryOnNetwork
	}

	return &config
}

// MergeRetryConfig merges two retry configurations, with config2 taking priority over config1.
//
// This function is used to combine OpenAPI extension values (config1) with CLI flag values (config2).
// CLI flags override OpenAPI extension values when both are provided.
//
// Merge rules:
//   - If config2 has a non-zero/non-empty value, it overrides config1
//   - If config2 has a zero/empty value, config1's value is kept
//   - For boolean fields, config2's value is used if config2.Enabled is true
//
// Typical usage:
//
//	openAPIConfig := ParseRetryConfigFromOpenAPI(doc)  // Base from OpenAPI
//	cliConfig := parseRetryConfigFromFlags(cmd)        // Override from CLI
//	finalConfig := MergeRetryConfig(*openAPIConfig, cliConfig)
//
// Parameters:
//   - config1: Base configuration (typically from OpenAPI extension)
//   - config2: Override configuration (typically from CLI flags)
//
// Returns:
//   - Merged RetryConfig with config2 values taking priority
func MergeRetryConfig(config1, config2 RetryConfig) RetryConfig {
	result := config1

	// If config2 has enabled set, use it
	if config2.Enabled {
		result.Enabled = config2.Enabled
	}

	// If config2 has non-zero values, use them (CLI flags override)
	if config2.MaxAttempts > 0 {
		result.MaxAttempts = config2.MaxAttempts
	}
	if config2.InitialDelay > 0 {
		result.InitialDelay = config2.InitialDelay
	}
	if config2.MaxDelay > 0 {
		result.MaxDelay = config2.MaxDelay
	}
	if config2.BackoffMultiplier > 0 {
		result.BackoffMultiplier = config2.BackoffMultiplier
	}
	if config2.Strategy != "" {
		result.Strategy = config2.Strategy
	}
	if len(config2.RetryableStatusCodes) > 0 {
		result.RetryableStatusCodes = config2.RetryableStatusCodes
	}
	// retryOnNetworkErrors is a boolean, so we check if it was explicitly set
	// For now, we'll use config2's value if config2.Enabled is true
	if config2.Enabled {
		result.RetryOnNetworkErrors = config2.RetryOnNetworkErrors
	}

	return result
}
