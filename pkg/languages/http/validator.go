// Package http provides HTTP library validation functionality.
//
//nolint:revive // Package name 'http' is intentional and does not conflict with stdlib in this context
package http

import "fmt"

// ValidateLibrary validates that an HTTP library is valid for the given language
// Returns error if invalid, nil if valid
func ValidateLibrary(language, httpLib string) error {
	if !IsValidLibrary(language, httpLib) {
		validLibs := GetValidLibraries(language)
		return fmt.Errorf("invalid HTTP library '%s' for language '%s'. Valid options: %v",
			httpLib, language, validLibs)
	}
	return nil
}

// ValidateOrGetDefault validates the HTTP library or returns the default if empty
func ValidateOrGetDefault(language, httpLib string) (string, error) {
	// If no HTTP library provided, use default
	if httpLib == "" {
		defaultLib := GetDefaultLibrary(language)
		if defaultLib == "" {
			return "", fmt.Errorf("no default HTTP library found for language: %s", language)
		}
		return defaultLib, nil
	}

	// Validate the provided library
	if err := ValidateLibrary(language, httpLib); err != nil {
		return "", err
	}

	return httpLib, nil
}
