package generator

// Python type constants
const (
	pythonTypeAny     = "Any"
	pythonTypeDict    = "Dict[str, Any]"
	pythonTypeArray   = "array"
	pythonTypeObject  = "object"
	pythonTypeString  = "string"
	pythonTypeInteger = "integer"
	pythonTypeNumber  = "number"
	pythonTypeBoolean = "boolean"
)

// Parameter location constants
const (
	paramLocationPath   = "path"
	paramLocationQuery  = "query"
	paramLocationHeader = "header"
)

// Security scheme constants
const (
	securitySchemeAPIKey        = "apiKey"
	securitySchemeHTTP          = "http"
	securitySchemeBearer        = "bearer"
	securitySchemeBasic         = "basic"
	securitySchemeDigest        = "digest"
	securitySchemeOAuth2        = "oauth2"
	securitySchemeOpenIDConnect = "openIdConnect"
	securitySchemeMutualTLS     = "mutualTLS"
)
