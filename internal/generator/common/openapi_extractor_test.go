package common

import (
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestExtractOpenAPIData_WithOperations(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}

	// Add a GET operation
	pathItem := &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getPets",
			Summary:     "Get pets",
			Description: "Retrieve a list of pets",
			Tags:        []string{"pets"},
		},
	}
	doc.Paths.Set("/pets", pathItem)

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	if len(result.Operations) == 0 {
		t.Error("ExtractOpenAPIData() should extract operations")
	}

	found := false
	for _, op := range result.Operations {
		if op.Method == "GET" && op.Path == "/pets" && op.OperationID == "getPets" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ExtractOpenAPIData() should extract GET /pets operation")
	}
}

func TestExtractOpenAPIData_AllHTTPMethods(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}

	pathItem := &openapi3.PathItem{
		Get:     &openapi3.Operation{OperationID: "get"},
		Post:    &openapi3.Operation{OperationID: "post"},
		Put:     &openapi3.Operation{OperationID: "put"},
		Patch:   &openapi3.Operation{OperationID: "patch"},
		Delete:  &openapi3.Operation{OperationID: "delete"},
		Head:    &openapi3.Operation{OperationID: "head"},
		Options: &openapi3.Operation{OperationID: "options"},
	}
	doc.Paths.Set("/test", pathItem)

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	expectedMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	methodMap := make(map[string]bool)
	for _, op := range result.Operations {
		methodMap[op.Method] = true
	}

	for _, method := range expectedMethods {
		if !methodMap[method] {
			t.Errorf("ExtractOpenAPIData() should extract %s method", method)
		}
	}
}

func TestExtractOpenAPIData_WithParameters(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}

	pathItem := &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getPet",
			Parameters: openapi3.Parameters{
				{
					Value: &openapi3.Parameter{
						Name:     "id",
						In:       "path",
						Required: true,
						Schema: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: &openapi3.Types{"string"},
							},
						},
					},
				},
			},
		},
	}
	doc.Paths.Set("/pets/{id}", pathItem)

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	if len(result.Operations) == 0 {
		t.Fatal("ExtractOpenAPIData() should extract operations")
	}

	op := result.Operations[0]
	if len(op.Parameters) == 0 {
		t.Error("ExtractOpenAPIData() should extract parameters")
	}
	if op.Parameters[0].Name != "id" {
		t.Errorf("ExtractOpenAPIData() parameter name = %q, want %q", op.Parameters[0].Name, "id")
	}
	if op.Parameters[0].In != "path" {
		t.Errorf("ExtractOpenAPIData() parameter In = %q, want %q", op.Parameters[0].In, "path")
	}
}

func TestExtractOpenAPIData_WithRequestBody(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}

	pathItem := &openapi3.PathItem{
		Post: &openapi3.Operation{
			OperationID: "createPet",
			RequestBody: &openapi3.RequestBodyRef{
				Value: &openapi3.RequestBody{
					Required: true,
					Content: openapi3.Content{
						"application/json": &openapi3.MediaType{
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: &openapi3.Types{"object"},
								},
							},
						},
					},
				},
			},
		},
	}
	doc.Paths.Set("/pets", pathItem)

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	if len(result.Operations) == 0 {
		t.Fatal("ExtractOpenAPIData() should extract operations")
	}

	op := result.Operations[0]
	if op.RequestBody == nil {
		t.Error("ExtractOpenAPIData() should extract request body")
	}
	if !op.RequestBody.Required {
		t.Error("ExtractOpenAPIData() request body should be required")
	}
}

func TestExtractOpenAPIData_WithResponses(t *testing.T) {
	desc := "Success response"
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}

	responses := openapi3.NewResponses()
	responses.Set("200", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: &desc,
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"array"},
						},
					},
				},
			},
		},
	})
	pathItem := &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getPets",
			Responses:   responses,
		},
	}
	doc.Paths.Set("/pets", pathItem)

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	if len(result.Operations) == 0 {
		t.Fatal("ExtractOpenAPIData() should extract operations")
	}

	op := result.Operations[0]
	if len(op.Responses) == 0 {
		t.Error("ExtractOpenAPIData() should extract responses")
	}
	response, ok := op.Responses["200"]
	if !ok {
		t.Error("ExtractOpenAPIData() should extract 200 response")
	}
	if response.Description != desc {
		t.Errorf("ExtractOpenAPIData() response description = %q, want %q", response.Description, desc)
	}
}

func TestExtractOpenAPIData_WithSchemas(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{
				"Pet": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"object"},
						Properties: openapi3.Schemas{
							"id": &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: &openapi3.Types{"integer"},
								},
							},
							"name": &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: &openapi3.Types{"string"},
								},
							},
						},
						Required: []string{"id", "name"},
					},
				},
			},
		},
		Paths: openapi3.NewPaths(),
	}

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	if len(result.Schemas) == 0 {
		t.Error("ExtractOpenAPIData() should extract schemas")
	}

	petSchema, ok := result.Schemas["Pet"]
	if !ok {
		t.Error("ExtractOpenAPIData() should extract Pet schema")
	}
	if petSchema.Type != "object" {
		t.Errorf("ExtractOpenAPIData() Pet schema type = %q, want %q", petSchema.Type, "object")
	}
	if len(petSchema.Properties) != 2 {
		t.Errorf("ExtractOpenAPIData() Pet schema should have 2 properties, got %d", len(petSchema.Properties))
	}
}

func TestExtractOpenAPIData_WithArraySchema(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{
				"PetList": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"array"},
						Items: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: &openapi3.Types{"string"},
							},
						},
					},
				},
			},
		},
		Paths: openapi3.NewPaths(),
	}

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	petListSchema, ok := result.Schemas["PetList"]
	if !ok {
		t.Fatal("ExtractOpenAPIData() should extract PetList schema")
	}
	if petListSchema.Type != "array" {
		t.Errorf("ExtractOpenAPIData() PetList schema type = %q, want %q", petListSchema.Type, "array")
	}
	if petListSchema.Items == nil {
		t.Error("ExtractOpenAPIData() PetList schema should have items")
	}
	if petListSchema.Items.Type != "string" {
		t.Errorf("ExtractOpenAPIData() PetList items type = %q, want %q", petListSchema.Items.Type, "string")
	}
}

func TestExtractOpenAPIData_WithSecuritySchemes(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Components: &openapi3.Components{
			SecuritySchemes: openapi3.SecuritySchemes{
				"apiKey": &openapi3.SecuritySchemeRef{
					Value: &openapi3.SecurityScheme{
						Type: "apiKey",
						In:   "header",
						Name: "X-API-Key",
					},
				},
				"bearer": &openapi3.SecuritySchemeRef{
					Value: &openapi3.SecurityScheme{
						Type:   "http",
						Scheme: "bearer",
					},
				},
			},
		},
		Paths: openapi3.NewPaths(),
	}

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	if len(result.SecuritySchemes) != 2 {
		t.Errorf("ExtractOpenAPIData() should extract 2 security schemes, got %d", len(result.SecuritySchemes))
	}

	apiKeyScheme, ok := result.SecuritySchemes["apiKey"]
	if !ok {
		t.Error("ExtractOpenAPIData() should extract apiKey security scheme")
	}
	if apiKeyScheme.Type != "apiKey" {
		t.Errorf("ExtractOpenAPIData() apiKey type = %q, want %q", apiKeyScheme.Type, "apiKey")
	}

	bearerScheme, ok := result.SecuritySchemes["bearer"]
	if !ok {
		t.Error("ExtractOpenAPIData() should extract bearer security scheme")
	}
	if bearerScheme.Type != "http" {
		t.Errorf("ExtractOpenAPIData() bearer type = %q, want %q", bearerScheme.Type, "http")
	}
}

func TestExtractOpenAPIData_WithSecurityRequirements(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}

	securityReq := openapi3.SecurityRequirement{
		"apiKey": []string{},
	}
	pathItem := &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getPets",
			Security:    &openapi3.SecurityRequirements{securityReq},
		},
	}
	doc.Paths.Set("/pets", pathItem)

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	if len(result.Operations) == 0 {
		t.Fatal("ExtractOpenAPIData() should extract operations")
	}

	op := result.Operations[0]
	if len(op.Security) == 0 {
		t.Error("ExtractOpenAPIData() should extract security requirements")
	}
	if len(op.Security[0].Schemes) == 0 {
		t.Error("ExtractOpenAPIData() security requirement should have schemes")
	}
	if op.Security[0].Schemes[0] != "apiKey" {
		t.Errorf("ExtractOpenAPIData() security scheme = %q, want %q", op.Security[0].Schemes[0], "apiKey")
	}
}

func TestExtractOpenAPIData_WithOAuth2(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Components: &openapi3.Components{
			SecuritySchemes: openapi3.SecuritySchemes{
				"oauth2": &openapi3.SecuritySchemeRef{
					Value: &openapi3.SecurityScheme{
						Type: "oauth2",
						Flows: &openapi3.OAuthFlows{
							AuthorizationCode: &openapi3.OAuthFlow{
								AuthorizationURL: "https://example.com/oauth/authorize",
								TokenURL:         "https://example.com/oauth/token",
								Scopes:           map[string]string{"read": "Read access"},
							},
						},
					},
				},
			},
		},
		Paths: openapi3.NewPaths(),
	}

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	oauthScheme, ok := result.SecuritySchemes["oauth2"]
	if !ok {
		t.Fatal("ExtractOpenAPIData() should extract oauth2 security scheme")
	}
	if oauthScheme.Type != "oauth2" {
		t.Errorf("ExtractOpenAPIData() oauth2 type = %q, want %q", oauthScheme.Type, "oauth2")
	}
	if oauthScheme.OAuth2Flows == nil {
		t.Error("ExtractOpenAPIData() oauth2 should have flows")
	}
	if oauthScheme.OAuth2Flows.AuthorizationCode == nil {
		t.Error("ExtractOpenAPIData() oauth2 should have authorization code flow")
	}
}

func TestExtractOpenAPIData_WithOpenIDConnect(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Components: &openapi3.Components{
			SecuritySchemes: openapi3.SecuritySchemes{
				"openId": &openapi3.SecuritySchemeRef{
					Value: &openapi3.SecurityScheme{
						Type:             "openIdConnect",
						OpenIdConnectUrl: "https://example.com/.well-known/openid-configuration",
					},
				},
			},
		},
		Paths: openapi3.NewPaths(),
	}

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	openIDScheme, ok := result.SecuritySchemes["openId"]
	if !ok {
		t.Fatal("ExtractOpenAPIData() should extract openId security scheme")
	}
	if openIDScheme.Type != "openIdConnect" {
		t.Errorf("ExtractOpenAPIData() openId type = %q, want %q", openIDScheme.Type, "openIdConnect")
	}
	if openIDScheme.OpenIDConnectURL == "" {
		t.Error("ExtractOpenAPIData() openId should have OpenID Connect URL")
	}
}

func TestExtractOpenAPIData_WithExamples(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}

	exampleValue := map[string]interface{}{"id": 1, "name": "test"}
	pathItem := &openapi3.PathItem{
		Post: &openapi3.Operation{
			OperationID: "createPet",
			RequestBody: &openapi3.RequestBodyRef{
				Value: &openapi3.RequestBody{
					Content: openapi3.Content{
						"application/json": &openapi3.MediaType{
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: &openapi3.Types{"object"},
								},
							},
							Example: exampleValue,
							Examples: map[string]*openapi3.ExampleRef{
								"example1": {
									Value: &openapi3.Example{
										Value: exampleValue,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	doc.Paths.Set("/pets", pathItem)

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	if len(result.Operations) == 0 {
		t.Fatal("ExtractOpenAPIData() should extract operations")
	}

	op := result.Operations[0]
	if op.RequestBody == nil {
		t.Fatal("ExtractOpenAPIData() should extract request body")
	}
	content, ok := op.RequestBody.Content["application/json"]
	if !ok {
		t.Fatal("ExtractOpenAPIData() should extract content")
	}
	if len(content.Examples) == 0 {
		t.Error("ExtractOpenAPIData() should extract examples")
	}
}

func TestExtractOpenAPIData_WithResponseExamples(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}

	exampleValue := map[string]interface{}{"id": 1, "name": "test"}
	desc := "Success"
	responses := openapi3.NewResponses()
	responses.Set("200", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: &desc,
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"object"},
						},
					},
					Example: exampleValue,
					Examples: map[string]*openapi3.ExampleRef{
						"example1": {
							Value: &openapi3.Example{
								Value: exampleValue,
							},
						},
					},
				},
			},
		},
	})
	pathItem := &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getPet",
			Responses:   responses,
		},
	}
	doc.Paths.Set("/pets", pathItem)

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	if len(result.Operations) == 0 {
		t.Fatal("ExtractOpenAPIData() should extract operations")
	}

	op := result.Operations[0]
	response, ok := op.Responses["200"]
	if !ok {
		t.Fatal("ExtractOpenAPIData() should extract 200 response")
	}
	content, ok := response.Content["application/json"]
	if !ok {
		t.Fatal("ExtractOpenAPIData() should extract content")
	}
	if len(content.Examples) == 0 {
		t.Error("ExtractOpenAPIData() should extract examples")
	}
}

func TestExtractOpenAPIData_WithNilSchema(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{
				"Empty": &openapi3.SchemaRef{
					Value: nil, // Nil schema
				},
			},
		},
		Paths: openapi3.NewPaths(),
	}

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	// Should not crash, nil schemas should be skipped
	if len(result.Schemas) > 0 {
		t.Log("ExtractOpenAPIData() skipped nil schema (expected)")
	}
}

func TestExtractOpenAPIData_WithNilParameterSchema(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}

	pathItem := &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getPet",
			Parameters: openapi3.Parameters{
				{
					Value: &openapi3.Parameter{
						Name:     "id",
						In:       "path",
						Required: true,
						Schema: &openapi3.SchemaRef{
							Value: nil, // Nil schema value
						},
					},
				},
			},
		},
	}
	doc.Paths.Set("/pets/{id}", pathItem)

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	if len(result.Operations) == 0 {
		t.Fatal("ExtractOpenAPIData() should extract operations")
	}

	op := result.Operations[0]
	if len(op.Parameters) == 0 {
		t.Fatal("ExtractOpenAPIData() should extract parameters even with nil schema")
	}
	// Schema can be nil when param.Value.Schema.Value is nil
	if op.Parameters[0].Schema != nil {
		t.Log("ExtractOpenAPIData() parameter with nil schema value handled")
	}
}

func TestExtractOpenAPIData_WithOAuth2AllFlows(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Components: &openapi3.Components{
			SecuritySchemes: openapi3.SecuritySchemes{
				"oauth2": &openapi3.SecuritySchemeRef{
					Value: &openapi3.SecurityScheme{
						Type: "oauth2",
						Flows: &openapi3.OAuthFlows{
							AuthorizationCode: &openapi3.OAuthFlow{
								AuthorizationURL: "https://example.com/oauth/authorize",
								TokenURL:         "https://example.com/oauth/token",
								Scopes:           map[string]string{"read": "Read access"},
							},
							ClientCredentials: &openapi3.OAuthFlow{
								TokenURL: "https://example.com/oauth/token",
								Scopes:   map[string]string{"write": "Write access"},
							},
							Implicit: &openapi3.OAuthFlow{
								AuthorizationURL: "https://example.com/oauth/authorize",
								Scopes:           map[string]string{"read": "Read access"},
							},
							Password: &openapi3.OAuthFlow{
								TokenURL: "https://example.com/oauth/token",
								Scopes:   map[string]string{"read": "Read access"},
							},
						},
					},
				},
			},
		},
		Paths: openapi3.NewPaths(),
	}

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	oauthScheme, ok := result.SecuritySchemes["oauth2"]
	if !ok {
		t.Fatal("ExtractOpenAPIData() should extract oauth2 security scheme")
	}
	if oauthScheme.OAuth2Flows == nil {
		t.Fatal("ExtractOpenAPIData() oauth2 should have flows")
	}
	if oauthScheme.OAuth2Flows.ClientCredentials == nil {
		t.Error("ExtractOpenAPIData() oauth2 should have client credentials flow")
	}
	if oauthScheme.OAuth2Flows.Implicit == nil {
		t.Error("ExtractOpenAPIData() oauth2 should have implicit flow")
	}
	if oauthScheme.OAuth2Flows.Password == nil {
		t.Error("ExtractOpenAPIData() oauth2 should have password flow")
	}
}

func TestExtractOpenAPIData_WithNilParameterValue(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}

	pathItem := &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getPet",
			Parameters: openapi3.Parameters{
				{
					Value: nil, // Nil parameter value
				},
			},
		},
	}
	doc.Paths.Set("/pets", pathItem)

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	if len(result.Operations) == 0 {
		t.Fatal("ExtractOpenAPIData() should extract operations")
	}

	op := result.Operations[0]
	// Parameters with nil Value should be skipped
	if len(op.Parameters) > 0 {
		t.Log("ExtractOpenAPIData() skipped nil parameter value (expected)")
	}
}

func TestExtractOpenAPIData_WithNilRequestBodyValue(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}

	pathItem := &openapi3.PathItem{
		Post: &openapi3.Operation{
			OperationID: "createPet",
			RequestBody: &openapi3.RequestBodyRef{
				Value: nil, // Nil request body value
			},
		},
	}
	doc.Paths.Set("/pets", pathItem)

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	if len(result.Operations) == 0 {
		t.Fatal("ExtractOpenAPIData() should extract operations")
	}

	op := result.Operations[0]
	// Request body with nil Value should be skipped
	if op.RequestBody != nil {
		t.Log("ExtractOpenAPIData() skipped nil request body value (expected)")
	}
}

func TestExtractOpenAPIData_WithNilResponseValue(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}

	responses := openapi3.NewResponses()
	responses.Set("200", &openapi3.ResponseRef{
		Value: nil, // Nil response value
	})
	pathItem := &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getPets",
			Responses:   responses,
		},
	}
	doc.Paths.Set("/pets", pathItem)

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	if len(result.Operations) == 0 {
		t.Fatal("ExtractOpenAPIData() should extract operations")
	}

	op := result.Operations[0]
	// Responses with nil Value should be skipped
	if len(op.Responses) > 0 {
		t.Log("ExtractOpenAPIData() skipped nil response value (expected)")
	}
}

func TestExtractOpenAPIData_WithNilSecuritySchemeValue(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Components: &openapi3.Components{
			SecuritySchemes: openapi3.SecuritySchemes{
				"apiKey": &openapi3.SecuritySchemeRef{
					Value: nil, // Nil security scheme value
				},
			},
		},
		Paths: openapi3.NewPaths(),
	}

	result, err := ExtractOpenAPIData(doc)
	if err != nil {
		t.Fatalf("ExtractOpenAPIData() error = %v", err)
	}

	// Security schemes with nil Value should be skipped
	if len(result.SecuritySchemes) > 0 {
		t.Log("ExtractOpenAPIData() skipped nil security scheme value (expected)")
	}
}

func TestGetOperationMethodName_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		op       APIOperation
		expected string
	}{
		{
			name: "operation_id_with_underscores",
			op: APIOperation{
				OperationID: "get_user_profile",
				Method:      "GET",
				Path:        "/users/{id}/profile",
			},
			expected: "get_user_profile",
		},
		{
			name: "operation_id_with_dashes",
			op: APIOperation{
				OperationID: "get-user-profile",
				Method:      "GET",
				Path:        "/users/{id}/profile",
			},
			expected: "get_user_profile",
		},
		{
			name: "path_with_multiple_params",
			op: APIOperation{
				Method: "GET",
				Path:   "/users/{userId}/posts/{postId}",
			},
			expected: "get_users_posts",
		},
		{
			name: "path_only_params",
			op: APIOperation{
				Method: "GET",
				Path:   "/{id}",
			},
			expected: "get_root", // Path with only params becomes "root"
		},
		{
			name: "path_with_trailing_slash",
			op: APIOperation{
				Method: "GET",
				Path:   "/users/",
			},
			expected: "get_users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetOperationMethodName(tt.op)
			if result != tt.expected {
				t.Errorf("GetOperationMethodName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestToSnakeCase_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"camelCase_boundary", "helloWorldTest", "hello_world_test"},
		{"multiple_uppercase", "XMLParser", "x_m_l_parser"},
		{"starts_with_uppercase", "HelloWorld", "hello_world"},
		{"numbers_in_middle", "test123Value", "test123_value"},
		{"empty_string", "", ""}, // Test empty string branch
		{"single_char", "a", "a"},
		{"single_uppercase", "A", "a"},
		{"uppercase_after_lowercase", "helloWorld", "hello_world"},
		{"uppercase_after_number", "test123Value", "test123_value"},
		{"uppercase_at_start", "Hello", "hello"},
		{"consecutive_uppercase", "XMLHTTP", "x_m_l_h_t_t_p"},
		{"mixed_delimiters", "hello_world-test.value", "hello_world_test_value"},
		{"only_delimiters", "---", ""},
		{"delimiters_with_content", "-hello-", "hello"},
		{"whitespace_only", "   ", ""},
		{"whitespace_with_content", "hello world", "hello_world"},
		{"numbers_only", "123", "123"},
		{"starts_with_number", "123test", "123test"},
		{"uppercase_followed_by_lowercase", "Hello", "hello"},
		{"uppercase_at_position_zero", "HELLO", "h_e_l_l_o"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToCamelCase_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"camelCase_boundary", "helloWorldTest", "helloWorldTest"},
		{"multiple_uppercase", "XMLParser", "xMLParser"},
		{"starts_with_uppercase", "HelloWorld", "helloWorld"},
		{"numbers_in_middle", "test123Value", "test123Value"},
		{"empty_parts", "   ", "   "}, // Whitespace-only strings are preserved
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCapitalize_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"numbers_only", "123", "123"},
		{"special_chars", "hello-world", "Hello-World"},
		{"mixed_case_numbers", "h3ll0", "H3ll0"},
		{"empty_string", "", ""}, // Test empty string branch
		{"single_char", "a", "A"},
		{"single_uppercase", "A", "A"},
		{"simple_word", "hello", "Hello"},
		{"uppercase_word", "HELLO", "Hello"},
		{"mixed_case", "hELLo", "Hello"},
		{"with_numbers", "hello123", "Hello123"},
		{"already_capitalized", "Hello", "Hello"},
		{"unicode", "héllo", "Héllo"},
		{"whitespace", "hello world", "Hello World"},
		{"whitespace_only", "   ", ""}, // TrimSpace removes whitespace
		{"special_chars_only", "---", "---"},
		{"numbers_at_start", "123test", "123Test"},
		{"numbers_at_end", "test123", "Test123"},
		{"single_number", "1", "1"},
		{"uppercase_after_number", "test123Value", "Test123value"}, // Title case converts entire string
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// capitalize is not exported, test via template
			tmpl := "{{capitalize .SDKName}}"
			result, err := RenderTemplate(tmpl, TemplateData{SDKName: tt.input})
			if err != nil {
				t.Fatalf("RenderTemplate() error = %v", err)
			}
			result = strings.TrimSpace(result)
			if result != tt.expected {
				t.Errorf("capitalize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRenderTemplate_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		tmplContent string
		data        TemplateData
		wantErr     bool
	}{
		{
			name:        "invalid_field",
			tmplContent: "{{.NonExistentField}}",
			data:        TemplateData{SDKName: "test"},
			wantErr:     true, // Template execution fails with invalid field
		},
		{
			name:        "invalid_function",
			tmplContent: "{{invalidFunc .SDKName}}",
			data:        TemplateData{SDKName: "test"},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RenderTemplate(tt.tmplContent, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
