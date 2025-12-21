// Package generator provides retry mechanism configuration and utilities.
package generator

import (
	"time"
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

