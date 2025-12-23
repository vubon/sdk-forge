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

// TestRunInteractive_AllFlagsSet tests RunInteractive when all flags are already set
// This should skip all prompts and return immediately
func TestRunInteractive_AllFlagsSet(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("schema", "test-schema.yaml", "")
	cmd.Flags().String("lang", "python", "")
	cmd.Flags().String("name", "test-sdk", "")
	cmd.Flags().String("http-lib", "requests", "")
	cmd.Flags().String("python-version", "3.11", "")
	cmd.Flags().String("output", "output", "")
	cmd.Flags().Bool("ignore-minor-issues", true, "") // Set to true to skip prompt
	cmd.Flags().Bool("force", true, "")               // Set to true to skip prompt
	cmd.Flags().String("sdk-version", "1.0.0", "")
	cmd.Flags().Bool("skip-tests", true, "") // Set to true to skip prompt

	// Since all flags are set, RunInteractive should return without prompting
	// This test verifies the early return paths
	err := RunInteractive(cmd)
	if err != nil {
		t.Errorf("RunInteractive() with all flags set should not return error, got: %v", err)
	}
}

// TestRunInteractive_WithLanguageAlias tests RunInteractive with "language" flag instead of "lang"
func TestRunInteractive_WithLanguageAlias(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("schema", "test-schema.yaml", "")
	cmd.Flags().String("language", "go", "") // Using "language" alias
	cmd.Flags().String("name", "test-sdk", "")
	cmd.Flags().String("http-lib", "nethttp", "") // Set http-lib to avoid prompt
	cmd.Flags().String("go-version", "1.24", "")
	cmd.Flags().String("output", "output", "")
	cmd.Flags().Bool("ignore-minor-issues", true, "") // Set to true to skip prompt
	cmd.Flags().Bool("force", true, "")               // Set to true to skip prompt
	cmd.Flags().String("sdk-version", "1.0.0", "")
	cmd.Flags().Bool("skip-tests", true, "") // Set to true to skip prompt

	err := RunInteractive(cmd)
	if err != nil {
		t.Errorf("RunInteractive() with language alias should not return error, got: %v", err)
	}

	// Verify language was normalized (it uses the language flag internally)
	lang, _ := cmd.Flags().GetString("lang")
	language, _ := cmd.Flags().GetString("language")
	if lang == "" && language == "" {
		t.Error("RunInteractive() should have language set")
	}
}

// TestRunInteractive_TypeScriptWithAlias tests RunInteractive with TypeScript and ts-version alias
func TestRunInteractive_TypeScriptWithAlias(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("schema", "test-schema.yaml", "")
	cmd.Flags().String("lang", "typescript", "")
	cmd.Flags().String("name", "test-sdk", "")
	cmd.Flags().String("http-lib", "axios", "") // Set http-lib to avoid prompt
	cmd.Flags().String("ts-version", "5.0", "") // Using "ts-version" alias
	cmd.Flags().String("output", "output", "")
	cmd.Flags().Bool("ignore-minor-issues", true, "") // Set to true to skip prompt
	cmd.Flags().Bool("force", true, "")               // Set to true to skip prompt
	cmd.Flags().String("sdk-version", "1.0.0", "")
	cmd.Flags().Bool("skip-tests", true, "") // Set to true to skip prompt

	err := RunInteractive(cmd)
	if err != nil {
		t.Errorf("RunInteractive() with ts-version alias should not return error, got: %v", err)
	}
}

// TestRunInteractive_FlagsAlreadySet tests that flags that are already set are not prompted
func TestRunInteractive_FlagsAlreadySet(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*cobra.Command)
	}{
		{
			name: "all_optional_flags_set",
			setup: func(cmd *cobra.Command) {
				cmd.Flags().String("schema", "schema.yaml", "")
				cmd.Flags().String("lang", "python", "")
				cmd.Flags().String("name", "test-sdk", "")
				cmd.Flags().String("http-lib", "httpx", "")
				cmd.Flags().String("python-version", "3.12", "")
				cmd.Flags().String("output", "custom-output", "")
				cmd.Flags().Bool("ignore-minor-issues", true, "")
				cmd.Flags().Bool("force", true, "")
				cmd.Flags().String("sdk-version", "2.0.0", "")
				cmd.Flags().Bool("skip-tests", true, "")
			},
		},
		{
			name: "go_with_version",
			setup: func(cmd *cobra.Command) {
				cmd.Flags().String("schema", "schema.yaml", "")
				cmd.Flags().String("lang", "go", "")
				cmd.Flags().String("name", "test-sdk", "")
				cmd.Flags().String("http-lib", "nethttp", "")
				cmd.Flags().String("go-version", "1.25", "")
				cmd.Flags().String("output", "output", "")
				cmd.Flags().Bool("ignore-minor-issues", true, "") // Set to true to skip prompt
				cmd.Flags().Bool("force", true, "")               // Set to true to skip prompt
				cmd.Flags().String("sdk-version", "1.0.0", "")
				cmd.Flags().Bool("skip-tests", true, "") // Set to true to skip prompt
			},
		},
		{
			name: "javascript_language",
			setup: func(cmd *cobra.Command) {
				cmd.Flags().String("schema", "schema.yaml", "")
				cmd.Flags().String("lang", "javascript", "")
				cmd.Flags().String("name", "test-sdk", "")
				cmd.Flags().String("http-lib", "axios", "")
				cmd.Flags().String("typescript-version", "5.1", "")
				cmd.Flags().String("output", "output", "")
				cmd.Flags().Bool("ignore-minor-issues", true, "") // Set to true to skip prompt
				cmd.Flags().Bool("force", true, "")               // Set to true to skip prompt
				cmd.Flags().String("sdk-version", "1.0.0", "")
				cmd.Flags().Bool("skip-tests", true, "") // Set to true to skip prompt
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			tt.setup(cmd)

			err := RunInteractive(cmd)
			if err != nil {
				t.Errorf("RunInteractive() should not return error when all flags are set, got: %v", err)
			}
		})
	}
}
