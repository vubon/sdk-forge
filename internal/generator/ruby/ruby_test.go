package ruby

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/vubon/sdk-forge/internal/generator/common"
	httplib "github.com/vubon/sdk-forge/pkg/languages/http"
)

func TestGenerateRubyGemspec_Basic(t *testing.T) {
	libConfig, err := httplib.GetLibraryConfig("ruby", "faraday")
	if err != nil {
		t.Fatalf("failed to get lib config: %v", err)
	}

	extracted := &common.ExtractedData{Description: "Example SDK"}
	out, err := generateRubyGemspec("MySDK", "0.1.0", common.GetRubyDefaultVersion(), libConfig, extracted)
	if err != nil {
		t.Fatalf("generateRubyGemspec returned error: %v", err)
	}

	expectedName := common.ToSnakeCase("MySDK")
	if !strings.Contains(out, "spec.name") || !strings.Contains(out, expectedName) {
		t.Fatalf("gemspec missing expected name %s; got: %s", expectedName, out)
	}
	if !strings.Contains(out, "spec.version       = '0.1.0'") {
		t.Fatalf("gemspec missing version; got: %s", out)
	}
	if !strings.Contains(out, "spec.add_dependency 'faraday'") && !strings.Contains(out, "spec.add_dependency 'faraday',") {
		t.Fatalf("gemspec missing dependency for faraday; got: %s", out)
	}
}

func TestGenerateRubyGemfile(t *testing.T) {
	got := generateRubyGemfile()
	if !strings.Contains(got, "source 'https://rubygems.org'") || !strings.Contains(got, "gemspec") {
		t.Fatalf("unexpected Gemfile content: %s", got)
	}
}

func TestGenerateRubyMainLib_IncludesModelsAndAPIs(t *testing.T) {
	extracted := &common.ExtractedData{
		Schemas: map[string]*common.Schema{"Pet": {}},
		Operations: []common.APIOperation{
			{
				OperationID: "listPets",
				Summary:     "List pets",
				Tags:        []string{"pets"},
			},
		},
	}

	got := generateRubyMainLib("my_sdk", extracted)
	if !strings.Contains(got, "require_relative 'my_sdk/models/pet'") {
		t.Fatalf("main lib missing model require; got: %s", got)
	}
	if !strings.Contains(got, "require_relative 'my_sdk/api/pets_api'") {
		t.Fatalf("main lib missing api require; got: %s", got)
	}
	if !strings.Contains(got, "module MySdk") {
		t.Fatalf("main lib missing module declaration; got: %s", got)
	}
}

func TestGenerateRubyVersion(t *testing.T) {
	got := generateRubyVersion("my_sdk", "2.3.4")
	if !strings.Contains(got, "module MySdk") || !strings.Contains(got, "VERSION = '2.3.4'") {
		t.Fatalf("version file unexpected content: %s", got)
	}
}

func TestGenerateRubyException(t *testing.T) {
	got := generateRubyException("my_sdk")
	if !strings.Contains(got, "class ApiException < StandardError") || !strings.Contains(got, "attr_reader :status_code") {
		t.Fatalf("exception file unexpected content: %s", got)
	}
}

func TestGenerateRubyQualityConfigs_WritesFiles(t *testing.T) {
	dir := t.TempDir()
	// call function under test
	if err := generateRubyQualityConfigs(dir); err != nil {
		t.Fatalf("generateRubyQualityConfigs returned error: %v", err)
	}

	// rubocop
	rubocopPath := filepath.Join(dir, ".rubocop.yml")
	b, err := os.ReadFile(rubocopPath)
	if err != nil {
		t.Fatalf("failed to read .rubocop.yml: %v", err)
	}
	if string(b) != rubocopTemplate {
		t.Fatalf(".rubocop.yml content mismatch; got: %s", string(b))
	}

	// yardopts
	yardPath := filepath.Join(dir, ".yardopts")
	b2, err := os.ReadFile(yardPath)
	if err != nil {
		t.Fatalf("failed to read .yardopts: %v", err)
	}
	if string(b2) != yardoptsTemplate {
		t.Fatalf(".yardopts content mismatch; got: %s", string(b2))
	}
}

func TestGenerateRubyReadmeAndExample(t *testing.T) {
	data := common.TemplateData{SDKName: "my_sdk"}
	readme, err := generateRubyReadme(data, "0.0.1")
	if err != nil {
		t.Fatalf("generateRubyReadme error: %v", err)
	}
	if !strings.Contains(readme, "# MySdk") || !strings.Contains(readme, "Current version: 0.0.1") {
		t.Fatalf("readme content unexpected: %s", readme)
	}

	// example with no operations/security should produce placeholder
	ex := generateRubyExample(data, "my_sdk")
	if !strings.Contains(ex, "require 'my_sdk'") || !strings.Contains(ex, "client = MySdk::Client.new") {
		t.Fatalf("example content unexpected: %s", ex)
	}
}

func TestGenerateRubyClient_BasicAndRetry(t *testing.T) {
	// Basic client without retry
	data := common.TemplateData{
		SDKName:       "my_sdk",
		HTTPLib:       "faraday",
		HTTPLibImport: "faraday",
		RetryConfig:   common.DefaultRetryConfig(),
	}

	out, err := generateRubyClient(data, common.GetRubyDefaultVersion())
	if err != nil {
		t.Fatalf("generateRubyClient error: %v", err)
	}
	if !strings.Contains(out, "module MySdk") || !strings.Contains(out, "require 'faraday'") {
		t.Fatalf("client output unexpected: %s", out)
	}

	// With retry enabled
	rc := common.DefaultRetryConfig()
	rc.Enabled = true
	rc.MaxAttempts = 5
	rc.RetryableStatusCodes = []int{500, 502}

	extracted := &common.ExtractedData{
		SecuritySchemes: map[string]common.SecurityScheme{"api_key": {Type: "apiKey", In: "header", Name: "X-API-KEY"}},
	}

	data2 := common.TemplateData{
		SDKName:       "my_sdk",
		HTTPLib:       "faraday",
		HTTPLibImport: "faraday",
		RetryConfig:   rc,
		OpenAPIDoc:    extracted,
	}

	out2, err := generateRubyClient(data2, common.GetRubyDefaultVersion())
	if err != nil {
		t.Fatalf("generateRubyClient with retry error: %v", err)
	}
	if !strings.Contains(out2, "attr_accessor :retry_enabled") || !strings.Contains(out2, "@retry_status_codes") {
		t.Fatalf("retry-related content missing: %s", out2)
	}
	if !strings.Contains(out2, "headers['X-API-KEY']") {
		t.Fatalf("auth header handling missing: %s", out2)
	}
}

func TestGenerateRubyModel_ObjectAndSimple(t *testing.T) {
	schema := &common.Schema{
		Type: "object",
		Properties: map[string]*common.Schema{
			"Name": {Type: "string"},
			"Age":  {Type: "integer"},
		},
		Required: []string{"Name"},
	}

	out := generateRubyModel("Person", schema, "my_sdk")
	if !strings.Contains(out, "class Person") || !strings.Contains(out, "def to_h") || !strings.Contains(out, "def self.from_hash") {
		t.Fatalf("model output missing expected pieces: %s", out)
	}

	// simple (non-object) schema
	simple := &common.Schema{Type: "string"}
	out2 := generateRubyModel("Value", simple, "my_sdk")
	if !strings.Contains(out2, "attr_accessor :value") || !strings.Contains(out2, "def to_json") {
		t.Fatalf("simple model output unexpected: %s", out2)
	}
}

func TestGetRubyTypeAndContains(t *testing.T) {
	if getRubyType(&common.Schema{Type: "string"}) != "String" {
		t.Fatalf("expected String type")
	}
	if getRubyType(&common.Schema{Type: "array", Items: &common.Schema{Type: "integer"}}) != "Array<Integer>" {
		t.Fatalf("expected Array<Integer>")
	}
	if getRubyType(nil) != "Object" {
		t.Fatalf("expected Object for nil schema")
	}

	slice := []string{"a", "b"}
	if !contains(slice, "b") || contains(slice, "z") {
		t.Fatalf("contains behavior unexpected")
	}
}

func TestGenerateRubyAPIModuleAndMethod(t *testing.T) {
	op := common.APIOperation{
		Method:      "GET",
		Path:        "/pets/{id}",
		OperationID: "getPet",
		Summary:     "Get a pet",
		Parameters: []common.Parameter{
			{Name: "id", In: "path", Required: true, Schema: &common.Schema{Type: "string"}},
			{Name: "limit", In: "query", Required: false, Schema: &common.Schema{Type: "integer"}},
		},
	}

	data := common.TemplateData{SDKName: "my_sdk"}
	out := generateRubyAPIModule("pets", []common.APIOperation{op}, data, "my_sdk")
	if !strings.Contains(out, "module PetsApi") {
		t.Fatalf("api module missing module declaration: %s", out)
	}
	if !strings.Contains(out, "def self.get_pet(client") {
		t.Fatalf("api method signature missing: %s", out)
	}
	if !strings.Contains(out, "params = {}") {
		t.Fatalf("expected params initialization in method: %s", out)
	}
	if !strings.Contains(out, "client.request(:get, path, params: params") {
		t.Fatalf("expected client.request call for GET: %s", out)
	}

	// test helper functions
	if getMethodName(op) == "" {
		t.Fatalf("getMethodName returned empty")
	}
	if getRubyTypeFromSchema("number") != "Float" {
		t.Fatalf("getRubyTypeFromSchema unexpected")
	}
}

func TestGenerateRubyTests_WritesSpecs(t *testing.T) {
	dir := t.TempDir()
	sdkLibDir := filepath.Join(dir, "lib", "my_sdk")
	_ = os.MkdirAll(sdkLibDir, 0750)

	extracted := &common.ExtractedData{
		Schemas:    map[string]*common.Schema{"Pet": {Type: "object", Properties: map[string]*common.Schema{"Name": {Type: "string"}}}},
		Operations: []common.APIOperation{{Method: "GET", Path: "/pets", OperationID: "listPets", Tags: []string{"pets"}}},
	}

	data := common.TemplateData{SDKName: "my_sdk", OpenAPIDoc: extracted}

	if err := generateRubyTests(dir, sdkLibDir, data, extracted, "my_sdk"); err != nil {
		t.Fatalf("generateRubyTests error: %v", err)
	}

	// check files
	specs := []string{"spec/spec_helper.rb", "spec/client_spec.rb", "spec/models/pet_spec.rb", "spec/api/pets_api_spec.rb"}
	for _, p := range specs {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Fatalf("expected spec file %s to exist: %v", p, err)
		}
	}
}

func TestGenerateRubySDKFromExtracted_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	sdkName := "PetStore"

	extracted := &common.ExtractedData{
		Title:       "Pet Store",
		Version:     "0.5.0",
		Description: "Pet store API",
		Schemas: map[string]*common.Schema{
			"Pet": {Type: "object", Properties: map[string]*common.Schema{"id": {Type: "integer"}, "name": {Type: "string"}}, Required: []string{"id"}},
		},
		Operations: []common.APIOperation{
			{Method: "GET", Path: "/pets", OperationID: "listPets", Summary: "List pets", Tags: []string{"pets"}},
		},
		SecuritySchemes: map[string]common.SecurityScheme{"bearer": {Type: "http", Scheme: "bearer"}},
	}

	// Use faraday (valid)
	if err := generateRubySDKFromExtracted(dir, sdkName, "faraday", extracted, common.GetRubyDefaultVersion(), "0.5.0", true, func() common.RetryConfig { rc := common.DefaultRetryConfig(); rc.Enabled = true; return rc }()); err != nil {
		t.Fatalf("generateRubySDKFromExtracted error: %v", err)
	}

	sanitized := common.ToSnakeCase(sdkName)
	// check several expected files
	wantPaths := []string{
		filepath.Join(dir, sanitized+".gemspec"),
		filepath.Join(dir, "Gemfile"),
		filepath.Join(dir, "lib", sanitized+".rb"),
		filepath.Join(dir, "lib", sanitized, "client.rb"),
		filepath.Join(dir, "lib", sanitized, "version.rb"),
		filepath.Join(dir, "lib", sanitized, "models", "pet.rb"),
		filepath.Join(dir, "lib", sanitized, "api", "pets_api.rb"),
		filepath.Join(dir, "lib", sanitized, "exceptions", "api_exception.rb"),
		filepath.Join(dir, "README.md"),
		filepath.Join(dir, "examples", "basic_usage.rb"),
		filepath.Join(dir, "spec", "spec_helper.rb"),
	}

	for _, p := range wantPaths {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected file %s to exist: %v", p, err)
		}
	}
}

func TestGetRubyTypeFromSchema_AllTypes(t *testing.T) {
	cases := map[string]string{
		"string":  "String",
		"integer": "Integer",
		"number":  "Float",
		"boolean": "Boolean",
		"array":   "Array",
		"object":  "Hash",
		"unknown": "Object",
	}

	for in, want := range cases {
		got := getRubyTypeFromSchema(in)
		if got != want {
			t.Fatalf("type %s: expected %s, got %s", in, want, got)
		}
	}
}

func TestGetMethodName_Fallbacks(t *testing.T) {
	op := common.APIOperation{Method: "GET", Path: "/pets/{petId}"}
	// Ensure empty OperationID triggers fallback
	if got := getMethodName(op); got == "" {
		t.Fatal("expected non-empty method name")
	}

	op2 := common.APIOperation{Method: "POST", Path: "/"}
	if got := getMethodName(op2); got == "" || !strings.Contains(got, "post") {
		t.Fatalf("unexpected method name for root path: %s", got)
	}

	op3 := common.APIOperation{Method: "PUT", Path: "/users/{id}/photos"}
	if got := getMethodName(op3); !strings.Contains(got, "put_users_photos") {
		t.Fatalf("unexpected constructed name: %s", got)
	}
}

func TestGenerateRubyAPIMethod_DocsAndParams(t *testing.T) {
	op := common.APIOperation{
		OperationID: "createItem",
		Method:      "POST",
		Path:        "/items/{itemId}",
		Summary:     "Create item",
		Description: "Creates a new item\nwith details",
		Parameters: []common.Parameter{
			{Name: "itemId", In: "path", Required: true, Description: "the id", Schema: &common.Schema{Type: "string"}},
			{Name: "filter", In: "query", Required: false, Description: "filter param", Schema: &common.Schema{Type: "integer"}},
		},
		RequestBody: &common.RequestBody{Required: true, Content: map[string]common.ContentType{"application/json": {Schema: &common.Schema{Type: "object"}}}},
	}

	txt := generateRubyAPIMethod(op, common.TemplateData{})
	if !strings.Contains(txt, "Create item") || !strings.Contains(txt, "Creates a new item") {
		t.Fatal("expected doc comments to include summary and description")
	}
	if !strings.Contains(txt, "def self.create_item") {
		t.Fatal("expected method signature")
	}
	if !strings.Contains(txt, "client.request(:post") {
		t.Fatal("expected client.request call for POST")
	}
}

func TestGenerateRubyAPIModule_IncludesMethods(t *testing.T) {
	op := common.APIOperation{OperationID: "list", Method: "GET", Path: "/list"}
	mod := generateRubyAPIModule("pets", []common.APIOperation{op}, common.TemplateData{}, "my_sdk")
	if !strings.Contains(mod, "module PetsApi") || !strings.Contains(mod, "def self.list") {
		t.Fatal("expected API module and method")
	}
}

func TestGenerateRubyModelSpec_Object(t *testing.T) {
	schema := &common.Schema{Type: "object", Properties: map[string]*common.Schema{"name": {Type: "string"}}}
	txt := generateRubyModelSpec("Pet", schema, "my_sdk")
	if !strings.Contains(txt, "from_hash") || !strings.Contains(txt, "to_h") {
		t.Fatal("expected model spec helpers")
	}
}

func TestGenerateRubyClientSpec_WithAuth(t *testing.T) {
	data := common.TemplateData{}
	extracted := &common.ExtractedData{SecuritySchemes: map[string]common.SecurityScheme{"apiKey": {Type: "apiKey"}, "bearer": {Type: "http", Scheme: "bearer"}}}
	data.OpenAPIDoc = extracted
	txt := generateRubyClientSpec(data, "my_sdk")
	if !strings.Contains(txt, "sets API key") && !strings.Contains(txt, "sets bearer token") {
		t.Fatal("expected auth spec for apiKey or bearer")
	}
}

func TestGenerateRubyTests_WritesSpecs_All(t *testing.T) {
	out := t.TempDir()
	// prepare data with schemas and operations
	schema := &common.Schema{Type: "object", Properties: map[string]*common.Schema{"id": {Type: "string"}}}
	ops := []common.APIOperation{{OperationID: "getPet", Method: "GET", Path: "/pets/{id}", Tags: []string{"pets"}, Parameters: []common.Parameter{{Name: "id", In: "path", Required: true}}}}
	data := common.TemplateData{OpenAPIDoc: &common.ExtractedData{SecuritySchemes: map[string]common.SecurityScheme{"apiKey": {Type: "apiKey"}}}}
	extracted := &common.ExtractedData{Schemas: map[string]*common.Schema{"Pet": schema}, Operations: ops}

	if err := generateRubyTests(out, filepath.Join(out, "lib", "my_sdk"), data, extracted, "my_sdk"); err != nil {
		t.Fatalf("generateRubyTests failed: %v", err)
	}

	// Verify files
	mustExist := []string{
		filepath.Join(out, "spec", "spec_helper.rb"),
		filepath.Join(out, "spec", "client_spec.rb"),
		filepath.Join(out, "spec", "models", "pet_spec.rb"),
		filepath.Join(out, "spec", "api", "pets_api_spec.rb"),
	}
	for _, p := range mustExist {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected %s to exist: %v", p, err)
		}
	}
}

func TestGenerateRubySDK_InvalidDocType(t *testing.T) {
	out := t.TempDir()
	err := GenerateRubySDK(out, "my_sdk", "faraday", struct{}{}, nil, "", false, common.RetryConfig{})
	if err == nil || !strings.Contains(err.Error(), "invalid OpenAPI document type") {
		t.Fatalf("expected invalid OpenAPI document type error, got: %v", err)
	}
}

func TestGenerateRubyExample_WithBasicAuth(t *testing.T) {
	data := common.TemplateData{OpenAPIDoc: &common.ExtractedData{SecuritySchemes: map[string]common.SecurityScheme{"basic": {Type: "http", Scheme: "basic"}}}}
	got := generateRubyExample(data, "my_sdk")
	if !strings.Contains(got, "username") || !strings.Contains(got, "password") {
		t.Fatalf("expected basic auth example to include username/password; got: %s", got)
	}
}

func TestGetMethodNameFallbackAndRubyTypeMapping(t *testing.T) {
	op := common.APIOperation{Method: "POST", Path: "/", OperationID: ""}
	name := getMethodName(op)
	if name == "" {
		t.Fatalf("expected non-empty method name for fallback")
	}

	if getRubyType(&common.Schema{Type: "boolean"}) != "Boolean" {
		t.Fatalf("expected Boolean type")
	}
	if getRubyTypeFromSchema("object") != "Hash" {
		t.Fatalf("expected Hash for object")
	}
}

func TestGenerateRubyClient_NetHTTPAndHTTPRb(t *testing.T) {
	dataNet := common.TemplateData{SDKName: "my_sdk", HTTPLib: "net-http", HTTPLibImport: "net/http", RetryConfig: common.DefaultRetryConfig()}
	outNet, err := generateRubyClient(dataNet, common.GetRubyDefaultVersion())
	if err != nil {
		t.Fatalf("generateRubyClient net-http error: %v", err)
	}
	if !strings.Contains(outNet, "Net::HTTP") || !strings.Contains(outNet, "Net::HTTP::Get.new") {
		t.Fatalf("net-http client missing pieces: %s", outNet)
	}

	dataHTTP := common.TemplateData{SDKName: "my_sdk", HTTPLib: "httprb", HTTPLibImport: "http", RetryConfig: common.DefaultRetryConfig()}
	outHTTP, err := generateRubyClient(dataHTTP, common.GetRubyDefaultVersion())
	if err != nil {
		t.Fatalf("generateRubyClient httprb error: %v", err)
	}
	if !strings.Contains(outHTTP, "@http_client = HTTP") {
		t.Fatalf("httprb client missing assignment: %s", outHTTP)
	}
}

func TestGenerateRubyExample_WithOperationsAndSecurity(t *testing.T) {
	extracted := &common.ExtractedData{
		SecuritySchemes: map[string]common.SecurityScheme{"apiKey": {Type: "apiKey", In: "header", Name: "X-API-KEY"}},
		Operations:      []common.APIOperation{{OperationID: "listPets", Summary: "List pets", Tags: []string{"pets"}, Parameters: []common.Parameter{{Name: "limit", In: "query", Required: false}}}},
	}
	data := common.TemplateData{SDKName: "my_sdk", OpenAPIDoc: extracted}
	ex := generateRubyExample(data, "my_sdk")
	if !strings.Contains(ex, "client.api_key") && !strings.Contains(ex, "client.bearer_token") {
		t.Fatalf("example missing auth examples: %s", ex)
	}
	if !strings.Contains(ex, "API::PetsApi") || !strings.Contains(ex, "list_pets") {
		t.Fatalf("example missing API call example: %s", ex)
	}
}

func TestGenerateRubyClientSpec_AuthBranch(t *testing.T) {
	extracted := &common.ExtractedData{SecuritySchemes: map[string]common.SecurityScheme{"bearer": {Type: "http", Scheme: "bearer"}}}
	data := common.TemplateData{OpenAPIDoc: extracted}
	out := generateRubyClientSpec(data, "my_sdk")
	if !strings.Contains(out, "sets bearer token") {
		t.Fatalf("client spec missing bearer token test: %s", out)
	}
}

func TestGenerateRubyClient_BasicAuthBranch(t *testing.T) {
	extracted := &common.ExtractedData{SecuritySchemes: map[string]common.SecurityScheme{"basic": {Type: "http", Scheme: "basic"}}}
	data := common.TemplateData{OpenAPIDoc: extracted, SDKName: "my_sdk", HTTPLib: "faraday", HTTPLibImport: "faraday", RetryConfig: common.DefaultRetryConfig()}
	out, err := generateRubyClient(data, common.GetRubyDefaultVersion())
	if err != nil {
		t.Fatalf("generateRubyClient basic auth error: %v", err)
	}
	if !strings.Contains(out, "Base64.strict_encode64") || !strings.Contains(out, "headers['Authorization'] = \"Basic") {
		t.Fatalf("expected basic auth encoding code in client: %s", out)
	}
}

func TestGetMethodName_MultiPartFallback(t *testing.T) {
	op := common.APIOperation{Method: "DELETE", Path: "/users/{userId}/pets/{petId}", OperationID: ""}
	name := getMethodName(op)
	if !strings.Contains(name, "delete") || !strings.Contains(name, "users") || !strings.Contains(name, "pets") {
		t.Fatalf("getMethodName fallback did not include expected parts: %s", name)
	}
}

func TestGetMethodName_WithOperationID(t *testing.T) {
	op := common.APIOperation{OperationID: "ListPets"}
	name := getMethodName(op)
	if name != "list_pets" {
		t.Fatalf("expected list_pets, got %s", name)
	}
}

func TestGetRubyType_AdditionalCases(t *testing.T) {
	if getRubyType(&common.Schema{Type: "number"}) != "Float" {
		t.Fatalf("expected Float for number")
	}
	if getRubyType(&common.Schema{Type: "array"}) != "Array" {
		t.Fatalf("expected Array for array without items")
	}
	if getRubyType(&common.Schema{Type: "unknown"}) != "Object" {
		t.Fatalf("expected Object for unknown")
	}
}

func TestGenerateRubySDK_Wrapper_UsingExtracted(t *testing.T) {
	dir := t.TempDir()
	extracted := &common.ExtractedData{Title: "X", Version: "0.1.0", Description: "desc"}
	if err := GenerateRubySDK(dir, "WrapperSDK", "faraday", extracted, nil, "", false, common.DefaultRetryConfig()); err != nil {
		t.Fatalf("GenerateRubySDK returned error: %v", err)
	}
	// check gemspec exists
	sanitized := common.ToSnakeCase("WrapperSDK")
	if _, err := os.Stat(filepath.Join(dir, sanitized+".gemspec")); err != nil {
		t.Fatalf("expected gemspec file after GenerateRubySDK: %v", err)
	}
}

func TestGenerateRubySDK_WithOpenAPI3Doc(t *testing.T) {
	dir := t.TempDir()
	doc := &openapi3.T{}
	doc.Info = &openapi3.Info{Title: "API", Version: "1.0.0"}
	doc.Servers = openapi3.Servers{{URL: "https://api.example.com"}}

	if err := GenerateRubySDK(dir, "FromOpenAPI", "faraday", doc, nil, "", false, common.DefaultRetryConfig()); err != nil {
		t.Fatalf("GenerateRubySDK with openapi3 doc error: %v", err)
	}
	sanitized := common.ToSnakeCase("FromOpenAPI")
	if _, err := os.Stat(filepath.Join(dir, sanitized+".gemspec")); err != nil {
		t.Fatalf("expected gemspec for openapi3 output: %v", err)
	}
}

func TestGenerateRubySDK_NoSchemasNoOps_NetHTTP(t *testing.T) {
	dir := t.TempDir()
	extracted := &common.ExtractedData{Title: "Empty", Version: "0.0.1"}
	// use net-http to hit different client branch
	if err := generateRubySDKFromExtracted(dir, "EmptySDK", "net-http", extracted, common.GetRubyDefaultVersion(), "0.0.1", true, common.DefaultRetryConfig()); err != nil {
		t.Fatalf("generateRubySDKFromExtracted net-http error: %v", err)
	}
	sanitized := common.ToSnakeCase("EmptySDK")
	if _, err := os.Stat(filepath.Join(dir, sanitized+".gemspec")); err != nil {
		t.Fatalf("expected gemspec: %v", err)
	}
}

func TestGenerateRubySDK_Httprb_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	extracted := &common.ExtractedData{
		Title:       "Httprb API",
		Version:     "0.2.0",
		Description: "Example",
		Schemas:     map[string]*common.Schema{"Item": {Type: "object", Properties: map[string]*common.Schema{"id": {Type: "integer"}}}},
		Operations:  []common.APIOperation{{Method: "GET", Path: "/items", OperationID: "listItems", Tags: []string{"items"}}},
	}

	if err := generateRubySDKFromExtracted(dir, "HttprbSDK", "httprb", extracted, common.GetRubyDefaultVersion(), "0.2.0", true, common.DefaultRetryConfig()); err != nil {
		t.Fatalf("generateRubySDKFromExtracted httprb error: %v", err)
	}
	sanitized := common.ToSnakeCase("HttprbSDK")
	if _, err := os.Stat(filepath.Join(dir, "lib", sanitized, "client.rb")); err != nil {
		t.Fatalf("expected client.rb for httprb: %v", err)
	}
}

func TestGenerateRubyClient_Combinations(t *testing.T) {
	libs := []string{"faraday", "net-http", "httprb", "unknown-lib"}
	schemes := []common.SecurityScheme{
		{Type: "apiKey", In: "header", Name: "X-API-KEY"},
		{Type: "http", Scheme: "bearer"},
		{Type: "http", Scheme: "basic"},
	}

	for _, lib := range libs {
		for _, sch := range schemes {
			rc := common.DefaultRetryConfig()
			rc.Enabled = true
			data := common.TemplateData{SDKName: "combo_sdk", HTTPLib: lib, HTTPLibImport: lib, RetryConfig: rc, OpenAPIDoc: &common.ExtractedData{SecuritySchemes: map[string]common.SecurityScheme{"s": sch}}}
			if _, err := generateRubyClient(data, common.GetRubyDefaultVersion()); err != nil {
				t.Fatalf("generateRubyClient combo failed for %s/%v: %v", lib, sch, err)
			}
		}
	}
}

func TestGenerateRubySDK_InvalidHTTPLibError(t *testing.T) {
	dir := t.TempDir()
	extracted := &common.ExtractedData{Title: "T"}
	err := generateRubySDKFromExtracted(dir, "X", "badlib", extracted, common.GetRubyDefaultVersion(), "", false, common.DefaultRetryConfig())
	if err == nil || !strings.Contains(err.Error(), "failed to get HTTP library config") {
		t.Fatalf("expected HTTP library config error, got: %v", err)
	}
}

func TestGetMethodName_Root(t *testing.T) {
	op := common.APIOperation{Method: "GET", Path: "/", OperationID: ""}
	name := getMethodName(op)
	if name == "" || !strings.HasPrefix(name, "get") {
		t.Fatalf("expected method name to start with 'get', got %s", name)
	}
}

func TestGenerateRubyClientSpec_NoSecurity(t *testing.T) {
	data := common.TemplateData{}
	out := generateRubyClientSpec(data, "my_sdk")
	if !strings.Contains(out, "RSpec.describe") || !strings.Contains(out, "#initialize") {
		t.Fatalf("client spec missing basic content: %s", out)
	}
}

func TestGenerateRubySpecHelper_Content(t *testing.T) {
	out := generateRubySpecHelper("my_sdk")
	if !strings.Contains(out, "RSpec.configure") {
		t.Fatalf("spec helper missing RSpec.configure: %s", out)
	}
}

func TestGenerateRubyAPIMethod_WithBodyAndParams(t *testing.T) {
	rb := &common.RequestBody{Required: true, Content: map[string]common.ContentType{"application/json": {Schema: &common.Schema{Type: "object"}}}}
	op := common.APIOperation{
		Method:      "POST",
		Path:        "/items/{id}",
		Parameters:  []common.Parameter{{Name: "id", In: "path", Required: true, Schema: &common.Schema{Type: "string"}}, {Name: "q", In: "query", Required: true, Schema: &common.Schema{Type: "string"}}},
		RequestBody: rb,
	}
	out := generateRubyAPIMethod(op, common.TemplateData{})
	if !strings.Contains(out, "body: nil") || !strings.Contains(out, ", body: body") {
		t.Fatalf("API method did not include body parameter or request body usage: %s", out)
	}
}

func TestGenerateRubyModel_NilAndArrayTypes(t *testing.T) {
	if got := generateRubyModel("X", nil, "sdk"); got != "" {
		t.Fatalf("expected empty string for nil schema, got: %s", got)
	}

	arr := &common.Schema{Type: "array", Items: &common.Schema{Type: "integer"}}
	if got := getRubyType(arr); got != "Array<Integer>" {
		t.Fatalf("expected Array<Integer>, got: %s", got)
	}
}

func TestGenerateRubyQualityConfigs_WriteFailure(t *testing.T) {
	dir := t.TempDir()
	// create a directory where .yardopts should be so WriteFile fails
	if err := os.MkdirAll(filepath.Join(dir, ".yardopts"), 0750); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := generateRubyQualityConfigs(dir); err == nil || !strings.Contains(err.Error(), ".yardopts") {
		t.Fatalf("expected error writing .yardopts, got: %v", err)
	}
}

func TestGenerateRubyQualityConfigs_RubocopWriteFailure(t *testing.T) {
	dir := t.TempDir()
	// create directory at .rubocop.yml to cause write failure
	if err := os.MkdirAll(filepath.Join(dir, ".rubocop.yml"), 0750); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := generateRubyQualityConfigs(dir); err == nil || !strings.Contains(err.Error(), ".rubocop.yml") {
		t.Fatalf("expected error writing .rubocop.yml, got: %v", err)
	}
}
