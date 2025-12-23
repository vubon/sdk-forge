package commands

import (
	"bufio"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestPromptOptionalInput(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		defaultValue string
		expected     string
	}{
		{"with_input", "custom-value", "default", "custom-value"},
		{"empty_input_uses_default", "", "default", "default"},
		{"whitespace_input_uses_default", "   ", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input + "\n"))
			cmd := &cobra.Command{}
			cmd.Flags().String("test-flag", "", "test flag")

			err := promptOptionalInput(reader, "Test: ", "test-flag", tt.defaultValue, cmd)
			if err != nil {
				t.Fatalf("promptOptionalInput() error = %v", err)
			}

			result, _ := cmd.Flags().GetString("test-flag")
			if result != tt.expected {
				t.Errorf("promptOptionalInput() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPromptYesNo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"yes_lowercase", "y\n", true},
		{"yes_uppercase", "Y\n", true},
		{"yes_word", "yes\n", true},
		{"no_lowercase", "n\n", false},
		{"no_uppercase", "N\n", false},
		{"no_word", "no\n", false},
		{"empty", "\n", false},
		{"invalid", "maybe\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			cmd := &cobra.Command{}
			cmd.Flags().Bool("test-flag", false, "test flag")

			err := promptYesNo(reader, "Test (y/n): ", "test-flag", cmd)
			if err != nil {
				t.Fatalf("promptYesNo() error = %v", err)
			}

			result, _ := cmd.Flags().GetBool("test-flag")
			if result != tt.expected {
				t.Errorf("promptYesNo() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPromptYesNoWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		defaultValue bool
		flagName     string
		expected     bool
	}{
		{"yes_with_default_true", "y\n", true, "test-flag", true},
		{"yes_with_default_false", "y\n", false, "test-flag", true},
		{"no_with_default_true", "n\n", true, "test-flag", false},
		{"no_with_default_false", "n\n", false, "test-flag", false},
		{"empty_uses_default_true", "\n", true, "test-flag", true},
		{"empty_uses_default_false", "\n", false, "test-flag", false},
		{"skip_tests_yes", "y\n", false, "skip-tests", false},               // Inverted logic
		{"skip_tests_no", "n\n", false, "skip-tests", true},                 // Inverted logic
		{"skip_tests_empty_default_false", "\n", false, "skip-tests", true}, // Empty input with default false converts to "n", which sets skip-tests=true
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			cmd := &cobra.Command{}
			cmd.Flags().Bool(tt.flagName, false, "test flag")

			err := promptYesNoWithDefault(reader, "Test (y/n): ", tt.flagName, tt.defaultValue, cmd)
			if err != nil {
				t.Fatalf("promptYesNoWithDefault() error = %v", err)
			}

			result, _ := cmd.Flags().GetBool(tt.flagName)
			if result != tt.expected {
				t.Errorf("promptYesNoWithDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}
