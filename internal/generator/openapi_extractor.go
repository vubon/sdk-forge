package generator

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// APIOperation represents a single API operation (GET, POST, etc.)
type APIOperation struct {
	Method      string                // GET, POST, PUT, DELETE, etc.
	Path        string                // /pets, /pets/{id}, etc.
	OperationID string                // listPets, createPet, etc.
	Summary     string                // Operation summary
	Description string                // Operation description
	Tags        []string              // Operation tags
	Parameters  []Parameter           // Path, query, header parameters
	RequestBody *RequestBody          // Request body (if any)
	Responses   map[string]Response   // Status code -> Response
	Security    []SecurityRequirement // Security requirements
}

// Parameter represents an API parameter
type Parameter struct {
	Name        string
	In          string // path, query, header, cookie
	Required    bool
	Description string
	Schema      *Schema
}

// RequestBody represents a request body
type RequestBody struct {
	Required bool
	Content  map[string]ContentType // media type -> content
}

// ContentType represents content type information
type ContentType struct {
	Schema   *Schema
	Examples map[string]interface{} // Example name -> example value
}

// Response represents an API response
type Response struct {
	Description string
	Content     map[string]ContentType // media type -> content
}

// Schema represents a JSON schema
type Schema struct {
	Type        string
	Format      string
	Description string
	Properties  map[string]*Schema
	Items       *Schema
	Required    []string
	Ref         string // $ref
}

// SecurityRequirement represents security requirements
type SecurityRequirement struct {
	Schemes []string // Names of security schemes
}

// OAuth2Flow represents a single OAuth2 flow
type OAuth2Flow struct {
	AuthorizationURL string            // authorizationCode, implicit
	TokenURL         string            // authorizationCode, clientCredentials, password
	RefreshURL       string            // Optional refresh URL
	Scopes           map[string]string // Available scopes
}

// OAuth2Flows represents OAuth2 flow configuration
type OAuth2Flows struct {
	AuthorizationCode *OAuth2Flow // authorizationCode flow
	ClientCredentials *OAuth2Flow // clientCredentials flow
	Implicit          *OAuth2Flow // implicit flow
	Password          *OAuth2Flow // password flow
}

// ExtractedData contains all extracted data from OpenAPI schema
type ExtractedData struct {
	BaseURL         string
	Title           string
	Version         string
	Description     string
	Operations      []APIOperation
	Schemas         map[string]*Schema
	SecuritySchemes map[string]SecurityScheme
}

// SecurityScheme represents a security scheme definition
type SecurityScheme struct {
	Type             string       // apiKey, http, oauth2, openIdConnect, mutualTLS
	Scheme           string       // For http: basic, bearer, digest
	In               string       // For apiKey: query, header, cookie
	Name             string       // For apiKey: header name, etc.
	Description      string       // Scheme description
	OAuth2Flows      *OAuth2Flows // For oauth2: flows configuration
	OpenIDConnectURL string       // For openIdConnect: discovery URL
	BearerFormat     string       // For http bearer: bearer format
}

// ExtractOpenAPIData extracts all relevant data from OpenAPI document
func ExtractOpenAPIData(doc *openapi3.T) (*ExtractedData, error) {
	data := &ExtractedData{
		Operations:      []APIOperation{},
		Schemas:         make(map[string]*Schema),
		SecuritySchemes: make(map[string]SecurityScheme),
	}

	// Extract basic info
	if doc.Info != nil {
		data.Title = doc.Info.Title
		data.Version = doc.Info.Version
		data.Description = doc.Info.Description
	}

	// Extract base URL from servers
	if len(doc.Servers) > 0 {
		data.BaseURL = doc.Servers[0].URL
	} else {
		data.BaseURL = "https://api.example.com/v1" // Default fallback
	}

	// Extract security schemes
	if doc.Components != nil && doc.Components.SecuritySchemes != nil {
		for name, schemeRef := range doc.Components.SecuritySchemes {
			if schemeRef.Value == nil {
				continue
			}
			scheme := schemeRef.Value

			secScheme := SecurityScheme{
				Type:        scheme.Type,
				Description: scheme.Description,
			}

			// Handle API Key authentication
			if scheme.Type == "apiKey" {
				secScheme.In = scheme.In
				secScheme.Name = scheme.Name
			}

			// Handle HTTP authentication
			if scheme.Type == "http" {
				secScheme.Scheme = scheme.Scheme
				secScheme.Name = "Authorization" // Default for HTTP schemes
				secScheme.BearerFormat = scheme.BearerFormat
			}

			// Handle OAuth2 authentication
			if scheme.Type == "oauth2" && scheme.Flows != nil {
				secScheme.OAuth2Flows = &OAuth2Flows{}
				if scheme.Flows.AuthorizationCode != nil {
					secScheme.OAuth2Flows.AuthorizationCode = &OAuth2Flow{
						AuthorizationURL: scheme.Flows.AuthorizationCode.AuthorizationURL,
						TokenURL:         scheme.Flows.AuthorizationCode.TokenURL,
						RefreshURL:       scheme.Flows.AuthorizationCode.RefreshURL,
						Scopes:           scheme.Flows.AuthorizationCode.Scopes,
					}
				}
				if scheme.Flows.ClientCredentials != nil {
					secScheme.OAuth2Flows.ClientCredentials = &OAuth2Flow{
						TokenURL:   scheme.Flows.ClientCredentials.TokenURL,
						RefreshURL: scheme.Flows.ClientCredentials.RefreshURL,
						Scopes:     scheme.Flows.ClientCredentials.Scopes,
					}
				}
				if scheme.Flows.Implicit != nil {
					secScheme.OAuth2Flows.Implicit = &OAuth2Flow{
						AuthorizationURL: scheme.Flows.Implicit.AuthorizationURL,
						RefreshURL:       scheme.Flows.Implicit.RefreshURL,
						Scopes:           scheme.Flows.Implicit.Scopes,
					}
				}
				if scheme.Flows.Password != nil {
					secScheme.OAuth2Flows.Password = &OAuth2Flow{
						TokenURL:   scheme.Flows.Password.TokenURL,
						RefreshURL: scheme.Flows.Password.RefreshURL,
						Scopes:     scheme.Flows.Password.Scopes,
					}
				}
			}

			// Handle OpenID Connect authentication
			if scheme.Type == "openIdConnect" {
				secScheme.OpenIDConnectURL = scheme.OpenIdConnectUrl
			}

			// Handle Mutual TLS (OpenAPI 3.1+)
			// mTLS doesn't need additional fields in the scheme
			// Client certificate configuration will be handled in SDK generation
			_ = scheme.Type // For future mTLS support

			data.SecuritySchemes[name] = secScheme
		}
	}

	// Extract paths and operations
	if doc.Paths != nil {
		for path, pathItem := range doc.Paths.Map() {
			operations := extractOperationsFromPath(path, pathItem, data.SecuritySchemes)
			data.Operations = append(data.Operations, operations...)
		}
	}

	// Extract schemas
	if doc.Components != nil && doc.Components.Schemas != nil {
		for name, schemaRef := range doc.Components.Schemas {
			if schemaRef.Value != nil {
				data.Schemas[name] = extractSchema(schemaRef.Value)
			}
		}
	}

	return data, nil
}

// extractOperationsFromPath extracts all operations from a path item
func extractOperationsFromPath(path string, pathItem *openapi3.PathItem, _ map[string]SecurityScheme) []APIOperation {
	var operations []APIOperation

	// Helper to create operation
	createOp := func(method string, op *openapi3.Operation) APIOperation {
		operation := APIOperation{
			Method:      strings.ToUpper(method),
			Path:        path,
			OperationID: op.OperationID,
			Summary:     op.Summary,
			Description: op.Description,
			Tags:        op.Tags,
			Parameters:  []Parameter{},
			Responses:   make(map[string]Response),
		}

		// Extract parameters
		for _, param := range op.Parameters {
			if param.Value != nil {
				operation.Parameters = append(operation.Parameters, Parameter{
					Name:        param.Value.Name,
					In:          param.Value.In,
					Required:    param.Value.Required,
					Description: param.Value.Description,
					Schema:      extractSchema(param.Value.Schema.Value),
				})
			}
		}

		// Extract request body
		if op.RequestBody != nil && op.RequestBody.Value != nil {
			reqBody := RequestBody{
				Required: op.RequestBody.Value.Required,
				Content:  make(map[string]ContentType),
			}
			for mediaType, content := range op.RequestBody.Value.Content {
				contentType := ContentType{
					Schema:   extractSchema(content.Schema.Value),
					Examples: make(map[string]interface{}),
				}
				// Extract examples
				if content.Examples != nil {
					for name, exampleRef := range content.Examples {
						if exampleRef.Value != nil {
							contentType.Examples[name] = exampleRef.Value.Value
						}
					}
				}
				// Extract single example if present
				if content.Example != nil {
					contentType.Examples["default"] = content.Example
				}
				reqBody.Content[mediaType] = contentType
			}
			operation.RequestBody = &reqBody
		}

		// Extract responses
		if op.Responses != nil {
			for statusCode, resp := range op.Responses.Map() {
				if resp.Value != nil {
					desc := ""
					if resp.Value.Description != nil {
						desc = *resp.Value.Description
					}
					response := Response{
						Description: desc,
						Content:     make(map[string]ContentType),
					}
					for mediaType, content := range resp.Value.Content {
						var schema *Schema
						if content.Schema != nil && content.Schema.Value != nil {
							schema = extractSchema(content.Schema.Value)
						}
						contentType := ContentType{
							Schema:   schema,
							Examples: make(map[string]interface{}),
						}
						// Extract examples
						if content.Examples != nil {
							for name, exampleRef := range content.Examples {
								if exampleRef.Value != nil {
									contentType.Examples[name] = exampleRef.Value.Value
								}
							}
						}
						// Extract single example if present
						if content.Example != nil {
							contentType.Examples["default"] = content.Example
						}
						response.Content[mediaType] = contentType
					}
					operation.Responses[statusCode] = response
				}
			}
		}

		// Extract security requirements
		if op.Security != nil {
			for _, secReq := range *op.Security {
				secRequirement := SecurityRequirement{}
				for schemeName := range secReq {
					secRequirement.Schemes = append(secRequirement.Schemes, schemeName)
				}
				operation.Security = append(operation.Security, secRequirement)
			}
		}

		return operation
	}

	// Extract all HTTP methods
	if pathItem.Get != nil {
		operations = append(operations, createOp("GET", pathItem.Get))
	}
	if pathItem.Post != nil {
		operations = append(operations, createOp("POST", pathItem.Post))
	}
	if pathItem.Put != nil {
		operations = append(operations, createOp("PUT", pathItem.Put))
	}
	if pathItem.Patch != nil {
		operations = append(operations, createOp("PATCH", pathItem.Patch))
	}
	if pathItem.Delete != nil {
		operations = append(operations, createOp("DELETE", pathItem.Delete))
	}
	if pathItem.Head != nil {
		operations = append(operations, createOp("HEAD", pathItem.Head))
	}
	if pathItem.Options != nil {
		operations = append(operations, createOp("OPTIONS", pathItem.Options))
	}

	return operations
}

// extractSchema converts openapi3.Schema to our Schema type
func extractSchema(schema *openapi3.Schema) *Schema {
	if schema == nil {
		return nil
	}

	// Convert schema type to string
	typeStr := ""
	if schema.Type != nil {
		types := schema.Type.Slice()
		if len(types) > 0 {
			typeStr = types[0]
		}
	}

	result := &Schema{
		Type:        typeStr,
		Format:      schema.Format,
		Description: schema.Description,
		Required:    schema.Required,
		Properties:  make(map[string]*Schema),
	}

	// Extract properties
	if schema.Properties != nil {
		for name, prop := range schema.Properties {
			if prop.Value != nil {
				result.Properties[name] = extractSchema(prop.Value)
			}
		}
	}

	// Extract items (for arrays)
	if schema.Items != nil && schema.Items.Value != nil {
		result.Items = extractSchema(schema.Items.Value)
	}

	// Extract $ref (from schema reference if available)
	// Note: kin-openapi handles refs differently, we'll extract from the schema reference path if needed
	// For now, we'll skip explicit ref extraction as it's handled by the loader

	return result
}

// GetOperationMethodName generates a method name from an operation
func GetOperationMethodName(op APIOperation) string {
	if op.OperationID != "" {
		// Use operationId if available
		return toSnakeCase(op.OperationID)
	}

	// Generate from method + path
	method := strings.ToLower(op.Method)
	pathParts := strings.Split(strings.Trim(op.Path, "/"), "/")

	// Remove path parameters
	var cleanParts []string
	for _, part := range pathParts {
		if !strings.HasPrefix(part, "{") {
			cleanParts = append(cleanParts, part)
		}
	}

	if len(cleanParts) == 0 {
		return fmt.Sprintf("%s_root", method)
	}

	// Combine method and path
	name := method
	for _, part := range cleanParts {
		name += "_" + part
	}

	return toSnakeCase(name)
}
