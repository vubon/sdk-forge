package common

import (
	"github.com/getkin/kin-openapi/openapi3"
)

const (
	TestSDKName   = "test-sdk" // Test SDK name
	TestGoSDKName = "testsdk"  // Sanitized version of testSDKName for Go
)

// Contains checks if a string contains a substring
func Contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && containsHelper(s, substr)))
}

// containsHelper is a helper function for contains
func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// CreateTestExtractedData creates test ExtractedData for testing
func CreateTestExtractedData() *ExtractedData {
	return &ExtractedData{
		BaseURL:         "https://api.example.com/v1",
		Description:     "Test API",
		Version:         "1.0.0",
		Title:           "Test API",
		Operations:      []APIOperation{},
		Schemas:         make(map[string]*Schema),
		SecuritySchemes: make(map[string]SecurityScheme),
	}
}

// CreateTestOpenAPIDoc creates a minimal openapi3.T for testing
func CreateTestOpenAPIDoc() *openapi3.T {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Servers: openapi3.Servers{
			&openapi3.Server{
				URL: "https://api.example.com/v1",
			},
		},
		Paths: openapi3.NewPaths(),
	}
	return doc
}
