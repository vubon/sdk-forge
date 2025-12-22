package common

// Python type constants
const (
	PythonTypeAny     = "Any"
	PythonTypeDict    = "Dict[str, Any]"
	PythonTypeArray   = "array"
	PythonTypeObject  = "object"
	PythonTypeString  = "string"
	PythonTypeInteger = "integer"
	PythonTypeNumber  = "number"
	PythonTypeBoolean = "boolean"
)

// Parameter location constants
const (
	ParamLocationPath   = "path"
	ParamLocationQuery  = "query"
	ParamLocationHeader = "header"
)

// Security scheme constants
const (
	SecuritySchemeAPIKey        = "apiKey"
	SecuritySchemeHTTP          = "http"
	SecuritySchemeBearer        = "bearer"
	SecuritySchemeBasic         = "basic"
	SecuritySchemeDigest        = "digest"
	SecuritySchemeOAuth2        = "oauth2"
	SecuritySchemeOpenIDConnect = "openIdConnect"
	SecuritySchemeMutualTLS     = "mutualTLS"
)
