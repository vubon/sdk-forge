// Package commands provides interactive CLI functionality for SDK Forge.
package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vubon/sdk-forge/internal/generator"
	"github.com/vubon/sdk-forge/internal/validator"
	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

// promptInput prompts for user input and returns the trimmed result
func promptInput(reader *bufio.Reader, prompt string) (string, error) {
	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return strings.TrimSpace(input), nil
}

// promptRequiredInput prompts for required input and validates it's not empty
func promptRequiredInput(reader *bufio.Reader, prompt, flagName string, cmd *cobra.Command) error {
	input, err := promptInput(reader, prompt)
	if err != nil {
		return err
	}
	if input == "" {
		return fmt.Errorf("%s is required", flagName)
	}
	if err := cmd.Flags().Set(flagName, input); err != nil {
		return fmt.Errorf("failed to set %s flag: %w", flagName, err)
	}
	return nil
}

// promptOptionalInput prompts for optional input with a default value
func promptOptionalInput(reader *bufio.Reader, prompt, flagName, defaultValue string, cmd *cobra.Command) error {
	input, err := promptInput(reader, prompt)
	if err != nil {
		return err
	}
	if input == "" {
		input = defaultValue
	}
	if err := cmd.Flags().Set(flagName, input); err != nil {
		return fmt.Errorf("failed to set %s flag: %w", flagName, err)
	}
	return nil
}

// promptYesNo prompts for yes/no input and sets a boolean flag
func promptYesNo(reader *bufio.Reader, prompt, flagName string, cmd *cobra.Command) error {
	input, err := promptInput(reader, prompt)
	if err != nil {
		return err
	}
	response := strings.ToLower(input)
	if response == "y" || response == "yes" {
		if err := cmd.Flags().Set(flagName, "true"); err != nil {
			return fmt.Errorf("failed to set %s flag: %w", flagName, err)
		}
	}
	return nil
}

// RunInteractive prompts for missing required information
//
//nolint:gocyclo // Interactive prompts naturally have high cyclomatic complexity
func RunInteractive(cmd *cobra.Command) error {
	reader := bufio.NewReader(os.Stdin)

	// Get schema path
	schemaPath, _ := cmd.Flags().GetString("schema")
	if schemaPath == "" {
		if err := promptRequiredInput(reader, "Enter OpenAPI schema path or URL: ", "schema", cmd); err != nil {
			return err
		}
	}

	// Get language
	lang, _ := cmd.Flags().GetString("lang")
	if lang == "" {
		lang, _ = cmd.Flags().GetString("language")
	}
	if lang == "" {
		prompt := "Enter target language (python/go/php/javascript/typescript/all): "
		if err := promptRequiredInput(reader, prompt, "lang", cmd); err != nil {
			return err
		}
		lang, _ = cmd.Flags().GetString("lang")
	}

	// Normalize language
	normalizedLang := validator.NormalizeLanguage(lang)

	// Get SDK name
	sdkName, _ := cmd.Flags().GetString("name")
	if sdkName == "" {
		if err := promptRequiredInput(reader, "Enter SDK name: ", "name", cmd); err != nil {
			return err
		}
	}

	// Get HTTP library (optional - will use default if not provided)
	httpLib, _ := cmd.Flags().GetString("http-lib")
	if httpLib == "" {
		defaultLib := httplib.GetDefaultLibrary(normalizedLang)
		prompt := "Enter HTTP library: "
		if defaultLib != "" {
			prompt = fmt.Sprintf("Enter HTTP library (press Enter for default '%s'): ", defaultLib)
		}
		input, err := promptInput(reader, prompt)
		if err != nil {
			return err
		}
		if input != "" {
			if err := cmd.Flags().Set("http-lib", input); err != nil {
				return fmt.Errorf("failed to set http-lib flag: %w", err)
			}
		}
	}

	// Get language version (optional - will use default if not provided)
	switch normalizedLang {
	case "go":
		goVer, _ := cmd.Flags().GetString("go-version")
		if goVer == "" {
			availableVersions := generator.GetGoAvailableVersions()
			defaultVersion := generator.GetGoDefaultVersion()
			versionList := make([]string, len(availableVersions))
			for i, v := range availableVersions {
				versionList[i] = v.String()
			}
			prompt := fmt.Sprintf("Enter Go version (press Enter for default '%s', available: %v): ",
				defaultVersion.String(), versionList)
			input, err := promptInput(reader, prompt)
			if err != nil {
				return err
			}
			if input != "" {
				// Validate the version
				parsed, err := generator.ParseVersion(input)
				if err != nil {
					return fmt.Errorf("invalid Go version format: %w", err)
				}
				if err := generator.ValidateGoVersion(parsed); err != nil {
					return err
				}
				if err := cmd.Flags().Set("go-version", input); err != nil {
					return fmt.Errorf("failed to set go-version flag: %w", err)
				}
			}
		}
	case "python":
		pythonVer, _ := cmd.Flags().GetString("python-version")
		if pythonVer == "" {
			availableVersions := generator.GetPythonAvailableVersions()
			defaultVersion := generator.GetPythonDefaultVersion()
			versionList := make([]string, len(availableVersions))
			for i, v := range availableVersions {
				versionList[i] = v.String()
			}
			prompt := fmt.Sprintf("Enter Python version (press Enter for default '%s', available: %v): ",
				defaultVersion.String(), versionList)
			input, err := promptInput(reader, prompt)
			if err != nil {
				return err
			}
			if input != "" {
				// Validate the version
				parsed, err := generator.ParseVersion(input)
				if err != nil {
					return fmt.Errorf("invalid Python version format: %w", err)
				}
				if err := generator.ValidatePythonVersion(parsed); err != nil {
					return err
				}
				if err := cmd.Flags().Set("python-version", input); err != nil {
					return fmt.Errorf("failed to set python-version flag: %w", err)
				}
			}
		}
	}

	// Get output directory (optional)
	outputDir, _ := cmd.Flags().GetString("output")
	if outputDir == "" {
		prompt := "Enter output directory (press Enter for 'output'): "
		if err := promptOptionalInput(reader, prompt, "output", "output", cmd); err != nil {
			return err
		}
	}

	// Get ignore minor issues flag (optional)
	ignoreMinor, _ := cmd.Flags().GetBool("ignore-minor-issues")
	if !ignoreMinor {
		prompt := "Ignore minor OpenAPI validation issues? (y/N): "
		if err := promptYesNo(reader, prompt, "ignore-minor-issues", cmd); err != nil {
			return err
		}
	}

	// Get force flag (optional)
	force, _ := cmd.Flags().GetBool("force")
	if !force {
		if err := promptYesNo(reader, "Overwrite existing SDK directory? (y/N): ", "force", cmd); err != nil {
			return err
		}
	}

	// Get SDK version (optional - OpenAPI schema version takes precedence, then user input, then default)
	sdkVer, _ := cmd.Flags().GetString("sdk-version")
	if sdkVer == "" {
		prompt := "Enter SDK version (press Enter to use OpenAPI schema version if available, or default '1.0.0'): "
		input, err := promptInput(reader, prompt)
		if err != nil {
			return err
		}
		if input != "" {
			if err := cmd.Flags().Set("sdk-version", input); err != nil {
				return fmt.Errorf("failed to set sdk-version flag: %w", err)
			}
		}
	}

	return nil
}
