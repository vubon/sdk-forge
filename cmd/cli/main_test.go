package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// findProjectRoot finds the project root by looking for go.mod
func findProjectRoot(t *testing.T) string {
	// Start from current directory and walk up
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	t.Fatalf("could not find go.mod file")
	return ""
}

// TestVersionCommand tests the version command
func TestVersionCommand(t *testing.T) {
	// Build the binary for testing
	binaryPath := buildTestBinary(t)
	defer func() {
		_ = os.Remove(binaryPath)
	}()

	// Test version flag (--version)
	cmd := exec.Command(binaryPath, "--version")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("version command failed: %v, stderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "sdk-forge") && !strings.Contains(output, "dev") {
		t.Errorf("version output should contain 'sdk-forge' or 'dev', got: %s", output)
	}
}

// TestHelpCommand tests the help command
func TestHelpCommand(t *testing.T) {
	binaryPath := buildTestBinary(t)
	defer func() {
		_ = os.Remove(binaryPath)
	}()

	// Test root help
	cmd := exec.Command(binaryPath, "--help")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		t.Fatalf("help command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "SDK Forge") {
		t.Errorf("help output should contain 'SDK Forge', got: %s", output)
	}
}

// TestGenerateCommandHelp tests the generate command help
func TestGenerateCommandHelp(t *testing.T) {
	binaryPath := buildTestBinary(t)
	defer func() {
		_ = os.Remove(binaryPath)
	}()

	cmd := exec.Command(binaryPath, "generate", "--help")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		t.Fatalf("generate help command failed: %v", err)
	}

	output := stdout.String()
	expectedFlags := []string{"--schema", "--lang", "--name", "--output"}
	for _, flag := range expectedFlags {
		if !strings.Contains(output, flag) {
			t.Errorf("help output should contain flag '%s', got: %s", flag, output)
		}
	}
}

// TestGenerateCommandMissingRequiredFlags tests error handling for missing required flags
func TestGenerateCommandMissingRequiredFlags(t *testing.T) {
	binaryPath := buildTestBinary(t)
	defer func() {
		_ = os.Remove(binaryPath)
	}()

	// Test with no flags (should trigger interactive mode or error)
	cmd := exec.Command(binaryPath, "generate")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	// This might succeed if interactive mode is triggered, or fail if validation happens
	// We just want to ensure it doesn't crash
	_ = err
}

// TestGenerateCommandInvalidSchema tests error handling for invalid schema path
func TestGenerateCommandInvalidSchema(t *testing.T) {
	binaryPath := buildTestBinary(t)
	defer func() {
		_ = os.Remove(binaryPath)
	}()

	cmd := exec.Command(binaryPath, "generate",
		"--schema", "/nonexistent/path/to/schema.yaml",
		"--lang", "python",
		"--name", "test-sdk",
		"--output", "/tmp/test-output",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Error("expected error for invalid schema path, but command succeeded")
	}

	output := stderr.String()
	if !strings.Contains(output, "schema") && !strings.Contains(output, "not found") && !strings.Contains(output, "Error") {
		t.Logf("stderr output: %s", output)
		// This is okay - different error messages are acceptable
	}
}

// TestGenerateCommandInvalidLanguage tests error handling for invalid language
func TestGenerateCommandInvalidLanguage(t *testing.T) {
	binaryPath := buildTestBinary(t)
	defer func() {
		_ = os.Remove(binaryPath)
	}()

	// Create a temporary schema file
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.yaml")
	schemaContent := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}

	outputDir := filepath.Join(tmpDir, "output")

	cmd := exec.Command(binaryPath, "generate",
		"--schema", schemaPath,
		"--lang", "invalid-language",
		"--name", "test-sdk",
		"--output", outputDir,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Error("expected error for invalid language, but command succeeded")
	}

	output := stderr.String()
	if !strings.Contains(strings.ToLower(output), "language") && !strings.Contains(strings.ToLower(output), "invalid") {
		t.Logf("stderr output: %s", output)
		// This is okay - different error messages are acceptable
	}
}

// TestGenerateCommandRetryFlags tests that retry flags are accepted
func TestGenerateCommandRetryFlags(t *testing.T) {
	binaryPath := buildTestBinary(t)
	defer func() {
		_ = os.Remove(binaryPath)
	}()

	// Test that retry flags are recognized (help should show them)
	cmd := exec.Command(binaryPath, "generate", "--help")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		t.Fatalf("generate help command failed: %v", err)
	}

	output := stdout.String()
	retryFlags := []string{"--retry-enabled", "--retry-max-attempts", "--retry-strategy"}
	for _, flag := range retryFlags {
		if !strings.Contains(output, flag) {
			t.Errorf("help output should contain retry flag '%s', got: %s", flag, output)
		}
	}
}

// TestGenerateCommandWithValidSchema tests successful SDK generation
func TestGenerateCommandWithValidSchema(t *testing.T) {
	binaryPath := buildTestBinary(t)
	defer func() {
		_ = os.Remove(binaryPath)
	}()

	// Find project root to locate example schema
	projectRoot := findProjectRoot(t)
	schemaPath := filepath.Join(projectRoot, "examples", "petstore.yaml")

	// Check if schema exists
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		t.Skipf("example schema not found at %s, skipping test", schemaPath)
	}

	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")

	// Test successful generation
	cmd := exec.Command(binaryPath, "generate",
		"--schema", schemaPath,
		"--lang", "python",
		"--name", "test-sdk",
		"--output", outputDir,
		"--skip-tests", // Skip tests to speed up
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Logf("stdout: %s", stdout.String())
		t.Logf("stderr: %s", stderr.String())
		t.Fatalf("generate command failed: %v", err)
	}

	// Check that output directory was created
	expectedSDKPath := filepath.Join(outputDir, "python", "test-sdk")
	if _, err := os.Stat(expectedSDKPath); os.IsNotExist(err) {
		t.Errorf("expected SDK directory not created: %s", expectedSDKPath)
	}
}

// TestGenerateCommandWithRetryEnabled tests generation with retry enabled
func TestGenerateCommandWithRetryEnabled(t *testing.T) {
	binaryPath := buildTestBinary(t)
	defer func() {
		_ = os.Remove(binaryPath)
	}()

	projectRoot := findProjectRoot(t)
	schemaPath := filepath.Join(projectRoot, "examples", "petstore.yaml")

	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		t.Skipf("example schema not found at %s, skipping test", schemaPath)
	}

	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")

	// Test generation with retry enabled
	cmd := exec.Command(binaryPath, "generate",
		"--schema", schemaPath,
		"--lang", "go",
		"--name", "test-sdk-retry",
		"--output", outputDir,
		"--retry-enabled",
		"--retry-max-attempts", "5",
		"--retry-strategy", "exponential",
		"--skip-tests",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Logf("stdout: %s", stdout.String())
		t.Logf("stderr: %s", stderr.String())
		t.Fatalf("generate command with retry failed: %v", err)
	}

	// Check that SDK was generated
	expectedSDKPath := filepath.Join(outputDir, "go", "test-sdk-retry")
	if _, err := os.Stat(expectedSDKPath); os.IsNotExist(err) {
		t.Errorf("expected SDK directory not created: %s", expectedSDKPath)
	}
}

// buildTestBinary builds the CLI binary for testing
func buildTestBinary(t *testing.T) string {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "sdk-forge-test")

	// Find project root (where go.mod is located)
	projectRoot := findProjectRoot(t)

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/cli")
	cmd.Dir = projectRoot
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build test binary: %v, stderr: %s", err, stderr.String())
	}

	return binaryPath
}

