# SDK Forge

A CLI tool written in Go that generates SDKs for multiple programming languages from OpenAPI schemas.

## Status

🚧 **Early Development** - Project structure and planning phase

## Goals

Generate SDKs for:
- Python
- Go (Golang)
- PHP
- JavaScript/TypeScript
- (More languages to come)

## Usage

Generate SDKs one language at a time with your preferred HTTP library:

```bash
# Show help information
sdk-forge help
# or
sdk-forge --help
# or
sdk-forge -h

# Generate Python SDK with default HTTP library (requests) - http-lib is optional
sdk-forge generate \
  --schema ./openapi.yaml \
  --language python \
  --name my-api-sdk \
  --output ./sdks/python

# Generate Python SDK with custom HTTP library (httpx)
sdk-forge generate \
  --schema ./openapi.yaml \
  --language python \
  --http-lib httpx \
  --name my-api-sdk \
  --output ./sdks/python

# Generate Go SDK with default HTTP library (net/http) - http-lib is optional
sdk-forge generate \
  --schema https://api.example.com/openapi.json \
  --lang go \
  --name my-api-client \
  --output ./sdks/go

# Generate Go SDK with custom HTTP library (resty)
sdk-forge generate \
  --schema https://api.example.com/openapi.json \
  --lang go \
  --http-lib resty \
  --name my-api-client \
  --output ./sdks/go
```

### Required Parameters

- `--schema` (required): Path or URL to OpenAPI schema (YAML/JSON)
- `--lang` or `--language` (required): Target language (`python`, `go`, `php`, `js`, `ts`)
- `--name` (required): Name for the generated SDK
- `--output` (required): Output directory for generated SDK

### Optional Parameters

- `--http-lib` (optional): HTTP library to use. If not provided, defaults are used:
  - **Python**: `requests` (default)
  - **Go**: `nethttp` (default - standard library)
  - **PHP**: `guzzle` (default)
  - **JavaScript/TypeScript**: `axios` (default)

### Supported HTTP Libraries

- **Python**: `requests` (default), `httpx`, `aiohttp`, `urllib3`
- **Go**: `nethttp` (default), `resty`, `gentleman`
- **PHP**: `guzzle` (default), `curl`, `httpful`
- **JavaScript/TypeScript**: `axios` (default), `fetch`, `node-fetch`, `ky`

### Authentication

**SDK authentication is automatically generated based on OpenAPI schema definitions - STRICTLY:**

- **Schema-only**: Authentication methods are extracted ONLY from OpenAPI `securitySchemes` and `security` requirements
- **No assumptions**: The SDK will ONLY generate authentication methods that are explicitly defined in your OpenAPI schema
- **Nothing extra**: If an authentication method is not in the schema, it will NOT be generated in the SDK
- **Rule**: What's in the schema = What's in the SDK. Nothing more, nothing less.

**Supported schemes (if defined in schema):**
- API Key (query/header/cookie)
- HTTP Basic, HTTP Bearer, HTTP Digest
- OAuth2 (all flows)
- OpenID Connect
- Mutual TLS (mTLS) - client certificates (OpenAPI 3.1+)
- Custom authentication schemes

**Example:**
If your OpenAPI schema defines API key and Bearer token authentication, the generated SDK will ONLY include those two methods. If OAuth2 is not in the schema, the SDK will NOT include OAuth2 methods.

## Project Structure

```
.
├── cmd/
│   └── cli/          # CLI entry point
├── internal/
│   ├── parser/       # OpenAPI schema parser
│   ├── generator/    # Code generation logic
│   ├── templates/    # Language-specific templates (HTTP-lib agnostic)
│   │   ├── python/   # Uses {{.HttpLib}} variables
│   │   ├── go/       # Uses {{.HttpLib}} variables
│   │   ├── php/      # Uses {{.HttpLib}} variables
│   │   └── js/       # Uses {{.HttpLib}} variables
│   └── deps/         # Dependency file templates
├── pkg/
│   └── languages/    # Language-specific generators
│       └── http/     # HTTP library configuration/mapping
└── examples/         # Example OpenAPI schemas
```

### Architecture Notes

- **Parameterized Templates**: Templates are language-specific but HTTP-library-agnostic
- **HTTP Library as Variable**: Templates use variables like `{{.HttpLib}}` and `{{.HttpLibImport}}`
- **Dependency Management**: HTTP library is automatically added to dependency files (requirements.txt, go.mod, package.json, etc.)
- **Maintainable**: No need to maintain separate templates for each HTTP library combination

## Development

### Building

The project includes a `Makefile` with common development tasks:

```bash
# Show all available targets
make help

# Build the binary
make build

# Install to GOPATH/bin
make install

# Format code
make fmt

# Check code formatting
make fmt-check

# Run linter (golangci-lint)
make lint

# Run tests
make test

# Run tests with coverage
make test-coverage

# Run all checks (formatting + linting)
make check

# Run all checks and build
make all

# Clean build artifacts
make clean
```

### Manual Build

```bash
# Build
go build -o sdk-forge ./cmd/cli

# Run
./sdk-forge --help
```

### Code Quality

The project uses:
- **gofmt** for code formatting
- **golangci-lint** for linting (see `.golangci.yml` for configuration)
- **go vet** for static analysis

Run `make check` to verify code quality before committing.

## Roadmap & Recommendations

This project follows best practices for code generation, developer experience, and maintainability. Key focus areas include:

### Planned Features

- ✅ **Schema-Driven Generation**: Strict adherence to OpenAPI schema definitions
- ✅ **Multi-Language Support**: Python, Go, PHP, JavaScript/TypeScript
- ✅ **HTTP Library Flexibility**: Parameterized templates with default libraries
- 🔄 **Code Quality**: Auto-formatting, linting, and type safety
- 🔄 **Testing Strategy**: Unit tests, integration tests, and golden file testing
- 🔄 **Developer Experience**: Progress indicators, dry-run mode, verbose logging
- 🔄 **Documentation**: Auto-generated READMEs and usage examples
- 🔄 **CI/CD**: Automated testing, building, and releases

### Key Recommendations

1. **Code Quality & Testing**
   - Comprehensive test coverage (>80%)
   - Golden file testing for generated code
   - Language-specific code formatters

2. **Developer Experience**
   - `--dry-run` mode for preview
   - Progress indicators during generation
   - Config file support (`sdk-forge.yaml`)
   - Interactive mode for missing fields

3. **Generated SDK Features**
   - Auto-generated documentation
   - Type safety and validation
   - Retry logic and error handling
   - Pagination helpers

4. **Extensibility**
   - Plugin system for custom languages
   - Template override system
   - Pre/post generation hooks

For detailed recommendations and best practices, see [`../brainstorming/README.md`](../brainstorming/README.md).

## Contributing

Contributions are welcome! Please see the [Contributing Guide](../brainstorming/README.md#recommendations--best-practices) for guidelines.

## License

TBD

