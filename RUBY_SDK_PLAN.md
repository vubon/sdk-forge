# Ruby SDK Generation Implementation Plan

**Feature**: Ruby SDK Generation Support  
**Status**: 🚧 PLANNED  
**Priority**: High  
**Target Version**: v0.7.0  
**Branch**: `feature/ruby-sdk-support`

---

## Overview

Add support for generating production-ready Ruby SDKs from OpenAPI schemas. Ruby SDKs will follow Ruby community conventions, use Bundler for dependency management, and support Ruby 3.0+ with modern Ruby features.

## Goals

1. Generate Ruby SDKs that follow community best practices
2. Support Bundler-based gem structure with proper autoloading
3. Generate type-annotated Ruby 3.0+ code (using RBS/YARD where possible)
4. Support multiple HTTP libraries (Faraday, Net::HTTP, HTTP.rb)
5. Full authentication support (API Key, Bearer, Basic, OAuth2, etc.)
6. Automatic test generation with RSpec
7. Support for retry mechanism (reuse existing retry logic)
8. Generate comprehensive documentation and examples

---

## Requirements

### Functional Requirements

1. **Ruby Version Support**
   - Ruby 3.0+ (minimum)
   - Use modern Ruby features (pattern matching, keyword arguments, etc.)

2. **Package Structure**
   - Bundler-based gem structure
   - Proper module/class organization
   - `gemspec` with dependencies
   - `README.md` with usage examples
   - `CHANGELOG.md` for version history

3. **Code Standards**
   - RuboCop compliance (community Ruby style)
   - YARD doc comments for all public methods
   - Type signatures via RBS (optional)

4. **HTTP Client Libraries**
   - **Faraday** (default)
   - **Net::HTTP** (standard library)
   - **HTTP.rb** (alternative)
   - Support for async requests (if library supports)

5. **Authentication Support**
   - API Key (header, query, cookie)
   - HTTP Basic Authentication
   - HTTP Bearer Token Authentication
   - OAuth2 (all flows)
   - OpenID Connect
   - HTTP Digest Authentication
   - Mutual TLS (mTLS) - if supported by HTTP library

6. **Data Models**
   - Ruby classes with attribute accessors
   - JSON serialization/deserialization
   - Validation support
   - Support for nested objects and arrays
   - Enum support (using Ruby symbols or classes)

7. **API Methods**
   - Organized by OpenAPI tags
   - Method signatures with keyword arguments
   - Request/response models
   - Error handling with custom exceptions
   - Support for query parameters, path parameters, headers

8. **Retry Mechanism**
   - Reuse existing retry configuration structure
   - Support for all retry strategies (exponential, linear, fixed)
   - Configurable via constructor or setter methods
   - Integration with HTTP client libraries

9. **Testing**
   - RSpec test generation
   - Unit tests for models
   - Unit tests for API methods (with mocked HTTP responses)
   - Integration test templates
   - Test fixtures from OpenAPI examples

10. **Documentation**
    - YARD for all classes and methods
    - README.md with installation and usage
    - Examples directory with usage examples
    - API documentation generation (optional)

### Non-Functional Requirements

1. **Performance**
   - Efficient HTTP client usage
   - Minimal memory footprint
   - Fast serialization/deserialization
   - Support for streaming responses (large files)

2. **Usability**
   - Intuitive API design
   - Clear error messages
   - Comprehensive examples
   - Easy installation via Bundler

3. **Maintainability**
   - Clean code structure
   - Follow Ruby best practices
   - Well-documented code
   - Easy to extend

4. **Compatibility**
   - Backward compatible with Ruby 3.0+
   - Works with major Ruby frameworks (Rails, Sinatra, etc.)
   - Compatible with dependency injection containers

---

## Implementation Plan

### Phase 1: Foundation & Structure (Week 1-2)

#### 1.1 Create Ruby Generator Structure
- `internal/generator/ruby.go` - Main Ruby SDK generator
- `internal/generator/ruby_models.go` - Ruby model generation
- `internal/generator/ruby_client.go` - Ruby client generation
- `internal/generator/ruby_api.go` - Ruby API method generation
- `internal/generator/templates/ruby/` - Ruby templates directory

#### 1.2 Ruby Version Management
- Add Ruby version constants and validation in `internal/generator/versions.go`

#### 1.3 HTTP Library Support
- Add Ruby HTTP libraries and configurations in `pkg/languages/http/http.go`

#### 1.4 Template Structure
- Create template files:
  - `internal/generator/templates/ruby/client.rb.tmpl`
  - `internal/generator/templates/ruby/gemspec.tmpl`
  - `internal/generator/templates/ruby/README.md.tmpl`
  - `internal/generator/templates/ruby/models/model.rb.tmpl`
  - `internal/generator/templates/ruby/api/api_method.rb.tmpl`

### Phase 2: Core Generation (Week 3-4)

#### 2.1 Client Class Generation
- HTTP client initialization
- Authentication setup
- Base URL management
- Request/response handling
- Error handling
- Retry mechanism integration

#### 2.2 Model Generation
- Attribute accessors
- Constructor with keyword arguments
- JSON serialization
- Validation methods
- Enum support

#### 2.3 API Method Generation
- Method signatures with keyword arguments
- Parameter validation
- Request/response model usage
- Error handling
- Organized by OpenAPI tags

#### 2.4 Gem Package Generation
- Generate `.gemspec` file
- Generate gem structure

### Phase 3: Authentication & Advanced Features (Week 5)

#### 3.1 Authentication Implementation
- Support all authentication methods
- Example authentication setup in client

#### 3.2 Retry Mechanism Integration
- Reuse existing retry configuration
- Support all retry strategies
- Integrate with HTTP client libraries
- Configurable via constructor options

#### 3.3 Error Handling
- Custom exception classes

### Phase 4: Testing & Quality (Week 6)

#### 4.1 Test Generation
- Generate RSpec tests for models and API methods
- Integration test templates
- Test fixtures from OpenAPI examples

#### 4.2 Code Quality & Formatting
- Integrate RuboCop for code style
- Generate `.rubocop.yml` configuration file
- Automatically format generated Ruby code
- Validate generated code meets standards

### Phase 5: Documentation & Examples (Week 7)

#### 5.1 Documentation Generation
- README.md with installation and usage
- YARD for all classes and methods
- Examples directory with usage examples
- CHANGELOG.md template

#### 5.2 Example Generation
- Generate usage examples

### Phase 6: CLI Integration & Testing (Week 8)

#### 6.1 CLI Command Updates
- Add Ruby to supported languages
- Add `--ruby-version` flag
- Add Ruby HTTP library selection
- Update validation logic

#### 6.2 Integration Testing
- Test Ruby SDK generation with various OpenAPI schemas
- Test all HTTP libraries
- Test all authentication methods
- Test retry mechanism
- Verify RuboCop compliance
- Test gem installation

#### 6.3 Manual Testing
- Create manual test scripts
- Test against mock server
- Verify all features work correctly

---

## Technical Specifications

### File Structure

```
generated-ruby-sdk/
├── my_sdk.gemspec
├── Gemfile
├── README.md
├── CHANGELOG.md
├── lib/
│   ├── my_sdk.rb
│   ├── my_sdk/
│   │   ├── client.rb
│   │   ├── models/
│   │   │   ├── pet.rb
│   │   │   └── ...
│   │   ├── api/
│   │   │   ├── pets_api.rb
│   │   │   └── ...
│   │   └── exceptions/
│   │       └── api_exception.rb
├── spec/
│   ├── models/
│   │   └── pet_spec.rb
│   ├── api/
│   │   └── pets_api_spec.rb
│   └── ...
├── examples/
│   └── basic_usage.rb
├── .rubocop.yml
└── .yardopts
```

### Naming Conventions
- **Classes/Modules**: CamelCase (e.g., `MySdk::Client`, `Pet`)
- **Methods**: snake_case (e.g., `list_pets`, `create_pet`)
- **Constants**: UPPER_SNAKE_CASE
- **Files**: snake_case

### Type System
- Use Ruby 3.0+ type signatures (RBS, optional)
- YARD doc comments for all methods

### Error Handling
- Custom exception hierarchy
- Clear error messages
- HTTP status code mapping
- Response body in exceptions

---

## Testing Strategy

### Unit Tests
- Model serialization/deserialization
- Model validation
- API method parameter handling
- Error handling
- Authentication logic

### Integration Tests
- Full request/response cycle
- Authentication flows
- Error scenarios
- Retry mechanism
- Different HTTP libraries

### Manual Testing
- Generate SDK from real OpenAPI schemas
- Test against mock server
- Verify gem installation
- Test in different Ruby versions
- Test with different frameworks

---

## Documentation Requirements

### Generated Documentation
1. **README.md**
   - Installation instructions
   - Basic usage examples
   - Authentication setup
   - Configuration options
   - Error handling
   - Retry mechanism
2. **YARD**
   - All classes documented
   - All methods documented
   - Parameter descriptions
   - Return type descriptions
   - Exception documentation
3. **Examples**
   - Basic usage
   - Authentication examples
   - Error handling examples
   - Retry mechanism examples

---

## Dependencies

### Required Dependencies
- **Faraday** (default): `faraday >= 2.0`
- **RSpec** (for tests): `rspec >= 3.0`

### Optional Dependencies
- **Net::HTTP** (standard library)
- **HTTP.rb**: `http >= 5.0`

### Development Dependencies
- **RuboCop**: `rubocop >= 1.0`
- **YARD**: `yard >= 0.9`

---

## Migration & Compatibility

### Backward Compatibility
- Ruby 3.0+ required
- Use modern Ruby features
- Clear error messages for unsupported Ruby versions

### Framework Compatibility
- Works standalone (no framework required)
- Compatible with Rails, Sinatra, etc.
- Can be used with dependency injection containers

---

## Success Criteria

1. ✅ Generate valid Ruby SDKs from OpenAPI schemas
2. ✅ Follow Ruby community style (RuboCop compliance)
3. ✅ Gem installs correctly
4. ✅ All authentication methods work
5. ✅ Retry mechanism works correctly
6. ✅ Generated tests pass (RSpec)
7. ✅ Code quality tools pass (RuboCop)
8. ✅ All generated code is automatically formatted
9. ✅ Configuration files are auto-generated (`.rubocop.yml`, `.yardopts`)
10. ✅ Documentation is comprehensive
11. ✅ Examples work correctly
12. ✅ Integration tests pass

---

## Timeline

- **Week 1-2**: Foundation & Structure
- **Week 3-4**: Core Generation
- **Week 5**: Authentication & Advanced Features
- **Week 6**: Testing & Quality
- **Week 7**: Documentation & Examples
- **Week 8**: CLI Integration & Final Testing

**Total Estimated Time**: 8 weeks

---

## Risks & Mitigation

### Risk 1: HTTP Library Compatibility
- **Risk**: Different HTTP libraries have different APIs
- **Mitigation**: Abstract HTTP client interface, test all libraries

### Risk 2: Ruby Version Compatibility
- **Risk**: Ruby 3.0+ features may not be available in older versions
- **Mitigation**: Set minimum Ruby 3.0, test on multiple versions

### Risk 3: Community Standards Compliance
- **Risk**: Generated code may not follow Ruby community style
- **Mitigation**: Use RuboCop, generate compliant code

### Risk 4: Complex OpenAPI Schemas
- **Risk**: Some schemas may have complex structures
- **Mitigation**: Test with various schemas, handle edge cases

---

## Future Enhancements

1. **Async Support**: Full async support with concurrent-ruby
2. **Middleware System**: Customizable request/response middleware
3. **Caching**: Built-in response caching
4. **Rate Limiting**: Automatic rate limit handling
5. **Webhook Support**: Webhook signature verification
6. **GraphQL Support**: Generate GraphQL clients

---

## References
- [RubyGems Guides](https://guides.rubygems.org/)
- [Bundler Documentation](https://bundler.io/)
- [Faraday Documentation](https://lostisland.github.io/faraday/)
- [RSpec Documentation](https://rspec.info/)
- [RuboCop Documentation](https://docs.rubocop.org/)
- [YARD Documentation](https://yardoc.org/)

---

**Last Updated**: December 2025
**Status**: 🚧 PLANNED
