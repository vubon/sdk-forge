// Package generator provides retry mechanism configuration and utilities.
package generator

import (
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// RetryStrategy represents different retry delay strategies
type RetryStrategy string

const (
	RetryStrategyExponential RetryStrategy = "exponential"
	RetryStrategyLinear      RetryStrategy = "linear"
	RetryStrategyFixed        RetryStrategy = "fixed"
)

// RetryConfig holds configuration for retry mechanism
type RetryConfig struct {
	Enabled              bool          // Whether retry is enabled (default: false)
	MaxAttempts          int           // Maximum retry attempts (default: 3)
	InitialDelay         time.Duration // Initial delay (default: 1s)
	MaxDelay             time.Duration // Maximum delay (default: 60s)
	BackoffMultiplier    float64       // Exponential multiplier (default: 2.0)
	RetryableStatusCodes []int         // HTTP status codes to retry (default: [429, 500, 502, 503, 504])
	RetryOnNetworkErrors bool          // Retry on network errors (default: true)
	Strategy             RetryStrategy // Retry strategy (exponential, linear, fixed)
}

// DefaultRetryConfig returns a retry configuration with sensible defaults
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

// CalculateDelay calculates the delay for a given attempt based on the strategy
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

// IsRetryableStatusCode checks if a status code should trigger a retry
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

// ParseRetryConfigFromOpenAPI extracts retry configuration from OpenAPI document extensions
// Returns nil if no retry extension is found
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

// MergeRetryConfig merges two retry configs, with config2 taking priority over config1
// This allows CLI flags to override OpenAPI extension values
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

