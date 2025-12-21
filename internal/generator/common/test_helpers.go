package generator

const (
	testGoSDKName = "testsdk" // Sanitized version of testSDKName for Go
)

// contains checks if a string contains a substring
func contains(s, substr string) bool {
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
