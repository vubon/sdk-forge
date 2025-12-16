// Package main provides the CLI entry point for SDK Forge.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vubon/sdk-forge/cmd/cli/commands"
)

var version = "0.1.0"

func main() {
	var rootCmd = &cobra.Command{
		Use:   "sdk-forge",
		Short: "SDK Forge - Generate SDKs from OpenAPI schemas",
		Long: `SDK Forge is a CLI tool written in Go that generates SDKs for multiple 
programming languages from OpenAPI schemas.

Generate SDKs one language at a time with your preferred HTTP library.`,
		Version: version,
	}

	// Add commands
	rootCmd.AddCommand(commands.GetGenerateCmd())

	// Add validate command (placeholder for now)
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate OpenAPI schema",
		Long:  "Validate an OpenAPI schema file",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("🚧 Validation feature coming soon...")
		},
	}
	rootCmd.AddCommand(validateCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
