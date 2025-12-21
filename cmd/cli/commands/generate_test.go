package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestParseRetryConfig tests retry configuration parsing
func TestParseRetryConfig(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("retry-enabled", false, "")
	cmd.Flags().Int("retry-max-attempts", 3, "")
	cmd.Flags().Float64("retry-initial-delay", 1.0, "")
	cmd.Flags().Float64("retry-max-delay", 60.0, "")
	cmd.Flags().Float64("retry-backoff-multiplier", 2.0, "")
	cmd.Flags().String("retry-strategy", "exponential", "")
	cmd.Flags().String("retry-status-codes", "429,500,502,503,504", "")

	// Test default config (no flags set)
	config := parseRetryConfig(cmd)
	if config.Enabled {
		t.Error("expected retry to be disabled by default")
	}
	if config.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts=3, got %d", config.MaxAttempts)
	}

	// Test with retry enabled
	if err := cmd.Flags().Set("retry-enabled", "true"); err != nil {
		t.Fatalf("failed to set retry-enabled flag: %v", err)
	}
	if err := cmd.Flags().Set("retry-max-attempts", "5"); err != nil {
		t.Fatalf("failed to set retry-max-attempts flag: %v", err)
	}
	if err := cmd.Flags().Set("retry-strategy", "linear"); err != nil {
		t.Fatalf("failed to set retry-strategy flag: %v", err)
	}

	config = parseRetryConfig(cmd)
	if !config.Enabled {
		t.Error("expected retry to be enabled")
	}
	if config.MaxAttempts != 5 {
		t.Errorf("expected MaxAttempts=5, got %d", config.MaxAttempts)
	}
	if config.Strategy != "linear" {
		t.Errorf("expected Strategy=linear, got %s", config.Strategy)
	}
}

// TestGetGenerateCmd tests that GetGenerateCmd returns a valid command
func TestGetGenerateCmd(t *testing.T) {
	cmd := GetGenerateCmd()
	if cmd == nil {
		t.Fatal("GetGenerateCmd returned nil")
	}

	if cmd.Use != "generate" {
		t.Errorf("expected command Use='generate', got '%s'", cmd.Use)
	}

	// Check that required flags exist
	requiredFlags := []string{"schema", "lang", "name", "output"}
	for _, flag := range requiredFlags {
		if cmd.Flag(flag) == nil {
			t.Errorf("expected flag '--%s' to exist", flag)
		}
	}

	// Check that retry flags exist
	retryFlags := []string{"retry-enabled", "retry-max-attempts", "retry-strategy"}
	for _, flag := range retryFlags {
		if cmd.Flag(flag) == nil {
			t.Errorf("expected retry flag '--%s' to exist", flag)
		}
	}
}

