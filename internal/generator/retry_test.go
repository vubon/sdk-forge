package generator

import (
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.Enabled {
		t.Error("Default retry should be disabled")
	}
	if config.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts=3, got %d", config.MaxAttempts)
	}
	if config.InitialDelay != time.Second {
		t.Errorf("Expected InitialDelay=1s, got %v", config.InitialDelay)
	}
	if config.MaxDelay != 60*time.Second {
		t.Errorf("Expected MaxDelay=60s, got %v", config.MaxDelay)
	}
	if config.BackoffMultiplier != 2.0 {
		t.Errorf("Expected BackoffMultiplier=2.0, got %f", config.BackoffMultiplier)
	}
	if config.Strategy != RetryStrategyExponential {
		t.Errorf("Expected Strategy=exponential, got %s", config.Strategy)
	}
	if len(config.RetryableStatusCodes) != 5 {
		t.Errorf("Expected 5 retryable status codes, got %d", len(config.RetryableStatusCodes))
	}
	expectedCodes := []int{429, 500, 502, 503, 504}
	for i, code := range expectedCodes {
		if config.RetryableStatusCodes[i] != code {
			t.Errorf("Expected status code %d at index %d, got %d", code, i, config.RetryableStatusCodes[i])
		}
	}
	if !config.RetryOnNetworkErrors {
		t.Error("Expected RetryOnNetworkErrors=true")
	}
}

func TestCalculateDelay_Exponential(t *testing.T) {
	config := RetryConfig{
		Strategy:          RetryStrategyExponential,
		InitialDelay:     time.Second,
		MaxDelay:         60 * time.Second,
		BackoffMultiplier: 2.0,
	}

	tests := []struct {
		attempt    int
		expectedMs int64 // Expected delay in milliseconds (approximate)
	}{
		{0, 1000},  // 1s * 2^0 = 1s
		{1, 2000},  // 1s * 2^1 = 2s
		{2, 4000},  // 1s * 2^2 = 4s
		{3, 8000},  // 1s * 2^3 = 8s
		{10, 60000}, // Capped at maxDelay (60s)
	}

	for _, tt := range tests {
		delay := config.CalculateDelay(tt.attempt)
		expected := time.Duration(tt.expectedMs) * time.Millisecond
		// Allow 10ms tolerance for floating point calculations
		if delay < expected-10*time.Millisecond || delay > expected+10*time.Millisecond {
			t.Errorf("Attempt %d: expected ~%v, got %v", tt.attempt, expected, delay)
		}
	}
}

func TestCalculateDelay_Linear(t *testing.T) {
	config := RetryConfig{
		Strategy:      RetryStrategyLinear,
		InitialDelay: time.Second,
		MaxDelay:      60 * time.Second,
	}

	tests := []struct {
		attempt int
		expected time.Duration
	}{
		{0, 1 * time.Second},  // 1s * (0+1) = 1s
		{1, 2 * time.Second},  // 1s * (1+1) = 2s
		{2, 3 * time.Second},  // 1s * (2+1) = 3s
		{3, 4 * time.Second},  // 1s * (3+1) = 4s
		{100, 60 * time.Second}, // Capped at maxDelay
	}

	for _, tt := range tests {
		delay := config.CalculateDelay(tt.attempt)
		if delay != tt.expected {
			t.Errorf("Attempt %d: expected %v, got %v", tt.attempt, tt.expected, delay)
		}
	}
}

func TestCalculateDelay_Fixed(t *testing.T) {
	config := RetryConfig{
		Strategy:      RetryStrategyFixed,
		InitialDelay: 2 * time.Second,
		MaxDelay:      60 * time.Second,
	}

	// Fixed delay should always return initialDelay
	for attempt := 0; attempt < 10; attempt++ {
		delay := config.CalculateDelay(attempt)
		if delay != 2*time.Second {
			t.Errorf("Attempt %d: expected 2s, got %v", attempt, delay)
		}
	}
}

func TestIsRetryableStatusCode(t *testing.T) {
	config := RetryConfig{
		RetryableStatusCodes: []int{429, 500, 502, 503, 504},
	}

	tests := []struct {
		statusCode int
		expected   bool
	}{
		{200, false},
		{404, false},
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
		{505, false},
	}

	for _, tt := range tests {
		result := config.IsRetryableStatusCode(tt.statusCode)
		if result != tt.expected {
			t.Errorf("Status code %d: expected %v, got %v", tt.statusCode, tt.expected, result)
		}
	}
}

func TestParseRetryConfigFromOpenAPI(t *testing.T) {
	// Create a minimal OpenAPI document with retry extension
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Extensions: map[string]interface{}{
			"x-sdk-forge-retry": map[string]interface{}{
				"enabled":              true,
				"maxAttempts":           5.0,
				"initialDelay":          2.0,
				"maxDelay":              120.0,
				"backoffMultiplier":     2.5,
				"strategy":              "linear",
				"retryableStatusCodes":  []interface{}{429.0, 500.0, 502.0},
				"retryOnNetworkErrors": true,
			},
		},
	}

	config := ParseRetryConfigFromOpenAPI(doc)
	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	if !config.Enabled {
		t.Error("Expected Enabled=true")
	}
	if config.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts=5, got %d", config.MaxAttempts)
	}
	if config.InitialDelay != 2*time.Second {
		t.Errorf("Expected InitialDelay=2s, got %v", config.InitialDelay)
	}
	if config.MaxDelay != 120*time.Second {
		t.Errorf("Expected MaxDelay=120s, got %v", config.MaxDelay)
	}
	if config.BackoffMultiplier != 2.5 {
		t.Errorf("Expected BackoffMultiplier=2.5, got %f", config.BackoffMultiplier)
	}
	if config.Strategy != RetryStrategyLinear {
		t.Errorf("Expected Strategy=linear, got %s", config.Strategy)
	}
	if len(config.RetryableStatusCodes) != 3 {
		t.Errorf("Expected 3 retryable status codes, got %d", len(config.RetryableStatusCodes))
	}
	if !config.RetryOnNetworkErrors {
		t.Error("Expected RetryOnNetworkErrors=true")
	}
}

func TestParseRetryConfigFromOpenAPI_NoExtension(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Extensions: map[string]interface{}{},
	}

	config := ParseRetryConfigFromOpenAPI(doc)
	if config != nil {
		t.Error("Expected nil config when extension is missing")
	}
}

func TestParseRetryConfigFromOpenAPI_InvalidExtension(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Extensions: map[string]interface{}{
			"x-sdk-forge-retry": "invalid", // Not a map
		},
	}

	config := ParseRetryConfigFromOpenAPI(doc)
	if config != nil {
		t.Error("Expected nil config when extension is invalid")
	}
}

func TestMergeRetryConfig(t *testing.T) {
	// Base config (from OpenAPI extension)
	base := RetryConfig{
		Enabled:              true,
		MaxAttempts:          3,
		InitialDelay:         time.Second,
		MaxDelay:             60 * time.Second,
		BackoffMultiplier:    2.0,
		Strategy:             RetryStrategyExponential,
		RetryableStatusCodes: []int{429, 500, 502, 503, 504},
		RetryOnNetworkErrors: true,
	}

	// Override config (from CLI flags)
	override := RetryConfig{
		Enabled:              true,
		MaxAttempts:          5,
		InitialDelay:         2 * time.Second,
		MaxDelay:             120 * time.Second,
		BackoffMultiplier:    2.5,
		Strategy:             RetryStrategyLinear,
		RetryableStatusCodes: []int{429, 500},
		RetryOnNetworkErrors: false,
	}

	merged := MergeRetryConfig(base, override)

	if merged.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts=5 (from override), got %d", merged.MaxAttempts)
	}
	if merged.InitialDelay != 2*time.Second {
		t.Errorf("Expected InitialDelay=2s (from override), got %v", merged.InitialDelay)
	}
	if merged.MaxDelay != 120*time.Second {
		t.Errorf("Expected MaxDelay=120s (from override), got %v", merged.MaxDelay)
	}
	if merged.BackoffMultiplier != 2.5 {
		t.Errorf("Expected BackoffMultiplier=2.5 (from override), got %f", merged.BackoffMultiplier)
	}
	if merged.Strategy != RetryStrategyLinear {
		t.Errorf("Expected Strategy=linear (from override), got %s", merged.Strategy)
	}
	if len(merged.RetryableStatusCodes) != 2 {
		t.Errorf("Expected 2 retryable status codes (from override), got %d", len(merged.RetryableStatusCodes))
	}
	if merged.RetryOnNetworkErrors {
		t.Error("Expected RetryOnNetworkErrors=false (from override)")
	}
}

func TestMergeRetryConfig_PartialOverride(t *testing.T) {
	// Base config
	base := RetryConfig{
		Enabled:              true,
		MaxAttempts:          3,
		InitialDelay:         time.Second,
		MaxDelay:             60 * time.Second,
		BackoffMultiplier:    2.0,
		Strategy:             RetryStrategyExponential,
		RetryableStatusCodes: []int{429, 500, 502, 503, 504},
		RetryOnNetworkErrors: true,
	}

	// Partial override (only some fields set)
	override := RetryConfig{
		Enabled:    true,
		MaxAttempts: 5,
		Strategy:   RetryStrategyLinear,
		// Other fields use defaults (zero values)
	}

	merged := MergeRetryConfig(base, override)

	// Overridden fields
	if merged.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts=5, got %d", merged.MaxAttempts)
	}
	if merged.Strategy != RetryStrategyLinear {
		t.Errorf("Expected Strategy=linear, got %s", merged.Strategy)
	}

	// Fields that should keep base values (since override has zero values)
	if merged.InitialDelay != time.Second {
		t.Errorf("Expected InitialDelay=1s (from base), got %v", merged.InitialDelay)
	}
	if merged.MaxDelay != 60*time.Second {
		t.Errorf("Expected MaxDelay=60s (from base), got %v", merged.MaxDelay)
	}
}

